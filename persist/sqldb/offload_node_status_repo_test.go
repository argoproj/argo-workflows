package sqldb

import (
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testcontainers "github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

func Test_nodeStatusVersion(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		marshalled, version, err := nodeStatusVersion(nil)
		require.NoError(t, err)
		assert.NotEmpty(t, marshalled)
		assert.Equal(t, "fnv:784127654", version)
	})
	t.Run("NonEmpty", func(t *testing.T) {
		marshalled, version, err := nodeStatusVersion(wfv1.Nodes{"my-node": wfv1.NodeStatus{}})
		require.NoError(t, err)
		assert.NotEmpty(t, marshalled)
		assert.Equal(t, "fnv:2308444803", version)
	})
}

func setupOffloadNodeStatusRepo(t *testing.T) *nodeOffloadRepo {
	t.Helper()

	ctx := logging.TestContext(t.Context())

	postgresContainer, err := testpostgres.Run(ctx,
		"postgres:17.4-alpine",
		testpostgres.WithDatabase("argo"),
		testpostgres.WithUsername("argo"),
		testpostgres.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		terminateErr := testcontainers.TerminateContainer(postgresContainer)
		if terminateErr != nil {
			t.Logf("failed to terminate postgres container: %s", terminateErr)
		}
	})

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)

	mappedPort, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	port, err := strconv.Atoi(mappedPort.Port())
	require.NoError(t, err)

	sessionProxy, err := usqldb.NewSessionProxy(ctx, usqldb.SessionProxyConfig{
		DBConfig: config.DBConfig{
			PostgreSQL: &config.PostgreSQLConfig{
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

	t.Cleanup(func() {
		closeErr := sessionProxy.Close()
		if closeErr != nil {
			t.Logf("failed to close database session: %s", closeErr)
		}
	})

	const (
		clusterName = "test"
		tableName   = "argo_workflows"
	)

	err = Migrate(ctx, sessionProxy.Session(ctx), clusterName, tableName, sessionProxy.DBType())
	require.NoError(t, err)

	return &nodeOffloadRepo{
		sessionProxy: sessionProxy,
		clusterName:  clusterName,
		tableName:    tableName,
		ttl:          5 * time.Minute,
		log:          logging.RequireLoggerFromContext(ctx),
	}
}

func TestOffloadNodeStatusRepoSaveRefreshesTimestampOnDuplicate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Testcontainers are not supported on Windows")
	}

	ctx := logging.TestContext(t.Context())
	repo := setupOffloadNodeStatusRepo(t)

	uid := "test-workflow-uid"
	namespace := "default"

	nodesA := wfv1.Nodes{
		"my-node": wfv1.NodeStatus{
			Message: "ErrImagePull",
		},
	}

	nodesB := wfv1.Nodes{
		"my-node": wfv1.NodeStatus{
			Message: "ImagePullBackOff",
		},
	}

	// Save version A.
	versionA, err := repo.Save(ctx, uid, namespace, nodesA)
	require.NoError(t, err)

	// Save version B.
	versionB, err := repo.Save(ctx, uid, namespace, nodesB)
	require.NoError(t, err)

	assert.NotEqual(t, versionA, versionB)

	// Make version A old by backdating its updatedat timestamp.
	_, err = repo.sessionProxy.Session(ctx).SQL().Exec(
		`UPDATE argo_workflows
         SET updatedat = CURRENT_TIMESTAMP - INTERVAL '10 minutes'
         WHERE clustername = ? AND uid = ? AND version = ?`,
		repo.clusterName,
		uid,
		versionA,
	)
	require.NoError(t, err)

	// Verify version A is considered old.
	oldOffloads, err := repo.ListOldOffloads(ctx, namespace)
	require.NoError(t, err)
	assert.Contains(t, oldOffloads[uid], versionA)

	// Save version A again. This should refresh updatedat.
	reusedVersion, err := repo.Save(ctx, uid, namespace, nodesA)
	require.NoError(t, err)

	assert.Equal(t, versionA, reusedVersion)

	// Verify version A is no longer considered old.
	oldOffloads, err = repo.ListOldOffloads(ctx, namespace)
	require.NoError(t, err)
	assert.NotContains(t, oldOffloads[uid], versionA)

	// Verify the existing offloaded nodes are still retrievable.
	retrievedNodes, err := repo.Get(ctx, uid, versionA)
	require.NoError(t, err)
	assert.Equal(t, nodesA, retrievedNodes)
}

func TestDeleteIfOldPreservesRefreshedRecord(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Testcontainers are not supported on Windows")
	}

	ctx := logging.TestContext(t.Context())
	repo := setupOffloadNodeStatusRepo(t)

	uid := "test-wf-preserve"
	namespace := "default"
	nodes := wfv1.Nodes{"node1": wfv1.NodeStatus{Message: "running"}}

	// Save the record.
	version, err := repo.Save(ctx, uid, namespace, nodes)
	require.NoError(t, err)

	// Backdate its timestamp so ListOldOffloads sees it.
	_, err = repo.sessionProxy.Session(ctx).SQL().Exec(
		`UPDATE argo_workflows
         SET updatedat = CURRENT_TIMESTAMP - INTERVAL '10 minutes'
         WHERE clustername = ? AND uid = ? AND version = ?`,
		repo.clusterName, uid, version,
	)
	require.NoError(t, err)

	// GC lists old offloads and captures the version.
	oldOffloads, err := repo.ListOldOffloads(ctx, namespace)
	require.NoError(t, err)
	require.Contains(t, oldOffloads[uid], version)

	// Before GC deletes: Save() re-encounters the same content, refreshing updatedat.
	reusedVersion, err := repo.Save(ctx, uid, namespace, nodes)
	require.NoError(t, err)
	require.Equal(t, version, reusedVersion)

	// GC now calls DeleteIfOld with the previously listed version.
	// Because updatedat was refreshed, the TTL condition is false and no row is deleted.
	err = repo.DeleteIfOld(ctx, uid, version)
	require.NoError(t, err)

	// The record must still be retrievable.
	retrievedNodes, err := repo.Get(ctx, uid, version)
	require.NoError(t, err)
	assert.Equal(t, nodes, retrievedNodes)
}

func TestDeleteIfOldRemovesGenuinelyOldRecord(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Testcontainers are not supported on Windows")
	}

	ctx := logging.TestContext(t.Context())
	repo := setupOffloadNodeStatusRepo(t)

	uid := "test-wf-old"
	namespace := "default"
	nodes := wfv1.Nodes{"node1": wfv1.NodeStatus{Message: "done"}}

	// Save and backdate past the TTL.
	version, err := repo.Save(ctx, uid, namespace, nodes)
	require.NoError(t, err)

	_, err = repo.sessionProxy.Session(ctx).SQL().Exec(
		`UPDATE argo_workflows
         SET updatedat = CURRENT_TIMESTAMP - INTERVAL '10 minutes'
         WHERE clustername = ? AND uid = ? AND version = ?`,
		repo.clusterName, uid, version,
	)
	require.NoError(t, err)

	// DeleteIfOld should delete a record that is still past the TTL.
	err = repo.DeleteIfOld(ctx, uid, version)
	require.NoError(t, err)

	// The record must no longer be found.
	_, err = repo.Get(ctx, uid, version)
	require.Error(t, err)
}

func TestDeleteUnconditionallyDeletesRegardlessOfAge(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Testcontainers are not supported on Windows")
	}

	ctx := logging.TestContext(t.Context())
	repo := setupOffloadNodeStatusRepo(t)

	uid := "test-wf-unconditional"
	namespace := "default"
	nodes := wfv1.Nodes{"node1": wfv1.NodeStatus{Message: "done"}}

	// Save without backdating – record is fresh.
	version, err := repo.Save(ctx, uid, namespace, nodes)
	require.NoError(t, err)

	// Unconditional Delete should still remove it even though it's not old.
	err = repo.Delete(ctx, uid, version)
	require.NoError(t, err)

	_, err = repo.Get(ctx, uid, version)
	require.Error(t, err)
}

func TestSaveDuplicateZeroRowsReturnsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Testcontainers are not supported on Windows")
	}

	ctx := logging.TestContext(t.Context())
	repo := setupOffloadNodeStatusRepo(t)

	uid := "test-wf-zero-rows"
	namespace := "default"
	nodes := wfv1.Nodes{"node1": wfv1.NodeStatus{Message: "initial"}}

	// Save the initial record.
	version, err := repo.Save(ctx, uid, namespace, nodes)
	require.NoError(t, err)

	// Install a BEFORE UPDATE trigger in PostgreSQL that deletes the row and suppresses
	// the update. This realistically simulates a concurrent GC deletion occurring
	// between Save()'s failed duplicate INSERT and its subsequent timestamp-refresh UPDATE.
	_, err = repo.sessionProxy.Session(ctx).SQL().Exec(`
		CREATE OR REPLACE FUNCTION drop_row_before_update() RETURNS TRIGGER AS $$
		BEGIN
			DELETE FROM argo_workflows WHERE clustername = OLD.clustername AND uid = OLD.uid AND version = OLD.version;
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER trg_test_delete
		BEFORE UPDATE ON argo_workflows
		FOR EACH ROW EXECUTE FUNCTION drop_row_before_update();
	`)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = repo.sessionProxy.Session(ctx).SQL().Exec(`
			DROP TRIGGER IF EXISTS trg_test_delete ON argo_workflows;
			DROP FUNCTION IF EXISTS drop_row_before_update();
		`)
	})

	// Call Save() again with the same nodes.
	// The INSERT will fail with duplicate key, and the UPDATE will hit the trigger,
	// resulting in zero rows affected and the record no longer existing.
	// Save() must return a meaningful error rather than claiming success.
	_, err = repo.Save(ctx, uid, namespace, nodes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disappeared during duplicate-save: concurrent GC may have deleted it")

	// Verify the record is indeed gone.
	_, err = repo.Get(ctx, uid, version)
	require.Error(t, err)
}
