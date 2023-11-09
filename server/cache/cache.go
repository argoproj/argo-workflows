package cache

type Interface interface {
	Get(key string) (any, bool)
	Add(key string, value any)
}
