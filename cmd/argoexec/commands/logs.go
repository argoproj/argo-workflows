package commands

import (
	"context"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
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
	// Zero the umask so MkdirAll creates /var/run/argo/ctr (and below) with mode
	// 0o777. This directory is shared by every container in the pod, which may run
	// as different users (e.g. artifact plugin sidecars); if we created it with the
	// default umask the parent would be 0o755 and peers could not create their own
	// sibling directories. Matches the emissary/sidecar convention.
	osspecific.AllowGrantingAccessToEveryone()
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
