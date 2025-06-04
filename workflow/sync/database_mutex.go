package sync

import (
	log "github.com/sirupsen/logrus"
)

func newDatabaseMutex(name string, dbKey string, nextWorkflow NextWorkflow, info dbInfo) *databaseSemaphore {
	return &databaseSemaphore{
		name:         name,
		limitGetter:  &mutexLimit{},
		shortDBKey:   dbKey,
		nextWorkflow: nextWorkflow,
		log: log.WithFields(log.Fields{
			"lockType": lockTypeMutex,
			"name":     name,
		}),
		info:    info,
		isMutex: true,
	}
}
