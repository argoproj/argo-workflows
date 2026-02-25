package db

import (
	"context"
	"time"

	"github.com/upper/db/v4"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

type dbConfig struct {
	LimitTable                string
	StateTable                string
	ControllerTable           string
	LockTable                 string
	ControllerName            string
	InactiveControllerTimeout time.Duration
	SkipMigration             bool
}

type DBInfo struct {
	Config  dbConfig
	Session db.Session
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

func (d *DBInfo) Migrate(ctx context.Context) {
	if d.Session == nil {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx)
	logger.Info(ctx, "Setting up sync manager database")
	if !d.Config.SkipMigration {
		err := migrate(ctx, d.Session, &d.Config)
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

func DBConfigFromConfig(config *config.SyncConfig) dbConfig {
	if config == nil {
		return dbConfig{}
	}
	return dbConfig{
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

func DBSessionFromConfigWithCreds(config *config.SyncConfig, username, password string) db.Session {
	if config == nil {
		return nil
	}
	dbSession, err := sqldb.CreateDBSessionWithCreds(config.DBConfig, username, password)
	if err != nil {
		// Carry on anyway, but database sync locks won't work
		return nil
	}
	return dbSession
}

func DBSessionFromConfig(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, config *config.SyncConfig) db.Session {
	if config == nil {
		return nil
	}
	dbSession, err := sqldb.CreateDBSession(ctx, kubectlConfig, namespace, config.DBConfig)
	if err != nil {
		// Carry on anyway, but database sync locks won't work
		return nil
	}
	return dbSession
}
