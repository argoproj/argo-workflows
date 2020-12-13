package apiserver

import (
	"net/http"
	"sync"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/ticker"
	"github.com/felixge/httpsnoop"
)

func isTextStreamRequest(r *http.Request) bool {
	// We don't seem to be able to access the headers that are sent out in the response,
	// so we're going to detect an SSE stream by looking at the Accept header instead
	// and ensuring that it's the only valid response type accepted
	acceptHeader, ok := r.Header["Accept"]
	return ok && len(acceptHeader) == 1 && acceptHeader[0] == "text/event-stream"
}

type tickerFactoryFn func(time.Duration) ticker.Ticker

func serverSentEventKeepaliveMiddleware(next http.Handler, keepaliveInterval time.Duration) http.HandlerFunc {
	return serverSentEventKeepaliveMiddlewareAux(next, keepaliveInterval, nil, func(d time.Duration) ticker.Ticker {
		return ticker.NewTicker(d)
	})
}

func serverSentEventKeepaliveMiddlewareAux(next http.Handler, keepaliveInterval time.Duration, wg *sync.WaitGroup, tickerFactory tickerFactoryFn) http.HandlerFunc {
	return func(wr http.ResponseWriter, r *http.Request) {
		if !isTextStreamRequest(r) {
			next.ServeHTTP(wr, r)
			return
		}

		ticker := tickerFactory(keepaliveInterval)
		stopCh := r.Context().Done()

		var writeLock sync.Mutex

		writeKeepalive := func() {
			writeLock.Lock()
			defer writeLock.Unlock()

			// Per https://html.spec.whatwg.org/multipage/server-sent-events.html#event-stream-interpretation,
			// lines that start with a `:` must be ignored by the client.
			wr.Write([]byte(":\n"))

			if f, ok := wr.(http.Flusher); ok {
				f.Flush()
			}

			// The waitgroup is purely intended for unit tests and is always nil in production use cases
			if wg != nil {
				wg.Done()
			}
		}

		go func() {
			defer ticker.Stop()

			for {
				select {
				case <-stopCh:
					return

				case <-ticker.C():
					writeKeepalive()
				}
			}
		}()

		wrappedWr := httpsnoop.Wrap(wr, httpsnoop.Hooks{
			Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
				return func(p []byte) (int, error) {
					writeLock.Lock()
					defer writeLock.Unlock()

					ticker.Reset(keepaliveInterval)
					return next(p)
				}
			},
		})

		next.ServeHTTP(wrappedWr, r)
	}
}
