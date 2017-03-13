package main

import (
	"github.com/golang/glog"

	"k8s.io/client-go/1.4/rest"
	"k8s.io/client-go/1.4/kubernetes"

	"flag"
	"time"

	"memhpa/app"
	"memhpa/client"
	"memhpa/controller"
	"memhpa/controller/metrics"
)

var (
	stopCh chan struct{}

	promSvcScheme string
	promSvcNamespace string
	promSvcName string
	promSvcPort int
)

func init() {
	stopCh = make(chan struct{})

	flag.StringVar(&promSvcScheme, "prom-scheme", "http", "Scheme of Prometheus service")
	flag.StringVar(&promSvcNamespace, "prom-namespace", "kube-system",
		"Namespace of Prometheus service")
	flag.StringVar(&promSvcName, "prom-name", "prometheus","Name of Prometheus service")
	flag.IntVar(&promSvcPort, "prom-port", 9090,"Port of Prometheus service")
}

func main() {
	flag.Parse()

	// get in-cluster config
	config, err := rest.InClusterConfig()
	if nil != err {
		glog.Errorf("Failed to get in-cluster config: %#v\n", err)
		panic(err)
	}

	// get client set to query k8s resources
	cs := kubernetes.NewForConfigOrDie(config)

	// create custom resources
	app.CreateMemHPAResourceGroupOrDie(cs.Extensions())

	// get client to query custom resources
	scaleClient := client.NewForConfigOrDie(config)

	//list, _ := scaleClient.Scalers("huangxin").List(api.ListOptions{})
	//glog.Infof("list mem hpa: %#v\n", list)
	//hpalist, _ := cs.Autoscaling().HorizontalPodAutoscalers("zaizai").List(api.ListOptions{})
	//glog.Infof("list hpa: %#v\n", hpalist)
	//h, _ := scaleClient.Scalers("huangxin").Get("hpatest")

	// get client to query Prometheus
	metricsClient := metrics.NewInClusterPromClientOrDie(promSvcScheme, promSvcNamespace, promSvcName, promSvcPort)

	// create controller
	hpaController := controller.NewHPAController(cs.Core(), cs.Extensions(), scaleClient,
		controller.NewReplicaCalculator(metricsClient, cs.Core()), time.Second * 30)

	// run controller
	hpaController.Run(stopCh)
}
