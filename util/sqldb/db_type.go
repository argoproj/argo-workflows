package sqldb

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
	sqlite3 "github.com/mattn/go-sqlite3"
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
	case *sqlite3.SQLiteDriver:
		return SQLite
	}
	return Postgres
}

func (t DBType) IntType() string {
	if t == MySQL {
		return "signed"
	}
	return "int"
}
