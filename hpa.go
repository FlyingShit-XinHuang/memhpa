package main

import (
	"github.com/golang/glog"

	"github.com/prometheus/client_golang/api/prometheus"
	//"github.com/prometheus/common/model"

	"k8s.io/client-go/1.4/rest"
	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/pkg/api"

	"flag"
	"fmt"
	"context"
	"time"

	"memhpa/app"
	"memhpa/client"
	_ "memhpa/apis/install" // register custom resources group
	"memhpa/controller"
	"memhpa/controller/metrics"
	"github.com/prometheus/common/model"
)

func main() {
	flag.Parse()

	config, err := rest.InClusterConfig()
	if nil != err {
		glog.Errorf("Failed to get in-cluster config: %#v\n", err)
		return
	}

	cs, err := kubernetes.NewForConfig(config)
	if nil != err {
		glog.Errorf("Failed to init client set: %#v\n", err)
		return
	}

	// // test client set
	//pod, err := cs.Pods("kube-system").Get("kube-dns-2543703245-84jns")
	//if nil != err {
	//	glog.Errorf("Failed to fetch pods: %#v\n", err)
	//	return
	//}
	//glog.Infof("container %s of pod memory limit bytes: %d\n",
	//	pod.Spec.Containers[0].Name,
	//	pod.Spec.Containers[0].Resources.Limits[api.ResourceMemory].Value(),
	//)


	if err := app.CreateMemHPAResourceGroup(cs.Extensions()); nil != err {
		glog.Errorf("Failed to create mem-hpa resource: %#v\n", err)
		return
	}

	scaleClient, err := client.NewForConfig(config)
	if nil != err {
		glog.Errorf("Failed to init hpa scale client: %#v\n", err)
		return
	}

	scalers, err := scaleClient.Scalers("huangxin").List(api.ListOptions{})
	if nil != err {
		glog.Errorf("Failed to get scalers: %#v\n", err)
		return
	}
	glog.Infof("Numbers of hpa scalers in namespace huangxin: %d\n", len(scalers.Items))

	promSvcName := "prometheus-monitor"
	namespace := "kube-system"
	port := 9090
	addr := fmt.Sprintf("http://%s.%s:%s", promSvcName, namespace, port)
	promConf := prometheus.Config{
		Address: addr,
	}
	promClient, err := prometheus.New(promConf)
	if nil != err {
		glog.Errorf("Failed to init Prometheus client: %#v\n", err)
		return
	}

	podControllerNamespace := "kube-system"
	podControllerName := "kube-dns"
	query := fmt.Sprintf(
		`avg_over_time(
			container_memory_usage_bytes{
				namespace="%s",
				pod_name=~"%s-.*",
				image!~".*/pause-amd64.*"
			}[1m]
		)`, podControllerNamespace, podControllerName)
	value, err := prometheus.NewQueryAPI(promClient).Query(context.Background(), query, time.Now())
	if nil != err {
		glog.Errorf("Failed to query Prometheus: %#v\n", err)
		return
	}
	switch value.Type() {
	case model.ValVector:
		vector := value.(model.Vector)
		for _, s := range vector  {
			glog.Infof("Memory usage of container %s of pod %s: %v\n", s.Metric["container_name"], s.Metric["pod_name"], s.Value)
		}
	default:
		glog.Errorf("Error result type: %v\n", value.Type())
	}

	metricsClient, err := metrics.NewInClusterPromClient("http", namespace, promSvcName, port)
	if nil != err {
		glog.Errorf("Failed to init metrics client: %#v\n", err)
		return
	}

	controller.NewHPAController(cs.Core(), cs.Extensions(), scaleClient, controller.NewReplicaCalculator(metricsClient, cs.Core()), time.Minute)
}
