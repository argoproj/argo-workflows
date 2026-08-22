package sqldb

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/config"
)

// pqKeywords is the set of DSN keywords lib/pq consumes as driver settings. Anything else is
// forwarded to the server as a runtime parameter, so an unrecognised keyword makes the connection
// fail with SQLSTATE 42704 rather than being ignored.
var pqKeywords = map[string]bool{
	"connect_timeout":           true,
	"dbname":                    true,
	"host":                      true,
	"password":                  true,
	"port":                      true,
	"sslcert":                   true,
	"sslkey":                    true,
	"sslmode":                   true,
	"sslrootcert":               true,
	"user":                      true,
	"application_name":          true,
	"fallback_application_name": true,
}

func dsnKeys(t *testing.T, dsn string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, field := range strings.Fields(dsn) {
		k, v, ok := strings.Cut(field, "=")
		require.True(t, ok, "malformed DSN field %q in %q", field, dsn)
		out[k] = v
	}
	return out
}

// TestBuildPostgresDSNOnlyPqKeywords is the regression test for the Azure and AWS RDS token
// connectors: buildPostgresDSN used to be built with the upper/db postgresql adapter's
// ConnectionURL, which targets pgx and unconditionally appends default_query_exec_mode. The token
// connectors open that DSN with lib/pq, which forwards the unknown keyword to the server, so every
// connection failed with `unrecognized configuration parameter "default_query_exec_mode"`.
func TestBuildPostgresDSNOnlyPqKeywords(t *testing.T) {
	cfg := &config.PostgreSQLConfig{
		DatabaseConfig: config.DatabaseConfig{
			Host:     "example.eu-west-1.rds.amazonaws.com",
			Port:     5432,
			Database: "argo",
		},
		SSL:     true,
		SSLMode: "require",
	}

	dsn := buildPostgresDSN(cfg, "argo_user", 7*time.Second)

	assert.NotContains(t, dsn, "default_query_exec_mode")
	for k := range dsnKeys(t, dsn) {
		assert.True(t, pqKeywords[k], "DSN contains keyword %q which lib/pq does not recognise; it would be sent to the server as a runtime parameter", k)
	}
}

func TestBuildPostgresDSNParams(t *testing.T) {
	cfg := &config.PostgreSQLConfig{
		DatabaseConfig: config.DatabaseConfig{
			Host:     "db.example.com",
			Port:     6432,
			Database: "argo",
		},
		SSL:     true,
		SSLMode: "verify-full",
	}

	got := dsnKeys(t, buildPostgresDSN(cfg, "argo_user", 3*time.Second))

	assert.Equal(t, "argo_user", got["user"])
	assert.Equal(t, "db.example.com", got["host"])
	assert.Equal(t, "6432", got["port"])
	assert.Equal(t, "argo", got["dbname"])
	assert.Equal(t, "verify-full", got["sslmode"])
	assert.Equal(t, "3", got["connect_timeout"])
}

// A host with no port must not be split into an empty port, which lib/pq rejects.
func TestBuildPostgresDSNHostWithoutPort(t *testing.T) {
	cfg := &config.PostgreSQLConfig{
		DatabaseConfig: config.DatabaseConfig{
			Host:     "db.example.com",
			Database: "argo",
		},
		SSL: true,
	}

	got := dsnKeys(t, buildPostgresDSN(cfg, "argo_user", time.Second))

	assert.Equal(t, "db.example.com", got["host"])
	assert.NotContains(t, got, "port")
}

func TestPostgresSSLMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		ssl  bool
		mode string
		want string
	}{
		{"ssl disabled", false, "", "disable"},
		{"ssl disabled ignores mode", false, "require", "disable"},
		{"explicit mode", true, "verify-ca", "verify-ca"},
		{"adapter default preserved", true, "", "prefer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.PostgreSQLConfig{SSL: tc.ssl, SSLMode: tc.mode}
			assert.Equal(t, tc.want, postgresSSLMode(cfg))
		})
	}
}

// Values carrying DSN metacharacters must be escaped rather than splitting the DSN.
func TestBuildPostgresDSNEscapesValues(t *testing.T) {
	cfg := &config.PostgreSQLConfig{
		DatabaseConfig: config.DatabaseConfig{
			Host:     "db.example.com",
			Port:     5432,
			Database: "argo db",
		},
		SSL: true,
	}

	dsn := buildPostgresDSN(cfg, "user's name", time.Second)

	assert.Contains(t, dsn, `dbname=argo\ db`)
	assert.Contains(t, dsn, `user=user\'s\ name`)
}