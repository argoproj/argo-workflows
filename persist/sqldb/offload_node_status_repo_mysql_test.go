//go:build !windows

package sqldb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// setupMySQLOffloadTest starts a MySQL or MariaDB container and returns an offload repository.
func setupMySQLOffloadTest(ctx context.Context, t *testing.T, v usqldb.MySQLVariant) OffloadNodeStatusRepo {
	t.Helper()
	repo, err := NewOffloadNodeStatusRepo(ctx, logging.RequireLoggerFromContext(ctx), setupMySQLTest(ctx, t, v), "test", "argo_workflows")
	require.NoError(t, err)
	return repo
}

// saveOffload writes a node status and returns the version it was stored under.
func saveOffload(ctx context.Context, t *testing.T, repo OffloadNodeStatusRepo, uid, nodeName string) string {
	t.Helper()
	version, err := repo.Save(ctx, uid, "argo", wfv1.Nodes{nodeName: wfv1.NodeStatus{ID: nodeName}})
	require.NoError(t, err)
	return version
}

// TestMySQLListOnlyReturnsRequestedKeys covers the behaviour the list path depends on: the
// query is scoped to the keys it is given, so a caller that needs one page of workflows does
// not pull every offloaded blob in the namespace.
func TestMySQLListOnlyReturnsRequestedKeys(t *testing.T) {
	for name, variant := range usqldb.MySQLVariants {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			repo := setupMySQLOffloadTest(ctx, t, variant)

			wantedA := UUIDVersion{UID: "uid-a", Version: saveOffload(ctx, t, repo, "uid-a", "node-a")}
			wantedB := UUIDVersion{UID: "uid-b", Version: saveOffload(ctx, t, repo, "uid-b", "node-b")}
			unwanted := UUIDVersion{UID: "uid-c", Version: saveOffload(ctx, t, repo, "uid-c", "node-c")}

			got, err := repo.List(ctx, "argo", []UUIDVersion{wantedA, wantedB})
			require.NoError(t, err)

			assert.Len(t, got, 2)
			assert.Contains(t, got, wantedA)
			assert.Contains(t, got, wantedB)
			assert.NotContains(t, got, unwanted, "a key that was not asked for must not come back")
		})
	}
}

// TestMySQLListExcludesSupersededVersions pins down why the version has to be part of the
// condition. Older offloads of the same workflow stay in the table until they are garbage
// collected, so filtering on the uid alone would drag those blobs back too.
// Dialect portability is already covered on both engines by TestMySQLListOnlyReturnsRequestedKeys,
// so one engine is enough here.
func TestMySQLListExcludesSupersededVersions(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	repo := setupMySQLOffloadTest(ctx, t, usqldb.MySQLVariants["MySQL"])

	oldVersion := saveOffload(ctx, t, repo, "uid-a", "node-old")
	newVersion := saveOffload(ctx, t, repo, "uid-a", "node-new")
	require.NotEqual(t, oldVersion, newVersion, "the two writes must produce different versions")

	got, err := repo.List(ctx, "argo", []UUIDVersion{{UID: "uid-a", Version: newVersion}})
	require.NoError(t, err)

	assert.Len(t, got, 1)
	assert.Contains(t, got, UUIDVersion{UID: "uid-a", Version: newVersion})
	assert.NotContains(t, got, UUIDVersion{UID: "uid-a", Version: oldVersion})
}
