//go:build !windows

package sqldb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Regression for #16771: a failed reconnect must not disable later operations
// after the database recovers, even when the entire retry budget was exhausted.
func TestSessionRecoveryAfterFailedReconnect(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		name := "ExhaustedRetries"
		if canceled {
			name = "CanceledReconnect"
		}
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			postgres, cfg, err := runPostgresContainer(ctx, t)
			testcontainers.CleanupContainer(t, postgres)
			require.NoError(t, err)
			// Use the retry settings below rather than the slow ones from the helper.
			cfg.DBReconnectConfig = nil
			proxy, err := NewSessionProxy(ctx, SessionProxyConfig{
				DBConfig: cfg, Username: userName, Password: password,
				MaxRetries: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond,
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
					// A caller that has already given up gets its own error back, and
					// that must not permanently close the proxy either.
					canceledCtx, cancel := context.WithCancel(ctx)
					cancel()
					require.ErrorIs(t, proxy.With(canceledCtx, query), context.Canceled)
				}
				// Controller configuration reloads use Session() directly to
				// update pool settings, including while reconnect is failing.
				require.NotPanics(t, func() {
					ConfigureDBSession(proxy.Session(ctx), &config.ConnectionPool{MaxOpenConns: 4})
				})

				require.NoError(t, postgres.Start(ctx))
				// The old pool or a reconnect has to pick the database up again,
				// for plain operations and for transactions, which the archive uses.
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					assert.NoError(c, proxy.With(ctx, query))
				}, 10*time.Second, 100*time.Millisecond, "cycle %d", cycle)
				require.NoError(t, proxy.TxWith(ctx, func(tx *SessionProxy) error {
					return tx.With(ctx, query)
				}, nil), "cycle %d", cycle)
			}

			// Explicit Reconnect remains an opt-in way to reopen a closed proxy.
			require.NoError(t, proxy.Close())
			require.ErrorIs(t, proxy.With(ctx, query), errSessionProxyClosed)
			require.NoError(t, proxy.Reconnect(ctx))
			require.NoError(t, proxy.With(ctx, query))
		})
	}
}
