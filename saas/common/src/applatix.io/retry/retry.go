package retry

import (
	"applatix.io/axerror"

	"math"
	"time"
)

type RetryConfig struct {
	retryTimeout           int64           // 0 indicates no timeout
	retryInterval          int64           // Minimum 1
	retryIntervalMax       int64           // 0 indicates no maximum interval
	retryExponentialFactor int64           // Minimum 1
	retryErrors            map[string]bool // Errors to retry
}

func NewRetryConfig(retryTimeout, retryInterval, retryIntervalMax, retryExponentialFactor int64, retryErrors map[string]bool) *RetryConfig {
	if retryIntervalMax <= 0 {
		retryIntervalMax = math.MaxInt64
	}
	if retryInterval < 1 {
		retryInterval = 1
	}
	if retryExponentialFactor < 1 {
		retryExponentialFactor = 1
	}

	return &RetryConfig{
		retryTimeout:           retryTimeout,
		retryInterval:          retryInterval,
		retryIntervalMax:       retryIntervalMax,
		retryExponentialFactor: retryExponentialFactor,
		retryErrors:            retryErrors,
	}
}

func (rc *RetryConfig) Retry(fn func() *axerror.AXError) *axerror.AXError {
	var endTime int64
	var interval int64
	var axErr *axerror.AXError = nil

	if rc.retryTimeout == 0 {
		endTime = math.MaxInt64
	} else {
		endTime = time.Now().Unix() + rc.retryTimeout
	}
	interval = rc.retryInterval

	for {
		axErr = fn()

		if axErr == nil || time.Now().Unix() >= endTime {
			break
		}

		if rc.retryErrors != nil {
			if _, ok := rc.retryErrors[axErr.Code]; !ok {
				break
			}
		}

		time.Sleep(time.Duration(interval) * time.Second)
		interval *= rc.retryExponentialFactor
		if interval > rc.retryIntervalMax {
			interval = rc.retryIntervalMax
		}
	}

	return axErr
}
