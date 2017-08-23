package sessions

import (
	"testing"
)

var newCookieStore = func(_ *testing.T) Store {
	store := NewCookieStore([]byte("secret"))
	return store
}

func TestCookie_SessionGetSet(t *testing.T) {
	sessionGetSet(t, newCookieStore)
}

func TestCookie_SessionDeleteKey(t *testing.T) {
	sessionDeleteKey(t, newCookieStore)
}

func TestCookie_SessionFlashes(t *testing.T) {
	sessionFlashes(t, newCookieStore)
}

func TestCookie_SessionClear(t *testing.T) {
	sessionClear(t, newCookieStore)
}

func TestCookie_SessionOptions(t *testing.T) {
	sessionOptions(t, newCookieStore)
}
