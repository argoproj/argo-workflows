package sync

// newInternalMutex creates a size 1 semaphore
func newInternalMutex(name string, nextWorkflow NextWorkflow) *prioritySemaphore {
	return newInternalSemaphore(name, 1, nextWorkflow, lockTypeMutex)
}
