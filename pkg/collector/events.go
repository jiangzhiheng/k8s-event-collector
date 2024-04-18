package collector

import (
	"fmt"
	"github.com/jiangzhiheng/k8s-event-collector/pkg/elasticsearch"
	"github.com/jiangzhiheng/k8s-event-collector/pkg/options"
	v1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	coreV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

const (
	workNum       = 5
	IndexNameBase = "k8s-event-collector-%s"
)

var (
	IndexName = fmt.Sprintf(IndexNameBase, time.Now().Format("2006-01-02"))
)

type EventCollector struct {
	kc                kubernetes.Interface
	factory           informers.SharedInformerFactory
	eventLister       coreV1.EventLister
	eventListerSynced cache.InformerSynced
	queue             workqueue.RateLimitingInterface
	locker            sync.Mutex
	esClient          *elasticsearch.ESClient
}

func NewEventCollector(client kubernetes.Interface, factor informers.SharedInformerFactory, o *options.Options) *EventCollector {
	event := factor.Core().V1().Events()
	esClient, err := elasticsearch.NewES(&elasticsearch.ESConfig{
		Hosts:    o.ESEndpoint,
		Username: o.ESUsername,
		Password: o.ESPassword,
	})
	if err != nil {
		klog.Errorf("Init elastic connect failed")
	}
	// init index template
	elasticsearch.InitIndexTemplate(esClient.Client)
	// init ilm
	elasticsearch.InitIndexILMPolicy(esClient.Client)

	eventCollector := &EventCollector{
		kc:                client,
		factory:           factor,
		eventLister:       event.Lister(),
		eventListerSynced: event.Informer().HasSynced,
		queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		locker:            sync.Mutex{},
		esClient:          esClient,
	}
	event.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			eventCollector.enqueueEvent(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*v1api.Event)
			oldObj := old.(*v1api.Event)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			eventCollector.enqueueEvent(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			eventCollector.enqueueEvent(obj)
		},
	})

	return eventCollector
}

func (ec *EventCollector) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer ec.queue.ShutDown()
	// 创建 es 索引
	ec.esClient.CreateIndex(IndexName)

	klog.Info("starting eventCollector")
	if ok := cache.WaitForCacheSync(stopCh, ec.eventListerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	klog.Info("started eventCollector")

	for i := 0; i <= workNum; i++ {
		go wait.Until(ec.Worker, time.Minute, stopCh)
	}
	<-stopCh
	klog.Info("shutting down")
	return nil
}

func (ec *EventCollector) enqueueEvent(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %+v:%v", obj, err))
		return
	}
	ec.queue.Add(key)
}

func (ec *EventCollector) Worker() {
	for ec.processNextItem() {
	}
}

func (ec *EventCollector) processNextItem() bool {
	key, quit := ec.queue.Get()
	if quit {
		return false
	}

	err := ec.syncEventToES(key.(string))
	if err != nil {
		ec.queue.Forget(key)
		return true
	}
	ec.queue.AddRateLimited(key)
	return true
}

func (ec *EventCollector) syncEventToES(key string) interface{} {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	event, err := ec.eventLister.Events(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("event %s has been deleted", key)
			return nil
		}
		klog.Infof("get event failed: %v", err)
	}
	klog.Infof(
		"event name: %s,count: %d,involvedObject_namespace: %s,involvedObject_kind: %s,involvedObject_name: %s,reason: %s,type: %s, Msg: %s, Event time:%s",
		event.Name,
		event.Count,
		event.InvolvedObject.Namespace,
		event.InvolvedObject.Kind,
		event.InvolvedObject.Name,
		event.Reason,
		event.Type,
		event.Message,
		event.LastTimestamp,
	)
	ec.esClient.SyncEventItem(event, IndexName)

	return nil
}
