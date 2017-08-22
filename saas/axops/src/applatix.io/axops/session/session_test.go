// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package session_test

import (
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axdb/core"
	"applatix.io/axerror"
	"applatix.io/axops"
	"applatix.io/axops/session"
	"applatix.io/test"
	"fmt"
	"gopkg.in/check.v1"
	"math/rand"
	"strconv"
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
	session.SessionSchema,
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

func (s *S) TestSessionCreate(c *check.C) {
	ssn := &session.Session{
		UserID:   "aaaa",
		Username: "a@a.com",
		Scheme:   "native",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)
	c.Assert(ssn.ID, check.Not(check.HasLen), 0)
	c.Assert(ssn.Expiry, check.Not(check.Equals), 0)

	ssn, err = session.GetSessionById(ssn.ID)
	c.Assert(err, check.IsNil)
	c.Assert(ssn.UserID, check.Equals, "aaaa")
	c.Assert(ssn.Username, check.Equals, "a@a.com")
	c.Assert(ssn.Scheme, check.Equals, "native")
}

func (s *S) TestSessionReload(c *check.C) {
	ssn := &session.Session{
		UserID:   "aaaa",
		Username: "a@a.com",
		Scheme:   "native",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)

	id := ssn.ID

	ssn = &session.Session{
		ID: id,
	}

	ssn, err = ssn.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.NotNil)
	c.Assert(ssn.ID, check.Equals, ssn.ID)
	c.Assert(ssn.UserID, check.Equals, "aaaa")
	c.Assert(ssn.Username, check.Equals, "a@a.com")
	c.Assert(ssn.Scheme, check.Equals, "native")
}

func (s *S) TestSessionDelete(c *check.C) {
	ssn := &session.Session{
		UserID:   "aaaa",
		Username: "a@a.com",
		Scheme:   "native",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)

	id := ssn.ID
	err = ssn.Delete()
	c.Assert(err, check.IsNil)
	ssn, err = session.GetSessionById(id)
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.IsNil)
}

func (s *S) TestSessionValidate(c *check.C) {
	ssn := &session.Session{
		UserID:   "aaaa",
		Username: "a@a.com",
		Scheme:   "native",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)
	id := ssn.ID

	err = ssn.Validate()
	c.Assert(err, check.IsNil)

	ssn.Expiry = time.Now().Add(-1 * time.Minute).Unix()
	err = ssn.Validate()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_EXPIRED_SESSION.Code)
	ssn, err = session.GetSessionById(id)
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.IsNil)
}

//func (s *S) TestSessionExtendNative(c *check.C) {
//	ssn := &session.Session{
//		UserID:   "aaaa",
//		Username: "a@a.com",
//		Scheme:   "native",
//	}
//
//	ssn, err := ssn.Create()
//	c.Assert(err, check.IsNil)
//	id := ssn.ID
//	expiry := ssn.Expiry
//
//	err = ssn.Extend()
//	c.Assert(err, check.IsNil)
//
//	ssn, err = session.GetSessionById(id)
//	c.Assert(err, check.IsNil)
//	c.Assert(ssn, check.NotNil)
//	c.Assert(ssn.Expiry, check.Equals, expiry)
//
//	time.Sleep(time.Second)
//	ssn.Expiry = time.Now().Add(10 * time.Hour).Unix()
//	err = ssn.Extend()
//	c.Assert(err, check.IsNil)
//	ssn, err = session.GetSessionById(id)
//	c.Assert(err, check.IsNil)
//	c.Assert(ssn, check.NotNil)
//	c.Assert(ssn.Expiry, check.Not(check.Equals), expiry)
//}

//func (s *S) TestSessionExtendOther(c *check.C) {
//	ssn := &session.Session{
//		UserID:   "aaaa",
//		Username: "a@a.com",
//		Scheme:   "saml",
//	}
//
//	ssn, err := ssn.Create()
//	c.Assert(err, check.IsNil)
//	id := ssn.ID
//	expiry := ssn.Expiry
//
//	err = ssn.Extend()
//	c.Assert(err, check.IsNil)
//
//	ssn, err = session.GetSessionById(id)
//	c.Assert(err, check.IsNil)
//	c.Assert(ssn, check.NotNil)
//	c.Assert(ssn.Expiry, check.Equals, expiry)
//
//	time.Sleep(time.Second)
//	ssn.Expiry = time.Now().Add(1 * time.Hour).Unix()
//	err = ssn.Extend()
//	c.Assert(err, check.IsNil)
//	ssn, err = session.GetSessionById(id)
//	c.Assert(err, check.IsNil)
//	c.Assert(ssn, check.NotNil)
//	c.Assert(ssn.Expiry, check.Equals, expiry)
//}

func (s *S) TestSessionGetByUserID(c *check.C) {
	ssn := &session.Session{
		UserID:   strconv.Itoa(rand.Int()),
		Username: "a@a.com",
		Scheme:   "saml",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)

	ssn, err = ssn.Create()
	c.Assert(err, check.IsNil)

	ssns, err := session.GetSessionByUserID(ssn.UserID)
	c.Assert(err, check.IsNil)
	c.Assert(len(ssns), check.Equals, 2)
}

func (s *S) TestSessionGetByUserName(c *check.C) {
	ssn := &session.Session{
		UserID:   "aaaa",
		Username: strconv.Itoa(rand.Int()),
		Scheme:   "saml",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)

	ssn, err = ssn.Create()
	c.Assert(err, check.IsNil)

	ssns, err := session.GetSessionByUsername(ssn.Username)
	c.Assert(err, check.IsNil)
	c.Assert(len(ssns), check.Equals, 2)
}
