package sqldb

import (
	log "github.com/sirupsen/logrus"
	"upper.io/db.v3/lib/sqlbuilder"
)

type migrateCfg struct {
	tableName string
}

func migrate(cfg migrateCfg, session sqlbuilder.Database) error {
	// poor mans SQL migration
	_, err := session.Exec("create table if not exists schema_history(schema_version int not null)")
	if err != nil {
		return err
	}
	rs, err := session.Query("select schema_version from schema_history")
	if err != nil {
		return err
	}
	schemaVersion := -1
	if rs.Next() {
		err := rs.Scan(&schemaVersion)
		if err != nil {
			return err
		}
	} else {
		_, err := session.Exec("insert into schema_history values(-1)")
		if err != nil {
			return err
		}
	}
	err = rs.Close()
	if err != nil {
		return err
	}
	log.WithField("schemaVersion", schemaVersion).Info("Migrating database schema")

	// try and make changes idempotent, as it is possible for the change to apply, but the archive update to fail
	// and therefore try and apply again next try
	for changeSchemaVersion, change := range []string{
		`create table if not exists ` + cfg.tableName + ` (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp,
    finishedat timestamp,
    primary key (id, namespace)
)`,
		`create unique index idx_name on ` + cfg.tableName + ` (name)`,
		`create table if not exists argo_workflow_history (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp,
    finishedat timestamp,
    primary key (id, namespace)
)`,
		`alter table argo_workflow_history rename to argo_archived_workflows`,
		`drop index idx_name`,
		`create unique index idx_name on ` + cfg.tableName + `(name, namespace)`,
		`alter table ` + cfg.tableName + ` drop constraint ` + cfg.tableName + `_pkey`,
		`alter table ` + cfg.tableName + ` add primary key(name,namespace)`,
		`drop index idx_name`,
		// huh - why does the pkey not have the same name as the table - history!
		`alter table argo_archived_workflows drop constraint argo_workflow_history_pkey`,
		`alter table argo_archived_workflows add primary key(id)`,
	} {
		if changeSchemaVersion > schemaVersion {
			log.WithFields(log.Fields{"changeSchemaVersion": changeSchemaVersion, "change": change}).Info("Applying database change")
			_, err := session.Exec(change)
			if err != nil {
				return err
			}
			_, err = session.Exec("update schema_history set schema_version=?", changeSchemaVersion)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
