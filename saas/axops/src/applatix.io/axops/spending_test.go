// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

import (
	"applatix.io/axops"
	"gopkg.in/check.v1"
)

type SpendingListResult struct {
	Data []axops.PerfData `json:"data,omitempty"`
}

func (s *S) TestSpending(t *check.C) {
	var spending SpendingListResult
	axErr := axopsClient.Get("spendings/perf/3600", nil, &spending)
	checkError(t, axErr)

	var usage axops.Usage
	axErr = axopsClient.Get("spendings/detail/0/7200", nil, &usage)
	checkError(t, axErr)
}
