//go:build !windows

package sqldb

// Integration tests for #13290: offload now stores node status COMPRESSED in the
// compressednodes column, cutting the volume written on every update of a large
// workflow. The MySQL 8.4 container pins max_allowed_packet to 16MB, which also makes
// the size reduction observable: a Save of ~13MB of raw nodes only fits once compressed.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// setupOffloadRepo starts MySQL 8.4 with max_allowed_packet pinned to 16MB, migrates the
// argo_workflows offload table, and returns the offload repo plus the session proxy.
func setupOffloadRepo(ctx context.Context, t testing.TB) (OffloadNodeStatusRepo, *usqldb.SessionProxy) {
	t.Helper()

	c, err := testmysql.Run(ctx,
		"mysql:8.4",
		testmysql.WithDatabase("argo"),
		testmysql.WithUsername("argo"),
		testmysql.WithPassword("argo"),
		// Pin the ceiling to 16MB so ~13MB raw nodes would fail pre-fix but pass compressed.
		testcontainers.WithCmdArgs("--max-allowed-packet=16777216"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(120*time.Second),
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

// makeNodes builds a wfv1.Nodes whose marshalled JSON is >= target bytes.
func makeNodes(t *testing.T, target int) wfv1.Nodes {
	t.Helper()
	nodes := wfv1.Nodes{}
	chunk := strings.Repeat("x", 64*1024) // 64KB per node
	i := 0
	for {
		id := fmt.Sprintf("node-%06d", i)
		nodes[id] = wfv1.NodeStatus{ID: id, Name: id, Message: chunk}
		i++
		if i%16 == 0 {
			b, err := json.Marshal(nodes)
			require.NoError(t, err)
			if len(b) >= target {
				return nodes
			}
		}
	}
}

const mb = 1 << 20

// TestOffloadCompression_RoundTrip verifies a ~13MB node status (which pre-fix exceeded the
// 16MB packet ceiling once expanded) now saves compressed, round-trips via Get, and is stored
// with the raw nodes column holding only the "null" placeholder.
func TestOffloadCompression_RoundTrip(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	repo, proxy := setupOffloadRepo(ctx, t)

	nodes := makeNodes(t, 13*mb)
	uid := "uid-roundtrip"

	version, err := repo.Save(ctx, uid, "default", nodes)
	require.NoError(t, err, "compressed Save of ~13MB nodes should succeed under 16MB max_allowed_packet")

	got, err := repo.Get(ctx, uid, version)
	require.NoError(t, err)
	assert.Equal(t, nodes, got, "Get must return the original nodes")

	// Storage format: compressed payload present, raw nodes column is the placeholder.
	r := fetchRow(ctx, t, proxy, uid, version)
	assert.NotEmpty(t, r.CompressedNodes.String, "compressednodes should hold the compressed payload")
	assert.Equal(t, "null", r.Nodes, "nodes column should be the json null placeholder")
	assert.Less(t, len(r.CompressedNodes.String), 13*mb, "stored compressed payload should be far smaller than raw")
}

// TestOffloadCompression_LegacyRows covers both shapes a pre-compression row can have,
// so an upgrade needs no data migration: the empty string the migration backfills, and a
// genuine SQL NULL an old replica writes after that one-shot backfill has already run
// (see the CompressedNodes field comment).
//
// Worth having on both engines rather than trusting one: the column is longtext here and
// text on Postgres, and the two drivers scan NULL through different code.
func TestOffloadCompression_LegacyRows(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	repo, proxy := setupOffloadRepo(ctx, t)

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
			name: "backfilled empty string", uid: "uid-legacy", wantValid: true,
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
			name: "genuine SQL NULL", uid: "uid-null", wantValid: false,
			insert: func(s db.Session, uid string) error {
				_, execErr := s.SQL().Exec(
					"insert into argo_workflows (clustername, uid, version, namespace, nodes) values (?, ?, ?, ?, ?)",
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
			require.NoError(t, err, "Get must read legacy rows")
			assert.Equal(t, legacyNodes, got)

			list, err := repo.List(ctx, "default")
			require.NoError(t, err, "List must read legacy rows")
			assert.Equal(t, legacyNodes, list[UUIDVersion{UID: tc.uid, Version: version}])
		})
	}
}

func fetchRow(ctx context.Context, t *testing.T, proxy *usqldb.SessionProxy, uid, version string) nodesRecord {
	t.Helper()
	var r nodesRecord
	err := proxy.With(ctx, func(s db.Session) error {
		return s.SQL().SelectFrom("argo_workflows").
			Where(db.Cond{"uid": uid}).And(db.Cond{"version": version}).One(&r)
	})
	require.NoError(t, err)
	return r
}
