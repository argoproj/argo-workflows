package sqldb

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
