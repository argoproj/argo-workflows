//go:build !windows

package sqldb

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

func setupMySQLOffloadRepo(ctx context.Context, t *testing.T, v usqldb.MySQLVariant) *nodeOffloadRepo {
	t.Helper()

	c, err := testmysql.Run(ctx,
		v.Image,
		testmysql.WithDatabase("argo"),
		testmysql.WithUsername("argo"),
		testmysql.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog(v.WaitMessage).WithStartupTimeout(60*time.Second),
				wait.ForListeningPort("3306/tcp"),
			)),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(c); termErr != nil {
			t.Logf("failed to terminate container: %s", termErr)
		}
	})

	host, err := c.Host(ctx)
	require.NoError(t, err)
	p, err := c.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(p.Port())
	require.NoError(t, err)

	proxy, err := usqldb.NewSessionProxy(ctx, usqldb.SessionProxyConfig{
		DBConfig: config.DBConfig{
			MySQL: &config.MySQLConfig{
				DatabaseConfig: config.DatabaseConfig{
					Database: "argo",
					Host:     host,
					Port:     port,
				},
			},
		},
		Username: "argo",
		Password: "argo",
	})
	require.NoError(t, err)

	const (
		clusterName = "test"
		tableName   = "argo_workflows"
	)

	err = Migrate(ctx, proxy.Session(ctx), clusterName, tableName, proxy.DBType())
	require.NoError(t, err)

	t.Cleanup(func() { proxy.Close() })

	return &nodeOffloadRepo{
		sessionProxy: proxy,
		clusterName:  clusterName,
		tableName:    tableName,
		ttl:          5 * time.Minute,
		log:          logging.RequireLoggerFromContext(ctx),
	}
}

func TestMySQLOffloadNodeStatusRepo(t *testing.T) {
	for name, variant := range usqldb.MySQLVariants {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			repo := setupMySQLOffloadRepo(ctx, t, variant)

			uid := "mysql-test-wf"
			namespace := "default"
			nodes := wfv1.Nodes{"node1": wfv1.NodeStatus{Message: "msg1"}}

			// 1. Initial save
			v1, err := repo.Save(ctx, uid, namespace, nodes)
			require.NoError(t, err)

			// 2. Duplicate save in the exact same second:
			// In MySQL without clientFoundRows, this causes RowsAffected() == 0 because updatedat
			// was already CURRENT_TIMESTAMP. The implementation must check Exists() and NOT
			// return a false concurrent deletion error.
			v1Duplicate, err := repo.Save(ctx, uid, namespace, nodes)
			require.NoError(t, err, "duplicate Save in the same second must succeed in MySQL")
			assert.Equal(t, v1, v1Duplicate)

			// 3. DeleteIfOld preserves refreshed record:
			// Backdate the record past the TTL
			_, err = repo.sessionProxy.Session(ctx).SQL().Exec(
				`UPDATE argo_workflows
				 SET updatedat = CURRENT_TIMESTAMP - INTERVAL 10 MINUTE
				 WHERE clustername = ? AND uid = ? AND version = ?`,
				repo.clusterName, uid, v1,
			)
			require.NoError(t, err)

			// Verify ListOldOffloads finds it
			oldOffloads, err := repo.ListOldOffloads(ctx, namespace)
			require.NoError(t, err)
			require.Contains(t, oldOffloads[uid], v1)

			// Refresh timestamp with Save()
			reusedVersion, err := repo.Save(ctx, uid, namespace, nodes)
			require.NoError(t, err)
			assert.Equal(t, v1, reusedVersion)

			// DeleteIfOld must NOT delete the refreshed record
			err = repo.DeleteIfOld(ctx, uid, v1)
			require.NoError(t, err)

			// Verify still retrievable
			retrieved, err := repo.Get(ctx, uid, v1)
			require.NoError(t, err)
			assert.Equal(t, nodes, retrieved)

			// 4. DeleteIfOld deletes genuinely old record:
			_, err = repo.sessionProxy.Session(ctx).SQL().Exec(
				`UPDATE argo_workflows
				 SET updatedat = CURRENT_TIMESTAMP - INTERVAL 10 MINUTE
				 WHERE clustername = ? AND uid = ? AND version = ?`,
				repo.clusterName, uid, v1,
			)
			require.NoError(t, err)

			err = repo.DeleteIfOld(ctx, uid, v1)
			require.NoError(t, err)

			_, err = repo.Get(ctx, uid, v1)
			require.Error(t, err)

			// 5. Unconditional Delete
			v2, err := repo.Save(ctx, uid, namespace, wfv1.Nodes{"node2": wfv1.NodeStatus{Message: "msg2"}})
			require.NoError(t, err)
			err = repo.Delete(ctx, uid, v2)
			require.NoError(t, err)
			_, err = repo.Get(ctx, uid, v2)
			require.Error(t, err)
		})
	}
}

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
