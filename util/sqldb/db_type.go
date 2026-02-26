package sqldb

import "github.com/upper/db/v4"

type DBType string

const (
	MySQL    DBType = "mysql"
	Postgres DBType = "postgres"
	SQLite   DBType = "sqlite"
	Invalid  DBType = "invalid"
)

// DBTypeFor returns the DBType for a given database session by inspecting its adapter name.
func DBTypeFor(session db.Session) DBType {
	switch session.Name() {
	case "postgresql":
		return Postgres
	case "mysql":
		return MySQL
	case "sqlite3":
		return SQLite
	default:
		return Invalid
	}
}

func (t DBType) IntType() string {
	if t == MySQL {
		return "signed"
	}
	return "int"
}
