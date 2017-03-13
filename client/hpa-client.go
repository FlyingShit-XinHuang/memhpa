package client

import (
	"k8s.io/client-go/1.4/rest"
	"k8s.io/client-go/1.4/pkg/apimachinery/registered"
	"k8s.io/client-go/1.4/pkg/runtime/serializer"
	"k8s.io/client-go/1.4/pkg/api"

	"memhpa/apis/v1"
	_ "memhpa/apis/install" // register custom resources group

	"github.com/golang/glog"
)

type ScalingClient struct {
	*rest.RESTClient
}

func NewForConfig(c *rest.Config) (*ScalingClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ScalingClient{client}, nil
}

func NewForConfigOrDie(c *rest.Config) *ScalingClient {
	client, err := NewForConfig(c)
	if nil != err {
		glog.Errorf("Failed to init scaling client: %#v\n", err)
		panic(err)
	}
	return client
}

func (c *ScalingClient) Scalers(namespace string) MemHPAScalerInterface {
	return newMemHPAScalers(c, namespace)
}

func setConfigDefaults(config *rest.Config) error {
	// if extensions group is not registered, return an error
	g, err := registered.Group(v1.MemHPAResourcesGroup)
	if err != nil {
		return err
	}
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	// TODO: Unconditionally set the config.Version, until we fix the config.
	//if config.Version == "" {
	copyGroupVersion := g.GroupVersion
	config.GroupVersion = &copyGroupVersion
	//}

	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	return nil
}