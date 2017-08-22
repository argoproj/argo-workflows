// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package auth_test

import (
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axdb/core"
	"applatix.io/axerror"
	"applatix.io/axops"
	"applatix.io/axops/auth"
	"applatix.io/axops/auth/native"
	"applatix.io/axops/session"
	"applatix.io/axops/user"
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
	auth.AuthRequestSchema,
	user.UserSchema,
	user.SystemRequestSchema,
	session.SessionSchema,
}

var u *user.User
var scheme *auth.BaseScheme = &auth.BaseScheme{}

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

	u = &user.User{
		Username:    "admin@example.com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	user, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(user, check.NotNil)
	err = u.Active()
	c.Assert(err, check.IsNil)
	u = user
}

func (s *S) TestBaseSchemeAuth(c *check.C) {
	ssn := &session.Session{
		UserID:   u.ID,
		Username: u.Username,
		Scheme:   "native",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.NotNil)

	user, ssn, err := scheme.Auth(ssn.ID)
	c.Assert(err, check.IsNil)
	c.Assert(user, check.NotNil)
	c.Assert(ssn, check.NotNil)
	c.Assert(user.Username, check.Equals, "admin@example.com")
	c.Assert(user.ID, check.Equals, u.ID)
}

func (s *S) TestBaseSchemeAuthInvalidSession(c *check.C) {
	user, ssn, err := scheme.Auth("this is not a valid session")
	c.Assert(err, check.NotNil)
	c.Assert(user, check.IsNil)
	c.Assert(ssn, check.IsNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_AUTH_FAILED.Code)
}

func (s *S) TestBaseSchemeAuthMissingUser(c *check.C) {
	ssn := &session.Session{
		UserID:   "missing user id",
		Username: "missing user name",
		Scheme:   "native",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.NotNil)

	user, ssn, err := scheme.Auth(ssn.ID)
	c.Assert(err, check.NotNil)
	c.Assert(user, check.IsNil)
	c.Assert(ssn, check.IsNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_AUTH_FAILED.Code)
}

func (s *S) TestBaseSchemeAuthExpiredSession(c *check.C) {
	ssn := &session.Session{
		UserID:   u.ID,
		Username: u.Username,
		Scheme:   "native",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.NotNil)

	id := ssn.ID

	// turn session to be expired
	ssn.Expiry = time.Now().Add(-1 * time.Hour).Unix()
	err = ssn.Save()
	c.Assert(err, check.IsNil)

	user, ssn, err := scheme.Auth(ssn.ID)
	c.Assert(err, check.NotNil)
	c.Assert(user, check.IsNil)
	c.Assert(ssn, check.IsNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_EXPIRED_SESSION.Code)

	// expired session purged
	ssn, err = session.GetSessionById(id)
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.IsNil)
}

func (s *S) TestBaseSchemeLogout(c *check.C) {
	ssn := &session.Session{
		UserID:   u.ID,
		Username: u.Username,
		Scheme:   "native",
	}

	ssn, err := ssn.Create()
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.NotNil)

	id := ssn.ID

	err = scheme.Logout(ssn)
	c.Assert(err, check.IsNil)

	// session gone
	ssn, err = session.GetSessionById(id)
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.IsNil)
}

func (s *S) TestBaseSchemeDeleteUser(c *check.C) {
	// create a user
	usr := &user.User{
		Username:    "delete@example" + strconv.Itoa(rand.Int()) + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	usr, err := scheme.CreateUser(usr)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)

	ssn := &session.Session{
		UserID:   usr.ID,
		Username: usr.Username,
		Scheme:   "native",
	}

	// create two sessions
	ssn, err = ssn.Create()
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.NotNil)

	ssn, err = ssn.Create()
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.NotNil)

	err = scheme.DeleteUser(usr)
	c.Assert(err, check.IsNil)

	// sessions gone
	ssns, err := session.GetSessionByUserID(usr.ID)
	c.Assert(err, check.IsNil)
	c.Assert(ssns, check.HasLen, 0)

	// user gone
	usr, err = user.GetUserByName(usr.Username)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.IsNil)
}

func (s *S) TestBaseSchemeBanUser(c *check.C) {
	// create a user
	usr := &user.User{
		Username:    "ban@example" + strconv.Itoa(rand.Int()) + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	usr, err := scheme.CreateUser(usr)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)

	ssn := &session.Session{
		UserID:   usr.ID,
		Username: usr.Username,
		Scheme:   "native",
	}

	// create two sessions
	ssn, err = ssn.Create()
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.NotNil)

	ssn, err = ssn.Create()
	c.Assert(err, check.IsNil)
	c.Assert(ssn, check.NotNil)

	err = scheme.BanUser(usr)
	c.Assert(err, check.IsNil)

	// sessions gone
	ssns, err := session.GetSessionByUserID(usr.ID)
	c.Assert(err, check.IsNil)
	c.Assert(ssns, check.HasLen, 0)

	// user stay with ban state
	usr, err = user.GetUserByName(usr.Username)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)
	c.Assert(usr.State, check.Equals, user.UserStateBanned)
}

func (s *S) TestBaseSchemeActiveUser(c *check.C) {
	// create a user
	usr := &user.User{
		Username:    "active@example" + strconv.Itoa(rand.Int()) + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	usr, err := scheme.CreateUser(usr)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)

	err = scheme.ActiveUser(usr)
	c.Assert(err, check.IsNil)

	// user activated
	usr, err = user.GetUserByName(usr.Username)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)
	c.Assert(usr.State, check.Equals, user.UserStateActive)

	// ban user
	err = scheme.BanUser(usr)
	c.Assert(err, check.IsNil)

	usr, err = user.GetUserByName(usr.Username)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)
	c.Assert(usr.State, check.Equals, user.UserStateBanned)

	// active it again
	err = scheme.ActiveUser(usr)
	c.Assert(err, check.IsNil)

	// user activated
	usr, err = user.GetUserByName(usr.Username)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)
	c.Assert(usr.State, check.Equals, user.UserStateActive)
}

func (s *S) TestSchemeRegistration(c *check.C) {
	native := native.NativeScheme{&auth.BaseScheme{}}
	auth.RegisterScheme("native", &native)

	scheme, err := auth.GetScheme("native")
	c.Assert(err, check.IsNil)
	c.Assert(scheme, check.NotNil)

	auth.UnregisterScheme("native")
	scheme, err = auth.GetScheme("native")
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_RESOURCE_NOT_FOUND.Code)
}

func (s *S) TestSchemeCRUDSuperAdmin(c *check.C) {
	var err *axerror.AXError

	// create a user
	usr := &user.User{
		Username:    "active@example" + strconv.Itoa(rand.Int()) + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupSuperAdmin},
	}

	_, err = scheme.CreateUser(usr)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	err = scheme.DeleteUser(usr)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	err = scheme.BanUser(usr)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	err = scheme.ActiveUser(usr)
	c.Assert(err, check.NotNil)
	fmt.Println(err)
}
