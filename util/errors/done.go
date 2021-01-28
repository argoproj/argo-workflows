package errors

// this function is intended to be used with wait.ExponentialBackoff to retry transient errors
// * no error? we are done
// * transient error? we ignore and try again
// * non-transient error? we are done
func Done(err error) (bool, error) {
	if IsTransientErr(err) {
		return false, nil
	}
	return true, err
}
