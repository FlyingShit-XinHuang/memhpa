package controller

import (
	"time"

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
	"fmt"
)

const (
	scaleUpLimitMinimum = 4
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
	// update mem-hpa
	controller.hpaNamespacer.Scalers(hpa.Namespace).Update(hpa)
	return !modified
}