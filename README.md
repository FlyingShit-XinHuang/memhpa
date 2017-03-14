# Horizontal pod autoscaler through memory

## Design proposals

### Query K8S API

This HPA is designed to run in a pod (in "kube-system" namespace) in K8S cluster. The service account will be used to 
access K8S API.

### Pull metrics

Memory metrics are pulled from Prometheus which should be deployed in the cluster and expose its service with K8S Service.
Some parameters can be used to specify Prometheus Service:

```
  -prom-name string
        Name of Prometheus service (default "prometheus")
  -prom-namespace string
        Namespace of Prometheus service (default "kube-system")
  -prom-port int
        Port of Prometheus service (default 9090)
  -prom-scheme string
        Scheme of Prometheus service (default "http")
```

### HPA resources

A [3rd party resource](https://kubernetes.io/docs/user-guide/thirdpartyresources/) is created to define the 
memory-based HPA resource. It was defined similarly with K8S HorizontalPodAutoscaler:

```go
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
```

The client package in the project can be used to query the MemHpa resource

### Autoscaling Algorithm

It is similar with [K8S Horizontal Pod Autoscaling](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/horizontal-pod-autoscaler.md).

K8S list and watch API are used to watch modifications of MemHpa resources. Rescaling maybe triggered by one of 
following conditions:

* A MemHpa resource was created or modified
* or every 30 seconds

.spec.scaleTargetRef is used to fetch Pods and Scale subresource of the referenced pod controller. Pods are used to 
calculate sum of memory limits by which sum of metrics is divided to get utilization. 

## How to run

### Build

Docker must be installed in your environment. Then just run:
 
```
make docker-build
```

Then the image will be built. If you want to specify another image name:

```
make IMAGE=your-image-name docker-build
```

Run the following command to build and push image:

```
make push
```

### Run in K8S

You can use [deployment-in-cluster.yaml](k8s-compose/demo/deployment-in-cluster.yaml) to run this memory-based HPA 
controller in a K8S Deployment and create a MemHpa resource with [memhpa-demo.yaml](k8s-compose/demo/memhpa-demo.yaml)
to reference your pod controller
