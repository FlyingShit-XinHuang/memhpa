package v1

import (
	"k8s.io/client-go/1.4/pkg/runtime"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/api"
)

var (
	builder = runtime.NewSchemeBuilder(addKnownTypes, addDefaultingFuncs)
	AddToScheme = builder.AddToScheme
	SchemeGroupVersion = unversioned.GroupVersion{
		Group: MemHPAResourcesGroup,
		Version: MemHPAResourcesVersion,
	}
)


func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&MemHpa{},
		&MemHpaList{},
		&api.ListOptions{},
		&api.DeleteOptions{},
	)
	return nil
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return scheme.AddDefaultingFuncs(
		func(obj *MemHpa) {
			if obj.Spec.MinReplicas == nil {
				minReplicas := int32(1)
				obj.Spec.MinReplicas = &minReplicas
			}
			if obj.Spec.TargetUtilizationPercentage == nil {
				percentage := int32(80)
				obj.Spec.TargetUtilizationPercentage = &percentage
			}
		},
	)
}