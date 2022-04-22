package resource

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/lru"
)

type Interface interface {
	GetSecret(ctx context.Context, name, key string) (string, error)
	GetConfigMapKey(ctx context.Context, name, key string) (string, error)
}

// New creates a new instance of Interface. This is intended not to live very long, e.g. the length of one service request.
func New(kubeClient kubernetes.Interface, namespace string) Interface {
	return &cache{lru.New(1024), &impl{kubeClient, namespace}}
}
