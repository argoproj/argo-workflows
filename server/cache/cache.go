package cache

import (
	"context"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
)

type ResourceCache struct {
	ctx    context.Context
	client kubernetes.Interface
	v1.ServiceAccountLister
}

func NewResourceCache(client kubernetes.Interface, ctx context.Context, namespace string) *ResourceCache {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(client, time.Minute*20, informers.WithNamespace(namespace))
	cache := &ResourceCache{
		ctx:                  ctx,
		client:               client,
		ServiceAccountLister: informerFactory.Core().V1().ServiceAccounts().Lister(),
	}
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())
	return cache
}

func (c *ResourceCache) GetSecret(namespace string, secretName string) (*v12.Secret, error) {
	options := metav1.GetOptions{}
	return c.client.CoreV1().Secrets(namespace).Get(c.ctx, secretName, options)
}
