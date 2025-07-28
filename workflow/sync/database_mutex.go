package sync

func newDatabaseMutex(name string, dbKey string, nextWorkflow NextWorkflow, info dbInfo) *databaseSemaphore {
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
		isMutex:      true,
	}
}
