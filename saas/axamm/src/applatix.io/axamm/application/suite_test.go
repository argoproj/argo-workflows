// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package application_test

import (
	"fmt"
	"testing"
	"time"

	"applatix.io/axamm"
	"applatix.io/axamm/adc"
	"applatix.io/axamm/application"
	"applatix.io/axamm/axam"
	"applatix.io/axamm/deployment"
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axdb/core"
	"applatix.io/axerror"
	"applatix.io/test"
	"gopkg.in/check.v1"
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
	application.GetLatestApplicationSchema(),
	application.GetHistoryApplicationSchema(),
	deployment.GetLatestDeploymentSchema(),
	deployment.GetHistoryDeploymentSchema(),
}

var TEST_PREFIX = test.RandStr()

func (s *S) SetUpSuite(c *check.C) {
	var axErr *axerror.AXError

	//utils.InitLoggers("APPLICATION-TEST")
	//_, axErr = kube.GetKubeClient()
	//if axErr != nil {
	//	fmt.Println(axErr)
	//	if strings.Contains(axErr.Message, "unable to load in-cluster configuration") {
	//		c.Skip("Local testing environment, kube test can only be run inside kube cluster.")
	//		return
	//	}
	//	panic(axErr)
	//}

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

	for _, table := range tables {
		_, axErr = axdbClient.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, table)
		c.Assert(axErr, check.IsNil)
		fmt.Printf("Update the table %v", table.Name)
	}

	axamm.InitTest("UNIT-TEST", axdburl, axmonurl, axmonurl, axmonurl, axmonurl, axmonurl)

	application.Init()
	axam.EnableTest()
	deployment.EnableTest()
	adc.EnableTest()
}

func (s *S) TearDownSuite(c *check.C) {
	//nss, err := kube.GetAllNameSpaces()
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//for _, ns := range nss {
	//	if strings.HasPrefix(ns.Name, TEST_PREFIX) {
	//		_, err = kube.DeleteNameSpace(ns)
	//		if err != nil {
	//			fmt.Println(err)
	//		}
	//	}
	//}
}
