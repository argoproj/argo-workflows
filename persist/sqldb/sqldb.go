package sqldb

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/mysql"
	"upper.io/db.v3/postgresql"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util"
)

// CreateDBSession creates the dB session
func CreateDBSession(kubectlConfig kubernetes.Interface, namespace string, persistConfig *config.PersistConfig) (sqlbuilder.Database, string, error) {
	if persistConfig == nil {
		return nil, "", errors.InternalError("Persistence config is not found")
	}
	cfg := persistConfig.GetDatabaseConfig()
	if cfg == nil {
		return nil, "", fmt.Errorf("no databases are configured")
	}
	// get the connectionURL
	var connectionURL db.ConnectionURL
	{
		userNameByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
		if err != nil {
			return nil, "", err
		}
		passwordByte, err := util.GetSecrets(kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
		if err != nil {
			return nil, "", err
		}
		username := string(userNameByte)
		password := string(passwordByte)
		host := cfg.Host + ":" + cfg.Port
		database := cfg.Database
		options := cfg.GetOptions()
		if persistConfig.PostgreSQL != nil {
			connectionURL = postgresql.ConnectionURL{User: username, Password: password, Host: host, Database: database, Options: options}
		} else {
			connectionURL = mysql.ConnectionURL{User: username, Password: password, Host: host, Database: database, Options: options}
		}
		log.WithField("connectionURL", redactConnectionURL(connectionURL, password)).Info("Creating DB session")
	}
	var session sqlbuilder.Database
	var err error
	if persistConfig.PostgreSQL != nil {
		session, err = postgresql.Open(connectionURL)
		if err != nil {
			return nil, "", err
		}
	} else {
		session, err = mysql.Open(connectionURL)
		if err != nil {
			return nil, "", err
		}
	}
	// apply the connection pool values
	persistPool := persistConfig.ConnectionPool
	if persistPool != nil {
		log.WithFields(log.Fields{"maxOpenConns": persistPool.MaxOpenConns, "maxIdleConns": persistPool.MaxIdleConns, "connMaxLifetime": persistPool.ConnMaxLifetime}).Info("Setting connection pool fields on the database session")
		session.SetMaxOpenConns(persistPool.MaxOpenConns)
		session.SetMaxIdleConns(persistPool.MaxIdleConns)
		session.SetConnMaxLifetime(time.Duration(persistPool.ConnMaxLifetime))
	}
	if persistConfig.MySQL != nil {
		log.Info("Making MySQL run in a Golang-compatible UTF-8 character set")
		_, err = session.Exec("SET NAMES 'utf8mb4'")
		if err != nil {
			return nil, "", err
		}
		_, err = session.Exec("SET CHARACTER SET utf8mb4")
		if err != nil {
			return nil, "", err
		}
	}
	return session, cfg.GetTableName(), nil
}

func redactConnectionURL(connectionURL db.ConnectionURL, password string) string {
	return strings.Replace(strings.Replace(connectionURL.String(), "="+password, "=******", -1), ":"+password, ":******", -1)
}
