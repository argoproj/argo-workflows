//go:build !windows

package sqldb

// Integration tests for #15942: offload now stores node status COMPRESSED in the
// compressednodes column, removing the MySQL max_allowed_packet ceiling that raw
// uncompressed JSON hit. MySQL 8.4 container is pinned to max_allowed_packet=16MB so
// the pre-fix failure (Save of ~13MB nodes) is exercised as a passing case post-fix.

import (
	"context"
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
func setupOffloadRepo(ctx context.Context, t *testing.T) (OffloadNodeStatusRepo, *usqldb.SessionProxy) {
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
	assert.NotEmpty(t, r.CompressedNodes, "compressednodes should hold the compressed payload")
	assert.Equal(t, "null", r.Nodes, "nodes column should be the json null placeholder")
	assert.Less(t, len(r.CompressedNodes), 13*mb, "stored compressed payload should be far smaller than raw")
}

// TestOffloadCompression_BackwardCompat verifies that a legacy row (raw JSON in nodes,
// empty compressednodes) still reads correctly via both Get and List after the change.
func TestOffloadCompression_BackwardCompat(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	repo, proxy := setupOffloadRepo(ctx, t)

	legacyNodes := wfv1.Nodes{"n1": wfv1.NodeStatus{ID: "n1", Name: "n1", Phase: wfv1.NodeSucceeded}}
	raw, err := json.Marshal(legacyNodes)
	require.NoError(t, err)

	uid, version := "uid-legacy", "fnv:legacy"
	err = proxy.With(ctx, func(s db.Session) error {
		_, insErr := s.Collection("argo_workflows").Insert(&nodesRecord{
			ClusterName:     "test",
			UUIDVersion:     UUIDVersion{UID: uid, Version: version},
			Namespace:       "default",
			Nodes:           string(raw),
			CompressedNodes: "", // legacy: no compression
		})
		return insErr
	})
	require.NoError(t, err)

	got, err := repo.Get(ctx, uid, version)
	require.NoError(t, err)
	assert.Equal(t, legacyNodes, got, "Get must read legacy uncompressed rows")

	list, err := repo.List(ctx, "default")
	require.NoError(t, err)
	assert.Equal(t, legacyNodes, list[UUIDVersion{UID: uid, Version: version}], "List must read legacy uncompressed rows")
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
