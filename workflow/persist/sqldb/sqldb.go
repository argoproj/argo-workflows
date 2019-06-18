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
func CreateDBSession(kubectlConfig kubernetes.Interface, namespace string, persistConfig *config.PersistConfig) (sqlbuilder.Database, error) {
	if persistConfig == nil {
		return nil, errors.InternalError("Persistence config is not found")
	}
	if persistConfig.PostgreSQL != nil {
		return CreatePostGresDBSession(kubectlConfig, namespace, persistConfig.PostgreSQL, persistConfig.PersistConnectPool)
	} else if persistConfig.MySQL != nil {
		return CreateMySQLDBSession(kubectlConfig, namespace, persistConfig.MySQL, persistConfig.PersistConnectPool)
	}

	return nil, nil
}

// CreatePostGresDBSession creates postgresDB session
func CreatePostGresDBSession(kubectlConfig kubernetes.Interface, namespace string, postgresConfig *config.PostgreSQLConfig, persistPool *config.PersistConnectPool) (sqlbuilder.Database, error) {

	userNameByte, err := util.GetSecrets(kubectlConfig, namespace, postgresConfig.UsernameSecret.Name, postgresConfig.UsernameSecret.Key)
	if err != nil {
		return nil, err
	}
	passwordByte, err := util.GetSecrets(kubectlConfig, namespace, postgresConfig.PasswordSecret.Name, postgresConfig.PasswordSecret.Key)
	if err != nil {
		return nil, err
	}
	var settings = postgresql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     postgresConfig.Host + ":" + postgresConfig.Port,
		Database: postgresConfig.Database,
	}
	session, err := postgresql.Open(settings)

	return session, err

}

// CreatePostGresDBSession creates Mysql DB session
func CreateMySQLDBSession(kubectlConfig kubernetes.Interface, namespace string, postgresConfig *config.MySQLConfig, persistPool *config.PersistConnectPool) (sqlbuilder.Database, error) {

	userNameByte, err := util.GetSecrets(kubectlConfig, namespace, postgresConfig.UsernameSecret.Name, postgresConfig.UsernameSecret.Key)
	if err != nil {
		return nil, err
	}
	passwordByte, err := util.GetSecrets(kubectlConfig, namespace, postgresConfig.PasswordSecret.Name, postgresConfig.PasswordSecret.Key)
	if err != nil {
		return nil, err
	}
	var settings = mysql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     postgresConfig.Host + ":" + postgresConfig.Port,
		Database: postgresConfig.Database,
	}
	session, err := mysql.Open(settings)

	return session, err

}
