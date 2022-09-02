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

type ResourceCache struct {
	cache  Interface
	client kubernetes.Interface
	v1.ServiceAccountLister
	informerFactory informers.SharedInformerFactory
}

func NewResourceCacheWithTimeout(client kubernetes.Interface, namespace string, timeout time.Duration) *ResourceCache {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(client, time.Minute*20, informers.WithNamespace(namespace))
	cache := &ResourceCache{
		cache:                NewLRUTtlCache(timeout, 2000),
		client:               client,
		ServiceAccountLister: informerFactory.Core().V1().ServiceAccounts().Lister(),
		informerFactory:      informerFactory,
	}
	return cache
}

func NewResourceCache(client kubernetes.Interface, namespace string) *ResourceCache {
	return NewResourceCacheWithTimeout(client, namespace, time.Minute*1)
}

func (c *ResourceCache) Run(stopCh <-chan struct{}) {
	c.informerFactory.Start(stopCh)
	c.informerFactory.WaitForCacheSync(stopCh)
}

func (c *ResourceCache) GetSecret(ctx context.Context, namespace string, secretName string) (*corev1.Secret, error) {
	cacheKey := c.getSecretCacheKey(namespace, secretName)
	if secret, ok := c.cache.Get(cacheKey); ok {
		if secret, ok := secret.(*corev1.Secret); ok {
			return secret, nil
		}
	}

	secret, err := c.getSecretFromServer(ctx, namespace, secretName)
	if err != nil {
		return nil, err
	}

	c.cache.Add(cacheKey, secret)
	return secret, nil
}

func (c *ResourceCache) getSecretFromServer(ctx context.Context, namespace string, secretName string) (*corev1.Secret, error) {
	return c.client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
}

func (c *ResourceCache) getSecretCacheKey(namespace string, secretName string) string {
	return namespace + ":secret:" + secretName
}
