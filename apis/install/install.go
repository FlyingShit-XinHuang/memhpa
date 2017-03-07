package install

import (
	"fmt"

	"memhpa/apis/v1"
	
	"github.com/golang/glog"
	
	"k8s.io/client-go/1.4/pkg/api/meta"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/runtime"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/apimachinery"
	"k8s.io/client-go/1.4/pkg/apimachinery/registered"
	"k8s.io/client-go/1.4/pkg/util/sets"
)

var accessor = meta.NewAccessor()

var availableVersions = []unversioned.GroupVersion{ v1.SchemeGroupVersion }

func init() {
	registered.RegisterVersions(availableVersions)
	externalVersions := []unversioned.GroupVersion{}
	for _, v := range availableVersions {
		if registered.IsAllowedVersion(v) {
			externalVersions = append(externalVersions, v)
		}
	}
	if len(externalVersions) == 0 {
		glog.Infof("No version is registered for group %v", v1.MemHPAResourcesGroup)
		return
	}

	if err := registered.EnableVersions(externalVersions...); nil != err {
		glog.Warningf("Cannot enable versions %v: %#v\n", externalVersions, err)
		return
	}
	if err := enableVersions(externalVersions); nil != err {
		glog.Warningf("Cannot enable versions %v: %#v\n", externalVersions, err)
		return
	}
	glog.Infof("Installed versions: %v\n", externalVersions)
}

func enableVersions(externalVersions []unversioned.GroupVersion) error {
	addVersionsToScheme()
	preferredExternalVersion := externalVersions[0]
	
	groupMeta := apimachinery.GroupMeta{
		GroupVersion:  preferredExternalVersion,
		GroupVersions: externalVersions,
		RESTMapper:    newRESTMapper(externalVersions),
		SelfLinker:    runtime.SelfLinker(accessor),
		InterfacesFor: interfacesFor,
	}

	if err := registered.RegisterGroup(groupMeta); nil != err{
		return err
	}
	api.RegisterRESTMapper(groupMeta.RESTMapper)
	return nil
}

func newRESTMapper(externalVersions []unversioned.GroupVersion) meta.RESTMapper {
	return api.NewDefaultRESTMapper(externalVersions, interfacesFor,
		"memhpa/apis", sets.NewString(), sets.NewString())
}

func interfacesFor(version unversioned.GroupVersion) (*meta.VersionInterfaces, error) {
	switch version {
	case v1.SchemeGroupVersion:
		return &meta.VersionInterfaces{
			ObjectConvertor: api.Scheme,
			MetadataAccessor: accessor,
		}, nil
	default:
		g, _ := registered.Group(v1.MemHPAResourcesGroup)
		return nil, fmt.Errorf("unsupported storage version: %s (valid: %v)", version, g.GroupVersions)
	}
}

func addVersionsToScheme() {
	if err := v1.AddToScheme(api.Scheme); err != nil {
		panic(err)
	}
}

