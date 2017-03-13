package app

import (
	"net/http"

	"memhpa/apis/v1"

	"github.com/golang/glog"

	"k8s.io/client-go/1.4/kubernetes/typed/extensions/v1beta1"
	v1beta1types "k8s.io/client-go/1.4/pkg/apis/extensions/v1beta1"
	v1types "k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/api/errors"
)

const customResourceName = "mem-hpa.tenxcloud.com"

// Create custom resource if doesn't exist
func CreateMemHPAResourceGroup(getter v1beta1.ThirdPartyResourcesGetter) error {
	_, err := getter.ThirdPartyResources().Get(v1.MemHPAResourcesMetaName)
	if nil != err {
		if k8sErr, ok := err.(*errors.StatusError); ok && http.StatusNotFound == k8sErr.ErrStatus.Code {
			if _, err := getter.ThirdPartyResources().Create(newCustomResource()); nil != err {
				glog.Errorf("Failed to create custom mem-hpa resources: %#v\n", err)
				return err
			}
			glog.Infoln("Succeed to create custom mem-hpa resources")
			return nil
		}
		glog.Errorf("Failed to get custom mem-hpa resources: %#v\n", err)
		return err
	}
	glog.Infoln("Custom mem-hpa resources already exists")
	return nil
}

func CreateMemHPAResourceGroupOrDie(getter v1beta1.ThirdPartyResourcesGetter) {
	if err := CreateMemHPAResourceGroup(getter); nil != err {
		panic(err)
	}
}

func newCustomResource() *v1beta1types.ThirdPartyResource {
	return &v1beta1types.ThirdPartyResource{
		ObjectMeta: v1types.ObjectMeta{
			Name: v1.MemHPAResourcesMetaName,
		},
		Description: "Resources for controlling autoscale through memory limit",
		Versions: []v1beta1types.APIVersion{
			v1beta1types.APIVersion{v1.MemHPAResourcesVersion},
		},
	}
}