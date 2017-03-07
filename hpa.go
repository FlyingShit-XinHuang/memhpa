package main

import (
	"github.com/golang/glog"

	"k8s.io/client-go/1.4/rest"
	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/pkg/api"

	"flag"

	"memhpa/app"
	"memhpa/client"
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
}
