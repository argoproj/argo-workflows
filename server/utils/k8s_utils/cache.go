package k8s_utils

import (
	"context"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
)

type K8sCache struct {
	informers.SharedInformerFactory
	v1.ServiceAccountLister
	v1.SecretLister
}

func NewK8sCache(client kubernetes.Interface, ctx context.Context) *K8sCache {
	informerFactory := informers.NewSharedInformerFactory(client, time.Minute*20)
	cache := &K8sCache{
		SharedInformerFactory: informerFactory,
		ServiceAccountLister:  informerFactory.Core().V1().ServiceAccounts().Lister(),
		SecretLister:          informerFactory.Core().V1().Secrets().Lister(),
	}
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())
	return cache
}
