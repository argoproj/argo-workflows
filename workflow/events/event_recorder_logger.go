package events

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

// setupKlogAdapter configures klog to use our logging system
func setupKlogAdapter(ctx context.Context) {
	logging.SetupKlogAdapter(ctx)
}
