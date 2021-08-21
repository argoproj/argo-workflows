package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/events"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
)

type Controller struct {
	namespace            string
	kubeclientset        kubernetes.Interface
	cacheQueue           workqueue.RateLimitingInterface
	cacheController      cache.Controller
	cacheLister          cache.Store
	eventRecorderManager events.EventRecorderManager
}

const (
	cacheResyncPeriod = 20 * time.Minute
	cacheWorkers      = 8
)

func NewController(namespace string, kubeclientset kubernetes.Interface, metrics *metrics.Metrics, eventRecorderManager events.EventRecorderManager) *Controller {
	return &Controller{
		namespace:            namespace,
		kubeclientset:        kubeclientset,
		cacheQueue:           metrics.RateLimiterWithBusyWorkers(workqueue.DefaultControllerRateLimiter(), "cache_queue"),
		eventRecorderManager: eventRecorderManager,
	}
}

func (cc *Controller) Run(ctx context.Context) {
	defer cc.cacheQueue.ShutDown()
	log.Infof("Starting cache controller")
	restClient := cc.kubeclientset.CoreV1().RESTClient()
	resource := "configmaps"
	labelSelector, _ := labels.Parse(common.LabelKeyCacheGCAfterNotHitDuration)
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = labelSelector.String()
		req := restClient.Get().
			Namespace(cc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do(ctx).Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.LabelSelector = labelSelector.String()
		req := restClient.Get().
			Namespace(cc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch(ctx)
	}
	source := &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
	cc.cacheLister, cc.cacheController = cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		cacheResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					cc.cacheQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					cc.cacheQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					cc.cacheQueue.Add(key)
				}
			},
		})
	log.Info("Watching config map updates")
	go cc.cacheController.Run(ctx.Done())
	go wait.UntilWithContext(ctx, cc.syncAll, 10*time.Second)

	for i := 0; i < cacheWorkers; i++ {
		go wait.Until(cc.runCacheWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
}

func (cc *Controller) runCacheWorker() {
	for cc.processNextCacheItem() {
	}
}

func (cc *Controller) processNextCacheItem() bool {
	key, quit := cc.cacheQueue.Get()
	if quit {
		return false
	}
	defer cc.cacheQueue.Done(key)
	logCtx := log.WithField("cache", key)
	logCtx.Infof("Processing %s", key)
	return true
}

//nolint:unparam
func (cc *Controller) syncAll(ctx context.Context) {
	log.Info("Syncing all caches")

	caches := cc.cacheLister.List()

	for _, obj := range caches {
		cm, ok := obj.(*apiv1.ConfigMap)
		if !ok {
			log.Error("Unable to convert object to configmap when syncing ConfigMaps")
			continue
		}

		err := cc.syncConfigMap(cm)
		if err != nil {
			cc.eventRecorderManager.Get(cm.GetNamespace()).Event(cm, apiv1.EventTypeWarning, "SyncFailed", err.Error())
			log.WithError(err).Errorf("Unable to sync ConfigMap: %s", cm.Name)
			continue
		}
	}
}

func (cc *Controller) syncConfigMap(cm *apiv1.ConfigMap) error {
	log.Infof("Syncing ConfigMap: %s", cm.Name)
	if gcAfterNotHitDuration := cm.Labels[common.LabelKeyCacheGCAfterNotHitDuration]; gcAfterNotHitDuration != "" {
		gcAfterNotHitDurationTime, err := time.ParseDuration(gcAfterNotHitDuration)
		if err != nil {
			return err
		}

		var modified bool
		for key, rawEntry := range cm.Data {
			var entry Entry
			err = json.Unmarshal([]byte(rawEntry), &entry)
			if err != nil {
				return fmt.Errorf("malformed cache entry: could not unmarshal JSON; unable to parse: %w", err)
			}
			if time.Since(entry.LastHitTimestamp.Time) > gcAfterNotHitDurationTime {
				log.Infof("Deleting entry with key %s in ConfigMap %s since it's not been hit after %s", key, cm.Name, gcAfterNotHitDuration)
				delete(cm.Data, key)
				modified = true
			}
		}
		selfLink := fmt.Sprintf("api/v1/namespaces/%s/configmaps/%s",
			cm.Namespace, cm.Name)
		if len(cm.Data) == 0 {
			log.Infof("Deleting ConfigMap %s since it doesn't contain any cache entries", cm.Name)
			request := cc.kubeclientset.CoreV1().RESTClient().Delete().RequestURI(selfLink)
			stream, err := request.Stream(context.TODO())
			if err != nil {
				return fmt.Errorf("failed to delete ConfigMap %s: %w", cm.Name, err)
			}
			defer func() { _ = stream.Close() }()
		} else {
			if modified {
				log.Infof("Modified ConfigMap: %s: %v", cm.Name, cm)
				request := cc.kubeclientset.CoreV1().RESTClient().Put().RequestURI(selfLink).Body(cm)
				stream, err := request.Stream(context.TODO())
				if err != nil {
					return fmt.Errorf("failed to patch ConfigMap %s: %w", cm.Name, err)
				}
				defer func() { _ = stream.Close() }()
			}
		}
	}

	return nil
}
