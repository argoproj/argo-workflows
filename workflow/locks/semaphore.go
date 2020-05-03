package main

import (
	"context"
	"fmt"
	"github.com/argoproj/argo/workflow/common"
	"k8s.io/apimachinery/pkg/labels"

	"golang.org/x/sync/semaphore"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/controller"
)

type lock interface {
	Acquire() error
	Remove()
}

type WorkflowSemaphore struct{
	namespace string
	name string
	queue controller.Throttler
	sem semaphore.Weighted
	ctx context.Context
}

func (ws *WorkflowSemaphore) key () string {
	return fmt.Sprintf("%s/%s",ws.namespace, ws.name)
}

func (ws *WorkflowSemaphore) Acquire(wf *v1alpha1.Workflow) error {

	if ws.sem.TryAcquire(1) {
		ws.sem.Acquire(ws.ctx, 1)
		wf.Labels[common.LabelKeySemaphore] = ws.key()
		return nil
	}
}

var semaphoreMap map[string]WorkflowSemaphore








func main() {
	// a context is required for the weighted semaphore pkg.
	ctx := context.Background()
	for _, employee := range employeeList {
		if err := sem.Acquire(ctx, 1); err != nil {
			// handle error and maybe break
		}
		go func(){

			sem.Release(1)
		}()
	}
}