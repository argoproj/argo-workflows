//go:build !linux

package poolpod

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// BecomeSubreaper is Linux-only, as is loop mode.
func BecomeSubreaper() error {
	return fmt.Errorf("become child subreaper: %w", errors.ErrUnsupported)
}

// Reap is Linux-only, as is loop mode.
func Reap() (int, error) {
	return 0, fmt.Errorf("reap children: %w", errors.ErrUnsupported)
}

// Sweep needs /proc, so it is Linux-only, as is loop mode.
func Sweep(context.Context, time.Duration) error {
	return fmt.Errorf("sweep processes: %w", errors.ErrUnsupported)
}
