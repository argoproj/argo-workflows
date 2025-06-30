package hydrator

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/packer"
)

type Interface interface {
	// whether or not the workflow in hydrated
	IsHydrated(wf *wfv1.Workflow) bool
	// hydrate the workflow - doing nothing if it is already hydrated
	Hydrate(ctx context.Context, wf *wfv1.Workflow) error
	// dehydrate the workflow - doing nothing if already dehydrated
	Dehydrate(ctx context.Context, wf *wfv1.Workflow) error
	// hydrate the workflow using the provided nodes
	HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes)
}

func New(offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo) Interface {
	return &hydrator{offloadNodeStatusRepo}
}

var alwaysOffloadNodeStatus = os.Getenv("ALWAYS_OFFLOAD_NODE_STATUS") == "true"

func init() {
	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	log := logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())
	log.WithField(ctx, "alwaysOffloadNodeStatus", alwaysOffloadNodeStatus).Debug(ctx, "Hydrator config")
}

type hydrator struct {
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

func (h hydrator) IsHydrated(wf *wfv1.Workflow) bool {
	return wf.Status.CompressedNodes == "" && !wf.Status.IsOffloadNodeStatus()
}

func (h hydrator) HydrateWithNodes(wf *wfv1.Workflow, offloadedNodes wfv1.Nodes) {
	wf.Status.Nodes = offloadedNodes
	wf.Status.CompressedNodes = ""
	wf.Status.OffloadNodeStatusVersion = ""
}

// should be <10s
// Retry	Seconds
// 1	0.10
// 2	0.30
// 3	0.70
// 4	1.50
// 5	3.10
var readRetry = wait.Backoff{Steps: 5, Duration: 100 * time.Millisecond, Factor: 2}

// needs to be long
// http://backoffcalculator.com/?attempts=5&rate=2&interval=1
// Retry	Seconds
// 1	1.00
// 2	3.00
// 3	7.00
// 4	15.00
// 5	31.00
var writeRetry = wait.Backoff{Steps: 5, Duration: 1 * time.Second, Factor: 2}

func (h hydrator) Hydrate(ctx context.Context, wf *wfv1.Workflow) error {
	log := logging.GetLoggerFromContext(ctx)
	if log == nil {
		log = logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())
	}
	err := packer.DecompressWorkflow(ctx, wf)
	if err != nil {
		return err
	}
	if wf.Status.IsOffloadNodeStatus() {
		var offloadedNodes wfv1.Nodes
		err := waitutil.Backoff(readRetry, func() (bool, error) {
			offloadedNodes, err = h.offloadNodeStatusRepo.Get(ctx, string(wf.UID), wf.GetOffloadNodeStatusVersion())
			return !errorsutil.IsTransientErr(ctx, err), err
		})
		if err != nil {
			return err
		}
		h.HydrateWithNodes(wf, offloadedNodes)
		log.WithField(ctx, "Workflow Size", wf.Size()).Info(ctx, "Workflow hydrated")
	}

	return nil
}

func (h hydrator) Dehydrate(ctx context.Context, wf *wfv1.Workflow) error {
	if !h.IsHydrated(wf) {
		return nil
	}
	log := logging.GetLoggerFromContext(ctx)
	if log == nil {
		log = logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())
	}
	var err error
	log.WithField(ctx, "Workflow Size", wf.Size()).Info(ctx, "Workflow to be dehydrated")
	if !alwaysOffloadNodeStatus {
		err = packer.CompressWorkflowIfNeeded(ctx, wf)
		if err == nil {
			wf.Status.OffloadNodeStatusVersion = ""
			return nil
		}
	}
	if packer.IsTooLargeError(err) || alwaysOffloadNodeStatus {
		var offloadVersion string
		var errMsg string
		if err != nil {
			errMsg += err.Error()
		}
		offloadErr := waitutil.Backoff(writeRetry, func() (bool, error) {
			var offloadErr error
			offloadVersion, offloadErr = h.offloadNodeStatusRepo.Save(ctx, string(wf.UID), wf.Namespace, wf.Status.Nodes)
			return !errorsutil.IsTransientErr(ctx, offloadErr), offloadErr
		})
		if offloadErr != nil {
			return fmt.Errorf("%sTried to offload but encountered error: %s", errMsg, offloadErr.Error())
		}
		wf.Status.Nodes = nil
		wf.Status.CompressedNodes = ""
		wf.Status.OffloadNodeStatusVersion = offloadVersion
		return nil
	} else {
		return err
	}
}
