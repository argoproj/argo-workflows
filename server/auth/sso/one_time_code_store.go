package sso

import (
	"sync"
	"time"

	pkgrand "github.com/argoproj/argo-workflows/v3/util/rand"
)

type OneTimeCodeStore struct {
	codes sync.Map
}

func (s *OneTimeCodeStore) Store(token string) (string, error) {
	code, err := pkgrand.RandString(20) // make it long enough to be hard to guess
	if err != nil {
		return "", err
	}

	s.codes.Store(code, token)
	go func() {
		time.Sleep(10 * time.Second)
		s.codes.Delete(code) // remove after 10 seconds
	}()
	return code, nil
}

func (s *OneTimeCodeStore) Retrieve(code string) (string, bool) {
	val, ok := s.codes.LoadAndDelete(code) // remove on read
	if !ok {
		return "", false
	}
	return val.(string), true
}
