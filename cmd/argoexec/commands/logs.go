package commands

import (
	"context"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// teeContainerLogs adds the combined file as an additional log output destination,
// preserving all fields, format, and hooks from the context Logger.
// The returned context embeds the new Logger.
func teeContainerLogs(ctx context.Context, varRunArgo, containerName string) (context.Context, func(), error) {
	// The "combined" file written here is the same one the wait container's
	// saveContainerLogs reads back to upload as the <container>-logs artifact.
	// Permissions are permissive because the file is written by this container
	// and read by the wait container over the shared /var/run/argo volume.
	dir := filepath.Join(varRunArgo, "ctr", containerName)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return ctx, func() {}, err
	}
	combinedPath := filepath.Join(dir, "combined")
	f, err := os.OpenFile(combinedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return ctx, func() {}, err
	}
	logger := logging.RequireLoggerFromContext(ctx)
	newCtx := logging.WithLogger(ctx, logging.TeeLogger(logger, f))
	return newCtx, func() { _ = f.Close() }, nil
}
