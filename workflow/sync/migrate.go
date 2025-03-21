package sync

import (
	"context"

	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

const (
	versionTable = "sync_schema_history"
)

func migrate(ctx context.Context, session db.Session, config *dbConfig) (err error) {
	return sqldb.Migrate(ctx, session, versionTable, []sqldb.Change{
		sqldb.AnsiSQLChange(`create table if not exists ` + config.limitTable + ` (
    name varchar(256) not null,
    sizelimit int,
    primary key (name)
)`),
		sqldb.AnsiSQLChange(`create unique index ilimit_name on ` + config.limitTable + ` (name)`),
		sqldb.AnsiSQLChange(`create table if not exists ` + config.controllerTable + ` (
    controller varchar(64) not null,
    time timestamp,
    primary key (controller)
)`),
		sqldb.AnsiSQLChange(`create unique index icontroller_name on ` + config.controllerTable + ` (controller)`),
		sqldb.AnsiSQLChange(`create table if not exists ` + config.stateTable + ` (
    name varchar(256),
    workflowkey varchar(256),
    controller varchar(64) not null,
    mutex boolean,
    held boolean,
    priority int,
    time timestamp,
    primary key(name, workflowkey, controller, mutex)
)`),
		sqldb.AnsiSQLChange(`create index istate_name on ` + config.stateTable + ` (name)`),
		sqldb.AnsiSQLChange(`create index istate_workflowkey on ` + config.stateTable + ` (workflowkey)`),
		sqldb.AnsiSQLChange(`create index istate_controller on ` + config.stateTable + ` (controller)`),
		sqldb.AnsiSQLChange(`create index istate_held on ` + config.stateTable + ` (held)`),
	})
}
