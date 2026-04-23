//go:build corefunctional

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

type SQLDBMemoizeSuite struct {
	fixtures.E2ESuite
}

// memoWorkflow builds a workflow spec with a unique cache key per test run to avoid
// stale cache hits from previous runs.
func memoWorkflow(cacheKey string) string {
	return fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: sqldb-memoize-
spec:
  entrypoint: hello
  templates:
    - name: hello
      steps:
        - - name: run1
            template: memoized
            arguments:
              parameters: [{name: message, value: "%s"}]
        - - name: run2
            template: memoized
            arguments:
              parameters: [{name: message, value: "%s"}]
    - name: memoized
      inputs:
        parameters:
          - name: message
      memoize:
        key: "{{inputs.parameters.message}}"
        maxAge: "10m"
        cache:
          configMap:
            name: sqldb-memo-cache
      container:
        image: argoproj/argosay:v2
        command: [echo]
        args: ["{{inputs.parameters.message}}"]
`, cacheKey, cacheKey)
}

func (s *SQLDBMemoizeSuite) TestSQLDBMemoize() {
	if s.Config.Memoization == nil {
		s.T().Skip("memoization DB not configured; skipping SQL cache test")
	}

	ctx := logging.TestContext(s.T().Context())

	// Use a unique key so each test run starts with a cold cache.
	cacheKey := fmt.Sprintf("hello-sqldb-%d", time.Now().UnixNano())

	// Submit the workflow and wait for it to succeed.
	s.Given().
		Workflow(memoWorkflow(cacheKey)).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			memoHit := false
			memoSaved := false
			for _, node := range status.Nodes {
				if node.MemoizationStatus == nil {
					continue
				}
				if node.MemoizationStatus.Hit {
					memoHit = true
				} else {
					memoSaved = true
				}
			}
			assert.True(t, memoSaved, "expected at least one node to save to the cache")
			assert.True(t, memoHit, "expected at least one node to hit the cache")
		})

	// Also verify the entry landed in postgres, not a ConfigMap.
	s.assertDBCacheEntry(ctx, cacheKey)
	s.assertNoConfigMap(ctx, "sqldb-memo-cache")
}

// assertDBCacheEntry checks the memoization_cache table directly.
func (s *SQLDBMemoizeSuite) assertDBCacheEntry(ctx context.Context, key string) {
	memoCfg := s.Config.Memoization
	// E2E tests connect to postgres via a port-forward on localhost.
	cfg := *memoCfg
	if cfg.PostgreSQL != nil {
		pg := *cfg.PostgreSQL
		pg.Host = "localhost"
		cfg.PostgreSQL = &pg
	}
	if cfg.MySQL != nil {
		my := *cfg.MySQL
		my.Host = "localhost"
		cfg.MySQL = &my
	}

	session, _, err := sqldb.CreateDBSession(ctx, s.KubeClient, fixtures.Namespace, cfg.DBConfig)
	s.Require().NoError(err, "could not connect to memoization DB")
	defer session.Close()

	tableName := memodb.TableName(&cfg)

	var count int
	row, err := session.SQL().
		QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE namespace = ? AND cache_name = ? AND cache_key = ?`, tableName),
			fixtures.Namespace, "sqldb-memo-cache", key)
	s.Require().NoError(err)
	s.Require().NoError(row.Scan(&count))
	s.Equal(1, count, "expected exactly one cache entry in the database for key %q", key)

	// Also verify outputs are stored as valid JSON.
	var outputs string
	row, err = session.SQL().
		QueryRow(fmt.Sprintf(`SELECT outputs FROM %s WHERE namespace = ? AND cache_name = ? AND cache_key = ?`, tableName),
			fixtures.Namespace, "sqldb-memo-cache", key)
	s.Require().NoError(err)
	s.Require().NoError(row.Scan(&outputs))
	s.NotEmpty(outputs)
}

// assertNoConfigMap verifies the controller did NOT fall back to creating a ConfigMap cache.
func (s *SQLDBMemoizeSuite) assertNoConfigMap(ctx context.Context, name string) {
	_, err := s.KubeClient.CoreV1().ConfigMaps(fixtures.Namespace).Get(ctx, name, metav1.GetOptions{})
	s.Error(err, "ConfigMap %q should not exist when SQL memoization is configured", name)
}

func TestSQLDBMemoizeSuite(t *testing.T) {
	suite.Run(t, new(SQLDBMemoizeSuite))
}
