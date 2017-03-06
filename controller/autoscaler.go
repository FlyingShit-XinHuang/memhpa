package controller

//import (
//	"time"
//)

//type HorizontalController struct {
//	scaleNamespacer unversionedextensions.ScalesGetter
//	hpaNamespacer   unversionedautoscaling.HorizontalPodAutoscalersGetter
//
//	replicaCalc   *ReplicaCalculator
//	eventRecorder record.EventRecorder
//
//	// A store of HPA objects, populated by the controller.
//	store cache.Store
//	// Watches changes to all HPA objects.
//	controller *cache.Controller
//}
//
//func NewHorizontalController(evtNamespacer v1core.EventsGetter, scaleNamespacer unversionedextensions.ScalesGetter,
//	hpaNamespacer hpaclient.HorizontalPodAutoscalersGetter, replicaCalc *ReplicaCalculator,
//	resyncPeriod time.Duration) *HorizontalController {
//
//}
