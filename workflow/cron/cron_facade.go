package cron

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// cronFacade allows the client to operate using key rather than cron.EntryID,
// as well as providing sync guarantees
type cronFacade struct {
	mu       sync.Mutex
	cron     *cron.Cron
	entryIDs map[string][]cron.EntryID
}

type ScheduledTimeFunc func() time.Time

func newCronFacade() *cronFacade {
	return &cronFacade{
		cron:     cron.New(),
		entryIDs: make(map[string][]cron.EntryID),
	}
}

func (f *cronFacade) Start() {
	f.cron.Start()
}

func (f *cronFacade) Stop() {
	f.cron.Stop()
}

func (f *cronFacade) Delete(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entryIDs, ok := f.entryIDs[key]
	if !ok {
		return
	}
	for _, entryID := range entryIDs {
		f.cron.Remove(entryID)
	}
	delete(f.entryIDs, key)
}

func (f *cronFacade) AddJob(key, schedule string, cwoc *cronWfOperationCtx) (ScheduledTimeFunc, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entryID, err := f.cron.AddJob(schedule, cwoc)
	if err != nil {
		return nil, err
	}
	f.entryIDs[key] = append(f.entryIDs[key], entryID)

	// Return a function to return the last scheduled time.
	// If multiple schedules are configured, it will return
	// the most recent schedule time for the key
	return func() time.Time {
		f.mu.Lock()
		defer f.mu.Unlock()
		var t time.Time
		for _, entryID := range f.entryIDs[key] {
			prev := f.cron.Entry(entryID).Prev
			if prev.After(t) {
				t = prev
			}
		}
		return t
	}, nil
}

func (f *cronFacade) Load(key string) ([]*cronWfOperationCtx, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entryIDs, ok := f.entryIDs[key]
	if !ok {
		return nil, fmt.Errorf("entry ID for %s not found", key)
	}
	cwocs := make([]*cronWfOperationCtx, len(entryIDs))
	for i, entryID := range entryIDs {
		entry := f.cron.Entry(entryID).Job
		cwoc, ok := entry.(*cronWfOperationCtx)
		if !ok {
			return nil, fmt.Errorf("job entry ID for %s was not a *cronWfOperationCtx, was %v", key, reflect.TypeOf(entry))
		}
		cwocs[i] = cwoc
	}

	return cwocs, nil
}
