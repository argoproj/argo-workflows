package file

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// WaitForCreate blocks until a file or directory exists at path, or ctx is
// cancelled. The parent directory of path MUST already exist; callers are
// responsible for creating it (e.g. with os.MkdirAll) before calling.
func WaitForCreate(ctx context.Context, path string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var fired bool
	err := WatchFile(ctx, path, func() {
		fired = true
		cancel()
	})
	if fired {
		return nil
	}
	return err
}

// WatchFile invokes onChange whenever the file at path is created or
// written to. If the file already exists when the watcher starts, onChange
// is invoked once immediately. The parent directory of path MUST already
// exist.
//
// Uses inotify when available; falls back to 2s polling if the kernel's
// inotify limits (fs.inotify.max_user_instances / max_user_watches) are
// exhausted.
//
// Blocks until ctx is cancelled or an unrecoverable error occurs.
func WatchFile(ctx context.Context, path string, onChange func()) error {
	err := watchFileInotify(ctx, path, onChange)
	if isInotifyResourceExhausted(err) {
		logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx,
			"inotify unavailable; falling back to polling. Consider raising fs.inotify.max_user_instances / fs.inotify.max_user_watches")
		return watchFilePoll(ctx, path, onChange)
	}
	return err
}

func watchFileInotify(ctx context.Context, path string, onChange func()) error {
	path = filepath.Clean(path)
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating fsnotify watcher: %w", err)
	}
	defer w.Close()
	if err := w.Add(filepath.Dir(path)); err != nil {
		return fmt.Errorf("watching %q: %w", filepath.Dir(path), err)
	}
	if _, err := os.Stat(path); err == nil {
		onChange()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-w.Events:
			if !ok {
				return errors.New("fsnotify event channel closed")
			}
			if filepath.Clean(ev.Name) != path {
				continue
			}
			if ev.Has(fsnotify.Create) || ev.Has(fsnotify.Write) {
				onChange()
			}
		case err, ok := <-w.Errors:
			if !ok {
				return errors.New("fsnotify error channel closed")
			}
			return err
		}
	}
}

func watchFilePoll(ctx context.Context, path string, onChange func()) error {
	path = filepath.Clean(path)
	var last os.FileInfo
	if fi, err := os.Stat(path); err == nil {
		onChange()
		last = fi
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			fi, err := os.Stat(path)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					continue
				}
				return err
			}
			if last == nil || fi.ModTime() != last.ModTime() || fi.Size() != last.Size() {
				onChange()
				last = fi
			}
		}
	}
}

// isInotifyResourceExhausted reports whether err indicates the kernel has no
// more inotify instances or watches available for this user.
func isInotifyResourceExhausted(err error) bool {
	return errors.Is(err, syscall.EMFILE) || errors.Is(err, syscall.ENOSPC)
}
