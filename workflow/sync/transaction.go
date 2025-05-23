package sync

import (
	"github.com/upper/db/v4"
)

type transaction struct {
	db *db.Session
}
