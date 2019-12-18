package sqldb

import (
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
func CreatePostGresDBSession(kubectlConfig kubernetes.Interface, namespace string, cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, string, error) {

	if cfg.TableName == "" {
		return nil, "", errors.InternalError("TableName is empty")
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

	_, err = session.Exec(`create table if not exists ` + cfg.TableName + ` (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp,
    finishedat timestamp,
    primary key (id, namespace)
)`)
	if err != nil {
		return nil, "", err
	}

	return session, cfg.TableName, nil

}

// CreatePostGresDBSession creates Mysql DB session
func CreateMySQLDBSession(kubectlConfig kubernetes.Interface, namespace string, cfg *config.MySQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, string, error) {

	if cfg.TableName == "" {
		return nil, "", errors.InternalError("TableName is empty")
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

	_, err = session.Exec(`CREATE TABLE IF NOT EXISTS ` + cfg.TableName + ` (
  id varchar(128) NOT NULL DEFAULT "", 
  name varchar(128) DEFAULT NULL,
  phase varchar(24) DEFAULT NULL,
  namespace varchar(24) NOT NULL DEFAULT "" ,
  workflow longtext,
  startedat datetime DEFAULT NULL,
  finishedat datetime DEFAULT NULL,
  PRIMARY KEY (id(24), namespace(24))
) ENGINE=InnoDB DEFAULT CHARSET=utf8;`)
	if err != nil {
		return nil, "", err
	}
	return session, cfg.TableName, nil

}
