//go:build !windows

// Package sqldb provides disposable database fixtures shared by SQL tests.
package sqldb

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const (
	Database = "argo"
	Username = "argo"
	Password = "argo"
)

func Postgres(ctx context.Context, t *testing.T) config.DBConfig {
	t.Helper()
	cfg := start(ctx, t, testcontainers.ContainerRequest{
		Image: "postgres:17.4-alpine", ExposedPorts: []string{"5432/tcp"},
		Env:        map[string]string{"POSTGRES_DB": Database, "POSTGRES_USER": Username, "POSTGRES_PASSWORD": Password},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(time.Minute),
	})
	return config.DBConfig{PostgreSQL: &config.PostgreSQLConfig{DatabaseConfig: cfg}}
}

func MySQL(ctx context.Context, t *testing.T, v sqldb.MySQLVariant, args ...string) config.DBConfig {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image: v.Image, ExposedPorts: []string{"3306/tcp"}, Cmd: args,
		Env:        map[string]string{"MYSQL_DATABASE": Database, "MYSQL_USER": Username, "MYSQL_PASSWORD": Password, "MYSQL_ROOT_PASSWORD": Password},
		WaitingFor: wait.ForAll(wait.ForLog(v.WaitMessage).WithStartupTimeout(2*time.Minute), wait.ForListeningPort("3306/tcp")),
	}
	// MySQL 5.7 has no ARM image; Docker can emulate it on ARM development machines.
	if strings.HasPrefix(v.Image, "mysql:5.7") {
		req.ImagePlatform = "linux/amd64"
	}
	cfg := start(ctx, t, req)
	return config.DBConfig{MySQL: &config.MySQLConfig{DatabaseConfig: cfg}}
}

func start(ctx context.Context, t *testing.T, req testcontainers.ContainerRequest) config.DatabaseConfig {
	t.Helper()
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, testcontainers.TerminateContainer(c)) })
	endpoint, err := c.Endpoint(ctx, "")
	require.NoError(t, err)
	host, port, err := net.SplitHostPort(endpoint)
	require.NoError(t, err)
	n, err := strconv.Atoi(port)
	require.NoError(t, err)
	return config.DatabaseConfig{Host: host, Port: n, Database: Database}
}
