package sqldb

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/mysql"
	"upper.io/db.v3/postgresql"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util"
	"github.com/argoproj/argo/workflow/config"
)

const (
	CodeDBUpdateRowNotFound = "ERR_DB_UPDATE_ROW_NOT_FOUND"
)

func DBUpdateNoRowFoundError(err error) error {
	return errors.Wrap(err, CodeDBUpdateRowNotFound, err.Error())
}

// CreateDBSession creates the dB session
func CreateDBSession(kubectlConfig kubernetes.Interface, namespace string, persistConfig *config.PersistConfig) (sqlbuilder.Database, string, error) {
	if persistConfig == nil {
		return nil, "", errors.InternalError("Persistence config is not found")
	}

	if persistConfig.PostgreSQL != nil {
		return CreatePostGresDBSession(kubectlConfig, namespace, persistConfig.PostgreSQL, persistConfig.ConnectionPool)
	} else if persistConfig.MySQL != nil {
		return CreateMySQLDBSession(kubectlConfig, namespace, persistConfig.MySQL, persistConfig.ConnectionPool)
	}

	return nil, "", fmt.Errorf("no databases are configured")
}

// CreatePostGresDBSession creates postgresDB session
func CreatePostGresDBSession(kubectlConfig kubernetes.Interface, namespace string, cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, string, error) {

	if cfg.TableName == "" {
		return nil, "", errors.InternalError("tableName is empty")
	}

	userNameByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, "", err
	}
	passwordByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, "", err
	}

	var settings = postgresql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     cfg.Host + ":" + cfg.Port,
		Database: cfg.Database,
	}

	if cfg.SSL {
		settings.Options = map[string]string{
			"sslmode": "true",
		}
	}

	session, err := postgresql.Open(settings)
	if err != nil {
		return nil, "", err
	}

	if persistPool != nil {
		session.SetMaxOpenConns(persistPool.MaxOpenConns)
		session.SetMaxIdleConns(persistPool.MaxIdleConns)
	}

	err = migrate(migrateCfg{cfg.TableName}, session)
	if err != nil {
		return nil, "", err
	}

	return session, cfg.TableName, nil

}

// CreateMySQLDBSession creates Mysql DB session
func CreateMySQLDBSession(kubectlConfig kubernetes.Interface, namespace string, cfg *config.MySQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, string, error) {

	if cfg.TableName == "" {
		return nil, "", errors.InternalError("tableName is empty")
	}

	userNameByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, "", err
	}
	passwordByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, "", err
	}

	session, err := mysql.Open(mysql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     cfg.Host + ":" + cfg.Port,
		Database: cfg.Database,
	})
	if err != nil {
		return nil, "", err
	}

	if persistPool != nil {
		session.SetMaxOpenConns(persistPool.MaxOpenConns)
		session.SetMaxIdleConns(persistPool.MaxIdleConns)
	}

	err = migrate(migrateCfg{cfg.TableName}, session)
	if err != nil {
		return nil, "", err
	}

	return session, cfg.TableName, nil

}
