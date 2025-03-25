package sync

import (
	"time"

	log "github.com/sirupsen/logrus"
)

func newDatabaseMutex(name string, dbKey string, nextWorkflow NextWorkflow, info dbInfo) *databaseSemaphore {
	return &databaseSemaphore{
		name:              name,
		limit:             1,
		limitTimestamp:    time.Time{}, // not used
		syncLimitCacheTTL: 0,           // not used
		dbKey:             dbKey,
		nextWorkflow:      nextWorkflow,
		log: log.WithFields(log.Fields{
			"lockType": lockTypeMutex,
			"name":     name,
		}),
		info:    info,
		isMutex: true,
	}
}
