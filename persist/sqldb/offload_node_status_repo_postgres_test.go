//go:build !windows

package sqldb

// Postgres counterpart to offload_node_status_repo_mysql_test.go.
//
// The compression change touches two things that behave differently per database:
// Save writes the Go string "null" into the nodes column, which is `json not null`
// on both engines, and the migration adds compressednodes as `text` on Postgres
// (`longtext` on MySQL). Only the MySQL side was covered, so these tests prove the
// Postgres path.

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// setupOffloadRepoPostgres starts Postgres, migrates the argo_workflows offload table,
// and returns the offload repo plus the session proxy.
func setupOffloadRepoPostgres(ctx context.Context, t testing.TB) (OffloadNodeStatusRepo, *usqldb.SessionProxy) {
	t.Helper()

	c, err := testpostgres.Run(ctx,
		"postgres:17.4-alpine",
		testpostgres.WithDatabase("argo"),
		testpostgres.WithUsername("argo"),
		testpostgres.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(120*time.Second)),
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
			// SSL is left unset, which yields sslmode=disable.
			PostgreSQL: &config.PostgreSQLConfig{
				DatabaseConfig: config.DatabaseConfig{Database: "argo", Host: host, Port: port},
			},
		},
		Username: "argo",
		Password: "argo",
	})
	require.NoError(t, err)
	t.Cleanup(func() { proxy.Close() })

	require.NoError(t, Migrate(ctx, proxy.Session(), "test", "argo_workflows", proxy.DBType()))

	repo, err := NewOffloadNodeStatusRepo(ctx, logging.RequireLoggerFromContext(ctx), proxy, "test", "argo_workflows")
	require.NoError(t, err)
	return repo, proxy
}

// TestOffloadCompressionPostgres_RoundTrip verifies that the migration applies on Postgres
// and that a compressed Save round-trips through Get.
//
// 1MiB is deliberate: the MySQL test uses ~13MiB to clear the 16MB max_allowed_packet
// ceiling, but Postgres has no equivalent limit. What is unproven here is the storage
// format and the migration, and 1MiB exercises both without the runtime cost.
func TestOffloadCompressionPostgres_RoundTrip(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	repo, proxy := setupOffloadRepoPostgres(ctx, t)

	nodes := makeNodes(t, 1*mb)
	uid := "uid-pg-roundtrip"

	version, err := repo.Save(ctx, uid, "default", nodes)
	require.NoError(t, err, `Save must accept the "null" placeholder in the json not null nodes column`)

	got, err := repo.Get(ctx, uid, version)
	require.NoError(t, err)
	assert.Equal(t, nodes, got, "Get must return the original nodes")

	// List has its own decompression branch and its own explicit column list, so Get
	// passing does not imply List passes. Reuses the container rather than starting one.
	list, err := repo.List(ctx, "default")
	require.NoError(t, err)
	assert.Equal(t, nodes, list[UUIDVersion{UID: uid, Version: version}], "List must decompress compressed rows")

	r := fetchRow(ctx, t, proxy, uid, version)
	assert.NotEmpty(t, r.CompressedNodes.String, "compressednodes should hold the compressed payload")
	assert.Equal(t, "null", r.Nodes, "nodes column should be the json null placeholder")
	assert.Less(t, len(r.CompressedNodes.String), 1*mb, "stored compressed payload should be smaller than raw")
}

// TestOffloadCompressionPostgres_LegacyRows covers both shapes a pre-compression row
// can have, so an upgrade needs no data migration: the empty string the migration
// backfills, and a genuine SQL NULL an old replica writes after that one-shot backfill
// has already run (see the CompressedNodes field comment).
func TestOffloadCompressionPostgres_LegacyRows(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	repo, proxy := setupOffloadRepoPostgres(ctx, t)

	legacyNodes := wfv1.Nodes{"n1": wfv1.NodeStatus{ID: "n1", Name: "n1", Phase: wfv1.NodeSucceeded}}
	raw, err := json.Marshal(legacyNodes)
	require.NoError(t, err)

	const version = "fnv:legacy"
	for _, tc := range []struct {
		name, uid string
		wantValid bool
		insert    func(db.Session, string) error
	}{
		{
			name: "backfilled empty string", uid: "uid-pg-legacy", wantValid: true,
			insert: func(s db.Session, uid string) error {
				_, insErr := s.Collection("argo_workflows").Insert(&nodesRecord{
					ClusterName:     "test",
					UUIDVersion:     UUIDVersion{UID: uid, Version: version},
					Namespace:       "default",
					Nodes:           string(raw),
					CompressedNodes: sql.NullString{String: "", Valid: true},
				})
				return insErr
			},
		},
		{
			// Must be raw SQL: Insert(&nodesRecord{...}) always writes a non-NULL empty
			// string, so it cannot reproduce the shape an old replica leaves behind.
			name: "genuine SQL NULL", uid: "uid-pg-null", wantValid: false,
			insert: func(s db.Session, uid string) error {
				_, execErr := s.SQL().Exec(
					"insert into argo_workflows (clustername, uid, version, namespace, nodes) values ($1, $2, $3, $4, $5)",
					"test", uid, version, "default", string(raw))
				return execErr
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, proxy.With(ctx, func(s db.Session) error { return tc.insert(s, tc.uid) }))
			require.Equal(t, tc.wantValid, fetchRow(ctx, t, proxy, tc.uid, version).CompressedNodes.Valid,
				"row must have the column shape this case claims to test")

			got, err := repo.Get(ctx, tc.uid, version)
			require.NoError(t, err, "Get must read legacy rows on Postgres")
			assert.Equal(t, legacyNodes, got)

			list, err := repo.List(ctx, "default")
			require.NoError(t, err, "List must read legacy rows on Postgres")
			assert.Equal(t, legacyNodes, list[UUIDVersion{UID: tc.uid, Version: version}])
		})
	}
}
