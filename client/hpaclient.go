package client

import (
//	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/rest"
)

type ScalingClient struct {
	*rest.RESTClient
}

func New(c *rest.RESTClient) *ScalingClient {
	return &ScalingClient{c}
}

func (c *ScalingClient) Scalers(namespace string) MemHPAScalerInterface {
	return newMemHPAScalers(c, namespace)
}