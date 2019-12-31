package resource

type Interface interface {
	GetSecret(name, key string) (string, error)
	GetConfigMapKey(name, key string) (string, error)
}
