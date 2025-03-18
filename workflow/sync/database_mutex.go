package sync

import (
	log "github.com/sirupsen/logrus"
)

func newDatabaseMutex(name string, dbKey string, nextWorkflow NextWorkflow, info dbInfo) *databaseSemaphore {
	return &databaseSemaphore{
		name:         name,
		dbKey:        dbKey,
		nextWorkflow: nextWorkflow,
		log: log.WithFields(log.Fields{
			"lockType": lockTypeMutex,
			"name":     name,
		}),
		info:    info,
		limitFn: mutexLimit,
		isMutex: true,
	}
}

func mutexLimit() int {
	return 1
}
