package sqldb

// MySQLVariant defines a MySQL-compatible database image for integration testing.
type MySQLVariant struct {
	Image       string
	WaitMessage string
}

// MySQLVariants contains the set of MySQL-compatible databases to test against.
var MySQLVariants = map[string]MySQLVariant{
	"MySQL":   {Image: "mysql:8.4", WaitMessage: "port: 3306  MySQL Community Server"},
	"MariaDB": {Image: "mariadb:11.4", WaitMessage: "mariadbd: ready for connections"},
}
