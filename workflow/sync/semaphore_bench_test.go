package sync

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Tier 1 benchmark harness for semaphore promotion throughput.
//
// To reproduce the before/after comparison:
//
//	go test ./workflow/sync/ -run TestSemaphorePromotionCost -count=1 -v
//
// Both columns come from this one command. "before" emulates the pre-grant-set
// admission gate in the harness (see reconcileSim.singleFront) rather than
// requiring a checkout of the parent revision, so the comparison is a single
// deterministic run with no build juggling. Everything is seedless, so two runs
// on the same commit produce byte-identical numbers.
//
// This models the controller's reconcile loop against the in-memory
// prioritySemaphore directly: no Kubernetes, no pods, no database. The 6-minute
// workflow body is virtual time (a hold that spans a fixed number of rounds), so
// a run that takes ~25 minutes in a real cluster completes here in milliseconds.
//
// What it measures is the shape of promotion, not wall-clock makespan. The test
// reports total tryAcquire calls both per workflow and absolute. Every workflow
// acquires exactly once, so the per-workflow figure is one successful reconcile
// plus every wasted one -- the admission amplification factor, where 1.0 means
// each workflow is looked at once and admitted.
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
// The reconcile loop is a model of the controller, not the controller itself;
// Tier 2 (fake-clientset controller test) exists to corroborate these numbers.

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

	// singleFront emulates the pre-grant-set controller, so the "before" column is
	// reproducible from this commit instead of requiring a checkout of the parent
	// revision. Upstream had two separate widths: notifyWaiters woke the top
	// (limit - holders) waiters, but checkAcquire admitted only pending.items[0].
	// The gate lives here rather than in the semaphore because it is a property of
	// the old code, not a configuration anyone should be able to select. Emulating
	// it means both columns of the before/after comparison are reproducible with a
	// single `go test` on this commit; see the header comment for how to run it.
	singleFront bool

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

// newReconcileSim builds a simulation. Set singleFront to measure the
// pre-grant-set baseline (see reconcileSim.singleFront); leave it false to measure
// the grant-set implementation in this commit.
func newReconcileSim(ctx context.Context, tb testing.TB, limit, total, holdRounds, workers int, singleFront bool) *reconcileSim {
	tb.Helper()
	sim := &reconcileSim{
		limit:       limit,
		total:       total,
		holdRounds:  holdRounds,
		workers:     workers,
		singleFront: singleFront,
		enqueued:    make(map[string]bool, total),
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

// pendingHead returns the current head of the priority queue, i.e. the only key
// upstream's checkAcquire would have admitted. Used solely by the singleFront
// baseline.
func (s *reconcileSim) pendingHead() (string, bool) {
	if s.sem.pending.Len() == 0 {
		return "", false
	}
	return s.sem.pending.items[0].key, true
}

func (s *reconcileSim) isPendingHead(key string) bool {
	head, ok := s.pendingHead()
	return ok && head == key
}

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

		if s.singleFront {
			// Upstream had no grant set, so a waiter that had already been woken was
			// woken again on the next release. Clearing the grants each round restores
			// that repeated-wake behaviour; without this, grantWaiters would suppress
			// the re-wakes and understate the churn being measured.
			clear(s.sem.granted)
		}

		for _, key := range batch {
			if s.singleFront && !s.isPendingHead(key) {
				// Upstream checkAcquire admitted only pending.items[0]. Everyone else
				// bounced with "isn't at the front" after enqueueing the head.
				s.bounces++
				s.tryAcquires++
				if head, ok := s.pendingHead(); ok {
					s.enqueues++
					s.enqueued[head] = true
				}
				continue
			}
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

// variants are the two implementations compared side by side. "before" emulates
// the pre-grant-set controller (see reconcileSim.singleFront); "after" is the
// grant-set code in this commit. Both columns are therefore reproducible from a
// single checkout, with no need to build the parent revision.
var variants = []struct {
	name        string
	singleFront bool
}{
	{name: "before", singleFront: true},
	{name: "after", singleFront: false},
}

// TestSemaphorePromotionCost is the headline measurement, written as a test
// rather than a Benchmark because the interesting quantity is a structural count
// (rounds, amplification) that is identical run to run, not a time that needs
// repetition to stabilize.
//
// The optimum is ceil(total/limit) waves, each taking holdRounds. Any excess is
// promotion overhead.
func TestSemaphorePromotionCost(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// reconciles is total tryAcquire calls across the drain: the absolute load the
	// controller and apiserver carry. Per workflow it is the admission
	// amplification factor, since every workflow acquires exactly once -- one
	// successful reconcile plus every wasted one, where 1.0 would mean each
	// workflow is looked at once and admitted.
	//
	// One row per scenario with before and after side by side, emitted as a
	// markdown table so the result can be pasted into a PR or issue unedited.
	rows := make([]string, 0, len(benchScenarios))

	for _, sc := range benchScenarios {
		perVariant := make(map[string]int, len(variants))

		for _, v := range variants {
			t.Run(sc.name+"/"+v.name, func(t *testing.T) {
				sim := newReconcileSim(ctx, t, sc.limit, sc.total, sc.holdRounds, sc.workers, v.singleFront)

				// Cap generously: the quadratic case needs ~limit rounds per wave, so
				// allow that plus slack. Hitting the cap is reported, not hidden.
				waves := (sc.total + sc.limit - 1) / sc.limit
				maxRounds := waves * (sc.limit + sc.holdRounds) * 4

				start := time.Now()
				completed, hitCap := sim.run(ctx, maxRounds)
				elapsed := time.Since(start)

				// The counts are only meaningful if every workflow got through, so
				// this is an assertion rather than a reported column.
				if hitCap || completed != sc.total {
					t.Fatalf("only %d/%d workflows drained within %d rounds (%s)",
						completed, sc.total, maxRounds, elapsed.Round(time.Millisecond))
				}
				perVariant[v.name] = sim.tryAcquires
			})
		}

		// A subtest that failed leaves its variant unrecorded; skip the row rather
		// than printing a ratio against a zero.
		before, okBefore := perVariant["before"]
		after, okAfter := perVariant["after"]
		if !okBefore || !okAfter {
			continue
		}
		rows = append(rows, fmt.Sprintf("| %d | %d | %s | %s | %.0fx |",
			sc.limit, sc.total,
			reconcileCell(before, sc.total), reconcileCell(after, sc.total),
			float64(before)/float64(after)))
	}

	t.Logf("\n%s\n%s\n%s",
		"| limit | workflows | before | after | reduction |",
		"|------:|----------:|-------:|------:|----------:|",
		strings.Join(rows, "\n"))
}

// reconcileCell renders a reconcile count as "total (N.N/wf)", the absolute load
// alongside the same figure normalized per workflow.
func reconcileCell(reconciles, total int) string {
	return fmt.Sprintf("%s (%.1f/wf)", humanCount(reconciles),
		float64(reconciles)/float64(total))
}

// humanCount groups thousands with commas: 1394953 -> "1,394,953". Six-digit
// reconcile counts are the point of the table and are hard to compare unseparated.
func humanCount(n int) string {
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// BenchmarkSemaphorePromotion measures wall-clock cost of draining the reference
// workload, for the CPU side of the comparison. A fix that trades stalling for
// busy-looping would show up here as a regression even while rounds improve.
func BenchmarkSemaphorePromotion(b *testing.B) {
	ctx := logging.TestContext(b.Context())
	for _, sc := range benchScenarios {
		for _, v := range variants {
			b.Run(sc.name+"/"+v.name, func(b *testing.B) {
				waves := (sc.total + sc.limit - 1) / sc.limit
				maxRounds := waves * (sc.limit + sc.holdRounds) * 4
				b.ReportAllocs()
				for b.Loop() {
					sim := newReconcileSim(ctx, b, sc.limit, sc.total, sc.holdRounds, sc.workers, v.singleFront)
					sim.run(ctx, maxRounds)
					b.ReportMetric(float64(sim.rounds), "rounds")
					b.ReportMetric(float64(sim.bounces)/float64(sc.total), "bounces/wf")
				}
			})
		}
	}
}
