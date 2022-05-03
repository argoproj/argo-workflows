package cache

type Cache interface {
	Get(key string) (any, bool)
	Add(key string, value any)
}
