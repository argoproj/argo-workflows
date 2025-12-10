package sqldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/upper/db/v4"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/util/logging"
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

	// Current session and state
	sess   db.Session
	mu     sync.RWMutex
	closed bool

	// Retry configuration
	maxRetries    int
	baseDelay     time.Duration
	maxDelay      time.Duration
	retryMultiple float64

	insideTransaction bool
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

// invalid parameter found.
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

// NewSessionProxy creates a SessionProxy configured from the provided SessionProxyConfig,
// initializes retry/backoff defaults and validation, and establishes an initial database session.
// 
// The function applies DBReconnectConfig values when present and falls back to sensible defaults:
// maxRetries defaults to 5, baseDelay defaults to 100ms, maxDelay defaults to 30s, and very small
// retryMultipliers are normalized to 1.0. It returns an error if parameter validation fails or
// if the initial connection attempt cannot be established.
func NewSessionProxy(ctx context.Context, config SessionProxyConfig) (*SessionProxy, error) {
	proxy := &SessionProxy{
		kubectlConfig: config.KubectlConfig,
		namespace:     config.Namespace,
		dbConfig:      &config.DBConfig,
		username:      config.Username,
		password:      config.Password,
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

// NewSessionProxyFromSession creates a SessionProxy that wraps an existing db.Session and the
// provided DB configuration and credentials. It initializes sensible defaults for reconnection
// behavior: maxRetries=5, baseDelay=100ms, maxDelay=30s, and retryMultiple=2.0.
func NewSessionProxyFromSession(sess db.Session, dbConfig *config.DBConfig, username, password string) *SessionProxy {
	return &SessionProxy{
		sess:          sess,
		dbConfig:      dbConfig,
		username:      username,
		password:      password,
		maxRetries:    5,
		baseDelay:     100 * time.Millisecond,
		maxDelay:      30 * time.Second,
		retryMultiple: 2.0,
	}
}

// Tx marks the sessionproxy as being part of a transaction.
// This ensures we do not retry/reconnect
func (sp *SessionProxy) Tx() *SessionProxy {
	s := SessionProxy{sp.kubectlConfig, sp.namespace, sp.dbConfig, sp.username, sp.password, sp.Session(), sync.RWMutex{}, sp.closed, sp.maxRetries, sp.baseDelay, sp.maxDelay, sp.retryMultiple, sp.insideTransaction}
	s.insideTransaction = true
	return &s
}

// TxWith runs a With transaction
func (sp *SessionProxy) TxWith(ctx context.Context, fn func(*SessionProxy) error, opts *sql.TxOptions) error {
	return sp.With(ctx, func(s db.Session) error {
		return s.TxContext(ctx, func(sess db.Session) error {
			newSp := SessionProxy{sp.kubectlConfig, sp.namespace, sp.dbConfig, sp.username, sp.password, sess, sync.RWMutex{}, sp.closed, sp.maxRetries, sp.baseDelay, sp.maxDelay, sp.retryMultiple, true}
			return fn(&newSp)
		}, opts)
	})
}

func (sp *SessionProxy) connect(ctx context.Context) error {
	var sess db.Session
	var err error

	if sp.kubectlConfig != nil && sp.namespace != "" && sp.dbConfig != nil {
		// Use Kubernetes secrets for authentication
		sess, err = CreateDBSession(ctx, sp.kubectlConfig, sp.namespace, *sp.dbConfig)
	} else if sp.username != "" && sp.password != "" && sp.dbConfig != nil {
		// Use direct credentials
		sess, err = CreateDBSessionWithCreds(*sp.dbConfig, sp.username, sp.password)
	} else {
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

	sp.sess = sess
	return nil
}

func (sp *SessionProxy) isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

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
		"eof",
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
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}

	// Check for driver.ErrBadConn
	if err == driver.ErrBadConn {
		return true
	}

	// Check for sql.ErrConnDone
	if err == sql.ErrConnDone {
		return true
	}

	return false
}

// With executes with a db Session
func (sp *SessionProxy) With(ctx context.Context, fn func(db.Session) error) error {
	logger := logging.RequireLoggerFromContext(ctx)
	sp.mu.RLock()
	if sp.closed {
		sp.mu.RUnlock()
		logger.Warn(ctx, "session proxy is closed")
		return fmt.Errorf("session proxy is closed")
	}
	sp.mu.RUnlock()

	sp.mu.RLock()
	sess := sp.sess
	sp.mu.RUnlock()

	if sess == nil {
		return fmt.Errorf("no active session")
	}

	err := fn(sess)
	if err == nil {
		return nil
	}

	// If it's not a network error or inside a tx do not retry
	if !sp.isNetworkError(err) || sp.insideTransaction {
		return err
	}

	if reconnectErr := sp.Reconnect(ctx); reconnectErr != nil {
		return fmt.Errorf("operation failed and reconnection failed: %w", reconnectErr)
	}

	sp.mu.RLock()
	sess = sp.sess
	sp.mu.RUnlock()

	if sess == nil {
		return fmt.Errorf("no active session after reconnection")
	}

	if retryErr := fn(sess); retryErr != nil {
		return fmt.Errorf("operation failed after reconnection: %w", retryErr)
	}

	return nil
}

// Reconnect performs reconnection with retry logic and exponential backoff
func (sp *SessionProxy) Reconnect(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	sp.mu.Lock()
	defer sp.mu.Unlock()

	var err error

	for attempt := 0; attempt <= sp.maxRetries; attempt++ {
		// Perform the reconnection attempt
		// Close the bad connection if it exists
		if sp.sess != nil {
			sp.sess.Close()
			sp.closed = true
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

		// Calculate delay for next retry with exponential backoff
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

// Session returns the underlying session. Use With() for operations that need reconnection.
// This method is provided for cases where you need direct access to the session,
// but it won't provide automatic reconnection.
func (sp *SessionProxy) Session() db.Session {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.sess
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
		return sp.sess.Close()
	}

	return nil
}