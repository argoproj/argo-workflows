package sqldb

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
	"github.com/upper/db/v4"
)

type dbType string

const (
	MySQL    dbType = "mysql"
	Postgres dbType = "postgres"
)

func dbTypeFor(session db.Session) dbType {
	switch session.Driver().(*sql.DB).Driver().(type) {
	case *mysql.MySQLDriver:
		return MySQL
	}
	return Postgres
}

func (t dbType) intType() string {
	if t == MySQL {
		return "signed"
	}
	return "int"
}
