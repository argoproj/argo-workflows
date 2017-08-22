// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axdb_test

import (
	"applatix.io/axdb/axdbcl"
	"applatix.io/axdb/core"
	"applatix.io/axerror"
	"flag"
	"gopkg.in/check.v1"
	"runtime/debug"
	"testing"
	"time"
)

const (
	axdburl        = "http://localhost:8080/v1"
	invalidAxdburl = "http://localhost:8080/v0.7"
	verbose        = true
)

const (
	appName     = "test"
	successCode = "ERR_OK"
)

type S struct{}

func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&S{})

// Use a client explicitly. Later replace the client with one that uses TLS
var axdbClient = axdbcl.NewAXDBClientWithTimeout(axdburl, time.Second*60)

func fail(c *check.C) {
	debug.PrintStack()
	c.FailNow()
}

func checkError(c *check.C, err *axerror.AXError) {
	c.Assert(err, check.IsNil)
}

func (s *S) SetUpSuite(c *check.C) {
	flag.Parse()

	// We test against our REST API. So we need to start our main program here.
	core.InitLoggers()
	core.InitDB(core.CassandraNodes)
	core.ReloadDBTable()
	go core.StartRouter(true)

	// wait for http server to be running
	for i := 0; i < 120; i++ {
		var bodyArray []interface{}
		err := axdbClient.Get("axdb", "status", nil, &bodyArray)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
}
