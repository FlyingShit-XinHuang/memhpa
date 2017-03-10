package metrics

import (
	"time"
	"fmt"
	"context"

	"github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"

	"github.com/golang/glog"
)

type PodResourceInfo map[string]int64

type MetricsClient interface {
	GetMemMetric(refNamespace, refName string) (PodResourceInfo, time.Time, error)
}

type InClusterPromClient struct {
	queryAPI prometheus.QueryAPI
}

// Get new client to access Prometheus with specified scheme, namsespace, name and port of Prometheus Service in k8s cluster
func NewInClusterPromClient(scheme, svcNamespace, svcName string, port int) (MetricsClient, error) {
	promConf := prometheus.Config{
		Address: fmt.Sprintf("%s://%s.%s:%d", scheme, svcName, svcNamespace, port),
	}
	client, err := prometheus.New(promConf)
	if nil != err {
		glog.Errorf("Failed to init client of Prometheus: %#v\n", client)
		return nil, err
	}
	return &InClusterPromClient{
		prometheus.NewQueryAPI(client),
	}, nil
}

func (c *InClusterPromClient) GetMemMetric(refNamespace, refName string) (PodResourceInfo, time.Time, error) {
	query := fmt.Sprintf(
		`avg_over_time(
			container_memory_usage_bytes{
				namespace="%s",
				pod_name=~"%s-.*",
				image!~".*/pause-amd64.*"
			}[1m]
		)`, refNamespace, refName)
	result, err := c.queryAPI.Query(context.Background(), query, time.Now())
	if nil != err {
		glog.Errorf("Failed to query Prometheus: %#v\n", err)
		return nil, time.Time{}, err
	}

	info := PodResourceInfo{}
	switch result.Type() {
	case model.ValVector:
		vector := result.(model.Vector)
		if 1 > len(vector) {
			return nil, time.Time{}, fmt.Errorf("No metrics was returned from Prometheus")
		}
		for _, s := range vector  {
			glog.V(2).Infof("Memory usage of container %s of pod %s: %v\n",
				s.Metric["container_name"], s.Metric["pod_name"], s.Value)
			// sum up memory of all containers of each pod
			info[string(s.Metric["pod_name"])] += int64(s.Value)
		}
		return info, vector[0].Timestamp.Time(), nil
	default:
		glog.Errorf("Error metrics type: %v\n", result.Type())
		return nil, time.Time{}, fmt.Errorf("Unexpected metrics type was returned")
	}
}