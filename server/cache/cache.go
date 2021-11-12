package cache

import (
	"context"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
)

type ResourceCache struct {
	v1.ServiceAccountLister
	v1.SecretLister
}

func NewResourceCache(client kubernetes.Interface, ctx context.Context, namespace string) *ResourceCache {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(client, time.Minute*20, informers.WithNamespace(namespace))
	cache := &ResourceCache{
		ServiceAccountLister: informerFactory.Core().V1().ServiceAccounts().Lister(),
		SecretLister:         informerFactory.Core().V1().Secrets().Lister(),
	}
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())
	return cache
}
