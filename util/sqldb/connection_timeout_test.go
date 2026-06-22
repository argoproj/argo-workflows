package sqldb

import (
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/config"
)

// silentListener accepts TCP connections but never writes the server startup
// handshake, simulating a half-open / overloaded database (the failure mode where
// the OS connect timeout does not apply because TCP is already established).
type silentListener struct {
	ln    net.Listener
	mu    sync.Mutex
	conns []net.Conn
}

func newSilentListener(t *testing.T) *silentListener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	sl := &silentListener{ln: ln}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Hold the connection open (never write) and keep a reference so it is
			// not closed by a finalizer.
			sl.mu.Lock()
			sl.conns = append(sl.conns, conn)
			sl.mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		sl.mu.Lock()
		for _, c := range sl.conns {
			c.Close()
		}
		sl.mu.Unlock()
	})
	return sl
}

func (sl *silentListener) hostPort(t *testing.T) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(sl.ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	return host, port
}

// TestPostgresConnectionTimeoutHalfOpen is the regression test for issue #14112:
// against a half-open server (TCP accepted, handshake never completes),
// CreateDBSessionWithCreds must return an error within a bounded time instead of
// blocking forever. lib/pq's connect_timeout covers the startup handshake read,
// so the session construction (upper/db's BindDB -> Ping) fails fast.
func TestPostgresConnectionTimeoutHalfOpen(t *testing.T) {
	sl := newSilentListener(t)
	host, port := sl.hostPort(t)

	dbConfig := config.DBConfig{
		PostgreSQL: &config.PostgreSQLConfig{
			DatabaseConfig: config.DatabaseConfig{
				Host:     host,
				Port:     port,
				Database: "test",
			},
		},
		ConnectionTimeoutSeconds: 1,
	}

	done := make(chan error, 1)
	go func() {
		_, _, err := CreateDBSessionWithCreds(dbConfig, "user", "pass")
		done <- err
	}()

	select {
	case err := <-done:
		// Before the fix this never returns; after it, connect_timeout fires.
		require.Error(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("CreateDBSessionWithCreds did not return within 10s; connect_timeout not applied")
	}
}

func TestBuildPostgresDSNConnectTimeout(t *testing.T) {
	cfg := &config.PostgreSQLConfig{
		DatabaseConfig: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "argo",
		},
	}
	dsn := buildPostgresDSN(cfg, "user", 7*time.Second)
	assert.Contains(t, dsn, "connect_timeout=7")
}

func TestBuildMySQLConfigTimeout(t *testing.T) {
	cfg := &config.MySQLConfig{
		DatabaseConfig: config.DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			Database: "argo",
		},
	}
	mysqlCfg := buildMySQLConfig(cfg, "user", "pass", 7*time.Second)
	assert.Equal(t, 7*time.Second, mysqlCfg.Timeout)
	// ReadTimeout is intentionally left unset (see buildMySQLConfig).
	assert.Zero(t, mysqlCfg.ReadTimeout)
}
