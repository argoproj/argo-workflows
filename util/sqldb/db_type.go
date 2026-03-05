package sqldb

import "github.com/argoproj/argo-workflows/v4/config"

type DBType string

const (
	MySQL    DBType = "mysql"
	Postgres DBType = "postgres"
	SQLite   DBType = "sqlite"
	Invalid  DBType = "invalid"
)

func (t DBType) IntType() string {
	if t == MySQL {
		return "signed"
	}
	return "int"
}

// dbTypeFromConfig determines the DBType from a DBConfig.
func dbTypeFromConfig(cfg *config.DBConfig) DBType {
	if cfg == nil {
		return Invalid
	}
	if cfg.PostgreSQL != nil {
		return Postgres
	}
	if cfg.MySQL != nil {
		return MySQL
	}
	return Invalid
}
