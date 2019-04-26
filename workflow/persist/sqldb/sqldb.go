package sqldb

import (
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util"
	"github.com/argoproj/argo/workflow/config"
	"k8s.io/client-go/kubernetes"
	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/mysql"
	"upper.io/db.v3/postgresql"
)

// CreateDBSession creates the dB session
func CreateDBSession(kubectlConfig kubernetes.Interface, namespace string, persistConfig *config.PersistConfig) (sqlbuilder.Database, error) {
	if persistConfig == nil {
		return nil, errors.InternalError("Persistence config is not found")
	}
	if persistConfig.Postgresql != nil {
		return CreatePostGresDBSession(kubectlConfig, namespace, persistConfig.Postgresql, persistConfig.PersistConnectPool)
	} else if persistConfig.Mysql != nil {
		return CreateMySQLDBSession(kubectlConfig, namespace, persistConfig.Mysql, persistConfig.PersistConnectPool)
	}

	return nil, nil
}

// CreatePostGresDBSession creates postgresDB session
func CreatePostGresDBSession(kubectlConfig kubernetes.Interface, namespace string, postgresConfig *config.PostgresqlConfig, persistPool *config.PersistConnectPool) (sqlbuilder.Database, error) {

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
func CreateMySQLDBSession(kubectlConfig kubernetes.Interface, namespace string, postgresConfig *config.MysqlConfig, persistPool *config.PersistConnectPool) (sqlbuilder.Database, error) {

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
