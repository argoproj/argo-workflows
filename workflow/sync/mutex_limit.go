package sync

type mutexLimit struct{}

var _ limitProvider = &mutexLimit{}

func (_ *mutexLimit) get(_ string) (int, bool, error) {
	return 1, false, nil
}
