package db

import (
	"context"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// Config holds database configuration for sync operations.
type Config struct {
	LimitTable                string
	StateTable                string
	ControllerTable           string
	LockTable                 string
	ControllerName            string
	InactiveControllerTimeout time.Duration
	SkipMigration             bool
}

type Info struct {
	Config       Config
	SessionProxy *sqldb.SessionProxy
}

const (
	DefaultDBPollSeconds               = 10
	DefaultDBHeartbeatSeconds          = 60
	DefaultDBInactiveControllerSeconds = 600

	defaultLimitTableName      = "sync_limit"
	defaultStateTableName      = "sync_state"
	defaultControllerTableName = "sync_controller"
	defaultLockTableName       = "sync_lock"
)

func defaultTable(tableName, defaultName string) string {
	if tableName == "" {
		return defaultName
	}
	return tableName
}

func SecondsToDurationWithDefault(value *int, defaultSeconds int) time.Duration {
	dur := time.Duration(defaultSeconds) * time.Second
	if value != nil {
		dur = time.Duration(*value) * time.Second
	}
	return dur
}

func (d *Info) Migrate(ctx context.Context) {
	if d.SessionProxy == nil {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx)
	logger.Info(ctx, "Setting up sync manager database")
	if !d.Config.SkipMigration {
		err := migrate(ctx, d.SessionProxy, &d.Config)
		if err != nil {
			// Carry on anyway, but database sync locks won't work
			logger.WithError(err).Warn(ctx, "cannot initialize semaphore database, database sync locks won't work")
			d.SessionProxy = nil
		} else {
			logger.Info(ctx, "Sync db migration complete")
		}
	} else {
		logger.Info(ctx, "Sync db migration skipped")
	}
}

func ConfigFromConfig(syncConfig *config.SyncConfig) Config {
	if syncConfig == nil {
		return Config{}
	}
	return Config{
		LimitTable:      defaultTable(syncConfig.LimitTableName, defaultLimitTableName),
		StateTable:      defaultTable(syncConfig.StateTableName, defaultStateTableName),
		ControllerTable: defaultTable(syncConfig.ControllerTableName, defaultControllerTableName),
		LockTable:       defaultTable(syncConfig.LockTableName, defaultLockTableName),
		ControllerName:  syncConfig.ControllerName,
		InactiveControllerTimeout: SecondsToDurationWithDefault(syncConfig.InactiveControllerSeconds,
			DefaultDBInactiveControllerSeconds),
		SkipMigration: syncConfig.SkipMigration,
	}
}

func SessionFromConfigWithCreds(syncConfig *config.SyncConfig, username, password string) *sqldb.SessionProxy {
	if syncConfig == nil {
		return nil
	}
	sessionProxy, err := sqldb.NewSessionProxy(context.Background(), sqldb.SessionProxyConfig{
		DBConfig: syncConfig.DBConfig,
		Username: username,
		Password: password,
	})
	if err != nil {
		// Carry on anyway, but database sync locks won't work
		return nil
	}
	return sessionProxy
}

func SessionFromConfig(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, syncConfig *config.SyncConfig) *sqldb.SessionProxy {
	if syncConfig == nil {
		return nil
	}
	sessionProxy, err := sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{
		KubectlConfig: kubectlConfig,
		Namespace:     namespace,
		DBConfig:      syncConfig.DBConfig,
	})
	if err != nil {
		// Carry on anyway, but database sync locks won't work
		return nil
	}
	return sessionProxy
}
