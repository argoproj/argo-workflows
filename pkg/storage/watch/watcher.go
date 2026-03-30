package watch

import (
	"sync"

	"k8s.io/apimachinery/pkg/watch"
)

// watcher implements watch.Interface backed by a Go channel.
type watcher struct {
	id     uint64
	kind   string
	ns     string
	ch     chan watch.Event
	done   chan struct{}
	stop   sync.Once
	remove func(uint64)
}

func newWatcher(id uint64, kind, namespace string, bufferSize int, remove func(uint64)) *watcher {
	return &watcher{
		id:     id,
		kind:   kind,
		ns:     namespace,
		ch:     make(chan watch.Event, bufferSize),
		done:   make(chan struct{}),
		remove: remove,
	}
}

func (w *watcher) ResultChan() <-chan watch.Event {
	return w.ch
}

func (w *watcher) Stop() {
	w.stop.Do(func() {
		w.remove(w.id)
		close(w.done)
	})
}

// send sends an event to the watcher, dropping it if the watcher is stopped or the buffer is full.
func (w *watcher) send(event watch.Event) bool {
	select {
	case <-w.done:
		return false
	default:
	}

	select {
	case w.ch <- event:
		return true
	case <-w.done:
		return false
	default:
		// Buffer full, drop event. In production you'd want to handle this
		// more gracefully (e.g. close the watcher with an error).
		return false
	}
}
