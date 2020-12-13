package apiserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/ticker"
	"github.com/stretchr/testify/assert"
)

type fakeTicker struct {
	c          chan time.Time
	resetCalls int
}

func (ft *fakeTicker) Stop() {
}

func (ft *fakeTicker) Reset(time.Duration) {
	ft.resetCalls++
}

func (ft *fakeTicker) C() <-chan time.Time {
	return ft.c
}

func (ft *fakeTicker) tick() {
	ft.c <- time.Now()
}

func newFakeTicker(time.Duration) *fakeTicker {
	return &fakeTicker{
		c:          make(chan time.Time, 1),
		resetCalls: 0,
	}
}

func Test_serverSentEventKeepaliveMiddleware(t *testing.T) {
	rr := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "/api/workflows", nil)
	req.Header["Accept"] = []string{"text/event-stream"}

	if !assert.Nil(t, err) {
		return
	}

	var wrapped http.ResponseWriter

	handler := func(rw http.ResponseWriter, r *http.Request) {
		wrapped = rw
	}

	ft := newFakeTicker(time.Second * 1)

	var wg sync.WaitGroup

	mw := serverSentEventKeepaliveMiddlewareAux(http.HandlerFunc(handler), time.Second, &wg, func(time.Duration) ticker.Ticker {
		return ft
	})

	mw(rr, req)

	wg.Add(1)
	ft.tick()
	wg.Wait()

	wrapped.Write([]byte("data: 1\n"))
	assert.Equal(t, 1, ft.resetCalls)
	wrapped.Write([]byte("data: 1\n"))
	assert.Equal(t, 2, ft.resetCalls)

	wg.Add(1)
	ft.tick()
	wg.Wait()

	assert.Equal(t, 2, ft.resetCalls)
	assert.Equal(t, ":\ndata: 1\ndata: 1\n:\n", string(rr.Body.Bytes()))
}

func Test_serverSentEventKeepaliveMiddleware_NonEventstream(t *testing.T) {
	rr := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "/api/workflows", nil)
	req.Header["Accept"] = []string{"text/plain"}

	if !assert.Nil(t, err) {
		return
	}

	var wrapped http.ResponseWriter

	handler := func(rw http.ResponseWriter, r *http.Request) {
		wrapped = rw
	}

	ft := newFakeTicker(time.Second * 1)
	mw := serverSentEventKeepaliveMiddlewareAux(http.HandlerFunc(handler), time.Second, nil, func(time.Duration) ticker.Ticker {
		return ft
	})

	mw(rr, req)

	ft.tick()

	wrapped.Write([]byte("foobar"))

	assert.Equal(t, 0, ft.resetCalls)
	assert.Equal(t, "foobar", string(rr.Body.Bytes()))
}
