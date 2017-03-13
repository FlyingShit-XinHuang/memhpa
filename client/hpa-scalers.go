package client

import (
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/watch"

	"memhpa/apis/v1"
)

type MemHPAScalersGetter interface {
	Scalers(namespace string) MemHPAScalerInterface
}

type MemHPAScalerInterface interface {
	Create(scaler *v1.MemHpa) (*v1.MemHpa, error)
	Update(scaler *v1.MemHpa) (*v1.MemHpa, error)
	Delete(name string, options *api.DeleteOptions) error
	Get(name string) (*v1.MemHpa, error)
	List(opts api.ListOptions) (*v1.MemHpaList, error)
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

func (s *memHPAScalers) Create(scaler *v1.MemHpa) (*v1.MemHpa, error) {
	result := &v1.MemHpa{}
	err := s.client.Post().
		Namespace(s.ns).
		Resource(v1.MemHPAResourcesName).
		Body(scaler).
		Do().
		Into(result)
	return result, err
}

func (s *memHPAScalers) Update(scaler *v1.MemHpa) (*v1.MemHpa, error) {
	result := &v1.MemHpa{}
	err := s.client.Put().
		Namespace(s.ns).
		Resource(v1.MemHPAResourcesName).
		Name(scaler.MetaData.Name).
		Body(scaler).
		Do().
		Into(result)
	return result, err
}

func (s *memHPAScalers) Delete(name string, options *api.DeleteOptions) error {
	return s.client.Delete().
		Namespace(s.ns).
		Resource(v1.MemHPAResourcesName).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (s *memHPAScalers) Get(name string) (*v1.MemHpa, error) {
	result := &v1.MemHpa{}
	err := s.client.Get().
		Namespace(s.ns).
		Resource(v1.MemHPAResourcesName).
		Name(name).
		Do().
		Into(result)
	return result, err
}

func (s *memHPAScalers) List(opts api.ListOptions) (*v1.MemHpaList, error)  {
	result := &v1.MemHpaList{}
	err := s.client.Get().
		Namespace(s.ns).
		Resource(v1.MemHPAResourcesName).
		VersionedParams(&opts, api.ParameterCodec).
		Do().
		Into(result)
	return result, err
}

func (s *memHPAScalers) Watch(opts api.ListOptions) (watch.Interface, error) {
	return s.client.Get().
		Prefix("watch").
		Namespace(s.ns).
		Resource(v1.MemHPAResourcesName).
		VersionedParams(&opts, api.ParameterCodec).
		Watch()
}