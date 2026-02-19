package sqldb

import (
	"context"
	"fmt"

	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

type Change interface {
	Apply(ctx context.Context, session db.Session) error
}

type TypedChanges map[DBType]Change

func ByType(dbType DBType, changes TypedChanges) Change {
	if change, ok := changes[dbType]; ok {
		return change
	}
	return nil
}

func Migrate(ctx context.Context, session db.Session, dbType DBType, versionTableName string, changes []Change) error {
	ctx, logger := logging.RequireLoggerFromContext(ctx).WithField("dbType", dbType).InContext(ctx)
	logger.Info(ctx, "Migrating database schema")

	{
		// poor mans SQL migration
		_, err := session.SQL().Exec(fmt.Sprintf("create table if not exists %s(schema_version int not null, primary key(schema_version))", versionTableName))
		if err != nil {
			return err
		}

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
		err := applyChange(ctx, session, changeSchemaVersion, versionTableName, change)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyChange(ctx context.Context, session db.Session, changeSchemaVersion int, versionTableName string, c Change) error {
	// https://upper.io/blog/2020/08/29/whats-new-on-upper-v4/#transactions-enclosed-by-functions
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("change", c).Info(ctx, "apply change")
	err := session.Tx(func(tx db.Session) error {
		rs, err := tx.SQL().Exec(fmt.Sprintf("update %s set schema_version = ? where schema_version = ?", versionTableName), changeSchemaVersion, changeSchemaVersion-1)
		if err != nil {
			logger.WithFields(logging.Fields{"err": err, "change": c}).Error(ctx, "Error applying database change")
			return err
		}
		rowsAffected, err := rs.RowsAffected()
		if err != nil {
			logger.WithError(err).WithField("change", c).Error(ctx, "Rows affected problem")
			return err
		}
		if rowsAffected == 1 {
			logger.WithFields(logging.Fields{"changeSchemaVersion": changeSchemaVersion, "change": c}).Info(ctx, "applying database change")
			if c != nil {
				err := c.Apply(ctx, tx)
				if err != nil {
					return err
				}
			}
		}
		logger.WithFields(logging.Fields{"change": c, "rowsaffected": rowsAffected}).Info(ctx, "done")
		return nil
	})
	return err
}
