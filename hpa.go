package main

import (
	"github.com/golang/glog"

	"k8s.io/client-go/1.4/rest"
	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/pkg/api"
)

func main() {
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

	pods, err := cs.Pods("kube-system").List(api.ListOptions{})
	if nil != err {
		glog.Errorf("Failed to fetch pods: %#v\n", err)
		return
	}

	glog.Errorf("number of pods in kube-system: %v\n", len(pods.Items))
}
