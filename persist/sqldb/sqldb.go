package sqldb

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/mysql"
	"upper.io/db.v3/postgresql"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util"
	"github.com/argoproj/argo/workflow/config"
)

// CreateDBSession creates the dB session
func CreateDBSession(kubectlConfig kubernetes.Interface, namespace string, persistConfig *config.PersistConfig) (sqlbuilder.Database, dbModel, error) {
	if persistConfig == nil {
		return nil, dbModel{}, errors.InternalError("Persistence config is not found")
	}

	log.Info("Creating DB session")

	if persistConfig.PostgreSQL != nil {
		return CreatePostGresDBSession(kubectlConfig, namespace, persistConfig.PostgreSQL, persistConfig.ConnectionPool)
	} else if persistConfig.MySQL != nil {
		return CreateMySQLDBSession(kubectlConfig, namespace, persistConfig.MySQL, persistConfig.ConnectionPool)
	}
	return nil, dbModel{}, fmt.Errorf("no databases are configured")
}

// CreatePostGresDBSession creates postgresDB session
func CreatePostGresDBSession(kubectlConfig kubernetes.Interface, namespace string, cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, dbModel, error) {

	if cfg.TableName == "" {
		return nil, dbModel{}, errors.InternalError("tableName is empty")
	}

	userNameByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, dbModel{}, err
	}
	passwordByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, dbModel{}, err
	}

	var settings = postgresql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     cfg.Host + ":" + cfg.Port,
		Database: cfg.Database,
	}

	if cfg.SSL {
		if cfg.SSLMode != "" {
			options := map[string]string{
				"sslmode": cfg.SSLMode,
			}
			settings.Options = options
		}
	}

	session, err := postgresql.Open(settings)
	if err != nil {
		return nil, dbModel{}, err
	}

	if persistPool != nil {
		session.SetMaxOpenConns(persistPool.MaxOpenConns)
		session.SetMaxIdleConns(persistPool.MaxIdleConns)
	}

	dbm := NewDBModel(cfg.Schema, cfg.TableName)
	return session, dbm, nil
}

// CreateMySQLDBSession creates Mysql DB session
func CreateMySQLDBSession(kubectlConfig kubernetes.Interface, namespace string, cfg *config.MySQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, dbModel, error) {

	if cfg.TableName == "" {
		return nil, dbModel{}, errors.InternalError("tableName is empty")
	}

	userNameByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, dbModel{}, err
	}
	passwordByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, dbModel{}, err
	}

	session, err := mysql.Open(mysql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     cfg.Host + ":" + cfg.Port,
		Database: cfg.Database,
	})
	if err != nil {
		return nil, dbModel{}, err
	}

	if persistPool != nil {
		session.SetMaxOpenConns(persistPool.MaxOpenConns)
		session.SetMaxIdleConns(persistPool.MaxIdleConns)
	}
	// this is needed to make MySQL run in a Golang-compatible UTF-8 character set.
	_, err = session.Exec("SET NAMES 'utf8mb4'")
	if err != nil {
		return nil, dbModel{}, err
	}
	_, err = session.Exec("SET CHARACTER SET utf8mb4")
	if err != nil {
		return nil, dbModel{}, err
	}
	dbm := NewDBModel("", cfg.TableName)
	return session, dbm, nil
}
