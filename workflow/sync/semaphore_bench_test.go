package sync

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Benchmark harness for semaphore promotion throughput. See
// TestSemaphorePromotionCost below for how to run it and what it measures.
//
// This measures whichever semaphore implementation it is compiled against, and
// prints one RESULT line per scenario. It does not emulate the old implementation:
// hack/semaphore-before-after.sh checks out two revisions, runs the same harness
// at each, and joins the numbers. Everything is seedless, so a given revision
// produces byte-identical numbers run to run.
//
// The simulation tracks more than it prints, for use when investigating:
//
//	rounds      - reconcile rounds needed to drain every workflow. The controller
//	              cannot admit faster than one round per requeue, so rounds is the
//	              latency multiplier that turns into real time on a cluster.
//	bounces     - reconciles that attempted acquisition and failed, i.e.
//	              tryAcquires minus the one success per workflow.
//	utilization - held slots per round; meanUtilization reports occupancy as a
//	              fraction of the limit. Deep underutilization with a large backlog
//	              is the same bug seen from the semaphore's side rather than the
//	              workflow's.
//	enqueues    - raw nextWorkflow callbacks. NOTE: this counts repeat callbacks
//	              against the same key, which the real workqueue would dedupe, so
//	              it overstates distinct reconciles. Use bounces for per-workflow
//	              cost and rounds for latency.
//
// The reconcile loop is a model of the controller, not the controller itself: one
// round is one reconcile pass over everything woken, where real workers have
// staggered latency and retry backoff. Both sides of the comparison run the same
// model, so the ratio is more trustworthy than the absolute counts.

// reconcileSim drives a semaphore the way the workflow controller does: a set of
// pending workflows, each of which is reconciled once per round, attempting to
// acquire. A holder occupies its slot for holdRounds rounds, standing in for the
// workflow body (e.g. sleep 6m).
type reconcileSim struct {
	sem        *prioritySemaphore
	limit      int
	total      int
	holdRounds int
	// workers is the controller's --workflow-workers: how many reconciles are in
	// flight at once. Bounds how far reconcile order may deviate from queue order.
	workers int

	// enqueued is the controller workqueue: keys that nextWorkflow asked to be
	// reconciled. A real controller also periodically resyncs, but the point of
	// the comparison is precisely how much this queue is churned, so the sim
	// reconciles exactly the woken set plus any workflow not yet done.
	enqueued map[string]bool

	// instrumentation
	rounds      int
	tryAcquires int
	enqueues    int
	// bounces is the number of reconciles that attempted acquisition and failed.
	// Divided by total, this is the average number of times a workflow is woken,
	// fails, and has to wait to be woken again. Bounded by the wake path: a
	// workflow only bounces if something enqueued it.
	bounces int

	// utilization[i] is the number of held slots at the end of round i.
	utilization []int
}

// newReconcileSim builds a simulation against whatever semaphore implementation
// is compiled in. There is no before/after switch: the harness measures the tree
// it is built from, and hack/semaphore-before-after.sh runs it at two revisions.
func newReconcileSim(ctx context.Context, tb testing.TB, limit, total, holdRounds, workers int) *reconcileSim {
	tb.Helper()
	sim := &reconcileSim{
		limit:      limit,
		total:      total,
		holdRounds: holdRounds,
		workers:    workers,
		enqueued:   make(map[string]bool, total),
	}
	sem, err := newInternalSemaphore(ctx, "bench", func(key string) {
		sim.enqueues++
		sim.enqueued[key] = true
	}, func(context.Context, string) (int, error) { return limit, nil }, 0)
	if err != nil {
		tb.Fatalf("newInternalSemaphore: %v", err)
	}
	sim.sem = sem
	return sim
}

func benchKey(i int) string { return fmt.Sprintf("default/wf-%05d", i) }

// shuffleDeterministic permutes keys using a seedless xorshift, so reconcile
// order is decoupled from priority order without introducing run-to-run
// variance. round is mixed into the state so successive rounds differ.
func shuffleDeterministic(keys []string, round int) {
	shuffleInBatches(keys, round, 0)
}

// shuffleInBatches models the controller's worker pool. The controller runs
// --workflow-workers goroutines against a shared workqueue, so at any instant only
// W workflows are in flight; the queue hands out keys in arrival order but the
// workers finish in a nondeterministic order within each group of W.
//
// So: chop the wake set into consecutive batches of `workers` and permute within
// each batch, leaving batch order intact. A key that entered the queue early is
// still reconciled early -- it just races with the W-1 keys alongside it. This is
// strictly weaker than a whole-set shuffle, which lets a key at the back of the
// queue be reconciled first.
//
// workers <= 0 means unbounded (one batch, i.e. a full shuffle), which is the
// worst case for the single-front gate and the original harness behaviour.
//
// workers == 1 is rejected rather than supported. Batches of one cannot be
// permuted, so the wake set would be left in the caller's sorted order, which is
// priority order -- and iterating in priority order lets each acquire unblock the
// next key within the same round, cascading the whole backlog through and hiding
// the head-of-line blocking this harness exists to measure. It reports 88 rounds
// where every W >= 4 reports ~1200. Measured admissions per round are otherwise
// flat in W (1.76 at W=4, 1.70 at W=512), so no real setting needs W=1.
func shuffleInBatches(keys []string, round, workers int) {
	if workers == 1 {
		panic("shuffleInBatches: workers=1 leaves keys in priority order, which defeats the measurement; use 0 for a full shuffle")
	}
	if workers <= 0 || workers > len(keys) {
		workers = len(keys)
	}
	state := uint64(round)*2862933555777941757 + 3037000493
	for start := 0; start < len(keys); start += workers {
		end := min(start+workers, len(keys))
		for i := end - 1; i > start; i-- {
			state ^= state << 13
			state ^= state >> 7
			state ^= state << 17
			j := start + int(state%uint64(i-start+1))
			keys[i], keys[j] = keys[j], keys[i]
		}
	}
}

// run submits every workflow, then reconciles until all have acquired and
// released. It returns once every workflow is done or maxRounds is exceeded.
//
// maxRounds is a safety valve: the pathological case is quadratic, so an
// unbounded loop on a large input would appear to hang. Exceeding it is itself a
// meaningful result and is reported rather than silently truncated.
func (s *reconcileSim) run(ctx context.Context, maxRounds int) (completed int, hitCap bool) {
	now := time.Now()
	for i := range s.total {
		// Priority ties broken by creation time, mirroring real submission order.
		if err := s.sem.addToQueue(ctx, benchKey(i), 0, now.Add(time.Duration(i)*time.Millisecond)); err != nil {
			panic(err)
		}
		s.enqueued[benchKey(i)] = true
	}

	// releaseAt[key] is the round at which a holder's virtual work finishes.
	releaseAt := make(map[string]int, s.total)
	done := make(map[string]bool, s.total)

	for len(done) < s.total && s.rounds < maxRounds {
		s.rounds++

		// Releases land first: a holder whose work finished frees its slot before
		// this round's admissions, as in the controller where pod completion is
		// observed and Release runs during that workflow's reconcile.
		for key, at := range releaseAt {
			if at == s.rounds {
				s.sem.release(ctx, key)
				delete(releaseAt, key)
				done[key] = true
			}
		}

		// Reconcile every workflow the controller has been asked to look at.
		//
		// Order matters and must NOT be priority order. The controller's workqueue
		// is keyed by workflow name and shuffled by arrival, retry backoff, and
		// worker concurrency; it does not hand work to the reconciler in semaphore
		// priority order. Iterating in sorted (== priority) order would let each
		// acquire unblock the next key within a single round, which silently
		// defeats the very head-of-line blocking being measured and makes the
		// single-front implementation look optimal.
		//
		// So: collect deterministically (Go map order is randomized), then apply a
		// fixed permutation to decouple reconcile order from priority order. The
		// permutation is seedless and reproducible, so runs stay comparable.
		batch := make([]string, 0, len(s.enqueued))
		for key := range s.enqueued {
			if !done[key] && releaseAt[key] == 0 {
				batch = append(batch, key)
			}
		}
		sort.Strings(batch)
		clear(s.enqueued)
		shuffleInBatches(batch, s.rounds, s.workers)

		for _, key := range batch {
			s.tryAcquires++
			acquired, _, err := s.sem.tryAcquire(ctx, key, nil)
			if err != nil {
				panic(err)
			}
			if acquired {
				releaseAt[key] = s.rounds + s.holdRounds
			} else {
				s.bounces++
			}
			// A bounced workflow is NOT re-queued here. The controller marks it
			// Pending and returns; it is only reconciled again when something wakes
			// it (notifyWaiters on a release, or the not-at-front path in
			// checkAcquire enqueueing the head). Those wakes arrive through the
			// nextWorkflow callback, which populates s.enqueued. Re-adding every
			// bouncer here would reconcile workflows the controller left alone and
			// inflate tryAcquires past what the wake path can produce.
		}

		s.utilization = append(s.utilization, len(s.sem.lockHolder))
	}
	return len(done), s.rounds >= maxRounds
}

// meanUtilization reports average slot occupancy as a fraction of the limit,
// over rounds where work remained. Deep underutilization with a large backlog is
// the clearest evidence of head-of-line blocking.
func (s *reconcileSim) meanUtilization() float64 {
	if len(s.utilization) == 0 {
		return 0
	}
	sum := 0
	for _, u := range s.utilization {
		sum += u
	}
	return float64(sum) / float64(len(s.utilization)) / float64(s.limit)
}

// scenario is one workload shape. The reference case is limit=500/total=2000; the
// smaller shape shows how the cost scales with limit, which is the signature of
// the quadratic behaviour.
type scenario struct {
	name       string
	limit      int
	total      int
	holdRounds int
	// workers is the controller's --workflow-workers. Zero means unbounded, i.e.
	// every woken workflow may be reconciled in any order within a round.
	workers int
}

// Two shapes, an order of magnitude apart in both dimensions, which is enough to
// show that the amplification tracks the limit rather than being a property of one
// workload size.
//
// holdRounds=1 isolates promotion cost from hold duration -- a longer hold only
// adds a constant per wave, whereas promotion cost is what differs between
// implementations. workers=16 is half the --workflow-workers default of 32, so the
// numbers describe a conservatively-sized controller rather than the most
// favourable one; admissions per round are near-flat in W anyway (1.76 at W=4,
// 1.66 at W=32, 1.70 at W=512), so it is not a tuned choice.
var benchScenarios = []scenario{
	{name: "limit100_wf400", limit: 100, total: 400, holdRounds: 1, workers: 16},
	// The larger shape. Both dimensions are 5x the smaller one, so the gap between
	// the two "before" figures does not by itself say which dimension drives the
	// cost. Holding the backlog at 2000 and sweeping the limit answers that: 114
	// wasted reconciles per workflow at limit=200 rising to 451 at limit=1000, while
	// every one of those limits takes the same 1195 rounds to drain.
	//
	// So under the single-front gate the limit buys no throughput at all -- only the
	// queue head may acquire, so a round admits the head plus whichever successors
	// happen to be reconciled after it, about 1.7 workflows per round however many
	// slots are free. What the limit does buy is waste, since it widens the woken
	// set. The grant set is what makes the limit mean anything.
	{name: "limit500_wf2000", limit: 500, total: 2000, holdRounds: 1, workers: 16},
	// Same limit, 2.5x the backlog. This is the pair that does isolate a single
	// dimension: holding limit at 500, reconciles/wf rises 261 -> 279 while the
	// absolute count rises 522k -> 1.39M. Cost per workflow is roughly flat in the
	// backlog, so the total grows with it -- superlinearly overall, since each of
	// the extra workflows also costs ~limit/2 reconciles of its own.
	{name: "limit500_wf5000", limit: 500, total: 5000, holdRounds: 1, workers: 16},
}

// TestSemaphorePromotionCost is the headline measurement.
//
// For the before/after comparison, use the script -- it runs this test at the
// pre-grant-set revision and at HEAD, and prints a markdown table:
//
//	./hack/semaphore-before-after.sh
//
// To run just this revision:
//
//	go test ./workflow/sync/ -run TestSemaphorePromotionCost -count=1 -v
//
// -v is required, since results come out through t.Logf. Takes about 3s. Append a
// scenario name to narrow it (-run '.../limit500_wf2000').
//
// It reports whichever implementation it was built from; there is no before/after
// switch in the harness.
//
// What it does: queue `total` workflows against one semaphore of size `limit`,
// then reconcile in rounds until all have run. Each round reconciles every
// workflow the controller has been asked to look at, in an order deliberately
// decoupled from priority order; whoever acquires holds its slot for holdRounds
// rounds and then releases, which wakes waiters. A workflow that fails to acquire
// is not re-reconciled until something wakes it, exactly as in the controller. No
// Kubernetes, no pods, no database, and the workflow body is virtual time -- a
// drain that takes ~25 minutes on a cluster runs here in under a second.
//
// What it reports: total tryAcquire calls to drain the queue. Every workflow
// acquires exactly once, so divided by the workflow count this is one useful
// reconcile plus every wasted one, and 1.0 is the floor. That isolates the cost of
// the promotion strategy, since the work of running the workflows themselves is
// identical across revisions.
//
// It is a test rather than a Benchmark because the quantity is a structural count
// that is identical run to run, not a time needing repetition to stabilize.
// BenchmarkSemaphorePromotion below covers the wall-clock and allocation side.
func TestSemaphorePromotionCost(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// One RESULT line per scenario, in a fixed parseable format, because this test
	// measures whichever revision it was compiled from and cannot know the other
	// side. hack/semaphore-before-after.sh runs it at two revisions and joins the
	// lines into the before/after table.
	for _, sc := range benchScenarios {
		t.Run(sc.name, func(t *testing.T) {
			sim := newReconcileSim(ctx, t, sc.limit, sc.total, sc.holdRounds, sc.workers)

			// Cap generously: the quadratic case needs ~limit rounds per wave, so
			// allow that plus slack. Hitting the cap is reported, not hidden.
			waves := (sc.total + sc.limit - 1) / sc.limit
			maxRounds := waves * (sc.limit + sc.holdRounds) * 4

			start := time.Now()
			completed, hitCap := sim.run(ctx, maxRounds)
			elapsed := time.Since(start)

			// The counts are only meaningful if every workflow got through, so this
			// is an assertion rather than a reported column.
			if hitCap || completed != sc.total {
				t.Fatalf("only %d/%d workflows drained within %d rounds (%s)",
					completed, sc.total, maxRounds, elapsed.Round(time.Millisecond))
			}

			// reconciles is total tryAcquire calls across the drain: the absolute
			// load the controller and apiserver carry. Divided by the workflow count
			// it is the admission amplification factor, since every workflow acquires
			// exactly once -- one useful reconcile plus every wasted one, where 1.0
			// would mean each workflow is looked at once and admitted.
			t.Logf("RESULT scenario=%s limit=%d workflows=%d reconciles=%d rounds=%d",
				sc.name, sc.limit, sc.total, sim.tryAcquires, sim.rounds)
		})
	}
}

// BenchmarkSemaphorePromotion measures wall-clock cost of draining the reference
// workload, for the CPU side of the comparison. A fix that trades stalling for
// busy-looping would show up here as a regression even while rounds improve.
func BenchmarkSemaphorePromotion(b *testing.B) {
	ctx := logging.TestContext(b.Context())
	for _, sc := range benchScenarios {
		b.Run(sc.name, func(b *testing.B) {
			waves := (sc.total + sc.limit - 1) / sc.limit
			maxRounds := waves * (sc.limit + sc.holdRounds) * 4
			b.ReportAllocs()
			for b.Loop() {
				sim := newReconcileSim(ctx, b, sc.limit, sc.total, sc.holdRounds, sc.workers)
				sim.run(ctx, maxRounds)
				b.ReportMetric(float64(sim.rounds), "rounds")
				b.ReportMetric(float64(sim.bounces)/float64(sc.total), "bounces/wf")
			}
		})
	}
}
