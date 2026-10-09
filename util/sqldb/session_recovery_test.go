//go:build !windows

package sqldb

import (
	"context"
	"errors"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Regression for #16771: a failed reconnect must not disable later operations
// after the database recovers, even when the entire retry budget was exhausted.
func TestSessionRecoveryAfterFailedReconnect(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	for _, canceled := range []bool{false, true} {
		name := "ExhaustedRetries"
		if canceled {
			name = "CanceledReconnect"
		}
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			postgres, err := testpostgres.Run(ctx, "postgres:17.4-alpine",
				testpostgres.WithDatabase(dbName),
				testpostgres.WithUsername(userName),
				testpostgres.WithPassword(password),
				testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
					// Keep the endpoint stable across a real database restart.
					hostConfig.PortBindings = network.PortMap{
						network.MustParsePort("5432/tcp"): {{
							HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: strconv.Itoa(fixedPort),
						}},
					}
				}),
				testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).WithStartupTimeout(30*time.Second)),
			)
			testcontainers.CleanupContainer(t, postgres)
			require.NoError(t, err)
			host, err := postgres.Host(ctx)
			require.NoError(t, err)
			mappedPort, err := postgres.MappedPort(ctx, "5432/tcp")
			require.NoError(t, err)
			port, err := strconv.Atoi(mappedPort.Port())
			require.NoError(t, err)
			cfg := config.DBConfig{PostgreSQL: &config.PostgreSQLConfig{
				DatabaseConfig: config.DatabaseConfig{Database: dbName, Host: host, Port: port},
			}}
			// The canceled variant interrupts a reconnect in its backoff, so the
			// backoff has to be long enough to still be pending when that happens.
			delay := time.Millisecond
			if canceled {
				delay = time.Second
			}
			proxy, err := NewSessionProxy(ctx, SessionProxyConfig{
				DBConfig: cfg, Username: userName, Password: password,
				MaxRetries: 1, BaseDelay: delay, MaxDelay: delay,
			})
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, proxy.Close()) })

			query := func(s db.Session) error {
				row, queryErr := s.SQL().QueryRow("SELECT 1")
				if queryErr != nil {
					return queryErr
				}
				var value int
				return row.Scan(&value)
			}
			require.NoError(t, proxy.With(ctx, query))

			for cycle := range 2 {
				stopTimeout := time.Second
				require.NoError(t, postgres.Stop(ctx, &stopTimeout))
				err = proxy.With(ctx, func(s db.Session) error { return s.Ping() })
				require.ErrorContains(t, err, "reconnection failed after 1 retries")
				if canceled {
					// The proxy is now disconnected, so the next operation reconnects
					// before it runs. A caller that has already given up stops in the
					// reconnect backoff; that must not permanently close the proxy either.
					canceledCtx, cancel := context.WithCancel(ctx)
					cancel()
					err = proxy.With(canceledCtx, func(db.Session) error {
						t.Error("operation ran without a successful reconnection")
						return nil
					})
					require.ErrorIs(t, err, context.Canceled)
				}
				// Controller configuration reloads use Session() directly to
				// update pool settings, including while reconnect is failing.
				require.NotPanics(t, func() {
					ConfigureDBSession(proxy.Session(ctx), &config.ConnectionPool{MaxOpenConns: 4})
				})

				require.NoError(t, postgres.Start(ctx))
				restartedPort, err := postgres.MappedPort(ctx, "5432/tcp")
				require.NoError(t, err)
				require.Equal(t, mappedPort, restartedPort)
				// Verify that PostgreSQL is reachable independently, without
				// reconnecting or replacing the proxy under test.
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					sess, _, connectErr := CreateDBSessionWithCreds(cfg, userName, password)
					if assert.NoError(c, connectErr) {
						assert.NoError(c, sess.Ping())
						assert.NoError(c, sess.Close())
					}
				}, 10*time.Second, 100*time.Millisecond)

				// All callers keep the original proxy. They must share the
				// recovered session rather than close each other's connections.
				results := make(chan error, 4)
				for range cap(results) {
					go func() { results <- proxy.With(ctx, query) }()
				}
				var queryErrors []error
				for range cap(results) {
					queryErrors = append(queryErrors, <-results)
				}
				require.NoError(t, errors.Join(queryErrors...), "cycle %d", cycle)
			}

			// Explicit Reconnect remains an opt-in way to reopen a closed proxy.
			require.NoError(t, proxy.Close())
			require.ErrorContains(t, proxy.With(ctx, query), "session proxy is closed")
			require.NoError(t, proxy.Reconnect(ctx))
			require.NoError(t, proxy.With(ctx, query))
		})
	}
}
