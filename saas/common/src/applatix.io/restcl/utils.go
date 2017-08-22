// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package restcl

import "time"

type RetryConfig struct {
	Timeout      time.Duration
	TriableCodes []string
}
