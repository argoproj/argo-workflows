//go:build !windows

package sqldb

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// TestTemplateOffloadMigrationAndRepo runs the full schema migration against real Postgres and
// MySQL/MariaDB containers, then exercises the template repo end-to-end (save/get/delete/GC-query).
// It pins MySQL compatibility of the argo_offloaded_workflow_templates changes (MySQL has no
// CREATE INDEX IF NOT EXISTS, and such errors are not transaction-rolled back), Postgres jsonb
// conversion of the template column, and rerun-safety (the migration runs a second time).
func TestTemplateOffloadMigrationAndRepo(t *testing.T) {
	t.Run("Postgres", func(t *testing.T) {
		session := runPostgresMigrateTwice(t)
		exerciseTemplateRepo(t, session, "postgres")
	})
	for name, variant := range usqldb.MySQLVariants {
		t.Run(name, func(t *testing.T) {
			session := runMySQLMigrateTwice(t, variant)
			exerciseTemplateRepo(t, session, "mysql")
		})
	}
}

func postgresDBConfig(ctx context.Context, t *testing.T) (config.DBConfig, string, string, func()) {
	t.Helper()
	c, err := testpostgres.Run(ctx,
		"postgres:17.4-alpine",
		testpostgres.WithDatabase("argo"),
		testpostgres.WithUsername("argo"),
		testpostgres.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)
	host, err := c.Host(ctx)
	require.NoError(t, err)
	p, err := c.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(p.Port())
	require.NoError(t, err)
	return config.DBConfig{
		PostgreSQL: &config.PostgreSQLConfig{
			DatabaseConfig: config.DatabaseConfig{Database: "argo", Host: host, Port: port},
		},
	}, "argo", "argo", func() {
		if err := testcontainers.TerminateContainer(c); err != nil {
			t.Logf("terminate postgres: %s", err)
		}
	}
}

func runPostgresMigrateTwice(t *testing.T) *usqldb.SessionProxy {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	dbConfig, user, pass, term := postgresDBConfig(ctx, t)
	t.Cleanup(term)

	proxy, err := usqldb.NewSessionProxy(ctx, usqldb.SessionProxyConfig{
		DBConfig: dbConfig, Username: user, Password: pass,
	})
	require.NoError(t, err)
	t.Cleanup(func() { proxy.Close() })

	require.NoError(t, Migrate(ctx, proxy.Session(ctx), "test", "argo_workflows", proxy.DBType()),
		"first migration run must succeed")
	require.NoError(t, Migrate(ctx, proxy.Session(ctx), "test", "argo_workflows", proxy.DBType()),
		"migration must be idempotent")

	// The template column must have been converted to jsonb on Postgres.
	var typ string
	row, err := proxy.Session(ctx).SQL().
		QueryRow(`select data_type from information_schema.columns where table_name = 'argo_offloaded_workflow_templates' and column_name = 'template'`)
	require.NoError(t, err)
	require.NoError(t, row.Scan(&typ))
	assert.Equal(t, "jsonb", typ)

	return proxy
}

func runMySQLMigrateTwice(t *testing.T, variant usqldb.MySQLVariant) *usqldb.SessionProxy {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	c, err := testmysql.Run(ctx,
		variant.Image,
		testmysql.WithDatabase("argo"),
		testmysql.WithUsername("argo"),
		testmysql.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog(variant.WaitMessage).WithStartupTimeout(90*time.Second),
				wait.ForListeningPort("3306/tcp"),
			)),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(c); termErr != nil {
			t.Logf("terminate mysql: %s", termErr)
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
				DatabaseConfig: config.DatabaseConfig{Database: "argo", Host: host, Port: port},
			},
		},
		Username: "argo",
		Password: "argo",
	})
	require.NoError(t, err)
	t.Cleanup(func() { proxy.Close() })

	// Regression guard: the migration must succeed and be rerunnable on MySQL, which rejects
	// CREATE INDEX IF NOT EXISTS with a syntax error.
	require.NoError(t, Migrate(ctx, proxy.Session(ctx), "test", "argo_workflows", proxy.DBType()),
		"first migration run must succeed on MySQL")
	require.NoError(t, Migrate(ctx, proxy.Session(ctx), "test", "argo_workflows", proxy.DBType()),
		"migration must be idempotent on MySQL")

	return proxy
}

func exerciseTemplateRepo(t *testing.T, proxy *usqldb.SessionProxy, dbType string) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	repo := NewTemplateRepo(ctx, logging.RequireLoggerFromContext(ctx), proxy, "test", TemplateOffloadTableName, 30*time.Second)
	require.True(t, repo.IsEnabled())

	templates := []wfv1.Template{
		{Name: "whalesay", Container: &corev1.Container{Image: "docker/whalesay:latest"}},
		{Name: "sleep", Container: &corev1.Container{Image: "alpine:latest", Command: []string{"sleep", "1"}}},
	}

	require.NoError(t, repo.SaveTemplates(ctx, "uid-abc", "default", templates))

	got, err := repo.GetTemplates(ctx, "uid-abc")
	require.NoError(t, err)
	require.Len(t, got, 2)
	names := map[string]bool{}
	for _, tmpl := range got {
		names[tmpl.Name] = true
		if tmpl.Name == "whalesay" {
			require.NotNil(t, tmpl.Container)
			assert.Equal(t, "docker/whalesay:latest", tmpl.Container.Image)
		}
	}
	assert.True(t, names["whalesay"])
	assert.True(t, names["sleep"])

	// SaveTemplates is delete+insert: re-saving replaces, never duplicates.
	require.NoError(t, repo.SaveTemplates(ctx, "uid-abc", "default", templates[:1]))
	got, err = repo.GetTemplates(ctx, "uid-abc")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "whalesay", got[0].Name)

	// GC query uses the interval syntax + createdat column. Age boundaries are driven by rewriting
	// createdat in-DB rather than negative ages or sleeping: the app writes createdat in its local
	// zone into a `timestamp without time zone` while the query compares against the DB server's
	// current_timestamp.
	uids, err := repo.ListOldOffloads(ctx, time.Hour)
	require.NoError(t, err)
	assert.Empty(t, uids, "just-written rows must not be reported as old")

	// Age 6h: clears any realistic app/DB zone skew (host UTC+2 vs container UTC).
	if dbType == "mysql" {
		_, err = proxy.Session(ctx).SQL().Exec(`update argo_offloaded_workflow_templates set createdat = createdat - interval 6 hour where uid = 'uid-abc'`)
	} else {
		_, err = proxy.Session(ctx).SQL().Exec(`update argo_offloaded_workflow_templates set createdat = createdat - interval '6 hour' where uid = 'uid-abc'`)
	}
	require.NoError(t, err)

	uids, err = repo.ListOldOffloads(ctx, time.Hour)
	require.NoError(t, err)
	assert.Contains(t, uids, "uid-abc", "rows aged 6h must be reported for a 1h GC age")

	require.NoError(t, repo.DeleteTemplates(ctx, "uid-abc"))
	got, err = repo.GetTemplates(ctx, "uid-abc")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestMySQLRejectsCreateIndexIfNotExists pins the dialect constraint behind the skipIfIndexExists
// migration pattern: MySQL (unlike MariaDB and Postgres) has no CREATE INDEX IF NOT EXISTS, and the
// error is not transaction-rolled back, while initDB runs the migration unconditionally. If MySQL
// adds the syntax, this test failing is the signal that the guard can be simplified.
func TestMySQLRejectsCreateIndexIfNotExists(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	c, err := testmysql.Run(ctx,
		"mysql:8.4",
		testmysql.WithDatabase("argo"),
		testmysql.WithUsername("argo"),
		testmysql.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("MySQL Community Server").WithStartupTimeout(90*time.Second),
				wait.ForListeningPort("3306/tcp"),
			)),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(c); termErr != nil {
			t.Logf("terminate mysql: %s", termErr)
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
				DatabaseConfig: config.DatabaseConfig{Database: "argo", Host: host, Port: port},
			},
		},
		Username: "argo",
		Password: "argo",
	})
	require.NoError(t, err)
	t.Cleanup(func() { proxy.Close() })

	_, err = proxy.Session(ctx).SQL().Exec("create table if not exists syntax_probe (a int, b int)")
	require.NoError(t, err)

	_, err = proxy.Session(ctx).SQL().Exec("create index if not exists idx_syntax_probe on syntax_probe (a)")
	require.Error(t, err, "MySQL must reject CREATE INDEX IF NOT EXISTS (syntax probe)")
	assert.Contains(t, strings.ToUpper(err.Error()), "1064")

	// The replacement pattern must work.
	_, err = proxy.Session(ctx).SQL().Exec("create index idx_syntax_probe2 on syntax_probe (a, b)")
	assert.NoError(t, err)
}
