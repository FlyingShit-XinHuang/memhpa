package informer

import (
	"k8s.io/client-go/1.4/tools/cache"
	"k8s.io/client-go/1.4/pkg/runtime"
	utilruntime "k8s.io/client-go/1.4/pkg/util/runtime"

	"sync"
	"time"
	"k8s.io/client-go/1.4/pkg/util/wait"
)

type Informer struct {
	config Config
	reflector *cache.Reflector
	reflectorMutex sync.RWMutex
}

type Config struct {
	// A queue to store operations of k8s resources
	cache.Queue
	// Watch k8s resources operations with it
	cache.ListerWatcher
	// Process the object popped from Queue
	Process ProcessFunc
	// Watched object type
	ObjectType runtime.Object
	// Period of loop to resync all resouces into Queue
	FullResyncPeriod time.Duration
}

type ProcessFunc func(obj interface{}) error

type ResourceEventHandler interface {
	OnAdd(obj interface{})
	OnUpdate(oldObj, newObj interface{})
	OnDelete(obj interface{})
}

type ResourceEventHandlerFuncs struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(oldObj, newObj interface{})
	DeleteFunc func(obj interface{})
}

func (r ResourceEventHandlerFuncs) OnAdd(obj interface{}) {
	if r.AddFunc != nil {
		r.AddFunc(obj)
	}
}

func (r ResourceEventHandlerFuncs) OnUpdate(oldObj, newObj interface{}) {
	if r.UpdateFunc != nil {
		r.UpdateFunc(oldObj, newObj)
	}
}

func (r ResourceEventHandlerFuncs) OnDelete(obj interface{}) {
	if r.DeleteFunc != nil {
		r.DeleteFunc(obj)
	}
}

func DeletionHandlingMetaNamespaceKeyFunc(obj interface{}) (string, error) {
	if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		return d.Key, nil
	}
	return cache.MetaNamespaceKeyFunc(obj)
}

func NewInformer(
	lw cache.ListerWatcher,
	objType runtime.Object,
	resyncPeriod time.Duration,
	h ResourceEventHandler,
) (cache.Store, *Informer) {
	// clientState is a cache to store k8s resources
	clientState := cache.NewStore(DeletionHandlingMetaNamespaceKeyFunc)
	// fifo is a delta queue to store operations of k8s resources
	fifo := cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil, clientState)

	cfg := &Config{
		Queue: fifo,
		ListerWatcher: lw,
		ObjectType: objType,
		FullResyncPeriod: resyncPeriod,
		Process: func(obj interface{}) error {
			for _, d := range obj.(cache.Deltas) {
				switch d.Type {
				case cache.Sync, cache.Added, cache.Updated:
					if old, exists, err := clientState.Get(d.Object); nil == err && exists {
						if err := clientState.Update(d.Object); nil != err {
							return err
						}
						h.OnUpdate(old, d.Object)
					} else {
						if err := clientState.Add(d.Object); nil != err {
							return err
						}
						h.OnAdd(d.Object)
					}
				case cache.Deleted:
					if err := clientState.Delete(d.Object); nil != err {
						return err
					}
					h.OnDelete(d.Object)
				}
			}
			return nil
		},
	}
	return clientState, &Informer{
		config: *cfg,
	}
}

func (i *Informer) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	reflector := cache.NewReflector(i.config.ListerWatcher, i.config.ObjectType, i.config.Queue,
		i.config.FullResyncPeriod)

	i.reflectorMutex.Lock()
	i.reflector = reflector
	i.reflectorMutex.Unlock()

	// Run a reflector to watch resources operations and enqueue
	reflector.RunUntil(stopCh)
	// Run a loop to pop a object from Queue and call Process
	wait.Until(
		func() {
			i.config.Queue.Pop(cache.PopProcessFunc(i.config.Process))
		},
		time.Second,
		stopCh,
	)
}