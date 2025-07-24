package controller

import (
	"context"
	"errors"
	"strconv"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	authutil "github.com/argoproj/argo-workflows/v3/util/auth"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

var (
	limitReq, _        = labels.NewRequirement(common.LabelParallelismLimit, selection.Exists, nil)
	nsResyncPeriod     = 5 * time.Minute
	errUnableToExtract = errors.New("was unable to extract limit")
)

type (
	updateFunc = func(string, int)
	resetFunc  = func(string)
)

func (wfc *WorkflowController) newNamespaceInformer(ctx context.Context, kubeclientset kubernetes.Interface) (cache.SharedIndexInformer, error) {
	log := logging.GetLoggerFromContext(ctx)
	log = log.WithField("component", "ns_watcher")
	ctx = logging.WithLogger(ctx, log)
	can, _ := authutil.CanI(ctx, wfc.kubeclientset, []string{"get", "watch", "list"}, "", metav1.NamespaceAll, "namespaces")
	if !can {
		log.Warn(ctx, "was unable to get permissions for get/watch/list verbs on the namespace resource, per-namespace parallelism will not work")
	}
	c := kubeclientset.CoreV1().Namespaces()

	labelSelector := labels.NewSelector().
		Add(*limitReq)

	listFunc := func(opts metav1.ListOptions) (runtime.Object, error) {
		opts.LabelSelector = labelSelector.String()
		return c.List(ctx, opts)
	}

	watchFunc := func(opts metav1.ListOptions) (watch.Interface, error) {
		opts.Watch = true
		opts.LabelSelector = labelSelector.String()
		return c.Watch(ctx, opts)
	}

	source := &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
	informer := cache.NewSharedIndexInformer(source, &apiv1.Namespace{}, nsResyncPeriod, cache.Indexers{})

	_, err := informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ns, err := nsFromObj(obj)
				if err != nil {
					return
				}
				updateNS(ctx, ns, wfc.throttler.UpdateNamespaceParallelism, wfc.throttler.ResetNamespaceParallelism)
			},

			UpdateFunc: func(old, newVal interface{}) {
				ns, err := nsFromObj(newVal)
				if err != nil {
					return
				}
				oldNs, err := nsFromObj(old)
				if err == nil && !limitChanged(oldNs, ns) {
					return
				}
				updateNS(ctx, ns, wfc.throttler.UpdateNamespaceParallelism, wfc.throttler.ResetNamespaceParallelism)
			},

			DeleteFunc: func(obj interface{}) {
				ns, err := nsFromObj(obj)
				if err != nil {
					return
				}
				deleteNS(ctx, ns, wfc.throttler.ResetNamespaceParallelism)
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return informer, nil
}

func deleteNS(ctx context.Context, ns *apiv1.Namespace, resetFn resetFunc) {
	log := logging.GetLoggerFromContext(ctx)
	log.Infof(ctx, "reseting the namespace parallelism limits for %s due to deletion event", ns.Name)
	resetFn(ns.Name)
}

func updateNS(ctx context.Context, ns *apiv1.Namespace, updateFn updateFunc, resetFn resetFunc) {
	log := logging.GetLoggerFromContext(ctx)
	limit, err := extractLimit(ns)
	if errors.Is(err, errUnableToExtract) {
		resetFn(ns.Name)
		log.Infof(ctx, "removing per-namespace parallelism for %s, reverting to default", ns.Name)
		return
	} else if err != nil {
		log.Errorf(ctx, "was unable to extract the limit due to: %s", err)
		return
	}
	log.Infof(ctx, "changing namespace parallelism in %s to %d", ns.Name, limit)
	updateFn(ns.Name, limit)
}

func nsFromObj(obj interface{}) (*apiv1.Namespace, error) {
	ns, ok := obj.(*apiv1.Namespace)
	if !ok {
		return nil, errors.New("was unable to convert to namespace")
	}
	return ns, nil
}

func limitChanged(old *apiv1.Namespace, newNS *apiv1.Namespace) bool {
	oldLimit := old.GetLabels()[common.LabelParallelismLimit]
	newLimit := newNS.GetLabels()[common.LabelParallelismLimit]
	return oldLimit != newLimit
}

func extractLimit(ns *apiv1.Namespace) (int, error) {
	labels := ns.GetLabels()
	var limitString *string

	for lbl, value := range labels {
		if lbl == common.LabelParallelismLimit {
			limitString = &value
			break
		}
	}
	if limitString == nil {
		return 0, errUnableToExtract
	}

	integerValue, err := strconv.Atoi(*limitString)
	if err != nil {
		return 0, err
	}
	return integerValue, nil
}
