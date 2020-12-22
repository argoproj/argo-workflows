package resource

import "context"

// TODO JPZ13 12/21/2020 add description
type Interface interface {
	GetSecret(name, key string) (string, error)
	GetConfigMapKey(ctx context.Context, name, key string) (string, error)
}
