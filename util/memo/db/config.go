package db

import (
	"context"
	"fmt"
	"hash/fnv"
	"regexp"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const (
	defaultTableName = "cache_entries"
	versionTable     = "memoization_schema_history"
)

var validTableName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// Config holds resolved configuration for database-backed memoization.
type Config struct {
	TableName     string
	SkipMigration bool
}

func TableName(cfg *config.MemoizationConfig) string {
	if cfg == nil || cfg.TableName == "" {
		return defaultTableName
	}
	return cfg.TableName
}

func validateTableName(tableName string) error {
	if !validTableName.MatchString(tableName) {
		return fmt.Errorf("invalid table name %q: must match [A-Za-z0-9_]+", tableName)
	}
	return nil
}

// memoizationVersionTableName returns the schema history table name for the given memoization
// cache table. The default table name uses a fixed well-known name; custom table names get a
// deterministic hash suffix to avoid collisions.
func memoizationVersionTableName(tableName string) string {
	if tableName == defaultTableName {
		return versionTable
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(tableName))
	return fmt.Sprintf("memoization_schema_history_%x", hasher.Sum64())
}

// memoizationExpiresAtIndexName returns the name of the expires_at index for the given
// memoization cache table, using a deterministic hash suffix to avoid collisions across
// multiple cache tables.
func memoizationExpiresAtIndexName(tableName string) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(tableName))
	return fmt.Sprintf("memoization_expires_at_%x", hasher.Sum64())
}

// ConfigFromConfig converts a controller MemoizationConfig (with DB credentials, connection
// settings, etc.) into the smaller Config struct used by the migration and query layers.
// Returns sensible defaults when cfg is nil.
func ConfigFromConfig(cfg *config.MemoizationConfig) Config {
	if cfg == nil {
		return Config{TableName: defaultTableName}
	}
	return Config{
		TableName:     TableName(cfg),
		SkipMigration: cfg.SkipMigration,
	}
}

// SessionProxyFromConfig creates a connection and applies its pool settings.
func SessionProxyFromConfig(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, cfg *config.MemoizationConfig) (*sqldb.SessionProxy, error) {
	if cfg == nil {
		return nil, fmt.Errorf("memoization configuration is required")
	}
	sessionProxy, err := sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{
		KubectlConfig: kubectlConfig,
		Namespace:     namespace,
		DBConfig:      cfg.DBConfig,
	})
	if err != nil {
		return nil, err
	}
	sqldb.ConfigureDBSession(sessionProxy.Session(ctx), cfg.ConnectionPool)
	return sessionProxy, nil
}

// Migrate runs database migrations for the memoization cache table. It is a no-op when
// cfg.SkipMigration is true. Returns an error if migration fails.
func Migrate(ctx context.Context, sessionProxy *sqldb.SessionProxy, cfg Config) error {
	if sessionProxy == nil {
		return nil
	}
	logger := logging.RequireLoggerFromContext(ctx)
	if err := validateTableName(cfg.TableName); err != nil {
		return err
	}
	if cfg.SkipMigration {
		logger.Info(ctx, "Memoization db migration skipped")
		return nil
	}
	logger.Info(ctx, "Running memoization db migration")
	if err := migrate(ctx, sessionProxy.Session(ctx), sessionProxy.DBType(), cfg.TableName); err != nil {
		return err
	}
	logger.Info(ctx, "Memoization db migration complete")
	return nil
}
