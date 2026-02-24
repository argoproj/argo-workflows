package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/upper/db/v4"
	mysqladp "github.com/upper/db/v4/adapter/mysql"
	postgresqladp "github.com/upper/db/v4/adapter/postgresql"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util"
)

// CreateDBSession creates the DB session and returns the session along with its database type
func CreateDBSession(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, dbConfig config.DBConfig) (db.Session, DBType, error) {
	if dbConfig.PostgreSQL != nil {
		session, err := createPostGresDBSession(ctx, kubectlConfig, namespace, dbConfig.PostgreSQL, dbConfig.ConnectionPool)
		if err != nil {
			return nil, Invalid, err
		}
		return session, Postgres, nil
	} else if dbConfig.MySQL != nil {
		session, err := createMySQLDBSession(ctx, kubectlConfig, namespace, dbConfig.MySQL, dbConfig.ConnectionPool)
		if err != nil {
			return nil, Invalid, err
		}
		return session, MySQL, err
	}
	return nil, "", fmt.Errorf("no databases are configured")
}

// CreateDBSessionWithCreds creates a database session using direct username and password
func CreateDBSessionWithCreds(dbConfig config.DBConfig, username, password string) (db.Session, DBType, error) {
	if dbConfig.PostgreSQL != nil {
		session, err := createPostGresDBSessionWithCreds(dbConfig.PostgreSQL, dbConfig.ConnectionPool, username, password)
		if err != nil {
			return nil, Invalid, err
		}
		return session, Postgres, err
	} else if dbConfig.MySQL != nil {
		session, err := createMySQLDBSessionWithCreds(dbConfig.MySQL, dbConfig.ConnectionPool, username, password)
		if err != nil {
			return nil, Invalid, err
		}
		return session, MySQL, err
	}
	return nil, "", fmt.Errorf("no databases are configured")
}

// createPostGresDBSession creates postgresDB session
func createPostGresDBSession(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool) (db.Session, error) {
	userNameByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, err
	}

	if cfg.AzureToken != nil && cfg.AzureToken.Enabled {
		return createPostGresDBSessionWithAzure(cfg, persistPool, string(userNameByte))
	}

	passwordByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, err
	}

	return createPostGresDBSessionWithCreds(cfg, persistPool, string(userNameByte), string(passwordByte))
}

// createMySQLDBSession creates Mysql DB session
func createMySQLDBSession(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, cfg *config.MySQLConfig, persistPool *config.ConnectionPool) (db.Session, error) {
	userNameByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, err
	}
	passwordByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, err
	}

	return createMySQLDBSessionWithCreds(cfg, persistPool, string(userNameByte), string(passwordByte))
}

// createPostGresDBSessionWithAzure creates postgresDB session with azure token
func createPostGresDBSessionWithAzure(cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool, username string) (db.Session, error) {
	settings := postgresqladp.ConnectionURL{
		User:     username,
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

	scope := cfg.AzureToken.Scope
	if scope == "" {
		scope = "https://ossrdbms-aad.database.windows.net/.default"
	}

	connector := &azureConnector{
		dsn:   settings.String(),
		scope: scope,
	}

	sqlDB := sql.OpenDB(connector)

	session, err := postgresqladp.New(sqlDB)
	if err != nil {
		return nil, err
	}
	session = ConfigureDBSession(session, persistPool)
	return session, nil
}

// createPostGresDBSessionWithCreds creates postgresDB session with direct credentials
func createPostGresDBSessionWithCreds(cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool, username, password string) (db.Session, error) {
	settings := postgresqladp.ConnectionURL{
		User:     username,
		Password: password,
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

// createMySQLDBSessionWithCreds creates MySQL DB session with direct credentials
func createMySQLDBSessionWithCreds(cfg *config.MySQLConfig, persistPool *config.ConnectionPool, username, password string) (db.Session, error) {
	session, err := mysqladp.Open(mysqladp.ConnectionURL{
		User:     username,
		Password: password,
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
