package sync

import (
	syncdb "github.com/argoproj/argo-workflows/v3/util/sync/db"
)

// newDatabaseMutex creates a databaseSemaphore configured to act as a mutex for the provided name and database key.
// The returned semaphore is set up to use info.SessionProxy and info.Config for its sync queries and to call nextWorkflow when scheduling the next holder.
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