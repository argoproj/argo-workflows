package sqldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/upper/db/v4"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// SessionProxy is a wrapper for upperdb sessions that provides automatic reconnection
// on network failures through a With() method pattern.
type SessionProxy struct {
	// Connection configuration for reconnection
	kubectlConfig kubernetes.Interface
	namespace     string
	dbConfig      *config.DBConfig
	username      string
	password      string
	dbType        DBType

	// Current session and state
	sess   *liveSession
	mu     sync.RWMutex
	closed bool

	// Retry configuration
	maxRetries    int
	baseDelay     time.Duration
	maxDelay      time.Duration
	retryMultiple float64

	insideTransaction bool
}

// liveSession wraps a session with a reference count of the With callers
// using it. A reconnect or Close retires it instead of closing it directly,
// so it is only closed once the callers that took it have finished. Closing
// a session that is still in use makes the next statement on it panic.
type liveSession struct {
	sess      db.Session
	refs      atomic.Int32
	retired   atomic.Bool
	closeOnce sync.Once
}

// release drops a reference taken by acquire and closes the session if it
// was retired and this was the last reference.
func (l *liveSession) release() {
	l.refs.Add(-1)
	l.tryClose()
}

// retire closes the session once every caller that took a reference has
// released it. It must only be called after the proxy no longer hands out
// the session, so no new references can appear.
func (l *liveSession) retire() {
	l.retired.Store(true)
	l.tryClose()
}

func (l *liveSession) tryClose() {
	if l.retired.Load() && l.refs.Load() == 0 {
		l.closeOnce.Do(func() {
			_ = l.sess.Close()
		})
	}
}

// SessionProxyConfig contains configuration for creating a SessionProxy
type SessionProxyConfig struct {
	KubectlConfig kubernetes.Interface
	Namespace     string
	DBConfig      config.DBConfig
	Username      string
	Password      string
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
}

func validateProxyParams(proxy *SessionProxy) error {
	if proxy.maxRetries < 0 {
		return fmt.Errorf("maxRetries cannot be less than 0")
	}
	if proxy.baseDelay < 0 {
		return fmt.Errorf("baseDelay cannot be less than 0")
	}
	if proxy.maxDelay < 0 {
		return fmt.Errorf("maxDelay cannot be less than 0")
	}
	if proxy.retryMultiple < 0 {
		return fmt.Errorf("retryMultiple cannot be less than 0")
	}
	return nil
}

// NewSessionProxy creates a new SessionProxy with the given configuration
func NewSessionProxy(ctx context.Context, config SessionProxyConfig) (*SessionProxy, error) {
	dbType := dbTypeFromConfig(&config.DBConfig)
	proxy := &SessionProxy{
		kubectlConfig: config.KubectlConfig,
		namespace:     config.Namespace,
		dbConfig:      &config.DBConfig,
		username:      config.Username,
		password:      config.Password,
		dbType:        dbType,
		maxRetries:    config.MaxRetries,
		baseDelay:     config.BaseDelay,
		maxDelay:      config.MaxDelay,
		retryMultiple: 2.0,
	}

	if config.DBConfig.DBReconnectConfig != nil {
		reconnectConfig := config.DBConfig.DBReconnectConfig
		proxy.maxRetries = reconnectConfig.MaxRetries
		proxy.baseDelay = time.Duration(reconnectConfig.BaseDelaySeconds) * time.Second
		proxy.maxDelay = time.Duration(reconnectConfig.MaxDelaySeconds) * time.Second
		proxy.retryMultiple = reconnectConfig.RetryMultiple
	}

	if err := validateProxyParams(proxy); err != nil {
		return nil, err
	}

	if proxy.maxRetries == 0 {
		proxy.maxRetries = 5
	}
	if proxy.baseDelay == 0 {
		proxy.baseDelay = 100 * time.Millisecond
	}
	if proxy.maxDelay == 0 {
		proxy.maxDelay = 30 * time.Second
	}

	// just trying to account for float funkiness
	// a value between 0 and 1 is (almost) always non-sensical, but we allow it
	if proxy.retryMultiple <= 0.000000001 {
		proxy.retryMultiple = 1.0
	}

	if err := proxy.connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to create initial database session: %w", err)
	}

	return proxy, nil
}

// NewSessionProxyFromSession creates a SessionProxy from an existing session with credentials
func NewSessionProxyFromSession(sess db.Session, dbConfig *config.DBConfig, username, password string) *SessionProxy {
	return &SessionProxy{
		sess:          &liveSession{sess: sess},
		dbConfig:      dbConfig,
		username:      username,
		password:      password,
		dbType:        dbTypeFromConfig(dbConfig),
		maxRetries:    5,
		baseDelay:     100 * time.Millisecond,
		maxDelay:      30 * time.Second,
		retryMultiple: 2.0,
	}
}

// DBType returns the database type for this session proxy.
func (sp *SessionProxy) DBType() DBType {
	return sp.dbType
}

// TxWith runs a With transaction
func (sp *SessionProxy) TxWith(ctx context.Context, fn func(*SessionProxy) error, opts *sql.TxOptions) error {
	return sp.With(ctx, func(s db.Session) error {
		return s.TxContext(ctx, func(sess db.Session) error {
			newSp := &SessionProxy{
				kubectlConfig:     sp.kubectlConfig,
				namespace:         sp.namespace,
				dbConfig:          sp.dbConfig,
				username:          sp.username,
				password:          sp.password,
				dbType:            sp.dbType,
				sess:              &liveSession{sess: sess},
				closed:            sp.closed,
				maxRetries:        sp.maxRetries,
				baseDelay:         sp.baseDelay,
				maxDelay:          sp.maxDelay,
				retryMultiple:     sp.retryMultiple,
				insideTransaction: true,
			}

			return fn(newSp)
		}, opts)
	})
}

func (sp *SessionProxy) connect(ctx context.Context) error {
	var sess db.Session
	var err error

	switch {
	case sp.kubectlConfig != nil && sp.namespace != "" && sp.dbConfig != nil:
		// Use Kubernetes secrets for authentication
		sess, _, err = CreateDBSession(ctx, sp.kubectlConfig, sp.namespace, *sp.dbConfig)
	case sp.username != "" && sp.password != "" && sp.dbConfig != nil:
		// Use direct credentials
		sess, _, err = CreateDBSessionWithCreds(*sp.dbConfig, sp.username, sp.password)
	default:
		return fmt.Errorf("insufficient authentication information provided")
	}

	if err != nil {
		return err
	}

	err = sess.Ping()
	if err != nil {
		return err
	}
	sp.closed = false

	sp.sess = &liveSession{sess: sess}
	return nil
}

func (sp *SessionProxy) isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// context.DeadlineExceeded satisfies net.Error with Timeout() == true, so
	// check for it before the pattern and net.Error checks below.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	errStr := strings.ToLower(err.Error())

	if errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EPIPE) {
		return true
	}

	// Common network error patterns
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"connection lost",
		"connection closed",
		"network is unreachable",
		"no route to host",
		"timeout",
		"broken pipe",
		"connection timed out",
		"i/o timeout",
		"connection aborted",
		"connection dropped",
		"server closed the connection",
		"bad connection",
		"invalid connection",
		"connection is not available",
		"connection has been closed",
		"driver: bad connection",
	}

	for _, pattern := range networkErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Check for specific error types
	if netErr, ok := errors.AsType[net.Error](err); ok {
		return netErr.Timeout()
	}

	// Check for driver.ErrBadConn
	if errors.Is(err, driver.ErrBadConn) {
		return true
	}

	// Check for sql.ErrConnDone
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}

	return false
}

var (
	errProxyClosed = errors.New("session proxy is closed")
	errNoSession   = errors.New("no active session")
)

// acquire returns the current session with a reference that keeps it open
// until release is called, so a reconnect cannot close it mid-statement.
func (sp *SessionProxy) acquire() (*liveSession, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	if sp.closed {
		return nil, errProxyClosed
	}
	if sp.sess == nil {
		return nil, errNoSession
	}
	sp.sess.refs.Add(1)
	return sp.sess, nil
}

// With executes fn with a db Session bound to ctx, so cancelling ctx cancels
// the running statement.
func (sp *SessionProxy) With(ctx context.Context, fn func(db.Session) error) error {
	logger := logging.RequireLoggerFromContext(ctx)
	sess, err := sp.acquire()
	if err != nil {
		if errors.Is(err, errProxyClosed) {
			logger.Warn(ctx, "session proxy is closed")
		}
		return err
	}

	err = fn(sess.sess.WithContext(ctx))
	sess.release()
	if err == nil {
		return nil
	}

	// If the caller has given up, it's not a network error, or inside a tx do not retry
	if ctx.Err() != nil || !sp.isNetworkError(err) || sp.insideTransaction {
		return err
	}

	if reconnectErr := sp.reconnectIfStale(ctx, sess); reconnectErr != nil {
		return fmt.Errorf("operation failed and reconnection failed: %w", reconnectErr)
	}

	sess, err = sp.acquire()
	if err != nil {
		return fmt.Errorf("no active session after reconnection")
	}
	defer sess.release()

	if retryErr := fn(sess.sess.WithContext(ctx)); retryErr != nil {
		return fmt.Errorf("operation failed after reconnection: %w", retryErr)
	}

	return nil
}

// reconnectIfStale reconnects only if the current session is still the
// same one that produced the error. If another goroutine already
// reconnected (sp.sess != staleSess), this is a no-op.
func (sp *SessionProxy) reconnectIfStale(ctx context.Context, staleSess *liveSession) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.sess != staleSess {
		return nil
	}
	return sp.reconnectLocked(ctx)
}

// Reconnect performs reconnection with retry logic and linear backoff
func (sp *SessionProxy) Reconnect(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.reconnectLocked(ctx)
}

func (sp *SessionProxy) reconnectLocked(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)

	var err error

	for attempt := 0; attempt <= sp.maxRetries; attempt++ {
		// Perform the reconnection attempt
		// Retire the bad connection if it exists: it closes once the With
		// callers using it have finished, rather than right now.
		if sp.sess != nil {
			old := sp.sess
			sp.sess = nil
			sp.closed = true
			old.retire()
		}

		err = sp.connect(ctx)
		if err == nil {
			logger.WithField("attempt_number", attempt).Info(ctx, "connected to database")
			return nil
		}
		logger.WithField("attempt_number", attempt).Warn(ctx, "failed to connect to database")

		// If this is the last attempt, don't wait
		if attempt == sp.maxRetries || !sp.isNetworkError(err) {
			break
		}

		// Calculate delay for next retry with linear backoff
		delay := time.Duration(float64(sp.baseDelay) * float64(attempt+1) * sp.retryMultiple)
		delay = min(delay, sp.maxDelay)

		// Wait before retrying with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("reconnection failed after %d retries, last error: %w", sp.maxRetries, err)
}

// Session returns the underlying session bound to ctx. Use With() for operations that need reconnection.
// This method is provided for cases where you need direct access to the session,
// but it won't provide automatic reconnection.
func (sp *SessionProxy) Session(ctx context.Context) db.Session {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	if sp.sess == nil {
		return nil
	}
	return sp.sess.sess.WithContext(ctx)
}

// Close closes the session proxy and underlying session
func (sp *SessionProxy) Close() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.closed {
		return nil
	}

	sp.closed = true

	if sp.sess != nil {
		old := sp.sess
		sp.sess = nil
		old.retire()
	}

	return nil
}
