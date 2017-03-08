package main

import (
	"github.com/golang/glog"

	"github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"

	"k8s.io/client-go/1.4/rest"
	"k8s.io/client-go/1.4/kubernetes"
	//"k8s.io/client-go/1.4/pkg/api"

	"flag"
	"fmt"
	"context"
	"time"

	"memhpa/app"
	//"memhpa/client"
	_ "memhpa/apis/install" // register custom resources group
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
	//pods, err := cs.Pods("kube-system").List(api.ListOptions{})
	//if nil != err {
	//	glog.Errorf("Failed to fetch pods: %#v\n", err)
	//	return
	//}
	//
	//glog.Infof("number of pods in kube-system: %v\n", len(pods.Items))

	if err := app.CreateMemHPAResourceGroup(cs.Extensions()); nil != err {
		glog.Errorf("Failed to create mem-hpa resource: %#v\n", err)
		return
	}

	//scaleClient, err := client.NewForConfig(config)
	//if nil != err {
	//	glog.Errorf("Failed to init hpa scale client: %#v\n", err)
	//	return
	//}
	//
	//scalers, err := scaleClient.Scalers("huangxin").List(api.ListOptions{})
	//if nil != err {
	//	glog.Errorf("Failed to get scalers: %#v\n", err)
	//	return
	//}
	//glog.Infof("Numbers of hpa scalers in namespace huangxin: %d\n", len(scalers.Items))

	promSvcName := "prometheus-monitor"
	namespace := "kube-system"
	port := "9090"
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
}
