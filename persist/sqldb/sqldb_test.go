package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"upper.io/db.v3/mysql"
	"upper.io/db.v3/postgresql"
)

func Test_redactConnectionURL(t *testing.T) {
	t.Run("MySQL", func(t *testing.T) {
		url := mysql.ConnectionURL{
			User:     "my-user",
			Password: "my-password",
			Database: "my-database",
			Host:     "my-host",
			Socket:   "my-socket",
			Options:  map[string]string{"my-option": "my-option-value"},
		}
		assert.Equal(t, "my-user:******@unix(my-socket)/my-database?charset=utf8&my-option=my-option-value&parseTime=true", redactConnectionURL(url, "my-password"))
	})
	t.Run("Posgres", func(t *testing.T) {
		url := postgresql.ConnectionURL{
			User:     "my-user",
			Password: "my-password",
			Database: "my-database",
			Host:     "my-host",
			Socket:   "my-socket",
			Options:  map[string]string{"my-option": "my-option-value"},
		}
		assert.Equal(t, "user=my-user password=****** host=my-host host=my-socket dbname=my-database my-option=my-option-value sslmode=disable", redactConnectionURL(url, "my-password"))
	})
}
