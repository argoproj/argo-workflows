package sync

import (
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
)

func newDatabaseMutex(name string, dbKey string, nextWorkflow NextWorkflow, info syncdb.DBInfo) *databaseSemaphore {
	logger := syncLogger{
		name:     name,
		lockType: lockTypeMutex,
	}
	return &databaseSemaphore{
		name:         name,
		limitGetter:  &mutexLimit{},
		shortDBKey:   dbKey,
		nextWorkflow: nextWorkflow,
		logger:       logger.get,
		info:         info,
		queries:      syncdb.NewSyncQueries(info.SessionProxy, info.Config),
		isMutex:      true,
	}
}
