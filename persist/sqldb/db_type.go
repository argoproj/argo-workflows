package sqldb

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
	"github.com/mattn/go-sqlite3"
	"github.com/upper/db/v4"
)

type dbType string

const (
	MySQL    dbType = "mysql"
	Postgres dbType = "postgres"
	SQLite   dbType = "sqlite"
)

func dbTypeFor(session db.Session) dbType {
	switch session.Driver().(*sql.DB).Driver().(type) {
	case *mysql.MySQLDriver:
		return MySQL
	case *sqlite3.SQLiteDriver:
		return SQLite
	}
	return Postgres
}

func (t dbType) intType() string {
	if t == MySQL {
		return "signed"
	}
	return "int"
}
