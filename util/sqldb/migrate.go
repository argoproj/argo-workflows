package sqldb

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"
)

type Change interface {
	Apply(session db.Session) error
}

type TypedChanges map[DBType]Change

func ByType(dbType DBType, changes TypedChanges) Change {
	if change, ok := changes[dbType]; ok {
		return change
	}
	return nil
}

func Migrate(ctx context.Context, session db.Session, versionTableName string, changes []Change) error {
	dbType := DBTypeFor(session)
	log.WithFields(log.Fields{"dbType": dbType}).Info("Migrating database schema")

	{
		// poor mans SQL migration
		_, err := session.SQL().Exec(fmt.Sprintf("create table if not exists %s(schema_version int not null, primary key(schema_version))", versionTableName))
		if err != nil {
			return err
		}

		// For SQLite, no need to check for primary key existence - it's created in the table definition
		// For other databases, check if we need to add the primary key
		if dbType != SQLite {
			// Ensure the schema_history table has a primary key, creating it if necessary
			// This logic is implemented separately from regular migrations to improve compatibility with databases running in strict or HA modes
			dbIdentifierColumn := "table_schema"
			if dbType == Postgres {
				dbIdentifierColumn = "table_catalog"
			}

			// Check if primary key exists
			rows, err := session.SQL().Query(
				fmt.Sprintf("select 1 from information_schema.table_constraints where constraint_type = 'PRIMARY KEY' and table_name = '%s' and %s = ?",
					versionTableName, dbIdentifierColumn),
				session.Name())
			if err != nil {
				return err
			}
			defer func() {
				tmpErr := rows.Close()
				if err == nil {
					err = tmpErr
				}
			}()
			if !rows.Next() {
				_, err := session.SQL().Exec(fmt.Sprintf("alter table %s add primary key(schema_version)", versionTableName))
				if err != nil {
					return err
				}
			} else if err := rows.Err(); err != nil {
				return err
			}
		}

		rs, err := session.SQL().Query(fmt.Sprintf("select schema_version from %s", versionTableName))
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
			_, err := session.SQL().Exec(fmt.Sprintf("insert into %s values(-1)", versionTableName))
			if err != nil {
				return err
			}
		} else if err := rs.Err(); err != nil {
			return err
		}
	}

	// try and make changes idempotent, as it is possible for the change to apply, but the archive update to fail
	// and therefore try and apply again next try
	for changeSchemaVersion, change := range changes {
		err := applyChange(session, changeSchemaVersion, versionTableName, change)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyChange(session db.Session, changeSchemaVersion int, versionTableName string, c Change) error {
	// https://upper.io/blog/2020/08/29/whats-new-on-upper-v4/#transactions-enclosed-by-functions
	log.Infof("apply change %s", c)
	err := session.Tx(func(tx db.Session) error {
		rs, err := tx.SQL().Exec(fmt.Sprintf("update %s set schema_version = ? where schema_version = ?", versionTableName), changeSchemaVersion, changeSchemaVersion-1)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "change": c}).Error("Error applying database change")
			return err
		}
		rowsAffected, err := rs.RowsAffected()
		if err != nil {
			log.WithFields(log.Fields{"err": err, "change": c}).Error("Rows affected problem")
			return err
		}
		if rowsAffected == 1 {
			log.WithFields(log.Fields{"changeSchemaVersion": changeSchemaVersion, "change": c}).Info("applying database change")
			if c != nil {
				err := c.Apply(tx)
				if err != nil {
					return err
				}
			}
		}
		log.WithFields(log.Fields{"change": c, "rowsaffected": rowsAffected}).Info("done")
		return nil
	})
	return err
}
