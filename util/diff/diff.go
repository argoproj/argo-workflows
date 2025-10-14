package diff

import (
	"context"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func LogChanges(ctx context.Context, old, new interface{}) {
	logger := logging.RequireLoggerFromContext(ctx)
	// Note: We don't have a direct equivalent to log.IsLevelEnabled(log.DebugLevel)
	// The logger will handle level filtering internally
	a, _ := json.Marshal(old)
	b, _ := json.Marshal(new)
	patch, _ := jsonpatch.CreateMergePatch(a, b)
	logger.WithField("patch", string(patch)).Debug(ctx, "Log changes patch")
}
