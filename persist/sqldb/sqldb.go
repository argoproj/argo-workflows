package sqldb

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/mysql"
	"upper.io/db.v3/postgresql"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/util"
)

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
	return nil, "", fmt.Errorf("no databases are configured")
}

// CreatePostGresDBSession creates postgresDB session
func CreatePostGresDBSession(kubectlConfig kubernetes.Interface, namespace string, cfg *config.PostgreSQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, string, error) {
	if cfg.TableName == "" {
		return nil, "", errors.InternalError("tableName is empty")
	}

	ctx := context.Background()
	userNameByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, "", err
	}
	passwordByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, "", err
	}

	settings := postgresql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     cfg.GetHostname(),
		Database: cfg.Database,
	}

	if cfg.SSL {
		if cfg.SSLMode != "" && cfg.SSLMode != "disable" {
			err := os.MkdirAll(cfg.GetPGCertPath(), 0700)
			if err != nil {
				return nil, "", err
			}
			rootCertByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.CaCertSecret.Name, cfg.CaCertSecret.Key)
			if err != nil {
				return nil, "", err
			}
			err = os.WriteFile(cfg.GetPGCertPath()+"/ca.crt", rootCertByte, 0600)
			if err != nil {
				return nil, "", err
			}

			serverCertByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.ClientCertSecret.Name, cfg.ClientCertSecret.Key)
			if err != nil {
				return nil, "", err
			}
			err = os.WriteFile(cfg.GetPGCertPath()+"/tls.crt", serverCertByte, 0600)
			if err != nil {
				return nil, "", err
			}

			serverKeyByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.ClientKeySecret.Name, cfg.ClientKeySecret.Key)
			if err != nil {
				return nil, "", err
			}
			err = os.WriteFile(cfg.GetPGCertPath()+"/tls.key", serverKeyByte, 0400)
			if err != nil {
				return nil, "", err
			}

			options := map[string]string{
				"sslmode":     cfg.SSLMode,
				"sslrootcert": cfg.GetPGCertPath() + "/ca.crt",
				"sslkey":      cfg.GetPGCertPath() + "/tls.key",
				"sslcert":     cfg.GetPGCertPath() + "/tls.crt",
			}
			settings.Options = options
		}
	}

	session, err := postgresql.Open(settings)
	if err != nil {
		return nil, "", err
	}

	// default for connMaxLifetime to 300 seconds
	session.SetConnMaxLifetime(300 * time.Second)

	if persistPool != nil {
		session.SetMaxOpenConns(persistPool.MaxOpenConns)
		session.SetMaxIdleConns(persistPool.MaxIdleConns)
		session.SetConnMaxLifetime(time.Duration(persistPool.ConnMaxLifetime))
	}
	return session, cfg.TableName, nil
}

// CreateMySQLDBSession creates Mysql DB session
func CreateMySQLDBSession(kubectlConfig kubernetes.Interface, namespace string, cfg *config.MySQLConfig, persistPool *config.ConnectionPool) (sqlbuilder.Database, string, error) {
	if cfg.TableName == "" {
		return nil, "", errors.InternalError("tableName is empty")
	}

	ctx := context.Background()
	userNameByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.UsernameSecret.Name, cfg.UsernameSecret.Key)
	if err != nil {
		return nil, "", err
	}
	passwordByte, err := util.GetSecrets(ctx, kubectlConfig, namespace, cfg.PasswordSecret.Name, cfg.PasswordSecret.Key)
	if err != nil {
		return nil, "", err
	}

	session, err := mysql.Open(mysql.ConnectionURL{
		User:     string(userNameByte),
		Password: string(passwordByte),
		Host:     cfg.GetHostname(),
		Database: cfg.Database,
		Options:  cfg.Options,
	})
	if err != nil {
		return nil, "", err
	}

	if persistPool != nil {
		session.SetMaxOpenConns(persistPool.MaxOpenConns)
		session.SetMaxIdleConns(persistPool.MaxIdleConns)
		session.SetConnMaxLifetime(time.Duration(persistPool.ConnMaxLifetime))
	}
	// this is needed to make MySQL run in a Golang-compatible UTF-8 character set.
	_, err = session.Exec("SET NAMES 'utf8mb4'")
	if err != nil {
		return nil, "", err
	}
	_, err = session.Exec("SET CHARACTER SET utf8mb4")
	if err != nil {
		return nil, "", err
	}
	return session, cfg.TableName, nil
}
