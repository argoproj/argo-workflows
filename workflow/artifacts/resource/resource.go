package resource

import "context"

type Interface interface {
	GetSecret(ctx context.Context, name, key string) (string, error)
	GetConfigMapKey(ctx context.Context, name, key string) (string, error)
}
