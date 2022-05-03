package cache

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
)

type Cache interface {
	Get(key string) (any, bool)
	Add(key string, value any)
}

type ResourceCache struct {
	ctx    context.Context
	cache  Cache
	client kubernetes.Interface
	v1.ServiceAccountLister
}

func NewResourceCacheWithTimeout(client kubernetes.Interface, ctx context.Context, namespace string, timeout time.Duration) *ResourceCache {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(client, timeout, informers.WithNamespace(namespace))
	cache := &ResourceCache{
		ctx:                  ctx,
		cache:                NewLruTtlCache(timeout, 2000),
		client:               client,
		ServiceAccountLister: informerFactory.Core().V1().ServiceAccounts().Lister(),
	}
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())
	return cache
}

func NewResourceCache(client kubernetes.Interface, ctx context.Context, namespace string) *ResourceCache {
	return NewResourceCacheWithTimeout(client, ctx, namespace, time.Minute*1)
}

func (c *ResourceCache) GetSecret(namespace string, secretName string) (*corev1.Secret, error) {
	cacheKey := c.getSecretCacheKey(namespace, secretName)
	if secret, ok := c.cache.Get(cacheKey); ok {
		if secret, ok := secret.(*corev1.Secret); ok {
			return secret, nil
		}
	}

	secret, err := c.getSecretFromServer(namespace, secretName)
	if err != nil {
		return nil, err
	}

	c.cache.Add(cacheKey, secret)
	return secret, nil
}

func (c *ResourceCache) getSecretFromServer(namespace string, secretName string) (*corev1.Secret, error) {
	return c.client.CoreV1().Secrets(namespace).Get(c.ctx, secretName, metav1.GetOptions{})
}

func (c *ResourceCache) getSecretCacheKey(namespace string, secretName string) string {
	return namespace + ":secret:" + secretName
}
