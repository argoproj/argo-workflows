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
	CodeInvalidDBSession    = "ERR_INVALID_DB_SESSION"
	CodeDBUpdateRowNotFound = "ERR_DB_UPDATE_ROW_NOT_FOUND"
	CodeDBOperationError    = "ERR_DB_OPERATION_ERROR"
)

func DBInvalidSession(err error, message ...string) error {
	if len(message) == 0 {
		return errors.Wrap(err, CodeInvalidDBSession, err.Error())
	}
	return errors.Wrap(err, CodeInvalidDBSession, message[0])

}

func DBOperationError(err error, message ...string) error {
	if len(message) == 0 {
		return errors.Wrap(err, CodeDBOperationError, err.Error())
	}
	return errors.Wrap(err, CodeInvalidDBSession, message[0])

}

func DBUpdateNoRowFoundError(err error, message ...string) error {
	if len(message) == 0 {
		return errors.Wrap(err, CodeDBUpdateRowNotFound, err.Error())
	}
	return errors.Wrap(err, CodeDBUpdateRowNotFound, message[0])
}

// InternalWrapErrorf annotates the error with the ERR_INTERNAL code and a stack trace, optional message
func DBUpdateNoRowFoundErrorf(err error, format string, args ...interface{}) error {
	return errors.Wrap(err, CodeDBUpdateRowNotFound, fmt.Sprintf(format, args...))
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

	return nil, "", nil
}

// CreatePostGresDBSession creates postgresDB session
func CreatePostGresDBSession(kubectlConfig kubernetes.Interface, namespace string, postgresConfig *config.PostgreSQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, string, error) {

	if postgresConfig.TableName == "" {
		return nil, "", errors.InternalError("TableName is empty")
	}

	userNameByte, err := util.GetSecrets(kubectlConfig, namespace, postgresConfig.UsernameSecret.Name, postgresConfig.UsernameSecret.Key)

	if err != nil {
		return nil, postgresConfig.TableName, err
	}
	passwordByte, err := util.GetSecrets(kubectlConfig, namespace, postgresConfig.PasswordSecret.Name, postgresConfig.PasswordSecret.Key)
	if err != nil {
		return nil, postgresConfig.TableName, err
	}

	var settings = postgresql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     postgresConfig.Host + ":" + postgresConfig.Port,
		Database: postgresConfig.Database,
	}
	session, err := postgresql.Open(settings)

	if err != nil {
		return nil, postgresConfig.TableName, err
	}

	if persistPool != nil {
		session.SetMaxOpenConns(persistPool.MaxOpenConns)
		session.SetMaxIdleConns(persistPool.MaxIdleConns)
	}

	return session, postgresConfig.TableName, err

}

// CreatePostGresDBSession creates Mysql DB session
func CreateMySQLDBSession(kubectlConfig kubernetes.Interface, namespace string, mysqlConfig *config.MySQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, string, error) {

	if mysqlConfig.TableName == "" {
		return nil, "", errors.InternalError("TableName is empty")
	}

	userNameByte, err := util.GetSecrets(kubectlConfig, namespace, mysqlConfig.UsernameSecret.Name, mysqlConfig.UsernameSecret.Key)
	if err != nil {
		return nil, mysqlConfig.TableName, err
	}
	passwordByte, err := util.GetSecrets(kubectlConfig, namespace, mysqlConfig.PasswordSecret.Name, mysqlConfig.PasswordSecret.Key)
	if err != nil {
		return nil, mysqlConfig.TableName, err
	}
	var settings = mysql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     mysqlConfig.Host + ":" + mysqlConfig.Port,
		Database: mysqlConfig.Database,
	}

	session, err := mysql.Open(settings)

	if err != nil {
		return nil, mysqlConfig.TableName, err
	}

	if persistPool != nil {
		session.SetMaxOpenConns(persistPool.MaxOpenConns)
		session.SetMaxIdleConns(persistPool.MaxIdleConns)
	}

	return session, mysqlConfig.TableName, err

}
