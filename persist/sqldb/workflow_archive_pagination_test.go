//go:build !windows

package sqldb

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// setupPostgresArchiveTest starts a PostgreSQL container, runs migrations, and returns a WorkflowArchive.
func setupPostgresArchiveTest(ctx context.Context, t *testing.T) WorkflowArchive {
	t.Helper()

	c, err := testpostgres.Run(ctx,
		"postgres:17.4-alpine",
		testpostgres.WithDatabase("argo"),
		testpostgres.WithUsername("argo"),
		testpostgres.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second),
				wait.ForListeningPort("5432/tcp"),
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
	p, err := c.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(p.Port())
	require.NoError(t, err)

	proxy, err := usqldb.NewSessionProxy(ctx, usqldb.SessionProxyConfig{
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

	err = Migrate(ctx, proxy.Session(ctx), "test", "argo_workflows", proxy.DBType())
	require.NoError(t, err)

	t.Cleanup(func() { proxy.Close() })

	return NewWorkflowArchive(proxy, "test", "", instanceid.NewService(""))
}

// TestArchivedWorkflowKeysetPagination pages through archived workflows using
// cursors where several workflows share a startedat and a page boundary falls
// between them. Every workflow must be returned exactly once, and the count
// for a cursor must only include the workflows after it.
func TestArchivedWorkflowKeysetPagination(t *testing.T) {
	archives := map[string]func(ctx context.Context, t *testing.T) WorkflowArchive{
		"Postgres": setupPostgresArchiveTest,
	}
	for name, variant := range usqldb.MySQLVariants {
		archives[name] = func(ctx context.Context, t *testing.T) WorkflowArchive {
			return setupMySQLArchiveTest(ctx, t, variant)
		}
	}

	for name, setup := range archives {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			archive := setup(ctx, t)

			// Five workflows; wf-1..wf-3 started at the same second, so with a
			// page size of 2 the first page boundary splits the tie.
			later := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			earlier := later.Add(-time.Minute)
			startedAt := []time.Time{later, later, later, earlier, earlier.Add(-time.Minute)}
			for i, s := range startedAt {
				err := archive.ArchiveWorkflow(ctx, &wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:              fmt.Sprintf("wf-%d", i+1),
						Namespace:         "default",
						UID:               types.UID(fmt.Sprintf("uid-%d", i+1)),
						CreationTimestamp: metav1.Time{Time: s},
						Labels:            map[string]string{"env": "test"},
					},
					Status: wfv1.WorkflowStatus{
						Phase:      wfv1.WorkflowSucceeded,
						StartedAt:  metav1.Time{Time: s},
						FinishedAt: metav1.Time{Time: s.Add(time.Second)},
					},
				})
				require.NoError(t, err)
			}
			want := []string{"uid-1", "uid-2", "uid-3", "uid-4", "uid-5"}

			envTest, err := labels.ParseToRequirements("env=test")
			require.NoError(t, err)
			for filterName, requirements := range map[string]labels.Requirements{
				"no label filter": nil,
				"label filter":    envTest,
			} {
				t.Run(filterName, func(t *testing.T) {
					options := sutils.ListOptions{Namespace: "default", Limit: 2, LabelRequirements: requirements}
					var got []string
					for range len(want) {
						count, err := archive.CountWorkflows(ctx, options)
						require.NoError(t, err)
						assert.Equal(t, int64(len(want)-len(got)), count, "count after %v", got)

						page, err := archive.ListWorkflows(ctx, options)
						require.NoError(t, err)
						for _, wf := range page {
							got = append(got, string(wf.UID))
						}
						if len(page) < options.Limit {
							break
						}
						last := page[len(page)-1]
						options.CursorStartedAt = last.Status.StartedAt.Time
						options.CursorUID = string(last.UID)
					}
					assert.Equal(t, want, got)
				})
			}
		})
	}
}
