package file

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// WaitForCreate blocks until a file or directory exists at path, or ctx is
// cancelled. The parent directory of path MUST already exist; callers are
// responsible for creating it (e.g. with os.MkdirAll) before calling.
//
// There is no fallback poll — if the kernel does not deliver an event we
// will wait forever (until ctx is cancelled).
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
// Blocks until ctx is cancelled or the kernel reports an unrecoverable
// watcher error. There is no fallback poll.
func WatchFile(ctx context.Context, path string, onChange func()) error {
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
