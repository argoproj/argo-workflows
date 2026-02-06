package wait

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/wait"
)

// the underlying ExponentialBackoff does not retain the underlying error
// so this addresses this
func Backoff(b wait.Backoff, f func() (bool, error)) error {
	var err error
	waitErr := wait.ExponentialBackoff(b, func() (bool, error) {
		var done bool
		done, err = f()
		return done, nil
	})
	if waitErr != nil {
		if err != nil {
			return fmt.Errorf("%w: %w", waitErr, err)
		}
		return waitErr
	}
	return err
}
