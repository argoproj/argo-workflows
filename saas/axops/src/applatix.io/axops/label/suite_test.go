// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package label_test

import (
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axdb/core"
	"applatix.io/axerror"
	"applatix.io/axops"
	"applatix.io/axops/label"
	"applatix.io/test"
	"fmt"
	"gopkg.in/check.v1"
	"testing"
	"time"
)

const (
	axdburl        = "http://localhost:8080/v1"
	axmonurl       = "http://localhost:9090/v1"
	axnotifierurl  = "http://localhost:9090/v1"
	gatewayurl     = "http://localhost:9090/v1"
	workflowadcurl = "http://localhost:9090/v1"
)

var axdbClient = axdbcl.NewAXDBClientWithTimeout(axdburl, time.Second*60)

type S struct{}

func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&S{})

var tables []axdb.Table = []axdb.Table{
	label.LabelSchema,
}

func (s *S) SetUpSuite(c *check.C) {
	var axErr *axerror.AXError
	// We test against our REST API. So we need to start our main program here.
	core.InitLoggers()
	core.InitDB(core.CassandraNodes)
	core.ReloadDBTable()
	go core.StartRouter(true)

	// wait for axdb server to be running
	for i := 0; i < 60; i++ {
		var bodyArray []interface{}
		err := axdbClient.Get("axdb", "version", nil, &bodyArray)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	go test.StartFakeRouter(9090)

	// startup axops server
	axops.InitTest(axdburl, gatewayurl, workflowadcurl, axmonurl, axnotifierurl, "", "")
	for _, table := range tables {
		_, axErr = axdbClient.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, table)
		c.Assert(axErr, check.IsNil)
		fmt.Printf("Update the table %v", table.Name)
	}
}
