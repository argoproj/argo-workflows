package sessions

import (
	"testing"
)

const redisTestServer = "localhost:6379"

var newRedisStore = func(_ *testing.T) Store {
	store, err := NewRedisStore(10, "tcp", redisTestServer, "", []byte("secret"))
	if err != nil {
		panic(err)
	}
	return store
}

func TestRedis_SessionGetSet(t *testing.T) {
	sessionGetSet(t, newRedisStore)
}

func TestRedis_SessionDeleteKey(t *testing.T) {
	sessionDeleteKey(t, newRedisStore)
}

func TestRedis_SessionFlashes(t *testing.T) {
	sessionFlashes(t, newRedisStore)
}

func TestRedis_SessionClear(t *testing.T) {
	sessionClear(t, newRedisStore)
}

func TestRedis_SessionOptions(t *testing.T) {
	sessionOptions(t, newRedisStore)
}
