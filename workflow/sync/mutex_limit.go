package sync

import "context"

type mutexLimit struct{}

var _ limitProvider = &mutexLimit{}

func (*mutexLimit) get(_ context.Context, _ string) (int, bool, error) {
	return 1, false, nil
}
