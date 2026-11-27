//go:build !windows

package db_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/retry"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	sqldbtest "github.com/argoproj/argo-workflows/v4/test/util/sqldb"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const (
	testNamespace = "default"
	testCacheName = "my-cache"
)

var testTableName = memodb.TableName(nil)

func connectTestDB(ctx context.Context, t *testing.T, cfg config.DBConfig) *sqldb.SessionProxy {
	t.Helper()
	credentials := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "memo-db", Namespace: testNamespace},
		Data:       map[string][]byte{"username": []byte(sqldbtest.Username), "password": []byte(sqldbtest.Password)},
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
	backends := map[string]func(context.Context, *testing.T) config.DBConfig{"Postgres": sqldbtest.Postgres}
	for name, variant := range sqldb.MySQLVariants {
		backends[name] = func(ctx context.Context, t *testing.T) config.DBConfig { return sqldbtest.MySQL(ctx, t, variant) }
	}
	backends["MySQL57Strict"] = func(ctx context.Context, t *testing.T) config.DBConfig {
		return sqldbtest.MySQL(ctx, t, sqldb.MySQLVariant{Image: "mysql:5.7.44", WaitMessage: "port: 3306  MySQL Community Server"}, "--explicit-defaults-for-timestamp=OFF", "--sql-mode=STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE")
	}
	for name, setup := range backends {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			sp := connectTestDB(ctx, t, setup(ctx, t))
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
			t.Run("mixed-case-rejected", func(t *testing.T) {
				require.ErrorContains(t, memodb.Migrate(ctx, sp, memodb.Config{TableName: "MemoCache"}), "invalid table name")
			})
			t.Run("long-name-and-future-expiry", func(t *testing.T) {
				name := strings.Repeat("a.", 126) + "a"
				require.NoError(t, q.Save(ctx, testNamespace, name, "future", "node", nil, "175200h"))
				record, err := q.Load(ctx, testNamespace, name, "future")
				require.NoError(t, err)
				require.NotNil(t, record)
				require.NotNil(t, record.ExpiresAt)
				assert.True(t, record.ExpiresAt.After(time.Date(2038, 1, 19, 0, 0, 0, 0, time.UTC)))
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
				_, err = sp.Session(ctx).SQL().Exec("CREATE TABLE malformed_cache (id integer)")
				require.NoError(t, err)
				require.Error(t, memodb.Migrate(ctx, sp, memodb.Config{TableName: "malformed_cache"}))
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
			t.Run("case-sensitive-keys", func(t *testing.T) {
				for _, key := range []string{"case-key", "Case-key"} {
					require.NoError(t, q.Save(ctx, testNamespace, testCacheName, key, key, nil, ""))
				}
				for _, key := range []string{"case-key", "Case-key"} {
					record, err := q.Load(ctx, testNamespace, testCacheName, key)
					require.NoError(t, err)
					require.NotNil(t, record)
					assert.Equal(t, key, record.NodeID)
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
			t.Run("save-racing-prune", func(t *testing.T) {
				for i := range 50 {
					key := fmt.Sprintf("racing-%d", i)
					require.NoError(t, q.Save(ctx, testNamespace, testCacheName, key, "old", nil, "-1h"))
					start := make(chan struct{})
					done := make(chan error, 2)
					go func() {
						<-start
						done <- retry.OnError(retry.DefaultBackoff, sqldb.IsTransientError, func() error { return q.Save(ctx, testNamespace, testCacheName, key, "new", nil, "") })
					}()
					go func() {
						<-start
						done <- retry.OnError(retry.DefaultBackoff, sqldb.IsTransientError, func() error { _, err := q.Prune(ctx); return err })
					}()
					close(start)
					require.NoError(t, <-done)
					require.NoError(t, <-done)
					record, err := q.Load(ctx, testNamespace, testCacheName, key)
					require.NoError(t, err)
					require.NotNil(t, record, "a successful save must survive concurrent pruning")
					assert.Equal(t, "new", record.NodeID)
				}
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
