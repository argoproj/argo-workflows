package pool

import (
	"context"
	"sync"
)

// Task represents a single upload or download operation.
type Task struct {
	// SourcePath is a local filesystem path when uploading, or an S3 key when downloading.
	SourcePath string
	// DestKey is the destination S3 key when uploading, or a local filesystem path when downloading.
	DestKey string
	// IsUpload distinguishes upload vs download, useful for debugging or future hooks.
	IsUpload bool
}

// RunPool executes fn over the supplied tasks using a worker pool of size `workers`.
// It stops processing as soon as the first error is encountered and returns that error.
// If workers <= 0 it defaults to 1.
func RunPool(ctx context.Context, workers int, tasks []Task, fn func(Task) error) error {
	if workers <= 0 {
		workers = 1
	}

	// Derive a cancellable context so we can stop work early on error.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	taskCh := make(chan Task)
	errCh := make(chan error, 1) // buffer 1 so first writer never blocks

	var wg sync.WaitGroup
	// Launch workers.
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case t, ok := <-taskCh:
					if !ok {
						return
					}
					if err := fn(t); err != nil {
						// Capture only the first error.
						select {
						case errCh <- err:
						default:
						}
						cancel()
						return
					}
				}
			}
		}()
	}

	// Feed tasks to workers.
	go func() {
		for _, t := range tasks {
			select {
			case <-ctx.Done():
				break
			case taskCh <- t:
			}
		}
		close(taskCh)
	}()

	// Wait for workers to finish.
	wg.Wait()

	// Check for errors.
	select {
	case err := <-errCh:
		return err
	default:
	}

	// Propagate context cancellation if it was due to parent context.
	if ctx.Err() != nil && ctx.Err() != context.Canceled {
		return ctx.Err()
	}
	return nil
}
