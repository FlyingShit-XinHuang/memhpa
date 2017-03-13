package v1

import (
	"k8s.io/client-go/1.4/pkg/api/v1"
	autoscaling "k8s.io/client-go/1.4/pkg/apis/autoscaling/v1"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/api/meta"
	"encoding/json"
)

const (
	MemHPAResourcesGroup = "tenxcloud.com"
	MemHPAResourcesName = "memhpas"
	MemHPAResourcesVersion = "v1"
	MemHPAResourcesMetaName = "mem-hpa.tenxcloud.com"
)

type MemHpa struct {
	unversioned.TypeMeta `json:",inline"`
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

func (m *MemHpa) GetObjectKind() unversioned.ObjectKind {
	return &m.TypeMeta
}

func (m *MemHpa) GetObjectMeta() meta.Object {
	return &m.MetaData
}

type MemHpaCopy MemHpa

func (m *MemHpa) UnmarshalJSON(data []byte) error {
	tmp := MemHpaCopy{}
	if err := json.Unmarshal(data, &tmp); nil != err {
		return err
	}
	*m = MemHpa(tmp)
	return nil
}