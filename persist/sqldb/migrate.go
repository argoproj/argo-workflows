package sqldb

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"upper.io/db.v3/lib/sqlbuilder"
)

type Migrate interface {
	Exec(ctx context.Context) error
}

func NewMigrate(session sqlbuilder.Database, dbModel dbModel, clusterName string) Migrate {
	return migrate{session, dbModel, clusterName}
}

type migrate struct {
	session     sqlbuilder.Database
	dbModel     dbModel
	clusterName string
}

type change interface {
	apply(session sqlbuilder.Database) error
}

func ternary(condition bool, left, right change) change {
	if condition {
		return left
	} else {
		return right
	}
}

func (m migrate) Exec(ctx context.Context) error {
	{
		if m.dbModel.schema != "public" && m.dbModel.schema != "" {
			_, err := m.session.Exec(fmt.Sprintf("create schema if not exists %s", m.dbModel.schema))
			if err != nil {
				return err
			}
		}
		// poor mans SQL migration
		_, err := m.session.Exec(`create table if not exists ` + m.dbModel.tables.schemaHistory + `(schema_version int not null)`)
		if err != nil {
			return err
		}
		rs, err := m.session.Query(`select schema_version from ` + m.dbModel.tables.schemaHistory)
		if err != nil {
			return err
		}
		if !rs.Next() {
			_, err := m.session.Exec(fmt.Sprintf("insert into %s values(-1)", m.dbModel.tables.schemaHistory))
			if err != nil {
				return err
			}
		}
		err = rs.Close()
		if err != nil {
			return err
		}
	}
	dbType := dbTypeFor(m.session)

	log.WithFields(log.Fields{"clusterName": m.clusterName, "dbType": dbType}).Info("Migrating database schema")

	// try and make changes idempotent, as it is possible for the change to apply, but the archive update to fail
	// and therefore try and apply again next try

	// log.Info("Schema: " + )

	for changeSchemaVersion, change := range []change{
		ansiSQLChange(`create table if not exists ` + m.dbModel.tables.workflows + ` (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp,
    finishedat timestamp,
    primary key (id, namespace)
)`),
		ansiSQLChange(`create unique index idx_name on ` + m.dbModel.tables.workflows + ` (name)`),
		ansiSQLChange(`create table if not exists ` + m.dbModel.tables.workflowsHistory + `(
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp,
    finishedat timestamp,
    primary key (id, namespace)
)`),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflowsHistory + ` rename to argo_archived_workflows`),
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index idx_name on `+m.dbModel.tables.workflows),
			ansiSQLChange(`drop index `+m.dbModel.indexes.idxName),
		),
		ansiSQLChange(`create unique index idx_name on ` + m.dbModel.tables.workflows + `(name, namespace)`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` drop primary key`),
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` drop constraint `+m.dbModel.tableName+`_pkey`),
		),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` add primary key(name,namespace)`),
		// huh - why does the pkey not have the same name as the table - history
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` drop primary key`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` drop constraint argo_workflow_history_pkey`),
		),
		ansiSQLChange(`alter table ` + m.dbModel.tables.archivedWorkflows + ` add primary key(id)`),
		// ***
		// THE CHANGES ABOVE THIS LINE MAY BE IN PER-PRODUCTION SYSTEMS - DO NOT CHANGE THEM
		// ***
		ansiSQLChange(`alter table ` + m.dbModel.tables.archivedWorkflows + ` rename column id to uid`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column uid varchar(128) not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column uid set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column phase varchar(25) not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column phase set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column namespace varchar(256) not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column namespace set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column workflow text not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column workflow set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column startedat timestamp not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column startedat set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column finishedat timestamp not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column finishedat set not null`),
		),
		ansiSQLChange(`alter table ` + m.dbModel.tables.archivedWorkflows + ` add clustername varchar(64)`), // DNS entry can only be max 63 bytes
		ansiSQLChange(`update ` + m.dbModel.tables.archivedWorkflows + ` set clustername = '` + m.clusterName + `' where clustername is null`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column clustername varchar(64) not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column clustername set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` drop primary key`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` drop constraint argo_archived_workflows_pkey`),
		),
		ansiSQLChange(`alter table ` + m.dbModel.tables.archivedWorkflows + ` add primary key(clustername,uid)`),
		ansiSQLChange(`create index argo_archived_workflows_i1 on ` + m.dbModel.tables.archivedWorkflows + ` (clustername,namespace)`),
		// argo_archived_workflows now looks like:
		// clustername(not null) | uid(not null) | | name (null) | phase(not null) | namespace(not null) | workflow(not null) | startedat(not null)  | finishedat(not null)
		// remove unused columns
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` drop column phase`),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` drop column startedat`),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` drop column finishedat`),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` rename column id to uid`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` modify column uid varchar(128) not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` alter column uid set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` modify column namespace varchar(256) not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` alter column namespace set not null`),
		),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` add column clustername varchar(64)`), // DNS cannot be longer than 64 bytes
		ansiSQLChange(`update ` + m.dbModel.tables.workflows + ` set clustername = '` + m.clusterName + `' where clustername is null`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` modify column clustername varchar(64) not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` alter column clustername set not null`),
		),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` add column version varchar(64)`),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` add column nodes text`),
		backfillNodes{dbModel: m.dbModel},
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` modify column nodes text not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` alter column nodes set not null`),
		),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` drop column workflow`),
		// add a timestamp column to indicate updated time
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` add column updatedat timestamp not null default current_timestamp`),
		// remove the old primary key and add a new one
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` drop primary key`),
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` drop constraint `+m.dbModel.tableName+`_pkey`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index idx_name on `+m.dbModel.tables.workflows),
			ansiSQLChange(`drop index `+m.dbModel.indexes.idxName),
		),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` drop column name`),
		ansiSQLChange(`alter table ` + m.dbModel.tables.workflows + ` add primary key(clustername,uid,version)`),
		ansiSQLChange(`create index ` + m.dbModel.tableName + `_i1 on ` + m.dbModel.tables.workflows + ` (clustername,namespace)`),
		// argo_workflows now looks like:
		//  clustername(not null) | uid(not null) | namespace(not null) | version(not null) | nodes(not null) | updatedat(not null)
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column workflow json not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column workflow type json using workflow::json`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column name varchar(256) not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column name set not null`),
		),
		// clustername(not null) | uid(not null) | | name (not null) | phase(not null) | namespace(not null) | workflow(not null) | startedat(not null)  | finishedat(not null)
		ansiSQLChange(`create index ` + m.dbModel.tableName + `_i2 on ` + m.dbModel.tables.workflows + ` (clustername,namespace,updatedat)`),
		// The argo_archived_workflows_labels is really provided as a way to create queries on labels that are fast because they
		// use indexes. When displaying, it might be better to look at the `workflow` column.
		// We could have added a `labels` column to argo_archived_workflows, but then we would have had to do free-text
		// queries on it which would be slow due to having to table scan.
		// The key has an optional prefix(253 chars) + '/' + name(63 chars)
		// Why is the key called "name" not "key"? Key is an SQL reserved word.
		ansiSQLChange(`create table if not exists ` + m.dbModel.tables.archivedWorkflowsLabels + ` (
	clustername varchar(64) not null,
	uid varchar(128) not null,
    name varchar(317) not null,
    value varchar(63) not null,
    primary key (clustername, uid, name),
	foreign key (clustername, uid) references ` + m.dbModel.tables.archivedWorkflows + `(clustername, uid) on delete cascade
)`),
		// MySQL can only store 64k in a TEXT field, both MySQL and Posgres can store 1GB in JSON.
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` modify column nodes json not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.workflows+` alter column nodes type json using nodes::json`),
		),
		// add instanceid column to table argo_archived_workflows
		ansiSQLChange(`alter table ` + m.dbModel.tables.archivedWorkflows + ` add column instanceid varchar(64)`),
		ansiSQLChange(`update ` + m.dbModel.tables.archivedWorkflows + ` set instanceid = '' where instanceid is null`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` modify column instanceid varchar(64) not null`),
			ansiSQLChange(`alter table `+m.dbModel.tables.archivedWorkflows+` alter column instanceid set not null`),
		),
		// drop argo_archived_workflows index
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index `+m.dbModel.tables.archivedWorkflows+`_i1 on `+m.dbModel.tables.archivedWorkflows+``),
			ansiSQLChange(`drop index `+m.dbModel.tables.archivedWorkflows+`_i1`),
		),
		// add argo_archived_workflows index
		ansiSQLChange(`create index argo_archived_workflows_i1 on ` + m.dbModel.tables.archivedWorkflows + ` (clustername,instanceid,namespace)`),
		// drop m.dbModel.tableName indexes
		// xxx_i1 is not needed because xxx_i2 already covers it, drop both and recreat an index named xxx_i1
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index `+m.dbModel.tableName+`_i1 on `+m.dbModel.tables.workflows),
			ansiSQLChange(`drop index `+m.dbModel.tables.workflows+`_i1`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index `+m.dbModel.tableName+`_i2 on `+m.dbModel.tables.workflows),
			ansiSQLChange(`drop index `+m.dbModel.tables.workflows+`_i2`),
		),
		// add m.dbModel.tableName index
		ansiSQLChange(`create index ` + m.dbModel.tableName + `_i1 on ` + m.dbModel.tables.workflows + ` (clustername,namespace,updatedat)`),
	} {
		err := m.applyChange(ctx, changeSchemaVersion, change)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m migrate) applyChange(ctx context.Context, changeSchemaVersion int, c change) error {
	// schemaHistory := m.parseTableName("schema_history")
	tx, err := m.session.NewTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	update_query := fmt.Sprintf("update %s set schema_version = ? where schema_version = ?", m.dbModel.tables.schemaHistory)
	rs, err := tx.Exec(update_query, changeSchemaVersion, changeSchemaVersion-1)
	if err != nil {
		return err
	}
	rowsAffected, err := rs.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 1 {
		log.WithFields(log.Fields{"changeSchemaVersion": changeSchemaVersion, "change": c}).Info("applying database change")
		err := c.apply(m.session)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
