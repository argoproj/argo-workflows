package controller

import (
	"context"
	"strconv"
	"time"

	"errors"

	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

var (
	limitReq, _        = labels.NewRequirement(common.LabelParallelismLimit, selection.Exists, nil)
	nsResyncPeriod     = 5 * time.Minute
	errUnableToExtract = errors.New("was unable to extract limit")
)

type updateFunc = func(string, int)
type resetFunc = func(string)

func (wfc *WorkflowController) newNamespaceInformer(ctx context.Context, kubeclientset kubernetes.Interface) (cache.SharedIndexInformer, error) {

	c := kubeclientset.CoreV1().Namespaces()
	logger := logrus.WithField("scope", "ns_watcher")

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
				updateNS(logger, ns, wfc.throttler.UpdateNamespaceParallelism, wfc.throttler.ResetNamespaceParallelism)
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
				updateNS(logger, ns, wfc.throttler.UpdateNamespaceParallelism, wfc.throttler.ResetNamespaceParallelism)
			},

			DeleteFunc: func(obj interface{}) {
				ns, err := nsFromObj(obj)
				if err != nil {
					return
				}
				deleteNS(logger, ns, wfc.throttler.ResetNamespaceParallelism)
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return informer, nil
}

func deleteNS(log *logrus.Entry, ns *apiv1.Namespace, resetFn resetFunc) {
	log.Infof("resetting the namespace parallelism limits for %s due to deletion event", ns.Name)
	resetFn(ns.Name)
}

func updateNS(log *logrus.Entry, ns *apiv1.Namespace, updateFn updateFunc, resetFn resetFunc) {
	limit, err := extractLimit(ns)
	if errors.Is(err, errUnableToExtract) {
		resetFn(ns.Name)
		log.Infof("removing per-namespace parallelism for %s, reverting to default", ns.Name)
		return
	} else if err != nil {
		log.Errorf("was unable to extract the limit due to: %s", err)
		return
	}
	log.Infof("changing namespace parallelism in %s to %d", ns.Name, limit)
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
