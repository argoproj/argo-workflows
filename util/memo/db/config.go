package db

import (
	"context"
	"fmt"
	"regexp"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const defaultTableName = "cache_entries"

// Leave room for the schema-history suffix within PostgreSQL's 63-byte identifier limit.
var validTableName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,47}$`)

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
		return fmt.Errorf("invalid table name %q: must start with a lowercase letter and contain only lowercase letters, digits or underscores (maximum 48 characters)", tableName)
	}
	return nil
}

// VersionTableName returns the migration history table associated with a cache table.
func VersionTableName(tableName string) string {
	return tableName + "_schema_history"
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
	if err := sqldb.Migrate(ctx, sessionProxy.Session(ctx), sessionProxy.DBType(), VersionTableName(cfg.TableName), MigrateChanges(cfg.TableName, sessionProxy.DBType())); err != nil {
		return err
	}
	logger.Info(ctx, "Memoization db migration complete")
	return nil
}
