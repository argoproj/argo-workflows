package sqldb

import (
	"context"
	"fmt"
	"time"

	"github.com/upper/db/v4"
	mysqladp "github.com/upper/db/v4/adapter/mysql"
	postgresqladp "github.com/upper/db/v4/adapter/postgresql"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/util"
)

// CreateDBSession creates the dB session
func CreateDBSession(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, dbConfig config.DBConfig) (db.Session, error) {
	if dbConfig.PostgreSQL != nil {
		return createPostGresDBSession(ctx, kubectlConfig, namespace, dbConfig.PostgreSQL, dbConfig.ConnectionPool)
	} else if dbConfig.MySQL != nil {
		return createMySQLDBSession(ctx, kubectlConfig, namespace, dbConfig.MySQL, dbConfig.ConnectionPool)
	}
	return nil, fmt.Errorf("no databases are configured")
}

// createPostGresDBSession creates postgresDB session
func createPostGresDBSession(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool) (db.Session, error) {
	userNameByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, err
	}
	passwordByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, err
	}

	settings := postgresqladp.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     cfg.GetHostname(),
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

	session, err := postgresqladp.Open(settings)
	if err != nil {
		return nil, err
	}
	session = ConfigureDBSession(session, persistPool)
	return session, nil
}

// createMySQLDBSession creates Mysql DB session
func createMySQLDBSession(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, cfg *config.MySQLConfig, persistPool *config.ConnectionPool) (db.Session, error) {
	if cfg.TableName == "" {
		return nil, errors.InternalError("tableName is empty")
	}

	userNameByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, err
	}
	passwordByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, err
	}

	session, err := mysqladp.Open(mysqladp.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     cfg.GetHostname(),
		Database: cfg.Database,
		Options:  cfg.Options,
	})
	if err != nil {
		return nil, err
	}
	session = ConfigureDBSession(session, persistPool)
	// this is needed to make MySQL run in a Golang-compatible UTF-8 character set.
	_, err = session.SQL().Exec("SET NAMES 'utf8mb4'")
	if err != nil {
		return nil, err
	}
	_, err = session.SQL().Exec("SET CHARACTER SET utf8mb4")
	if err != nil {
		return nil, err
	}
	return session, nil
}

// ConfigureDBSession configures the DB session
func ConfigureDBSession(session db.Session, dbPool *config.ConnectionPool) db.Session {
	if dbPool != nil {
		session.SetMaxOpenConns(dbPool.MaxOpenConns)
		session.SetMaxIdleConns(dbPool.MaxIdleConns)
		session.SetConnMaxLifetime(time.Duration(dbPool.ConnMaxLifetime))
	}
	return session
}
