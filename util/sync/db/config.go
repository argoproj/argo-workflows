package db

import (
	"context"
	"time"

	"github.com/upper/db/v4"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
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
	Config  Config
	Session db.Session
	DBType  sqldb.DBType
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
	if d.Session == nil {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx)
	logger.Info(ctx, "Setting up sync manager database")
	if !d.Config.SkipMigration {
		err := migrate(ctx, d.Session, d.DBType, &d.Config)
		if err != nil {
			// Carry on anyway, but database sync locks won't work
			logger.WithError(err).Warn(ctx, "cannot initialize semaphore database, database sync locks won't work")
			d.Session = nil
		} else {
			logger.Info(ctx, "Sync db migration complete")
		}
	} else {
		logger.Info(ctx, "Sync db migration skipped")
	}
}

func ConfigFromConfig(config *config.SyncConfig) Config {
	if config == nil {
		return Config{}
	}
	return Config{
		LimitTable:      defaultTable(config.LimitTableName, defaultLimitTableName),
		StateTable:      defaultTable(config.StateTableName, defaultStateTableName),
		ControllerTable: defaultTable(config.ControllerTableName, defaultControllerTableName),
		LockTable:       defaultTable(config.LockTableName, defaultLockTableName),
		ControllerName:  config.ControllerName,
		InactiveControllerTimeout: SecondsToDurationWithDefault(config.InactiveControllerSeconds,
			DefaultDBInactiveControllerSeconds),
		SkipMigration: config.SkipMigration,
	}
}

func SessionFromConfigWithCreds(config *config.SyncConfig, username, password string) (db.Session, sqldb.DBType) {
	if config == nil {
		return nil, ""
	}
	dbSession, dbType, err := sqldb.CreateDBSessionWithCreds(config.DBConfig, username, password)
	if err != nil {
		// Carry on anyway, but database sync locks won't work
		return nil, ""
	}
	return dbSession, dbType
}

func SessionFromConfig(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, config *config.SyncConfig) (db.Session, sqldb.DBType) {
	if config == nil {
		return nil, ""
	}
	dbSession, dbType, err := sqldb.CreateDBSession(ctx, kubectlConfig, namespace, config.DBConfig)
	if err != nil {
		// Carry on anyway, but database sync locks won't work
		return nil, ""
	}
	return dbSession, dbType
}
