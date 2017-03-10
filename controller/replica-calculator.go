package controller

import (
	"memhpa/controller/metrics"

	"k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/labels"
	apiv1 "k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/util/sets"

	"github.com/golang/glog"

	"time"
	"fmt"
	"math"
)

const tolerance = 0.1

type ReplicaCalculator struct {
	metricsClient metrics.MetricsClient
	podsGetter v1.PodsGetter
}

func NewReplicaCalculator(mc metrics.MetricsClient, pg v1.PodsGetter) *ReplicaCalculator {
	return &ReplicaCalculator{metricsClient: mc, podsGetter: pg}
}

func (r *ReplicaCalculator) GetReplicas(currentReplicas int32, targetUtilization int32, namespace, name string,
	selector labels.Selector) (int32, int32, time.Time, error) {

	nilTime := time.Time{}
	metrics, time, err := r.metricsClient.GetMemMetric(namespace, name)
	if nil != err {
		glog.Errorf("Failed to get memory metrics: %#v\n", err)
		return 0, 0, nilTime, fmt.Errorf("Get metrics error")
	}

	podsList, err := r.podsGetter.Pods(namespace).List(api.ListOptions{LabelSelector: selector})
	if nil != err {
		glog.Errorf("Failed to list pods: %#v\n", err)
		return 0, 0, nilTime, fmt.Errorf("List pods error")
	}

	if 1 > len(podsList.Items) {
		return 0, 0, nilTime, fmt.Errorf("No pods found")
	}

	limits := make(map[string]int64, len(podsList.Items))
	// Because metrics could contain pods of other pod controllers with similar name,
	// so validMetrics is used to filter metrics of invalid pods.
	validMetrics := make(map[string]int64)
	unreadyPods := sets.NewString()
	missingPods := sets.NewString() // pods without metrics

	for _, p := range podsList.Items {
		var sum int64
		for _, c := range p.Spec.Containers {
			limit, found := c.Resources.Limits[apiv1.ResourceMemory]
			if !found {
				return 0, 0, nilTime, fmt.Errorf("Memory limit was not set of container %s", c.Name)
			}
			sum += limit.Value()
		}
		limits[p.Name] = sum

		// remove metrics of pods that are not running
		if p.Status.Phase != apiv1.PodRunning || !isPodReady(&p) {
			unreadyPods.Insert(p.Name)
			continue
		}

		m, found := metrics[p.Name]
		if !found {
			missingPods.Insert(p.Name)
			continue
		}
		validMetrics[p.Name] = m
	}

	if 1 > len(validMetrics) {
		return 0, 0, nilTime, fmt.Errorf("No valid metrics found")
	}
	ratio, utilization, validCount := getRatioAndUtilization(limits, validMetrics, targetUtilization)

	rebalanceUnready := unreadyPods.Len() > 0 && ratio > 1.0
	if !rebalanceUnready && missingPods.Len() == 0 {
		glog.V(2).Infoln("There is no need to rebalance")
		if isChangeSmall(ratio) {
			return currentReplicas, utilization, time, nil
		}
		// calculate desired replicas
		return calculateReplicas(ratio, validCount), utilization, time, nil
	}

	if missingPods.Len() > 0 {
		glog.V(2).Infof("Missing metrics of pods %v\n", missingPods)
		// if some metrics are missed
		if ratio > 1.0 {
			for name := range missingPods{
				// set metrics 0 to see whether it should still be scaled up
				validMetrics[name] = 0
			}
		} else if ratio < 1.0 {
			for name := range missingPods {
				// set metrics the limit setting to see whether it should still be scaled down
				validMetrics[name] = limits[name]
			}
		}
	}

	if rebalanceUnready {
		glog.V(2).Infof("Rebalance pods %v\n", unreadyPods)
		for name := range unreadyPods {
			// set metrics to be 0 to see whether it should still be scaled up
			metrics[name] = 0
		}
	}

	rebalancedRatio, _, validCount := getRatioAndUtilization(limits, validMetrics, targetUtilization)
	if isChangeSmall(rebalancedRatio) || (ratio > 1.0 && rebalancedRatio < 1.0) ||
		(ratio < 1.0 && rebalancedRatio > 1.0) {

		// return current replicas if change is still small or scale direction is changed after rebalance
		return currentReplicas, utilization, time, nil
	}
	return calculateReplicas(rebalancedRatio, validCount), utilization, time, nil
}

func calculateReplicas(ratio float64, replicas int32) int32 {
	return int32(math.Ceil(ratio * float64(replicas)))
}

func isChangeSmall(ratio float64) bool {
	// 0.9 <= ratio <= 1.1
	// It means the change would be too small
	return math.Abs(1.0 - ratio) <= tolerance
}

func isPodReady(pod *apiv1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == apiv1.PodReady && c.Status == apiv1.ConditionTrue {
			return true
		}
	}
	return false
}

func getRatioAndUtilization(limits, metrics map[string]int64, target int32) (float64, int32, int32) {
	var limitsTotal, metricsTotal int64
	var validCount int32
	for name, m := range metrics {
		l, found := limits[name]
		if !found {
			// filter value which not in limits
			continue
		}
		limitsTotal += l
		metricsTotal += m
		validCount++
	}

	utilization := int32((metricsTotal * 100) / limitsTotal)
	return float64(utilization) / float64(target), utilization, validCount
}