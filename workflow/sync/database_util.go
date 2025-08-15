package sync

import (
	"context"
	"time"

	"github.com/upper/db/v4"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

type dbConfig struct {
	limitTable                string
	stateTable                string
	controllerTable           string
	lockTable                 string
	controllerName            string
	inactiveControllerTimeout time.Duration
	skipMigration             bool
}

type dbInfo struct {
	config  dbConfig
	session db.Session
}

const (
	defaultDBPollSeconds               = 10
	defaultDBHeartbeatSeconds          = 60
	defaultDBInactiveControllerSeconds = 600

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

func secondsToDurationWithDefault(value *int, defaultSeconds int) time.Duration {
	dur := time.Duration(defaultSeconds) * time.Second
	if value != nil {
		dur = time.Duration(*value) * time.Second
	}
	return dur
}

func (d *dbInfo) migrate(ctx context.Context) {
	if d.session == nil {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx)
	logger.Info(ctx, "Setting up sync manager database")
	if !d.config.skipMigration {
		err := migrate(ctx, d.session, &d.config)
		if err != nil {
			// Carry on anyway, but database sync locks won't work
			logger.WithError(err).Warn(ctx, "cannot initialize semaphore database, database sync locks won't work")
			d.session = nil
		} else {
			logger.Info(ctx, "Sync db migration complete")
		}
	} else {
		logger.Info(ctx, "Sync db migration skipped")
	}
}

func dbConfigFromConfig(config *config.SyncConfig) dbConfig {
	if config == nil {
		return dbConfig{}
	}
	return dbConfig{
		limitTable:      defaultTable(config.LimitTableName, defaultLimitTableName),
		stateTable:      defaultTable(config.StateTableName, defaultStateTableName),
		controllerTable: defaultTable(config.ControllerTableName, defaultControllerTableName),
		lockTable:       defaultTable(config.LockTableName, defaultLockTableName),
		controllerName:  config.ControllerName,
		inactiveControllerTimeout: secondsToDurationWithDefault(config.InactiveControllerSeconds,
			defaultDBInactiveControllerSeconds),
		skipMigration: config.SkipMigration,
	}
}

func dbSessionFromConfigWithCreds(config *config.SyncConfig, username, password string) db.Session {
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

func dbSessionFromConfig(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, config *config.SyncConfig) db.Session {
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
