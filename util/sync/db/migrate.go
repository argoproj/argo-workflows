package db

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

const (
	versionTable = "sync_schema_history"
)

// migrate applies the schema changes required for synchronization by creating the limit, controller, state, and lock tables (and their associated indexes) using the table names provided in config.
// It returns an error if the underlying migration operation fails.
func migrate(ctx context.Context, sessionProxy *sqldb.SessionProxy, config *dbConfig) (err error) {
	return sqldb.Migrate(ctx, sessionProxy.Session(), versionTable, []sqldb.Change{
		sqldb.AnsiSQLChange(`create table if not exists ` + config.LimitTable + ` (
    name varchar(256) not null,
    sizelimit int,
    primary key (name)
)`),
		sqldb.AnsiSQLChange(`create unique index ilimit_name on ` + config.LimitTable + ` (name)`),
		sqldb.AnsiSQLChange(`create table if not exists ` + config.ControllerTable + ` (
    controller varchar(64) not null,
    time timestamp,
    primary key (controller)
)`),
		sqldb.AnsiSQLChange(`create unique index icontroller_name on ` + config.ControllerTable + ` (controller)`),
		sqldb.AnsiSQLChange(`create table if not exists ` + config.StateTable + ` (
    name varchar(256),
    workflowkey varchar(256),
    controller varchar(64) not null,
    held boolean,
    priority int,
    time timestamp,
    primary key(name, workflowkey, controller)
)`),
		sqldb.AnsiSQLChange(`create index istate_name on ` + config.StateTable + ` (name)`),
		sqldb.AnsiSQLChange(`create index istate_workflowkey on ` + config.StateTable + ` (workflowkey)`),
		sqldb.AnsiSQLChange(`create index istate_controller on ` + config.StateTable + ` (controller)`),
		sqldb.AnsiSQLChange(`create index istate_held on ` + config.StateTable + ` (held)`),
		sqldb.AnsiSQLChange(`create table if not exists ` + config.LockTable + ` (
    name varchar(256),
    controller varchar(64) not null,
    time timestamp,
    primary key(name)
)`),
		sqldb.AnsiSQLChange(`create unique index ilock_name on ` + config.LockTable + ` (name)`),
	})
}