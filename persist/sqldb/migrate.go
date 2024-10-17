package sqldb

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"
)

type Migrate interface {
	Exec(ctx context.Context) error
}

func NewMigrate(session db.Session, clusterName string, tableName string) Migrate {
	return migrate{session, clusterName, tableName}
}

type migrate struct {
	session     db.Session
	clusterName string
	tableName   string
}

type change interface {
	apply(session db.Session) error
}

type noop struct{}

func (s noop) apply(session db.Session) error {
	return nil
}

func ternary(condition bool, left, right change) change {
	if condition {
		return left
	} else {
		return right
	}
}

func (m migrate) Exec(ctx context.Context) (err error) {
	{
		// poor mans SQL migration
		_, err = m.session.SQL().Exec("create table if not exists schema_history(schema_version int not null)")
		if err != nil {
			return err
		}
		rs, err := m.session.SQL().Query("select schema_version from schema_history")
		if err != nil {
			return err
		}
		defer func() {
			tmpErr := rs.Close()
			if err == nil {
				err = tmpErr
			}
		}()
		if !rs.Next() {
			_, err := m.session.SQL().Exec("insert into schema_history values(-1)")
			if err != nil {
				return err
			}
		} else if err := rs.Err(); err != nil {
			return err
		}
	}
	dbType := dbTypeFor(m.session)

	log.WithFields(log.Fields{"clusterName": m.clusterName, "dbType": dbType}).Info("Migrating database schema")

	// try and make changes idempotent, as it is possible for the change to apply, but the archive update to fail
	// and therefore try and apply again next try

	for changeSchemaVersion, change := range []change{
		ansiSQLChange(`create table if not exists ` + m.tableName + ` (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp default CURRENT_TIMESTAMP,
    finishedat timestamp default CURRENT_TIMESTAMP,
    primary key (id, namespace)
)`),
		ansiSQLChange(`create unique index idx_name on ` + m.tableName + ` (name)`),
		ansiSQLChange(`create table if not exists argo_workflow_history (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp default CURRENT_TIMESTAMP,
    finishedat timestamp default CURRENT_TIMESTAMP,
    primary key (id, namespace)
)`),
		ansiSQLChange(`alter table argo_workflow_history rename to argo_archived_workflows`),
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index idx_name on `+m.tableName),
			ansiSQLChange(`drop index idx_name`),
		),
		ansiSQLChange(`create unique index idx_name on ` + m.tableName + `(name, namespace)`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.tableName+` drop primary key`),
			ansiSQLChange(`alter table `+m.tableName+` drop constraint `+m.tableName+`_pkey`),
		),
		ansiSQLChange(`alter table ` + m.tableName + ` add primary key(name,namespace)`),
		// huh - why does the pkey not have the same name as the table - history
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows drop primary key`),
			ansiSQLChange(`alter table argo_archived_workflows drop constraint argo_workflow_history_pkey`),
		),
		ansiSQLChange(`alter table argo_archived_workflows add primary key(id)`),
		// ***
		// THE CHANGES ABOVE THIS LINE MAY BE IN PER-PRODUCTION SYSTEMS - DO NOT CHANGE THEM
		// ***
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows change column id uid varchar(128)`),
			ansiSQLChange(`alter table argo_archived_workflows rename column id to uid`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column uid varchar(128) not null`),
			ansiSQLChange(`alter table argo_archived_workflows alter column uid set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column phase varchar(25) not null`),
			ansiSQLChange(`alter table argo_archived_workflows alter column phase set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column namespace varchar(256) not null`),
			ansiSQLChange(`alter table argo_archived_workflows alter column namespace set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column workflow text not null`),
			ansiSQLChange(`alter table argo_archived_workflows alter column workflow set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column startedat timestamp not null default CURRENT_TIMESTAMP`),
			ansiSQLChange(`alter table argo_archived_workflows alter column startedat set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column finishedat timestamp not null default CURRENT_TIMESTAMP`),
			ansiSQLChange(`alter table argo_archived_workflows alter column finishedat set not null`),
		),
		ansiSQLChange(`alter table argo_archived_workflows add clustername varchar(64)`), // DNS entry can only be max 63 bytes
		ansiSQLChange(`update argo_archived_workflows set clustername = '` + m.clusterName + `' where clustername is null`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column clustername varchar(64) not null`),
			ansiSQLChange(`alter table argo_archived_workflows alter column clustername set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows drop primary key`),
			ansiSQLChange(`alter table argo_archived_workflows drop constraint argo_archived_workflows_pkey`),
		),
		ansiSQLChange(`alter table argo_archived_workflows add primary key(clustername,uid)`),
		ansiSQLChange(`create index argo_archived_workflows_i1 on argo_archived_workflows (clustername,namespace)`),
		// argo_archived_workflows now looks like:
		// clustername(not null) | uid(not null) | | name (null) | phase(not null) | namespace(not null) | workflow(not null) | startedat(not null)  | finishedat(not null)
		// remove unused columns
		ansiSQLChange(`alter table ` + m.tableName + ` drop column phase`),
		ansiSQLChange(`alter table ` + m.tableName + ` drop column startedat`),
		ansiSQLChange(`alter table ` + m.tableName + ` drop column finishedat`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.tableName+` change column id uid varchar(128)`),
			ansiSQLChange(`alter table `+m.tableName+` rename column id to uid`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.tableName+` modify column uid varchar(128) not null`),
			ansiSQLChange(`alter table `+m.tableName+` alter column uid set not null`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.tableName+` modify column namespace varchar(256) not null`),
			ansiSQLChange(`alter table `+m.tableName+` alter column namespace set not null`),
		),
		ansiSQLChange(`alter table ` + m.tableName + ` add column clustername varchar(64)`), // DNS cannot be longer than 64 bytes
		ansiSQLChange(`update ` + m.tableName + ` set clustername = '` + m.clusterName + `' where clustername is null`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.tableName+` modify column clustername varchar(64) not null`),
			ansiSQLChange(`alter table `+m.tableName+` alter column clustername set not null`),
		),
		ansiSQLChange(`alter table ` + m.tableName + ` add column version varchar(64)`),
		ansiSQLChange(`alter table ` + m.tableName + ` add column nodes text`),
		backfillNodes{tableName: m.tableName},
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.tableName+` modify column nodes text not null`),
			ansiSQLChange(`alter table `+m.tableName+` alter column nodes set not null`),
		),
		ansiSQLChange(`alter table ` + m.tableName + ` drop column workflow`),
		// add a timestamp column to indicate updated time
		ansiSQLChange(`alter table ` + m.tableName + ` add column updatedat timestamp not null default current_timestamp`),
		// remove the old primary key and add a new one
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.tableName+` drop primary key`),
			ansiSQLChange(`alter table `+m.tableName+` drop constraint `+m.tableName+`_pkey`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index idx_name on `+m.tableName),
			ansiSQLChange(`drop index idx_name`),
		),
		ansiSQLChange(`alter table ` + m.tableName + ` drop column name`),
		ansiSQLChange(`alter table ` + m.tableName + ` add primary key(clustername,uid,version)`),
		ansiSQLChange(`create index ` + m.tableName + `_i1 on ` + m.tableName + ` (clustername,namespace)`),
		// argo_workflows now looks like:
		//  clustername(not null) | uid(not null) | namespace(not null) | version(not null) | nodes(not null) | updatedat(not null)
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column workflow json not null`),
			ansiSQLChange(`alter table argo_archived_workflows alter column workflow type json using workflow::json`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column name varchar(256) not null`),
			ansiSQLChange(`alter table argo_archived_workflows alter column name set not null`),
		),
		// clustername(not null) | uid(not null) | | name (not null) | phase(not null) | namespace(not null) | workflow(not null) | startedat(not null)  | finishedat(not null)
		ansiSQLChange(`create index ` + m.tableName + `_i2 on ` + m.tableName + ` (clustername,namespace,updatedat)`),
		// The argo_archived_workflows_labels is really provided as a way to create queries on labels that are fast because they
		// use indexes. When displaying, it might be better to look at the `workflow` column.
		// We could have added a `labels` column to argo_archived_workflows, but then we would have had to do free-text
		// queries on it which would be slow due to having to table scan.
		// The key has an optional prefix(253 chars) + '/' + name(63 chars)
		// Why is the key called "name" not "key"? Key is an SQL reserved word.
		ansiSQLChange(`create table if not exists argo_archived_workflows_labels (
	clustername varchar(64) not null,
	uid varchar(128) not null,
    name varchar(317) not null,
    value varchar(63) not null,
    primary key (clustername, uid, name),
 	foreign key (clustername, uid) references argo_archived_workflows(clustername, uid) on delete cascade
)`),
		// MySQL can only store 64k in a TEXT field, both MySQL and Posgres can store 1GB in JSON.
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table `+m.tableName+` modify column nodes json not null`),
			ansiSQLChange(`alter table `+m.tableName+` alter column nodes type json using nodes::json`),
		),
		// add instanceid column to table argo_archived_workflows
		ansiSQLChange(`alter table argo_archived_workflows add column instanceid varchar(64)`),
		ansiSQLChange(`update argo_archived_workflows set instanceid = '' where instanceid is null`),
		ternary(dbType == MySQL,
			ansiSQLChange(`alter table argo_archived_workflows modify column instanceid varchar(64) not null`),
			ansiSQLChange(`alter table argo_archived_workflows alter column instanceid set not null`),
		),
		// drop argo_archived_workflows index
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index argo_archived_workflows_i1 on argo_archived_workflows`),
			ansiSQLChange(`drop index argo_archived_workflows_i1`),
		),
		// add argo_archived_workflows index
		ansiSQLChange(`create index argo_archived_workflows_i1 on argo_archived_workflows (clustername,instanceid,namespace)`),
		// drop m.tableName indexes
		// xxx_i1 is not needed because xxx_i2 already covers it, drop both and recreat an index named xxx_i1
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index `+m.tableName+`_i1 on `+m.tableName),
			ansiSQLChange(`drop index `+m.tableName+`_i1`),
		),
		ternary(dbType == MySQL,
			ansiSQLChange(`drop index `+m.tableName+`_i2 on `+m.tableName),
			ansiSQLChange(`drop index `+m.tableName+`_i2`),
		),
		// add m.tableName index
		ansiSQLChange(`create index ` + m.tableName + `_i1 on ` + m.tableName + ` (clustername,namespace,updatedat)`),
		// index to find records that need deleting, this omits namespaces as this might be null
		ansiSQLChange(`create index argo_archived_workflows_i2 on argo_archived_workflows (clustername,instanceid,finishedat)`),
		// add argo_archived_workflows name index for prefix searching performance
		ansiSQLChange(`create index argo_archived_workflows_i3 on argo_archived_workflows (clustername,instanceid,name)`),
		// add indexes for list archived workflow performance. #8836
		ansiSQLChange(`create index argo_archived_workflows_i4 on argo_archived_workflows (startedat)`),
		ansiSQLChange(`create index argo_archived_workflows_labels_i1 on argo_archived_workflows_labels (name,value)`),
		// PostgreSQL only: convert argo_archived_workflows.workflow column to JSONB for performance. #13601
		ternary(dbType == MySQL,
			noop{},
			ansiSQLChange(`alter table argo_archived_workflows alter column workflow set data type jsonb using workflow::jsonb`),
		),
	} {
		err := m.applyChange(changeSchemaVersion, change)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m migrate) applyChange(changeSchemaVersion int, c change) error {
	// https://upper.io/blog/2020/08/29/whats-new-on-upper-v4/#transactions-enclosed-by-functions
	err := m.session.Tx(func(tx db.Session) error {
		rs, err := tx.SQL().Exec("update schema_history set schema_version = ? where schema_version = ?", changeSchemaVersion, changeSchemaVersion-1)
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
		return nil
	})
	return err
}
