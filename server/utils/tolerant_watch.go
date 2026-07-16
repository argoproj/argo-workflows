package utils

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// ptrObject constrains PT to be a pointer-to-T that also satisfies runtime.Object.
// The reflector consumes runtime.Object event payloads, so the wrapper must emit
// pointers to the typed struct, not the bare value type.
type ptrObject[T any] interface {
	*T
	runtime.Object
}

// TolerantWatch wraps a dynamic-client Watch with per-event tolerant decoding.
// Each upstream event whose Object is an *unstructured.Unstructured is converted
// into a fresh *T. Events that fail conversion are logged and dropped, so a
// single malformed resource cannot repeatedly tear down the consuming reflector.
//
// Error events are forwarded as-is. Bookmark events are decoded to a typed *T
// (an empty one carrying the bookmark's resourceVersion and annotations if decode
// fails) so they still drive the reflector's watch-list sync. Pass an empty
// namespace for cluster-scoped resources.
func TolerantWatch[T any, PT ptrObject[T]](
	ctx context.Context,
	dyn dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	opts metav1.ListOptions,
) (watch.Interface, error) {
	resource := dyn.Resource(gvr)
	var upstream watch.Interface
	var err error
	if namespace == "" {
		upstream, err = resource.Watch(ctx, opts)
	} else {
		upstream, err = resource.Namespace(namespace).Watch(ctx, opts)
	}
	if err != nil {
		return nil, err
	}
	return newTolerantWatchProxy[T, PT](ctx, upstream, gvr), nil
}

type tolerantWatchProxy[T any, PT ptrObject[T]] struct {
	upstream watch.Interface
	out      chan watch.Event
	done     chan struct{}
	stopOnce sync.Once
}

func newTolerantWatchProxy[T any, PT ptrObject[T]](ctx context.Context, upstream watch.Interface, gvr schema.GroupVersionResource) *tolerantWatchProxy[T, PT] {
	p := &tolerantWatchProxy[T, PT]{
		upstream: upstream,
		out:      make(chan watch.Event),
		done:     make(chan struct{}),
	}
	go p.run(ctx, logging.RequireLoggerFromContext(ctx), gvr)
	return p
}

func (p *tolerantWatchProxy[T, PT]) run(ctx context.Context, logger logging.Logger, gvr schema.GroupVersionResource) {
	defer close(p.out)
	for {
		select {
		case <-p.done:
			return
		case evt, ok := <-p.upstream.ResultChan():
			if !ok {
				return
			}
			out, drop := p.translate(ctx, logger, gvr, evt)
			if drop {
				continue
			}
			select {
			case p.out <- out:
			case <-p.done:
				return
			}
		}
	}
}

func (p *tolerantWatchProxy[T, PT]) translate(ctx context.Context, logger logging.Logger, gvr schema.GroupVersionResource, evt watch.Event) (watch.Event, bool) {
	// Error events carry *metav1.Status, not user resources. Forward as-is so
	// the consuming reflector can handle the error.
	if evt.Type == watch.Error {
		return evt, false
	}
	un, ok := evt.Object.(*unstructured.Unstructured)
	if !ok {
		// Already typed (or unexpected) — forward without conversion.
		return evt, false
	}
	var item T
	if err := DecodeUnstructured(un, &item); err != nil {
		// Bookmark events carry only metadata, no spec. They must still reach the
		// reflector to drive watch-list sync, so fall back to an empty typed object
		// carrying the bookmark's resourceVersion (an empty resourceVersion would
		// clobber the reflector's lastSync and force a full relist) and annotations
		// (the watch-list protocol signals end-of-initial-sync with the
		// k8s.io/initial-events-end annotation on a bookmark; dropping it would hang
		// the reflector's initial sync forever).
		if evt.Type == watch.Bookmark {
			obj := PT(&item)
			if accessor, accErr := meta.Accessor(obj); accErr == nil {
				accessor.SetResourceVersion(un.GetResourceVersion())
				accessor.SetAnnotations(un.GetAnnotations())
			}
			return watch.Event{Type: evt.Type, Object: obj}, false
		}
		// A Delete whose payload no longer decodes (e.g. a well-formed object was
		// mutated into a type-incompatible shape, then deleted) must still reach the
		// store, or it leaves a phantom cache entry that keeps appearing in cached
		// list responses until the next full relist. The store deletes by UID, so a
		// metadata-only typed object is enough to evict the right row. (An Add/Modify
		// that fails to decode is intentionally dropped: the malformed object was
		// never representable in the typed cache, so there is nothing to evict.)
		if evt.Type == watch.Deleted {
			obj := PT(&item)
			if accessor, accErr := meta.Accessor(obj); accErr == nil {
				accessor.SetUID(un.GetUID())
				accessor.SetName(un.GetName())
				accessor.SetNamespace(un.GetNamespace())
				accessor.SetResourceVersion(un.GetResourceVersion())
			}
			return watch.Event{Type: evt.Type, Object: obj}, false
		}
		logger.
			WithField("namespace", un.GetNamespace()).
			WithField("name", un.GetName()).
			WithField("resource", gvr.Resource).
			WithField("eventType", string(evt.Type)).
			WithField("error", err.Error()).
			Warn(ctx, "dropping malformed watch event")
		return watch.Event{}, true
	}
	return watch.Event{Type: evt.Type, Object: PT(&item)}, false
}

func (p *tolerantWatchProxy[T, PT]) ResultChan() <-chan watch.Event { return p.out }

func (p *tolerantWatchProxy[T, PT]) Stop() {
	p.stopOnce.Do(func() {
		close(p.done)
		p.upstream.Stop()
	})
}
