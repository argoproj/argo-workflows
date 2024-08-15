package sync

// NewMutex creates a size 1 semaphore
func NewMutex(name string, nextWorkflow NextWorkflow) *PrioritySemaphore {
	return NewSemaphore(name, 1, nextWorkflow, "mutex")
}
