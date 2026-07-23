package sqldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"strconv"
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
		session, err := createPostGresDBSession(ctx, kubectlConfig, namespace, dbConfig.PostgreSQL, dbConfig.ConnectionPool, dbConfig.ConnectionTimeout())
		if err != nil {
			return nil, Invalid, err
		}
		return session, Postgres, nil
	} else if dbConfig.MySQL != nil {
		session, err := createMySQLDBSession(ctx, kubectlConfig, namespace, dbConfig.MySQL, dbConfig.ConnectionPool, dbConfig.ConnectionTimeout())
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
		session, err := createPostGresDBSessionWithCreds(dbConfig.PostgreSQL, dbConfig.ConnectionPool, username, password, dbConfig.ConnectionTimeout())
		if err != nil {
			return nil, Invalid, err
		}
		return session, Postgres, err
	} else if dbConfig.MySQL != nil {
		session, err := createMySQLDBSessionWithCreds(dbConfig.MySQL, dbConfig.ConnectionPool, username, password, dbConfig.ConnectionTimeout())
		if err != nil {
			return nil, Invalid, err
		}
		return session, MySQL, err
	}
	return nil, "", fmt.Errorf("no databases are configured")
}

// createPostGresDBSession creates postgresDB session
func createPostGresDBSession(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool, connectTimeout time.Duration) (db.Session, error) {
	azureEnabled := cfg.AzureToken != nil && cfg.AzureToken.Enabled
	awsEnabled := cfg.AWSRDSToken != nil && cfg.AWSRDSToken.Enabled

	if azureEnabled && awsEnabled {
		return nil, fmt.Errorf("only one of azureToken or awsRDSToken may be enabled, not both")
	}

	if awsEnabled && !cfg.SSL {
		return nil, fmt.Errorf("SSL must be enabled (ssl: true) when using AWS RDS IAM authentication")
	}

	userNameByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, err
	}

	if azureEnabled {
		return createPostGresDBSessionWithAzure(cfg, persistPool, string(userNameByte), connectTimeout)
	}

	if awsEnabled {
		return createPostGresDBSessionWithAWSRDS(cfg, persistPool, string(userNameByte), connectTimeout)
	}

	passwordByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, err
	}

	return createPostGresDBSessionWithCreds(cfg, persistPool, string(userNameByte), string(passwordByte), connectTimeout)
}

// createMySQLDBSession creates Mysql DB session
func createMySQLDBSession(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, cfg *config.MySQLConfig, persistPool *config.ConnectionPool, connectTimeout time.Duration) (db.Session, error) {
	userNameByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, err
	}
	passwordByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, err
	}

	return createMySQLDBSessionWithCreds(cfg, persistPool, string(userNameByte), string(passwordByte), connectTimeout)
}

// buildPostgresDSN constructs a PostgreSQL DSN from config and username, with SSL options
// and a connection-establishment timeout configured.
func buildPostgresDSN(cfg *config.PostgreSQLConfig, username string, connectTimeout time.Duration) string {
	settings := postgresqladp.ConnectionURL{
		User:     username,
		Host:     cfg.GetHostname(),
		Database: cfg.Database,
	}

	// connect_timeout limits connection setup (dial + handshake) to ensure fast failure if the DB is unreachable.
	// lib/pq resets this deadline afterward, leaving subsequent queries unaffected.
	settings.Options = map[string]string{
		"connect_timeout": strconv.Itoa(int(connectTimeout.Seconds())),
	}
	if cfg.SSL && cfg.SSLMode != "" {
		settings.Options["sslmode"] = cfg.SSLMode
	}

	if cfg.Schema != "" {
		if settings.Options == nil {
			settings.Options = map[string]string{}
		}
		settings.Options["search_path"] = cfg.Schema
	}

	return settings.String()
}

// createPostGresDBSessionWithConnector creates a PostgreSQL session using the provided connector.
func createPostGresDBSessionWithConnector(cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool, connector driver.Connector) (db.Session, error) {
	sqlDB := otelsql.OpenDB(connector, otelSQLOptions(semconv.DBSystemNamePostgreSQL, cfg.Database)...)
	return newPostgresSession(sqlDB, persistPool)
}

// timeoutConnector wraps a driver.Connector so that Connect bounds only connection
// establishment (dial + handshake read) to timeout, leaving queries on the resulting
// connection unaffected. This gives MySQL the same "half-open server" protection that
// PostgreSQL gets from lib/pq's connect_timeout, without go-sql-driver's ReadTimeout
// (which would apply to every subsequent query read, not just the handshake).
type timeoutConnector struct {
	driver.Connector
	timeout time.Duration
}

func (c *timeoutConnector) Connect(ctx context.Context) (driver.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.Connector.Connect(ctx)
}

// createPostGresDBSessionWithAzure creates postgresDB session with azure token
func createPostGresDBSessionWithAzure(cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool, username string, connectTimeout time.Duration) (db.Session, error) {
	dsn := buildPostgresDSN(cfg, username, connectTimeout)

	scope := cfg.AzureToken.Scope
	if scope == "" {
		scope = "https://ossrdbms-aad.database.windows.net/.default"
	}

	connector := &azureConnector{
		dsn:   dsn,
		scope: scope,
	}

	return createPostGresDBSessionWithConnector(cfg, persistPool, connector)
}

// createPostGresDBSessionWithAWSRDS creates postgresDB session with AWS RDS IAM auth
func createPostGresDBSessionWithAWSRDS(cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool, username string, connectTimeout time.Duration) (db.Session, error) {
	dsn := buildPostgresDSN(cfg, username, connectTimeout)

	connector := &awsRDSConnector{
		dsn:      dsn,
		endpoint: cfg.GetHostname(),
		username: username,
		region:   cfg.AWSRDSToken.Region,
	}

	return createPostGresDBSessionWithConnector(cfg, persistPool, connector)
}

// createPostGresDBSessionWithCreds creates postgresDB session with direct credentials
func createPostGresDBSessionWithCreds(cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool, username, password string, connectTimeout time.Duration) (db.Session, error) {
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

	if cfg.Schema != "" {
		query.Set("search_path", cfg.Schema)
	}

	// connect_timeout limits connection setup (dial + handshake) to ensure fast failure if the DB is unreachable.
	// lib/pq resets this deadline afterward, leaving subsequent queries unaffected.
	query.Set("connect_timeout", strconv.Itoa(int(connectTimeout.Seconds())))
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
// buildMySQLConfig constructs the mysql.Config (DSN inputs) for a MySQL session,
// using mysql.Config to safely handle special characters in credentials and
// configuring the connection-establishment (dial) timeout.
//
// Start from mysql.NewConfig() rather than a struct literal so the driver
// defaults are applied — most importantly Loc: time.UTC. When the session was
// opened from a DSN string, ParseDSN restored those defaults; NewConnector
// consumes the config directly, so a bare literal would leave Loc nil and
// panic ("missing Location in call to Time.In") on the first time.Time written.
func buildMySQLConfig(cfg *config.MySQLConfig, username, password string, connectTimeout time.Duration) mysql.Config {
	mysqlCfg := mysql.NewConfig()
	mysqlCfg.User = username
	mysqlCfg.Passwd = password
	mysqlCfg.Net = "tcp"
	mysqlCfg.Addr = cfg.GetHostname()
	mysqlCfg.DBName = cfg.Database
	mysqlCfg.ParseTime = true
	mysqlCfg.AllowNativePasswords = true // Required for MariaDB which uses mysql_native_password by default
	mysqlCfg.Params = cfg.Options
	mysqlCfg.Timeout = connectTimeout
	return *mysqlCfg
}

func createMySQLDBSessionWithCreds(cfg *config.MySQLConfig, persistPool *config.ConnectionPool, username, password string, connectTimeout time.Duration) (db.Session, error) {
	mysqlCfg := buildMySQLConfig(cfg, username, password, connectTimeout)

	// Wrap the MySQL connector so Connect (dial + handshake read) is bounded by
	// connectTimeout, protecting against a half-open server the same way lib/pq's
	// connect_timeout protects PostgreSQL.
	connector, err := mysql.NewConnector(&mysqlCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql connector: %w", err)
	}
	wrapped := &timeoutConnector{Connector: connector, timeout: connectTimeout}

	// Create traced *sql.DB using otelsql
	sqlDB := otelsql.OpenDB(wrapped, otelSQLOptions(semconv.DBSystemNameMySQL, cfg.Database)...)

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
