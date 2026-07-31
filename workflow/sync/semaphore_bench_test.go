package sync

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Tier 1 benchmark harness for semaphore promotion throughput.
//
// This models the controller's reconcile loop against the in-memory
// prioritySemaphore directly: no Kubernetes, no pods, no database. The 6-minute
// workflow body is virtual time (a hold that spans a fixed number of rounds), so
// a run that takes ~25 minutes in a real cluster completes here in milliseconds.
//
// What it measures is the shape of promotion, not wall-clock makespan:
//
//	rounds      - reconcile rounds needed to drain every workflow. The controller
//	              cannot admit faster than one round per requeue, so rounds is the
//	              latency multiplier that turns into real time on a cluster.
//	tryAcquires - total tryAcquire calls. Divided by the number of workflows this
//	              is admission amplification: 1.0 is perfect, higher means
//	              workflows are bouncing off the semaphore and re-queueing.
//	enqueues    - nextWorkflow callbacks. This is the workqueue churn that the
//	              O(N^2) re-wake storm produces.
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

	// enqueued is the controller workqueue: keys that nextWorkflow asked to be
	// reconciled. A real controller also periodically resyncs, but the point of
	// the comparison is precisely how much this queue is churned, so the sim
	// reconciles exactly the woken set plus any workflow not yet done.
	enqueued map[string]bool

	// instrumentation
	rounds      int
	tryAcquires int
	enqueues    int

	// utilization[i] is the number of held slots at the end of round i.
	utilization []int
}

func newReconcileSim(ctx context.Context, tb testing.TB, limit, total, holdRounds int) *reconcileSim {
	tb.Helper()
	sim := &reconcileSim{
		limit:      limit,
		total:      total,
		holdRounds: holdRounds,
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
	state := uint64(round)*2862933555777941757 + 3037000493
	for i := len(keys) - 1; i > 0; i-- {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		j := int(state % uint64(i+1))
		keys[i], keys[j] = keys[j], keys[i]
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
		shuffleDeterministic(batch, s.rounds)

		for _, key := range batch {
			s.tryAcquires++
			acquired, _, err := s.sem.tryAcquire(ctx, key, nil)
			if err != nil {
				panic(err)
			}
			if acquired {
				releaseAt[key] = s.rounds + s.holdRounds
			} else {
				// Bounced: the controller re-queues it to try again. This is the
				// cost the grant set removes.
				s.enqueued[key] = true
			}
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

// scenario is one workload shape. The Databricks reference case is
// limit=400/total=1500; the smaller shapes show how the cost scales with limit,
// which is the signature of the quadratic behaviour.
type scenario struct {
	name       string
	limit      int
	total      int
	holdRounds int
}

var benchScenarios = []scenario{
	{name: "limit10_wf50", limit: 10, total: 50, holdRounds: 1},
	{name: "limit50_wf200", limit: 50, total: 200, holdRounds: 1},
	{name: "limit100_wf400", limit: 100, total: 400, holdRounds: 1},
	// The reference workload: 400 slots, 1500 workflows. holdRounds=1 isolates
	// promotion cost from hold duration -- a longer hold only adds a constant per
	// wave, whereas the promotion cost is what differs between implementations.
	{name: "limit400_wf1500", limit: 400, total: 1500, holdRounds: 1},
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

	t.Logf("%-18s %7s %7s %9s %9s %7s %7s %6s",
		"scenario", "limit", "wfs", "rounds", "optimum", "ratio", "ampl", "util")

	for _, sc := range benchScenarios {
		t.Run(sc.name, func(t *testing.T) {
			sim := newReconcileSim(ctx, t, sc.limit, sc.total, sc.holdRounds)

			// Cap generously: the quadratic case needs ~limit rounds per wave, so
			// allow that plus slack. Hitting the cap is reported, not hidden.
			waves := (sc.total + sc.limit - 1) / sc.limit
			maxRounds := waves * (sc.limit + sc.holdRounds) * 4

			start := time.Now()
			completed, hitCap := sim.run(ctx, maxRounds)
			elapsed := time.Since(start)

			optimum := waves * sc.holdRounds
			ratio := float64(sim.rounds) / float64(optimum)
			ampl := float64(sim.tryAcquires) / float64(sc.total)

			t.Logf("%-18s %7d %7d %9d %9d %6.1fx %6.1fx %5.0f%%",
				sc.name, sc.limit, sc.total, sim.rounds, optimum, ratio, ampl,
				sim.meanUtilization()*100)
			t.Logf("  completed=%d/%d enqueues=%d tryAcquires=%d wall=%s hitRoundCap=%v",
				completed, sc.total, sim.enqueues, sim.tryAcquires, elapsed.Round(time.Millisecond), hitCap)

			if completed != sc.total {
				t.Logf("  NOTE: only %d/%d workflows drained within %d rounds",
					completed, sc.total, maxRounds)
			}
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
				sim := newReconcileSim(ctx, b, sc.limit, sc.total, sc.holdRounds)
				sim.run(ctx, maxRounds)
				b.ReportMetric(float64(sim.rounds), "rounds")
				b.ReportMetric(float64(sim.tryAcquires)/float64(sc.total), "tryAcquire/wf")
			}
		})
	}
}
