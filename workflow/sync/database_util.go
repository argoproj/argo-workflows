package sync

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
	"k8s.io/client-go/kubernetes"
)

type dbConfig struct {
	limitTable                string
	stateTable                string
	controllerTable           string
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
	log.Infof("Setting up sync manager database")
	if !d.config.skipMigration {
		err := migrate(ctx, d.session, &d.config)
		if err != nil {
			// Carry on anyway, but database sync locks won't work
			log.Warnf("cannot initialize semaphore database: %v", err)
			d.session = nil
		} else {
			log.Infof("Sync db migration complete")
		}
	} else {
		log.Infof("Sync db migration skipped")
	}
}

// func (d *dbInfo) getConfig() dbConfig {
// 	return d.config
// }

// func (d *dbInfo) getSession() db.Session {
// 	return d.session
// }

func dbConfigFromConfig(config *config.SyncConfig) dbConfig {
	if config == nil {
		return dbConfig{}
	}
	return dbConfig{
		limitTable:      defaultTable(config.LimitTableName, defaultLimitTableName),
		stateTable:      defaultTable(config.StateTableName, defaultStateTableName),
		controllerTable: defaultTable(config.ControllerTableName, defaultControllerTableName),
		controllerName:  config.ControllerName,
		inactiveControllerTimeout: secondsToDurationWithDefault(config.InactiveControllerSeconds,
			defaultDBInactiveControllerSeconds),
		skipMigration: config.SkipMigration,
	}
}

func dbSessionFromConfigWithCreds(ctx context.Context, config *config.SyncConfig, username, password string) db.Session {
	if config == nil {
		return nil
	}
	dbSession, err := sqldb.CreateDBSessionWithCreds(ctx, config.DBConfig, username, password)
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
