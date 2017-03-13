package controller

import (
	"time"
	"fmt"
	"math"

	"memhpa/client"
	memhpav1 "memhpa/apis/v1"
	"memhpa/controller/informer"

	"k8s.io/client-go/1.4/kubernetes/typed/extensions/v1beta1"
	"k8s.io/client-go/1.4/tools/record"
	"k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	apiv1 "k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/tools/cache"
	"k8s.io/client-go/1.4/pkg/runtime"
	"k8s.io/client-go/1.4/pkg/watch"
	apisv1beta1 "k8s.io/client-go/1.4/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	utilruntime "k8s.io/client-go/1.4/pkg/util/runtime"

	"github.com/golang/glog"
)

const (
	scaleUpLimitMinimum = 4
	scaleUpLimitFactor = 2

        upscaleForbiddenWindow = 3 * time.Minute
	downscaleForbiddenWindow = 5 * time.Minute
)

type HPAController struct {
	scaleNamespacer v1beta1.ScalesGetter
	hpaNamespacer   client.MemHPAScalersGetter

	replicaCalc   *ReplicaCalculator
	eventRecorder record.EventRecorder

	// A store of HPA objects, populated by the controller.
	store cache.Store
	// Watches changes to all HPA objects.
	informer *informer.Informer
}

func NewHPAController(evtNamespacer v1.EventsGetter, scaleNamespacer v1beta1.ScalesGetter,
	hpaNamespacer client.MemHPAScalersGetter, replicaCalc *ReplicaCalculator,
	resyncPeriod time.Duration) *HPAController {

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v1.EventSinkImpl{Interface:evtNamespacer.Events("")})


	hpaController := &HPAController{
		scaleNamespacer: scaleNamespacer,
		hpaNamespacer: hpaNamespacer,
		replicaCalc: replicaCalc,
		eventRecorder: broadcaster.NewRecorder(apiv1.EventSource{Component:"custom-mem-hpa-controller"}),
	}

	hpaController.newInformer(resyncPeriod)

	return hpaController
}

func (controller *HPAController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Infof("Starting HPA Controller")
	go controller.informer.Run(stopCh)
	<-stopCh
	glog.Infof("Shutting down HPA Controller")
}

func (controller *HPAController) newInformer(resyncPeriod time.Duration) {
	controller.store, controller.informer = informer.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return controller.hpaNamespacer.Scalers(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return controller.hpaNamespacer.Scalers(api.NamespaceAll).Watch(options)
			},
		},
		&memhpav1.MemHpa{},
		resyncPeriod,
		informer.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				hpa := obj.(*memhpav1.MemHpa)
				controller.reconcile(hpa)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				hpa := newObj.(*memhpav1.MemHpa)
				controller.reconcile(hpa)
			},
		},
	)
}

func (controller *HPAController) reconcile(hpa *memhpav1.MemHpa) {
	modified := controller.validate(hpa)
	reference := fmt.Sprintf("%s/%s(%s)", hpa.Spec.ScaleTargetRef.Name, hpa.MetaData.Namespace,
		hpa.Spec.ScaleTargetRef.Kind)

	// get scale subresource
	scale, err := controller.scaleNamespacer.Scales(hpa.MetaData.Namespace).Get(hpa.Spec.ScaleTargetRef.Kind,
		hpa.Spec.ScaleTargetRef.Name)
	if nil != err {
		// if validate() modified the hpa, the resource should be updated
		if modified {
			controller.updateStatus(hpa, hpa.Status.CurrentReplicas, hpa.Status.DesiredReplicas,
				hpa.Status.CurrentUtilizationPercentage, false)
		}
		glog.Errorf("Failed to get scale subresource of %s: %v\n", reference, err)
		return
	}

	currentReplicas := scale.Status.Replicas
	desiredReplicas := int32(0)
	rescale := true
	rescaleReason := ""
	timestamp := time.Now()
	utilization := int32(0)

	if 0 == scale.Spec.Replicas {
		rescale = false
	} else if currentReplicas > hpa.Spec.MaxReplicas {
		desiredReplicas = hpa.Spec.MaxReplicas
		rescaleReason = "Current number is greater than .spec.maxReplicas"
	} else if currentReplicas < *hpa.Spec.MinReplicas {
		desiredReplicas = *hpa.Spec.MinReplicas
		rescaleReason = "Current number is less than .spec.minReplicas"
	} else {
		// calculate desired replicas
		desiredReplicas, utilization, timestamp, err = controller.computeReplicas(hpa, scale)
		if nil != err {
			controller.updateStatus(hpa, currentReplicas, hpa.Status.DesiredReplicas,
				hpa.Status.CurrentUtilizationPercentage, false)
			glog.Errorf("Failed to calculate desired replicas of %s: %v\n", reference, err)
			return
		}

		if desiredReplicas > currentReplicas {
			rescaleReason = "Utilization is greater than target"
		} else if desiredReplicas < currentReplicas {
			rescaleReason = "Utilization is less than target"
		}

		if desiredReplicas < *hpa.Spec.MinReplicas {
			desiredReplicas = *hpa.Spec.MinReplicas
		}
		if desiredReplicas > hpa.Spec.MaxReplicas {
			desiredReplicas = hpa.Spec.MaxReplicas
		}
		scaleUpLimit := getScaleUpLimit(currentReplicas)
		if desiredReplicas > scaleUpLimit {
			desiredReplicas = scaleUpLimit
		}

		// check whether it should be scaled
		rescale = shouldScale(hpa, currentReplicas, desiredReplicas, timestamp)
	}

	if rescale {
		// update scale subresource to scale
		scale.Spec.Replicas = desiredReplicas
		if _, err := controller.scaleNamespacer.Scales(hpa.MetaData.Namespace).
			Update(hpa.Spec.ScaleTargetRef.Kind, scale); nil != err {

			controller.eventRecorder.Eventf(hpa, api.EventTypeWarning, "FailedRescale",
				"New size: %d; reason: %s; error: %v", desiredReplicas, rescaleReason, err)
			glog.Errorf("Failed to scale: %v\n", err)
			return
		}
		controller.eventRecorder.Eventf(hpa, api.EventTypeNormal, "SuccessfulRescale", "" +
			"New size: %d; reason: %s", desiredReplicas, rescaleReason)
		glog.Infof("Successfull rescale of %s, old size: %d, new size: %d, reason: %s",
			hpa.MetaData.Name, currentReplicas, desiredReplicas, rescaleReason)
	} else {
		desiredReplicas = currentReplicas
	}

	// update mem hpa
	controller.updateStatus(hpa, currentReplicas, desiredReplicas, utilization, rescale)
}

func shouldScale(hpa *memhpav1.MemHpa, current, desired int32, timestamp time.Time) bool {
	if desired == current {
		return false
	}

	if hpa.Status.LastScaleTime == nil {
		return true
	}

	// Do not rescale too often
	if desired < current && hpa.Status.LastScaleTime.Time.Add(downscaleForbiddenWindow).Before(timestamp) {
		return true
	}
	if desired > current && hpa.Status.LastScaleTime.Time.Add(upscaleForbiddenWindow).Before(timestamp) {
		return true
	}

	if glog.V(2) {
		glog.Infof("desired: %d, current: %d, interval: %v\n", desired, current, timestamp.Sub(hpa.Status.LastScaleTime.Time))
	}
	return false
}

func getScaleUpLimit(currentReplicas int32) int32 {
	return int32(math.Max(scaleUpLimitFactor * float64(currentReplicas), scaleUpLimitMinimum))
}

func (controller *HPAController) updateStatus(hpa *memhpav1.MemHpa, current, desired, utilization int32, rescale bool) {
	hpa.Status = memhpav1.MemHPAScalerStatus{
		CurrentReplicas: current,
		DesiredReplicas: desired,
		CurrentUtilizationPercentage: utilization,
		LastScaleTime: hpa.Status.LastScaleTime,
	}

	if rescale {
		*hpa.Status.LastScaleTime = unversioned.NewTime(time.Now())
	}

	if _, err := controller.hpaNamespacer.Scalers(hpa.MetaData.Namespace).Update(hpa); nil != err {
		controller.eventRecorder.Event(hpa, api.EventTypeWarning, "FailedUpdateStatus", err.Error())
		glog.Errorf("Failed to update mem hpa status: %#v\n", err)
		return
	}
	glog.V(2).Infof("Successfully updated status for %s\n", hpa.MetaData.Name)
}

func (controller *HPAController) computeReplicas(hpa *memhpav1.MemHpa, scale *apisv1beta1.Scale) (int32, int32, time.Time, error) {
	targetUtilization := *hpa.Spec.TargetUtilizationPercentage
	currentReplicas := scale.Status.Replicas
	nilTime := time.Time{}

	if scale.Status.Selector == nil {
		err := "selector is required"
		controller.eventRecorder.Event(hpa, api.EventTypeWarning, "SelectorRequired", err)
		return 0, 0, nilTime, fmt.Errorf(err)
	}

	selector, err := unversioned.LabelSelectorAsSelector(&unversioned.LabelSelector{MatchLabels:scale.Status.Selector})
	if err != nil {
		errMsg := fmt.Sprintf("couldn't convert selector string to a corresponding selector object: %v", err)
		controller.eventRecorder.Event(hpa, api.EventTypeWarning, "InvalidSelector", errMsg)
		return 0, 0, nilTime, fmt.Errorf(errMsg)
	}

	desiredReplicas, utilization, timestamp, err :=
		controller.replicaCalc.GetReplicas(currentReplicas, targetUtilization, hpa.MetaData.Namespace,
			hpa.Spec.ScaleTargetRef.Name, selector)
	if nil != err {
		lastScaleTime := getLastScaleTime(hpa)
		if time.Now().After(lastScaleTime.Add(upscaleForbiddenWindow)) {
			controller.eventRecorder.Event(hpa, api.EventTypeWarning, "FailedGetMetrics", err.Error())
		} else {
			controller.eventRecorder.Event(hpa, api.EventTypeNormal, "MetricsNotAvailableYet", err.Error())
		}

		return 0, 0, nilTime, fmt.Errorf("failed to get memory utilization: %v", err)
	}

	if desiredReplicas != currentReplicas {
		controller.eventRecorder.Eventf(hpa, api.EventTypeNormal, "DesiredReplicasComputed",
			"Computed the desired num of replicas: %d (avgUtil: %d, current replicas: %d)",
			desiredReplicas, utilization, currentReplicas)
	}

	return desiredReplicas, utilization, timestamp, nil
}

func getLastScaleTime(hpa *memhpav1.MemHpa) time.Time {
	lastTime := hpa.Status.LastScaleTime
	if nil == lastTime {
		lastTime = &hpa.MetaData.CreationTimestamp
	}
	return lastTime.Time
}

// Validate fields and update invalid fields,
// return whether it is valid
func (controller *HPAController) validate(hpa *memhpav1.MemHpa) bool {
	var modified bool
	if nil == hpa.Spec.MinReplicas || *hpa.Spec.MinReplicas < 1 {
		if nil == hpa.Spec.MinReplicas {
			hpa.Spec.MinReplicas = new(int32)
		}
		*hpa.Spec.MinReplicas = 1
		controller.eventRecorder.Event(hpa, apiv1.EventTypeNormal, "ValidationPolicy",
			".spec.minReplicas is invalid and will be set to 1")
		modified = true
	}

	if hpa.Spec.MaxReplicas < *hpa.Spec.MinReplicas {
		hpa.Spec.MaxReplicas = *hpa.Spec.MinReplicas + scaleUpLimitMinimum
		controller.eventRecorder.Event(hpa, apiv1.EventTypeNormal, "ValidationPolicy",
			fmt.Sprintf(".spec.maxReplicas is invalid and will be set to %d", hpa.Spec.MaxReplicas ))
		modified = true
	}
	if nil == hpa.Spec.TargetUtilizationPercentage || *hpa.Spec.TargetUtilizationPercentage < 1 ||
		*hpa.Spec.TargetUtilizationPercentage > 100 {

		if nil == hpa.Spec.TargetUtilizationPercentage{
			hpa.Spec.TargetUtilizationPercentage = new(int32)
		}
		*hpa.Spec.TargetUtilizationPercentage = 80
		controller.eventRecorder.Event(hpa, apiv1.EventTypeNormal, "ValidationPolicy",
			".spec.targetUtilizationPercentage is invalid and will be set to 80")
		modified = true

	}
	return !modified
}