//go:build !windows

package controller

import (
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/config"
	persist "github.com/argoproj/argo-workflows/v4/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// A real database outage must not leave completed workflows permanently Pending
// after the proxy exhausts its reconnect budget. Kubernetes is a fake client here;
// this checks the archive path and TTL eligibility, not deletion by a live cluster.
func TestWorkflowController_ArchiveRecoversAfterDatabaseOutage(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := logging.TestContext(t.Context())
	postgres, err := testpostgres.Run(ctx, "postgres:17.4-alpine",
		testpostgres.WithDatabase("archive_recovery"),
		testpostgres.WithUsername("argo"),
		testpostgres.WithPassword("argo"),
		testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			// Keep the same endpoint after Stop/Start; util/sqldb tests use 15432.
			hostConfig.PortBindings = network.PortMap{
				network.MustParsePort("5432/tcp"): {{
					HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: "15433",
				}},
			}
		}),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(30*time.Second)),
	)
	testcontainers.CleanupContainer(t, postgres)
	require.NoError(t, err)
	host, err := postgres.Host(ctx)
	require.NoError(t, err)
	mappedPort, err := postgres.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(mappedPort.Port())
	require.NoError(t, err)
	cfg := config.DBConfig{PostgreSQL: &config.PostgreSQLConfig{
		DatabaseConfig: config.DatabaseConfig{Database: "archive_recovery", Host: host, Port: port},
	}}
	proxy, err := usqldb.NewSessionProxy(ctx, usqldb.SessionProxyConfig{
		DBConfig: cfg, Username: "argo", Password: "argo",
		MaxRetries: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, proxy.Close()) })
	require.NoError(t, persist.Migrate(ctx, proxy.Session(), "test", "argo_workflows", proxy.DBType()))
	archive := persist.NewWorkflowArchive(proxy, "test", "", instanceid.NewService(""))

	wf := pendingArchiveWorkflow()
	wf.UID = types.UID("archive-recovery-16771")
	wf.CreationTimestamp = metav1.NewTime(time.Now().Add(-3 * time.Minute))
	wf.Status.StartedAt = metav1.NewTime(time.Now().Add(-2 * time.Minute))
	wf.Status.FinishedAt = metav1.NewTime(time.Now().Add(-time.Minute))
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: new(int32(1))}
	cancel, controller := newController(ctx, wf, func(wfc *WorkflowController) {
		wfc.wfArchive = archive
	})
	t.Cleanup(cancel)
	t.Cleanup(controller.wfQueue.ShutDown)
	t.Cleanup(controller.wfArchiveQueue.ShutDown)
	un, err := util.ToUnstructured(wf)
	require.NoError(t, err)
	require.False(t, common.IsDone(un), "Pending archiving blocks the TTL controller")

	stopTimeout := time.Second
	require.NoError(t, postgres.Stop(ctx, &stopTimeout))
	err = controller.archiveWorkflowAux(ctx, un)
	require.ErrorContains(t, err, "reconnection failed after 1 retries")
	t.Logf("archive attempt exhausted the reconnect budget during PostgreSQL outage: %v", err)
	pending, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "Pending", pending.Labels[common.LabelKeyWorkflowArchivingStatus])
	pendingUn, err := util.ToUnstructured(pending)
	require.NoError(t, err)
	require.False(t, common.IsDone(pendingUn))

	require.NoError(t, postgres.Start(ctx))
	restartedPort, err := postgres.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	require.Equal(t, mappedPort, restartedPort)
	// Check the restored database independently, without reconnecting or replacing
	// the proxy, archive, or controller used by the failed operation.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		sess, _, connectErr := usqldb.CreateDBSessionWithCreds(cfg, "argo", "argo")
		if assert.NoError(c, connectErr) {
			assert.NoError(c, sess.Ping())
			assert.NoError(c, sess.Close())
		}
	}, 10*time.Second, 100*time.Millisecond)
	require.NoError(t, controller.archiveWorkflowAux(ctx, un))
	archived, err := archive.GetWorkflow(ctx, string(wf.UID), wf.Namespace, wf.Name)
	require.NoError(t, err)
	require.NotNil(t, archived)
	assert.Equal(t, wf.UID, archived.UID)
	assert.Equal(t, wf.Name, archived.Name)
	assert.Equal(t, wfv1.WorkflowSucceeded, archived.Status.Phase)

	updated, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "Archived", updated.Labels[common.LabelKeyWorkflowArchivingStatus])
	updatedUn, err := util.ToUnstructured(updated)
	require.NoError(t, err)
	require.True(t, common.IsDone(updatedUn), "successful archiving removes the TTL admission blocker")
	require.True(t, updated.Status.Successful())
	require.True(t, time.Now().After(updated.Status.FinishedAt.Add(time.Duration(*updated.GetTTLStrategy().SecondsAfterSuccess)*time.Second)))
	t.Logf("same controller and SessionProxy archived UID %s; workflow is Archived and TTL-eligible", archived.UID)
}
