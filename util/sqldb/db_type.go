package sqldb

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
	"github.com/upper/db/v4"
)

type DBType string

const (
	MySQL    DBType = "mysql"
	Postgres DBType = "postgres"
	SQLite   DBType = "sqlite"
)

func DBTypeFor(session db.Session) DBType {
	switch session.Driver().(*sql.DB).Driver().(type) {
	case *mysql.MySQLDriver:
		return MySQL
	}
	return Postgres
}

func (t DBType) IntType() string {
	if t == MySQL {
		return "signed"
	}
	return "int"
}
