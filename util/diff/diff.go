package diff

import (
	"context"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func LogChanges(ctx context.Context, old, newObj interface{}) {
	logger := logging.RequireLoggerFromContext(ctx)
	// Marshalling both objects and diffing them costs time and memory in
	// proportion to the workflow's size, so only do it when the patch is logged.
	if logger.Level() != logging.Debug {
		return
	}
	a, _ := json.Marshal(old)
	b, _ := json.Marshal(newObj)
	patch, _ := jsonpatch.CreateMergePatch(a, b)
	logger.WithField("patch", string(patch)).Debug(ctx, "Log changes patch")
}
