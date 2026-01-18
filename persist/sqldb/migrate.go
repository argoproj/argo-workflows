package sqldb

import (
	"context"

	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

const (
	versionTable = "schema_history"
)

func Migrate(ctx context.Context, session db.Session, clusterName, tableName string) (err error) {
	dbType := sqldb.DBTypeFor(session)
	return sqldb.Migrate(ctx, session, versionTable, []sqldb.Change{
		sqldb.AnsiSQLChange(`create table if not exists ` + tableName + ` (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp default CURRENT_TIMESTAMP,
    finishedat timestamp default CURRENT_TIMESTAMP,
    creationtimestamp timestamp default CURRENT_TIMESTAMP,
    primary key (id, namespace)
)`),
		sqldb.AnsiSQLChange(`create unique index idx_name on ` + tableName + ` (name)`),
		sqldb.AnsiSQLChange(`create table if not exists argo_workflow_history (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp default CURRENT_TIMESTAMP,
    finishedat timestamp default CURRENT_TIMESTAMP,
    primary key (id, namespace)
)`),
		sqldb.AnsiSQLChange(`alter table argo_workflow_history rename to argo_archived_workflows`),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`drop index idx_name on ` + tableName),
			sqldb.Postgres: sqldb.AnsiSQLChange(`drop index idx_name`),
		}),
		sqldb.AnsiSQLChange(`create unique index idx_name on ` + tableName + `(name, namespace)`),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table ` + tableName + ` drop primary key`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table ` + tableName + ` drop constraint ` + tableName + `_pkey`),
		}),
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` add primary key(name,namespace)`),
		// huh - why does the pkey not have the same name as the table - history
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows drop primary key`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows drop constraint argo_workflow_history_pkey`),
		}),
		sqldb.AnsiSQLChange(`alter table argo_archived_workflows add primary key(id)`),
		// ***
		// THE CHANGES ABOVE THIS LINE MAY BE IN PER-PRODUCTION SYSTEMS - DO NOT CHANGE THEM
		// ***
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows change column id uid varchar(128)`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows rename column id to uid`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column uid varchar(128) not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column uid set not null`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column phase varchar(25) not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column phase set not null`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column namespace varchar(256) not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column namespace set not null`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column workflow text not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column workflow set not null`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column startedat timestamp not null default CURRENT_TIMESTAMP`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column startedat set not null`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column finishedat timestamp not null default CURRENT_TIMESTAMP`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column finishedat set not null`),
		}),
		sqldb.AnsiSQLChange(`alter table argo_archived_workflows add clustername varchar(64)`), // DNS entry can only be max 63 bytes
		sqldb.AnsiSQLChange(`update argo_archived_workflows set clustername = '` + clusterName + `' where clustername is null`),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column clustername varchar(64) not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column clustername set not null`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows drop primary key`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows drop constraint argo_archived_workflows_pkey`),
		}),
		sqldb.AnsiSQLChange(`alter table argo_archived_workflows add primary key(clustername,uid)`),
		sqldb.AnsiSQLChange(`create index argo_archived_workflows_i1 on argo_archived_workflows (clustername,namespace)`),
		// argo_archived_workflows now looks like:
		// clustername(not null) | uid(not null) | | name (null) | phase(not null) | namespace(not null) | workflow(not null) | startedat(not null)  | finishedat(not null)
		// remove unused columns
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` drop column phase`),
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` drop column startedat`),
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` drop column finishedat`),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table ` + tableName + ` change column id uid varchar(128)`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table ` + tableName + ` rename column id to uid`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table ` + tableName + ` modify column uid varchar(128) not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table ` + tableName + ` alter column uid set not null`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table ` + tableName + ` modify column namespace varchar(256) not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table ` + tableName + ` alter column namespace set not null`),
		}),
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` add column clustername varchar(64)`), // DNS cannot be longer than 64 bytes
		sqldb.AnsiSQLChange(`update ` + tableName + ` set clustername = '` + clusterName + `' where clustername is null`),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table ` + tableName + ` modify column clustername varchar(64) not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table ` + tableName + ` alter column clustername set not null`),
		}),
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` add column version varchar(64)`),
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` add column nodes text`),
		backfillNodes{tableName: tableName},
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table ` + tableName + ` modify column nodes text not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table ` + tableName + ` alter column nodes set not null`),
		}),
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` drop column workflow`),
		// add a timestamp column to indicate updated time
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` add column updatedat timestamp not null default current_timestamp`),
		// remove the old primary key and add a new one
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table ` + tableName + ` drop primary key`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table ` + tableName + ` drop constraint ` + tableName + `_pkey`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`drop index idx_name on ` + tableName),
			sqldb.Postgres: sqldb.AnsiSQLChange(`drop index idx_name`),
		}),
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` drop column name`),
		sqldb.AnsiSQLChange(`alter table ` + tableName + ` add primary key(clustername,uid,version)`),
		sqldb.AnsiSQLChange(`create index ` + tableName + `_i1 on ` + tableName + ` (clustername,namespace)`),
		// argo_workflows now looks like:
		//  clustername(not null) | uid(not null) | namespace(not null) | version(not null) | nodes(not null) | updatedat(not null)
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column workflow json not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column workflow type json using workflow::json`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column name varchar(256) not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column name set not null`),
		}),
		// clustername(not null) | uid(not null) | | name (not null) | phase(not null) | namespace(not null) | workflow(not null) | startedat(not null)  | finishedat(not null)
		sqldb.AnsiSQLChange(`create index ` + tableName + `_i2 on ` + tableName + ` (clustername,namespace,updatedat)`),
		// The argo_archived_workflows_labels is really provided as a way to create queries on labels that are fast because they
		// use indexes. When displaying, it might be better to look at the `workflow` column.
		// We could have added a `labels` column to argo_archived_workflows, but then we would have had to do free-text
		// queries on it which would be slow due to having to table scan.
		// The key has an optional prefix(253 chars) + '/' + name(63 chars)
		// Why is the key called "name" not "key"? Key is an SQL reserved word.
		sqldb.AnsiSQLChange(`create table if not exists argo_archived_workflows_labels (
	clustername varchar(64) not null,
	uid varchar(128) not null,
    name varchar(317) not null,
    value varchar(63) not null,
    primary key (clustername, uid, name),
 	foreign key (clustername, uid) references argo_archived_workflows(clustername, uid) on delete cascade
)`),
		// MySQL can only store 64k in a TEXT field, both MySQL and Posgres can store 1GB in JSON.
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table ` + tableName + ` modify column nodes json not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table ` + tableName + ` alter column nodes type json using nodes::json`),
		}),
		// add instanceid column to table argo_archived_workflows
		sqldb.AnsiSQLChange(`alter table argo_archived_workflows add column instanceid varchar(64)`),
		sqldb.AnsiSQLChange(`update argo_archived_workflows set instanceid = '' where instanceid is null`),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column instanceid varchar(64) not null`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column instanceid set not null`),
		}),
		// drop argo_archived_workflows index
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`drop index argo_archived_workflows_i1 on argo_archived_workflows`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`drop index argo_archived_workflows_i1`),
		}),
		// add argo_archived_workflows index
		sqldb.AnsiSQLChange(`create index argo_archived_workflows_i1 on argo_archived_workflows (clustername,instanceid,namespace)`),
		// change argo_archived_workflows_i1 index to add startedat DESC to resolve MySQL out of sort memory issues. #14240
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`drop index argo_archived_workflows_i1 on argo_archived_workflows`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`drop index argo_archived_workflows_i1`),
		}),
		sqldb.AnsiSQLChange(`create index argo_archived_workflows_i1 on argo_archived_workflows (clustername, instanceid, namespace, startedat DESC)`),
		// drop tableName indexes
		// xxx_i1 is not needed because xxx_i2 already covers it, drop both and recreat an index named xxx_i1
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`drop index ` + tableName + `_i1 on ` + tableName),
			sqldb.Postgres: sqldb.AnsiSQLChange(`drop index ` + tableName + `_i1`),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`drop index ` + tableName + `_i2 on ` + tableName),
			sqldb.Postgres: sqldb.AnsiSQLChange(`drop index ` + tableName + `_i2`),
		}),
		// add tableName index
		sqldb.AnsiSQLChange(`create index ` + tableName + `_i1 on ` + tableName + ` (clustername,namespace,updatedat)`),
		// index to find records that need deleting, this omits namespaces as this might be null
		sqldb.AnsiSQLChange(`create index argo_archived_workflows_i2 on argo_archived_workflows (clustername,instanceid,finishedat)`),
		// add argo_archived_workflows name index for prefix searching performance
		sqldb.AnsiSQLChange(`create index argo_archived_workflows_i3 on argo_archived_workflows (clustername,instanceid,name)`),
		// add indexes for list archived workflow performance. #8836
		sqldb.AnsiSQLChange(`create index argo_archived_workflows_i4 on argo_archived_workflows (startedat)`),
		sqldb.AnsiSQLChange(`create index argo_archived_workflows_labels_i1 on argo_archived_workflows_labels (name,value)`),
		// PostgreSQL only: convert argo_archived_workflows.workflow column to JSONB for performance and consistency with MySQL. #13779
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column workflow set data type jsonb using workflow::jsonb`),
		}),
		// change argo_archived_workflows_i4 index to include clustername so MySQL uses it for listing archived workflows. #13601
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`drop index argo_archived_workflows_i4 on argo_archived_workflows`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`drop index argo_archived_workflows_i4`),
		}),
		sqldb.AnsiSQLChange(`create index argo_archived_workflows_i4 on argo_archived_workflows (clustername, startedat)`),
		// add creationtimestamp column to argo_archived_workflows table
		sqldb.AnsiSQLChange(`alter table argo_archived_workflows add column creationtimestamp timestamp null`),
		sqldb.AnsiSQLChange(`update argo_archived_workflows set creationtimestamp = startedat where creationtimestamp is null`),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.MySQL:    sqldb.AnsiSQLChange(`alter table argo_archived_workflows modify column creationtimestamp timestamp not null default CURRENT_TIMESTAMP`),
			sqldb.Postgres: sqldb.AnsiSQLChange(`alter table argo_archived_workflows alter column creationtimestamp set default CURRENT_TIMESTAMP`),
		}),
		// add index on creationtimestamp column
		sqldb.AnsiSQLChange(`create index argo_archived_workflows_i5 on argo_archived_workflows (creationtimestamp)`),
	})
}
