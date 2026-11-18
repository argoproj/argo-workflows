//go:build !windows

package db_test

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
	"github.com/upper/db/v4"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const (
	testDBName     = "memotest"
	testDBUser     = "user"
	testDBPassword = "pass"
	testNamespace  = "default"
	testCacheName  = "my-cache"
)

var testTableName = memodb.TableName(nil)

func setupPostgres(ctx context.Context, t *testing.T) *sqldb.SessionProxy {
	t.Helper()
	pg, err := testpostgres.Run(ctx, "postgres:17.4-alpine",
		testpostgres.WithDatabase(testDBName), testpostgres.WithUsername(testDBUser), testpostgres.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, testcontainers.TerminateContainer(pg)) })
	host, err := pg.Host(ctx)
	require.NoError(t, err)
	portStr, err := pg.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr.Port())
	require.NoError(t, err)
	return connectTestDB(ctx, t, config.DBConfig{PostgreSQL: &config.PostgreSQLConfig{DatabaseConfig: config.DatabaseConfig{Host: host, Port: port, Database: testDBName}}})
}

func setupMySQL(ctx context.Context, t *testing.T) *sqldb.SessionProxy {
	t.Helper()
	my, err := testmysql.Run(ctx, "mysql:8.4.5", testmysql.WithDatabase(testDBName), testmysql.WithUsername(testDBUser), testmysql.WithPassword(testDBPassword))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, testcontainers.TerminateContainer(my)) })
	host, err := my.Host(ctx)
	require.NoError(t, err)
	portStr, err := my.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr.Port())
	require.NoError(t, err)
	return connectTestDB(ctx, t, config.DBConfig{MySQL: &config.MySQLConfig{DatabaseConfig: config.DatabaseConfig{Host: host, Port: port, Database: testDBName}}})
}

func connectTestDB(ctx context.Context, t *testing.T, cfg config.DBConfig) *sqldb.SessionProxy {
	t.Helper()
	credentials := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "memo-db", Namespace: testNamespace},
		Data:       map[string][]byte{"username": []byte(testDBUser), "password": []byte(testDBPassword)},
	}
	var database *config.DatabaseConfig
	if cfg.PostgreSQL != nil {
		database = &cfg.PostgreSQL.DatabaseConfig
	} else {
		database = &cfg.MySQL.DatabaseConfig
	}
	database.UsernameSecret = apiv1.SecretKeySelector{LocalObjectReference: apiv1.LocalObjectReference{Name: credentials.Name}, Key: "username"}
	database.PasswordSecret = apiv1.SecretKeySelector{LocalObjectReference: apiv1.LocalObjectReference{Name: credentials.Name}, Key: "password"}
	cfg.ConnectionPool = &config.ConnectionPool{MaxOpenConns: 4, MaxIdleConns: 2, ConnMaxLifetime: config.TTL(time.Minute)}
	sp, err := memodb.SessionProxyFromConfig(ctx, fake.NewClientset(credentials), testNamespace, &config.MemoizationConfig{DBConfig: cfg})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sp.Close()) })
	assert.Equal(t, 4, sp.Session(ctx).MaxOpenConns())
	assert.Equal(t, 2, sp.Session(ctx).MaxIdleConns())
	assert.Equal(t, time.Minute, sp.Session(ctx).ConnMaxLifetime())
	require.NoError(t, memodb.Migrate(ctx, sp, memodb.Config{TableName: testTableName}))
	return sp
}

func sampleOutputs(message string) *wfv1.Outputs {
	return &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "result", Value: wfv1.AnyStringPtr(message)}}}
}

func TestQueries(t *testing.T) {
	for name, setup := range map[string]func(context.Context, *testing.T) *sqldb.SessionProxy{"postgres": setupPostgres, "mysql": setupMySQL} {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			sp := setup(ctx, t)
			q, err := memodb.NewQueries(testTableName, sp)
			require.NoError(t, err)
			assert.True(t, q.IsEnabled())
			require.NoError(t, memodb.Migrate(ctx, sp, memodb.Config{TableName: testTableName}), "migration is repeatable")

			t.Run("custom-table", func(t *testing.T) {
				cfg := memodb.Config{TableName: "custom_cache"}
				require.NoError(t, memodb.Migrate(ctx, sp, cfg))
				require.NoError(t, memodb.Migrate(ctx, sp, cfg), "custom table migration is repeatable")
				custom, err := memodb.NewQueries(cfg.TableName, sp)
				require.NoError(t, err)
				require.NoError(t, custom.Save(ctx, testNamespace, testCacheName, "custom-key", "node", sampleOutputs("custom"), ""))
				record, err := custom.Load(ctx, testNamespace, testCacheName, "custom-key")
				require.NoError(t, err)
				require.NotNil(t, record)
				assert.Contains(t, record.Outputs, "custom")
				record, err = q.Load(ctx, testNamespace, testCacheName, "custom-key")
				require.NoError(t, err)
				assert.Nil(t, record, "custom and default tables are independent")
			})
			t.Run("database-errors", func(t *testing.T) {
				missing, err := memodb.NewQueries("missing_cache", sp)
				require.NoError(t, err)
				record, err := missing.Load(ctx, testNamespace, testCacheName, "key")
				require.Error(t, err)
				assert.Nil(t, record)
				require.Error(t, missing.Save(ctx, testNamespace, testCacheName, "key", "node", nil, ""))
				pruned, err := missing.Prune(ctx)
				require.Error(t, err)
				assert.Zero(t, pruned)
				// A reserved SQL identifier passes the name validation but fails DDL.
				require.Error(t, memodb.Migrate(ctx, sp, memodb.Config{TableName: "select"}))
			})

			t.Run("save-load-replace", func(t *testing.T) {
				rec, err := q.Load(ctx, testNamespace, testCacheName, "key")
				require.NoError(t, err)
				assert.Nil(t, rec)
				for _, value := range []string{"first", "replacement"} {
					require.NoError(t, q.Save(ctx, testNamespace, testCacheName, "key", value, sampleOutputs(value), "1h"))
					rec, err = q.Load(ctx, testNamespace, testCacheName, "key")
					require.NoError(t, err)
					require.NotNil(t, rec)
					assert.Equal(t, value, rec.NodeID)
					assert.Contains(t, rec.Outputs, value)
					require.NotNil(t, rec.ExpiresAt)
					assert.WithinDuration(t, rec.CreatedAt.Add(time.Hour), *rec.ExpiresAt, time.Second)
				}
			})
			t.Run("namespace-isolation", func(t *testing.T) {
				for _, ns := range []string{"controller-a", "controller-b"} {
					require.NoError(t, q.Save(ctx, ns, testCacheName, "shared-key", ns, sampleOutputs(ns), ""))
				}
				for _, ns := range []string{"controller-a", "controller-b"} {
					rec, err := q.Load(ctx, ns, testCacheName, "shared-key")
					require.NoError(t, err)
					require.NotNil(t, rec)
					assert.Equal(t, ns, rec.NodeID)
				}
			})
			t.Run("no-default-expiry", func(t *testing.T) {
				require.NoError(t, q.Save(ctx, testNamespace, testCacheName, "no-expiry", "node", nil, ""))
				require.NoError(t, sp.With(ctx, func(sess db.Session) error {
					return sess.Collection(testTableName).Find(db.Cond{"cache_key": "no-expiry"}).Update(map[string]any{"created_at": time.Now().Add(-90 * 24 * time.Hour)})
				}))
				n, err := q.Prune(ctx)
				require.NoError(t, err)
				assert.Zero(t, n)
				rec, err := q.Load(ctx, testNamespace, testCacheName, "no-expiry")
				require.NoError(t, err)
				require.NotNil(t, rec)
				assert.Nil(t, rec.ExpiresAt)
				assert.Equal(t, "null", rec.Outputs)
				// Replacing an entry also removes a previously explicit expiry.
				require.NoError(t, q.Save(ctx, testNamespace, testCacheName, "no-expiry", "node", nil, "1h"))
				require.NoError(t, q.Save(ctx, testNamespace, testCacheName, "no-expiry", "node", nil, ""))
				rec, err = q.Load(ctx, testNamespace, testCacheName, "no-expiry")
				require.NoError(t, err)
				require.NotNil(t, rec)
				assert.Nil(t, rec.ExpiresAt)
			})
			t.Run("explicit-expiry-and-prune", func(t *testing.T) {
				for key, age := range map[string]string{"expired": "-1s", "zero": "0"} {
					require.NoError(t, q.Save(ctx, testNamespace, testCacheName, key, "node", sampleOutputs("old"), age))
					// MySQL's second-precision timestamp can round a zero-age expiry forward.
					require.EventuallyWithT(t, func(c *assert.CollectT) {
						rec, err := q.Load(ctx, testNamespace, testCacheName, key)
						require.NoError(c, err)
						assert.Nil(c, rec)
					}, 2*time.Second, 10*time.Millisecond)
				}
				n, err := q.Prune(ctx)
				require.NoError(t, err)
				assert.EqualValues(t, 2, n)
				n, err = q.Prune(ctx)
				require.NoError(t, err)
				assert.Zero(t, n)
			})
			t.Run("invalid-max-age", func(t *testing.T) {
				require.ErrorContains(t, q.Save(ctx, testNamespace, testCacheName, "invalid", "node", nil, "bad"), "invalid maxAge")
			})
			t.Run("large-outputs", func(t *testing.T) {
				value := strings.Repeat("x", 1024*1024+1) + "\x00"
				require.NoError(t, q.Save(ctx, testNamespace, testCacheName, "large", "node", sampleOutputs(value), ""))
				rec, err := q.Load(ctx, testNamespace, testCacheName, "large")
				require.NoError(t, err)
				require.NotNil(t, rec)
				var outputs wfv1.Outputs
				wfv1.MustUnmarshal([]byte(rec.Outputs), &outputs)
				assert.Equal(t, value, outputs.Parameters[0].Value.String())
			})
		})
	}
}
