package sqldb

type DBType string

const (
	MySQL    DBType = "mysql"
	Postgres DBType = "postgres"
	SQLite   DBType = "sqlite"
)

func (t DBType) IntType() string {
	if t == MySQL {
		return "signed"
	}
	return "int"
}
