package client

import (
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/watch"

	"memhpa/apis/v1"
)

const (
	resourceName = "memhpa"
)

type MemHPAScalersGetter interface {
	Scalers(namespace string) MemHPAScalerInterface
}

type MemHPAScalerInterface interface {
	Create(scaler *v1.MemHPAScaler) (*v1.MemHPAScaler, error)
	Update(scaler *v1.MemHPAScaler) (*v1.MemHPAScaler, error)
	Delete(name string, options *api.DeleteOptions) error
	Get(name string) (*v1.MemHPAScaler, error)
	List(opts api.ListOptions) (*v1.MemHPAScalerList, error)
	Watch(opts api.ListOptions) (watch.Interface, error)
}

type memHPAScalers struct {
	client *ScalingClient
	ns string
}

func newMemHPAScalers(c *ScalingClient, namespace string) *memHPAScalers {
	return &memHPAScalers{
		client: c,
		ns: namespace,
	}
}

func (s *memHPAScalers) Create(scaler *v1.MemHPAScaler) (*v1.MemHPAScaler, error) {
	result := &v1.MemHPAScaler{}
	err := s.client.Post().
		Namespace(s.ns).
		Resource(resourceName).
		Body(scaler).
		Do().
		Into(result)
	return result, err
}

func (s *memHPAScalers) Update(scaler *v1.MemHPAScaler) (*v1.MemHPAScaler, error) {
	result := &v1.MemHPAScaler{}
	err := s.client.Put().
		Namespace(s.ns).
		Resource(resourceName).
		Name(scaler.Name).
		Body(scaler).
		Do().
		Into(result)
	return result, err
}

func (s *memHPAScalers) Delete(name string, options *api.DeleteOptions) error {
	return s.client.Delete().
		Namespace(s.ns).
		Resource(resourceName).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (s *memHPAScalers) Get(name string) (*v1.MemHPAScaler, error) {
	result := &v1.MemHPAScaler{}
	err := s.client.Get().
		Namespace(s.ns).
		Resource(resourceName).
		Name(name).
		Do().
		Into(result)
	return result, err
}