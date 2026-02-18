package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/go-sql-driver/mysql"
	"github.com/upper/db/v4"
	mysqladp "github.com/upper/db/v4/adapter/mysql"
	postgresqladp "github.com/upper/db/v4/adapter/postgresql"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util"

	// Database drivers - imported for side effects
	_ "github.com/lib/pq"
)

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

	sqlDB := otelsql.OpenDB(connector, otelSQLOptions(semconv.DBSystemNamePostgreSQL, cfg.Database)...)
	return newPostgresSession(sqlDB, persistPool)
}

// createPostGresDBSessionWithCreds creates postgresDB session with direct credentials
func createPostGresDBSessionWithCreds(cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool, username, password string) (db.Session, error) {
	// Build PostgreSQL DSN using url.URL for safe percent-encoding of credentials
	connURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(username, password),
		Host:   cfg.GetHostname(),
		Path:   cfg.Database,
	}
	query := url.Values{}
	switch {
	case !cfg.SSL:
		query.Set("sslmode", "disable")
	case cfg.SSLMode != "":
		query.Set("sslmode", cfg.SSLMode)
	default:
		// Preserve the default behavior of the upper/db postgresql adapter,
		// which used sslmode=prefer. lib/pq defaults to sslmode=require.
		query.Set("sslmode", "prefer")
	}
	connURL.RawQuery = query.Encode()
	dsn := connURL.String()

	// Create traced *sql.DB using otelsql
	sqlDB, err := otelsql.Open("postgres", dsn, otelSQLOptions(semconv.DBSystemNamePostgreSQL, cfg.Database)...)
	if err != nil {
		return nil, fmt.Errorf("failed to open traced postgres connection: %w", err)
	}
	return newPostgresSession(sqlDB, persistPool)
}

// createMySQLDBSessionWithCreds creates MySQL DB session with direct credentials
func createMySQLDBSessionWithCreds(cfg *config.MySQLConfig, persistPool *config.ConnectionPool, username, password string) (db.Session, error) {
	// Build MySQL DSN using mysql.Config to safely handle special characters in credentials
	mysqlCfg := mysql.Config{
		User:      username,
		Passwd:    password,
		Net:       "tcp",
		Addr:      cfg.GetHostname(),
		DBName:    cfg.Database,
		ParseTime: true,
		Params:    cfg.Options,
	}
	dsn := mysqlCfg.FormatDSN()

	// Create traced *sql.DB using otelsql
	sqlDB, err := otelsql.Open("mysql", dsn, otelSQLOptions(semconv.DBSystemNameMySQL, cfg.Database)...)
	if err != nil {
		return nil, fmt.Errorf("failed to open traced mysql connection: %w", err)
	}

	// Wrap with upper/db
	session, err := mysqladp.New(sqlDB)
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to create upper/db session: %w", err)
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

// otelSQLOptions returns the common otelsql tracing options for a database connection.
func otelSQLOptions(systemName attribute.KeyValue, database string) []otelsql.Option {
	return []otelsql.Option{
		otelsql.WithAttributes(systemName, semconv.DBNamespace(database)),
	}
}

// newPostgresSession wraps a *sql.DB with the upper/db PostgreSQL adapter and configures pooling.
func newPostgresSession(sqlDB *sql.DB, persistPool *config.ConnectionPool) (db.Session, error) {
	session, err := postgresqladp.New(sqlDB)
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to create upper/db session: %w", err)
	}
	session = ConfigureDBSession(session, persistPool)
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
