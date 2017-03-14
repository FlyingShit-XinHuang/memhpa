package v1

import (
	"k8s.io/client-go/1.4/pkg/api/v1"
	autoscaling "k8s.io/client-go/1.4/pkg/apis/autoscaling/v1"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/api/meta"
	"encoding/json"
)

const (
	MemHPAResourcesGroup = "xinhuang.com"
	MemHPAResourcesName = "memhpas"
	MemHPAResourcesVersion = "v1"
	MemHPAResourcesMetaName = "mem-hpa.xinhuang.com"
)

type MemHpa struct {
	unversioned.TypeMeta `json:",inline"`
	// There is a bug when using 3rd party resources: https://github.com/kubernetes/client-go/issues/8
	// so ObjectMeta was combined not embedded
	MetaData v1.ObjectMeta `json:"metadata,omitempty"`
	Spec MemHPASpec `json:"spec,omitempty"`
	Status MemHPAScalerStatus `json:"status,omitempty"`
}

type MemHPASpec struct {
	ScaleTargetRef autoscaling.CrossVersionObjectReference `json:"scaleTargetRef"`
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	MaxReplicas int32 `json:"maxReplicas"`
	TargetUtilizationPercentage *int32 `json:"targetUtilizationPercentage,omitempty"`
}

type MemHPAScalerStatus struct {
	ObservedGeneration *int64 `json:"observedGeneration,omitempty"`
	LastScaleTime *unversioned.Time `json:"lastScaleTime,omitempty"`
	CurrentReplicas int32 `json:"currentReplicas"`
	DesiredReplicas int32 `json:"desiredReplicas"`
	CurrentUtilizationPercentage int32 `json:"currentCPUUtilizationPercentage"`
}

type MemHpaList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items []MemHpa `json:"items"`
}

// Implement runtime.Object interface
func (m *MemHpa) GetObjectKind() unversioned.ObjectKind {
	return &m.TypeMeta
}

// Implement meta.ObjectMetaAccessor interface
func (m *MemHpa) GetObjectMeta() meta.Object {
	return &m.MetaData
}

// Workaround for decoding 3rd party resource.
// Define a copy type so that the call of json.Unmarshal cannot cause an endless loop
type MemHpaCopy MemHpa

func (m *MemHpa) UnmarshalJSON(data []byte) error {
	tmp := MemHpaCopy{}
	if err := json.Unmarshal(data, &tmp); nil != err {
		return err
	}
	*m = MemHpa(tmp)
	return nil
}