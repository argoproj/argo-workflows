package sync

import "context"

type syncMetrics interface {
	AddMutex(ctx context.Context, name, namespace string)
	RemoveMutex(ctx context.Context, name, namespace string)
	AddSemaphore(ctx context.Context, name, namespace string)
	RemoveSemaphore(ctx context.Context, name, namespace string)
}

type testMetricsRecorder struct {
	mutexAdds        map[string]int
	mutexRemoves     map[string]int
	semaphoreAdds    map[string]int
	semaphoreRemoves map[string]int
}

func newTestMetricsRecorder() *testMetricsRecorder {
	return &testMetricsRecorder{
		mutexAdds:        make(map[string]int),
		mutexRemoves:     make(map[string]int),
		semaphoreAdds:    make(map[string]int),
		semaphoreRemoves: make(map[string]int),
	}
}

func (t *testMetricsRecorder) AddMutex(_ context.Context, name, _ string)         { t.mutexAdds[name]++ }
func (t *testMetricsRecorder) RemoveMutex(_ context.Context, name, _ string)      { t.mutexRemoves[name]++ }
func (t *testMetricsRecorder) AddSemaphore(_ context.Context, name, _ string)     { t.semaphoreAdds[name]++ }
func (t *testMetricsRecorder) RemoveSemaphore(_ context.Context, name, _ string)  { t.semaphoreRemoves[name]++ }

func sum(m map[string]int) int {
	s := 0
	for _, v := range m {
		s += v
	}
	return s
}
