//go:build !windows

package sqldb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// TestMySQLListOldOffloadsInstanceID verifies that controllers with different instance IDs sharing a database and
// cluster name only garbage collect their own offloaded node status records. #17387
func TestMySQLListOldOffloadsInstanceID(t *testing.T) {
	t.Setenv("OFFLOAD_NODE_STATUS_TTL", "1s")
	for name, variant := range usqldb.MySQLVariants {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			proxy := setupMySQLSessionProxy(ctx, t, variant)
			log := logging.RequireLoggerFromContext(ctx)

			repoA, err := NewOffloadNodeStatusRepo(ctx, log, proxy, "test", "", "argo_workflows")
			require.NoError(t, err)
			repoB, err := NewOffloadNodeStatusRepo(ctx, log, proxy, "test", "b", "argo_workflows")
			require.NoError(t, err)

			nodes := wfv1.Nodes{"my-node": wfv1.NodeStatus{}}
			_, err = repoA.Save(ctx, "uid-a", "default", nodes)
			require.NoError(t, err)
			_, err = repoB.Save(ctx, "uid-b", "default", nodes)
			require.NoError(t, err)
			// a record written before the instanceid column existed
			_, err = proxy.Session(ctx).SQL().Exec(`insert into argo_workflows (clustername, uid, version, namespace, nodes) values ('test', 'uid-legacy', 'fnv:1', 'default', '{}')`)
			require.NoError(t, err)

			time.Sleep(2500 * time.Millisecond)

			oldA, err := repoA.ListOldOffloads(ctx, "")
			require.NoError(t, err)
			assert.Contains(t, oldA, "uid-a")
			assert.Contains(t, oldA, "uid-legacy")
			assert.NotContains(t, oldA, "uid-b")

			oldB, err := repoB.ListOldOffloads(ctx, "")
			require.NoError(t, err)
			assert.Contains(t, oldB, "uid-b")
			assert.Contains(t, oldB, "uid-legacy")
			assert.NotContains(t, oldB, "uid-a")

			_, err = repoA.Get(ctx, "uid-legacy", "fnv:1")
			require.NoError(t, err)
		})
	}
}
