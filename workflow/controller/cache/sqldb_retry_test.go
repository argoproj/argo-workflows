package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

type retryingMemoDB struct {
	memodb.MemoizationDB
	load func() (*memodb.CacheRecord, error)
	save func() error
}

func (q *retryingMemoDB) Load(context.Context, string, string, string) (*memodb.CacheRecord, error) {
	return q.load()
}
func (q *retryingMemoDB) Save(context.Context, string, string, string, string, *wfv1.Outputs, string) error {
	return q.save()
}

func TestSQLCacheRetriesTransientFailures(t *testing.T) {
	for _, operation := range []string{"load", "save"} {
		t.Run(operation, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			calls := 0
			attempt := func() error {
				calls++
				if calls == 1 {
					return fmt.Errorf("failover: %w", &pq.Error{Code: "57P03", Message: "database restarting"})
				}
				return nil
			}
			q := &retryingMemoDB{save: attempt, load: func() (*memodb.CacheRecord, error) {
				err := attempt()
				if err != nil {
					return nil, err
				}
				return &memodb.CacheRecord{NodeID: "node", Outputs: "null"}, nil
			}}
			c := newSQLDBCache("ns", "cache", func(fn func(memodb.MemoizationDB) error) error { return fn(q) })
			if operation == "load" {
				entry, err := c.Load(ctx, "key")
				require.NoError(t, err)
				require.NotNil(t, entry)
				assert.Equal(t, "node", entry.NodeID)
			} else {
				require.NoError(t, c.Save(ctx, "key", "node", nil, ""))
			}
			assert.Equal(t, 2, calls)
		})
	}
}

func TestMemoizationRetryStopsOnPermanentErrorOrCancellation(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	calls := 0
	denied := errors.New("permission denied")
	require.ErrorIs(t, retryMemoization(ctx, func() error { calls++; return denied }), denied)
	assert.Equal(t, 1, calls)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	calls = 0
	require.ErrorIs(t, retryMemoization(ctx, func() error { calls++; cancel(); return &pq.Error{Code: "57P01"} }), context.Canceled)
	assert.Equal(t, 1, calls)
}
