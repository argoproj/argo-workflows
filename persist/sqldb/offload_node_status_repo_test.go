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
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Logf("failed to terminate postgres container: %s", err)
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
		if err := sessionProxy.Close(); err != nil {
			t.Logf("failed to close database session: %s", err)
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
