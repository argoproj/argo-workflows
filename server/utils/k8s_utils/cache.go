package k8s_utils

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/core/v1"
)

type K8sCache struct {
	informers.SharedInformerFactory
	v1.ServiceAccountLister
}

func NewK8sCache(client kubernetes.Interface) *K8sCache {
	informerFactory := informers.NewSharedInformerFactory(client, time.Minute*20)
	cache := &K8sCache{
		SharedInformerFactory: informerFactory,
		ServiceAccountLister:  informerFactory.Core().V1().ServiceAccounts().Lister(),
	}
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
	return cache
}
