package dispatch

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/util/auth"
	informerutil "github.com/argoproj/argo-workflows/v4/util/informer"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// Watches is the watch-backed half of a worker's Kubernetes Consumer:
// informers on the metadata of its pool's TaskSets and results, which find
// its candidates and tell its claims when they are interrupted or lost.
// Sources gives them to NewConsumer; Run starts them.
//
// The caches lag the API server, so a claim is reported lost only on
// evidence: its own result's deletion, or a live read finding the result
// gone or replaced. A result of the claim's name with another UID in the
// cache can be an older one not yet seen deleted.
type Watches struct {
	worker   Worker
	store    store
	taskSets cache.SharedIndexInformer
	results  cache.SharedIndexInformer

	mu sync.Mutex
	// changed is closed and replaced on every event.
	changed chan struct{}
	// claims are the watched claims by result name.
	claims map[string][]*claimWatch
}

type claimWatch struct {
	// ctx is the watch's, for its live reads: it outlives the call that
	// started the watch.
	//nolint:containedctx
	ctx    context.Context
	claim  Claim
	notify func(StopReason)
	// interrupted is set once notify was told of the interrupt.
	interrupted bool
}

// NewWatches returns the watches of worker's pool, over metadata-only
// informers on the TaskSets and results in its namespace with its pool
// hash. client reads a result live when the cache suggests its claim is
// lost.
func NewWatches(client versioned.Interface, metadataClient metadata.Interface, worker Worker) *Watches {
	selector := poolSelector(worker)
	informer := func(resource string) cache.SharedIndexInformer {
		// resyncPeriod 0: a resync replays the cache, which holds no news.
		informer := metadatainformer.NewFilteredMetadataInformer(
			metadataClient,
			wfv1.SchemeGroupVersion.WithResource(resource),
			worker.Namespace,
			0,
			cache.Indexers{},
			func(options *metav1.ListOptions) { options.LabelSelector = selector },
		).Informer()
		//nolint:errcheck // the error only happens if the informer was already started, and it hasn't been
		informer.SetTransform(informerutil.StripManagedFields)
		return informer
	}
	return newWatches(kubeStore{client: client}, worker, informer(ResourceTaskSets), informer(ResourceResults))
}

// poolSelector selects the TaskSets and results of worker's pool.
func poolSelector(worker Worker) string {
	return labels.SelectorFromSet(labels.Set{common.LabelKeyPoolHash: worker.PoolHash}).String()
}

// workerAccess is every verb a worker uses on each resource: its Consumer's
// and claims' calls, a claim's live read of its result, and the list and
// watch of its Watches' informers. The hot-worker Role grants these.
var workerAccess = []struct {
	resource string
	verbs    []string
}{
	{ResourceTaskSets, []string{"get", "list", "watch"}},
	{ResourceResults, []string{"create", "get", "patch", "delete", "list", "watch"}},
}

// CheckAccess fails if worker isn't allowed every verb it uses on its
// namespace's TaskSets and results, as when its service account lacks the
// hot-worker Role's verbs. Without a list or watch, the watches' informers
// would retry for ever, and without a write every claim would fail, so the
// worker would never run a task while its pod looked ready. It asks the API
// server with SelfSubjectAccessReviews, which every authenticated account
// may create. A review that fails is only logged: the worker goes on, and
// finds out from its calls.
func CheckAccess(ctx context.Context, kubeClient kubernetes.Interface, worker Worker) error {
	var denied []string
	for _, access := range workerAccess {
		var verbs []string
		for _, verb := range access.verbs {
			allowed, err := auth.CanI(ctx, kubeClient, []string{verb}, workflow.Group, worker.Namespace, access.resource)
			if err != nil {
				logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "Couldn't check the worker's access; going on without")
				return nil
			}
			if !allowed {
				verbs = append(verbs, verb)
			}
		}
		if len(verbs) > 0 {
			denied = append(denied, strings.Join(verbs, ", ")+" "+access.resource)
		}
	}
	if len(denied) == 0 {
		return nil
	}
	account := "the service account the worker runs as"
	if review, err := kubeClient.AuthenticationV1().SelfSubjectReviews().Create(ctx, &authenticationv1.SelfSubjectReview{}, metav1.CreateOptions{}); err == nil && review.Status.UserInfo.Username != "" {
		account += " (" + review.Status.UserInfo.Username + ")"
	}
	return fmt.Errorf("the worker isn't allowed to %s in namespace %s, so it can't find, claim or report its pool's tasks: "+
		"bind the hot-worker Role, or grant its verbs, to %s, "+
		"which is the pool's or, when executor.serviceAccountName is set, the executor's", strings.Join(denied, "; "), worker.Namespace, account)
}

// newWatches returns the watches over the informers, which may hold any
// objects with metadata. st reads results live.
func newWatches(st store, worker Worker, taskSets, results cache.SharedIndexInformer) *Watches {
	w := &Watches{
		worker:   worker,
		store:    st,
		taskSets: taskSets,
		results:  results,
		changed:  make(chan struct{}),
		claims:   map[string][]*claimWatch{},
	}
	//nolint:errcheck // the error only happens if the informer was stopped, and it hasn't even started
	taskSets.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(any) { w.wake() },
		UpdateFunc: func(any, any) { w.wake() },
		DeleteFunc: func(any) { w.wake() },
	})
	//nolint:errcheck // as above
	results.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { w.resultChanged(obj, false) },
		UpdateFunc: func(_, obj any) { w.resultChanged(obj, false) },
		DeleteFunc: func(obj any) { w.resultChanged(informerutil.Untombstone(obj), true) },
	})
	return w
}

// Sources returns the watches as a Consumer's sources.
func (w *Watches) Sources() WatchSources {
	return WatchSources{Candidates: w, Claims: w}
}

// Run starts the informers, which stop when ctx ends.
func (w *Watches) Run(ctx context.Context) {
	go w.taskSets.Run(ctx.Done())
	go w.results.Run(ctx.Done())
}

// Candidates implements CandidateSource. It waits for the caches to sync
// first, so an empty cache isn't mistaken for every TaskSet being claimed.
func (w *Watches) Candidates(ctx context.Context) ([]Candidate, error) {
	if !cache.WaitForCacheSync(ctx.Done(), w.taskSets.HasSynced, w.results.HasSynced) {
		return nil, ctx.Err()
	}
	for {
		w.mu.Lock()
		changed := w.changed
		w.mu.Unlock()
		if found := w.candidates(); len(found) > 0 {
			return found, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

// candidates are the cached TaskSets of the worker's pool with no cached
// result, oldest first.
func (w *Watches) candidates() []Candidate {
	var found []Candidate
	for _, obj := range w.taskSets.GetStore().List() {
		m, err := meta.Accessor(obj)
		if err != nil {
			continue
		}
		c, ok := CandidateFromTaskSet(m)
		if !ok || c.Namespace != w.worker.Namespace || c.PoolHash != w.worker.PoolHash {
			continue
		}
		if _, claimed, _ := w.results.GetStore().GetByKey(c.Namespace + "/" + c.ResultName()); !claimed {
			found = append(found, c)
		}
	}
	slices.SortFunc(found, func(a, b Candidate) int {
		return cmp.Or(a.CreatedAt.Compare(b.CreatedAt.Time), cmp.Compare(a.NodeID, b.NodeID))
	})
	return found
}

// wake wakes everything waiting for a change.
func (w *Watches) wake() {
	w.mu.Lock()
	defer w.mu.Unlock()
	close(w.changed)
	w.changed = make(chan struct{})
}

// Watch implements ClaimWatcher. Anything the cache already shows is
// reported at once.
func (w *Watches) Watch(ctx context.Context, claim Claim, notify func(StopReason)) func() {
	cw := &claimWatch{ctx: ctx, claim: claim, notify: notify}
	name := claim.ResultName()
	w.mu.Lock()
	w.claims[name] = append(w.claims[name], cw)
	w.mu.Unlock()
	if obj, ok, _ := w.results.GetStore().GetByKey(claim.Namespace + "/" + name); ok {
		w.resultChanged(obj, false)
	}
	return func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.unwatchLocked(cw)
	}
}

// unwatchLocked stops watching cw, reporting whether it was watched. w.mu
// must be held.
func (w *Watches) unwatchLocked(cw *claimWatch) bool {
	name := cw.claim.ResultName()
	watched := slices.Contains(w.claims[name], cw)
	w.claims[name] = slices.DeleteFunc(w.claims[name], func(other *claimWatch) bool { return other == cw })
	if len(w.claims[name]) == 0 {
		delete(w.claims, name)
	}
	return watched
}

// resultChanged tells the claims on a result what its event means for
// them: a claim whose own result was deleted is lost, one whose result is
// labelled for interrupt is interrupted, and one whose name another result
// holds is checked live.
func (w *Watches) resultChanged(obj any, deleted bool) {
	defer w.wake()
	m, err := meta.Accessor(obj)
	if err != nil || m.GetNamespace() != w.worker.Namespace {
		return
	}
	var calls []func()
	var check []*claimWatch
	w.mu.Lock()
	kept := w.claims[m.GetName()][:0]
	for _, cw := range w.claims[m.GetName()] {
		switch {
		case m.GetUID() != cw.claim.UID:
			// A deletion of another object says nothing about this claim.
			if !deleted {
				check = append(check, cw)
			}
		case deleted:
			calls = append(calls, func() { cw.notify(StopLost) })
			continue
		case m.GetLabels()[common.LabelKeyInterrupt] == "true" && !cw.interrupted:
			cw.interrupted = true
			calls = append(calls, func() { cw.notify(StopInterrupted) })
		}
		kept = append(kept, cw)
	}
	if len(kept) == 0 {
		delete(w.claims, m.GetName())
	} else {
		w.claims[m.GetName()] = kept
	}
	w.mu.Unlock()
	for _, call := range calls {
		call()
	}
	for _, cw := range check {
		go w.checkLive(cw)
	}
}

// checkLive reads the claim's result and reports the claim lost if it is
// gone or another result has its name. A failed read is left for the
// claim's next write to find out.
func (w *Watches) checkLive(cw *claimWatch) {
	result, err := w.store.getResult(cw.ctx, cw.claim.Namespace, cw.claim.ResultName())
	switch {
	case errors.Is(err, ErrNotFound):
	case err != nil:
		logging.RequireLoggerFromContext(cw.ctx).WithError(err).
			WithFields(logging.Fields{"namespace": cw.claim.Namespace, "result": cw.claim.ResultName()}).
			Warn(cw.ctx, "Failed to read a claimed WorkflowTaskResult")
		return
	case result.UID == cw.claim.UID:
		return
	}
	w.mu.Lock()
	watched := w.unwatchLocked(cw)
	w.mu.Unlock()
	if watched {
		cw.notify(StopLost)
	}
}
