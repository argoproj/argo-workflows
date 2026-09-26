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
// query matches whole (uid, version) pairs, so a caller that needs one page of workflows does
// not pull every offloaded blob in the namespace.
//
// Save deliberately leaves superseded rows behind for the garbage collector, so uid-a and uid-b
// each have two versions here and only one version of each is requested.
//
// The version is a hash of the node contents, so two workflows only share a version value when
// they store identical nodes. That is arranged deliberately: without it every version value is
// unique to one uid, and a `uid IN (...) AND version IN (...)` cross-product happens to return
// the right rows anyway. With shared version values the cross-product matches all four rows,
// while matching whole pairs returns two.
func TestMySQLListOnlyReturnsRequestedKeys(t *testing.T) {
	for name, variant := range usqldb.MySQLVariants {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			repo := setupMySQLOffloadTest(ctx, t, variant)

			versionX := saveOffload(ctx, t, repo, "uid-a", "node-x")
			versionY := saveOffload(ctx, t, repo, "uid-a", "node-y")
			require.NotEqual(t, versionX, versionY, "different nodes must produce different versions")
			require.Equal(t, versionX, saveOffload(ctx, t, repo, "uid-b", "node-x"), "identical nodes must share a version")
			require.Equal(t, versionY, saveOffload(ctx, t, repo, "uid-b", "node-y"), "identical nodes must share a version")
			unwanted := UUIDVersion{UID: "uid-c", Version: saveOffload(ctx, t, repo, "uid-c", "node-c")}

			wantedA := UUIDVersion{UID: "uid-a", Version: versionX}
			wantedB := UUIDVersion{UID: "uid-b", Version: versionY}

			got, err := repo.List(ctx, "argo", []UUIDVersion{wantedA, wantedB})
			require.NoError(t, err)

			assert.Len(t, got, 2)
			assert.Contains(t, got, wantedA)
			assert.Contains(t, got, wantedB)
			assert.NotContains(t, got, UUIDVersion{UID: "uid-a", Version: versionY}, "a version that was not asked for must not come back")
			assert.NotContains(t, got, UUIDVersion{UID: "uid-b", Version: versionX}, "a version that was not asked for must not come back")
			assert.NotContains(t, got, unwanted, "a uid that was not asked for must not come back")
		})
	}
}
