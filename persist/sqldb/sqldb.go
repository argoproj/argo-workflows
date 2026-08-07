package sqldb

import (
	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/errors"
)

func GetTableName(persistConfig *config.PersistConfig) (string, error) {
	var tableName string
	if persistConfig.PostgreSQL != nil {
		tableName = persistConfig.PostgreSQL.TableName
	} else if persistConfig.MySQL != nil {
		tableName = persistConfig.MySQL.TableName
	}
	if tableName == "" {
		return "", errors.InternalError("TableName is empty")
	}
	return tableName, nil
}

func GetSchema(persistConfig *config.PersistConfig) string {
	var schemaName string
	if persistConfig.PostgreSQL != nil {
		schemaName = persistConfig.PostgreSQL.Schema
	}
	return schemaName
}
