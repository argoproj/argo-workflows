// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package native_test

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
	user.UserSchema,
	user.SystemRequestSchema,
	user.GroupSchema,
	session.SessionSchema,
}

var u *user.User
var scheme auth.ManagedScheme = &native.NativeScheme{&auth.BaseScheme{}}

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

	user.InitGroups()

	u = &user.User{
		Username:    "native@example.com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	uu, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(uu, check.NotNil)

	err = uu.Active()
	c.Assert(err, check.IsNil)

	u = uu
}

func (s *S) TestNativeSchemeLogin(c *check.C) {
	params := map[string]string{
		"username": u.Username,
		"password": "Test@test100",
	}
	usr, ssn, err := scheme.Login(params)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)
	c.Assert(usr.Username, check.Equals, u.Username)
	c.Assert(ssn, check.NotNil)
	c.Assert(ssn.Username, check.Equals, u.Username)
}

func (s *S) TestNativeSchemeLoginMissingUsername(c *check.C) {
	params := map[string]string{
		"username": "",
		"password": "Test@test100",
	}
	usr, ssn, err := scheme.Login(params)
	c.Assert(err, check.NotNil)
	c.Assert(usr, check.IsNil)
	c.Assert(ssn, check.IsNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_AXDB_INVALID_PARAM.Code)
}

func (s *S) TestNativeSchemeLoginMissingPassword(c *check.C) {
	params := map[string]string{
		"username": u.Username,
		"password": "",
	}
	usr, ssn, err := scheme.Login(params)
	c.Assert(err, check.NotNil)
	c.Assert(usr, check.IsNil)
	c.Assert(ssn, check.IsNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_AUTH_FAILED.Code)
}

func (s *S) TestNativeSchemeLoginMissingUser(c *check.C) {
	params := map[string]string{
		"username": "missing user",
		"password": "Test@test100",
	}
	usr, ssn, err := scheme.Login(params)
	c.Assert(err, check.NotNil)
	c.Assert(usr, check.IsNil)
	c.Assert(ssn, check.IsNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_AUTH_FAILED.Code)
}

func (s *S) TestNativeSchemeLoginWrongPassword(c *check.C) {
	params := map[string]string{
		"username": u.Username,
		"password": "wrong password",
	}
	usr, ssn, err := scheme.Login(params)
	c.Assert(err, check.NotNil)
	c.Assert(usr, check.IsNil)
	c.Assert(ssn, check.IsNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_AUTH_FAILED.Code)
}

func (s *S) TestNativeSchemeBannedUser(c *check.C) {
	u := &user.User{
		Username:    "banneduser@" + strconv.Itoa(rand.Int()) + "example.com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(u, check.NotNil)

	u.State = user.UserStateBanned
	err = u.Update()
	c.Assert(err, check.IsNil)

	params := map[string]string{
		"username": u.Username,
		"password": u.Password,
	}
	usr, ssn, err := scheme.Login(params)
	c.Assert(err, check.NotNil)
	c.Assert(usr, check.IsNil)
	c.Assert(ssn, check.IsNil)
	c.Assert(err.Code, check.Equals, auth.ErrUserBanned.Code)
}

func (s *S) TestNativeSchemeNotNative(c *check.C) {
	u := &user.User{
		Username:    "notnative@" + strconv.Itoa(rand.Int()) + "example.com",
		Password:    "Test@test100",
		AuthSchemes: []string{"saml"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(u, check.NotNil)

	u.State = user.UserStateActive
	err = u.Update()
	c.Assert(err, check.IsNil)

	params := map[string]string{
		"username": u.Username,
		"password": u.Password,
	}
	usr, ssn, err := scheme.Login(params)
	c.Assert(err, check.NotNil)
	c.Assert(usr, check.IsNil)
	c.Assert(ssn, check.IsNil)
	c.Assert(err.Code, check.Equals, native.ErrNotNativeScheme.Code)
}

func (s *S) TestNativeSchemeChangePassword(c *check.C) {
	u := &user.User{
		Username:    "changepassword@" + strconv.Itoa(rand.Int()) + "example.com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(u, check.NotNil)

	err = u.Active()
	c.Assert(err, check.IsNil)

	params := map[string]string{
		"username": u.Username,
		"password": "Test@test100",
	}
	usr, ssn, err := scheme.Login(params)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)
	c.Assert(usr.Username, check.Equals, u.Username)
	c.Assert(ssn, check.NotNil)
	c.Assert(ssn.Username, check.Equals, u.Username)

	ssns, err := session.GetSessionByUsername(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(len(ssns), check.Not(check.Equals), 0)

	err = scheme.ChangePassword(u, "Test@test100", "Test@test1001")
	c.Assert(err, check.IsNil)

	ssns, err = session.GetSessionByUsername(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(len(ssns), check.Equals, 0)

	params = map[string]string{
		"username": u.Username,
		"password": "Test@test1001",
	}
	usr, ssn, err = scheme.Login(params)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)
	c.Assert(usr.Username, check.Equals, u.Username)
	c.Assert(ssn, check.NotNil)
	c.Assert(ssn.Username, check.Equals, u.Username)
}

func (s *S) TestNativeSchemeChangePasswordOldWrong(c *check.C) {
	u := &user.User{
		Username:    "changepasswordoldwrong@" + strconv.Itoa(rand.Int()) + "example.com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(u, check.NotNil)

	err = scheme.ChangePassword(u, "wrongpassword", "APPLATIX")
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_AUTH_FAILED.Code)
}

func (s *S) TestNativeSchemeChangePasswordNewWeak(c *check.C) {
	u := &user.User{
		Username:    "changepasswordnewweek@applatix" + strconv.Itoa(rand.Int()) + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(u, check.NotNil)

	err = scheme.ChangePassword(u, "Test@test100", "short")
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_WEAK_PASSWORD.Code)
}

func (s *S) TestNativeSchemeResetPassword(c *check.C) {
	u := &user.User{
		Username:    "resetpassword@" + strconv.Itoa(rand.Int()) + "example.com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(u, check.NotNil)

	err = u.Active()
	c.Assert(err, check.IsNil)

	params := map[string]string{
		"username": u.Username,
		"password": "Test@test100",
	}
	usr, ssn, err := scheme.Login(params)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)
	c.Assert(usr.Username, check.Equals, u.Username)
	c.Assert(ssn, check.NotNil)
	c.Assert(ssn.Username, check.Equals, u.Username)

	ssns, err := session.GetSessionByUsername(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(len(ssns), check.Not(check.Equals), 0)

	err = scheme.ResetPassword(u, "Test@test1001")
	c.Assert(err, check.IsNil)

	ssns, err = session.GetSessionByUsername(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(len(ssns), check.Equals, 0)

	params = map[string]string{
		"username": u.Username,
		"password": "Test@test1001",
	}
	usr, ssn, err = scheme.Login(params)
	c.Assert(err, check.IsNil)
	c.Assert(usr, check.NotNil)
	c.Assert(usr.Username, check.Equals, u.Username)
	c.Assert(ssn, check.NotNil)
	c.Assert(ssn.Username, check.Equals, u.Username)
}

func (s *S) TestNativeSchemeResetPasswordWeakPassword(c *check.C) {
	u := &user.User{
		Username:    "resetpasswordweakpassword@" + strconv.Itoa(rand.Int()) + "example.com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(u, check.NotNil)

	err = scheme.ResetPassword(u, "short")
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_WEAK_PASSWORD.Code)
}

func (s *S) TestNativeSchemeStartResetPassword(c *check.C) {
	u := &user.User{
		Username:    "startresetpassword@" + strconv.Itoa(rand.Int()) + "example.com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	c.Assert(u, check.NotNil)

	err = scheme.StartPasswordReset(u)
	c.Assert(err, check.IsNil)

	reqs, err := user.GetSysReqsByTarget(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(reqs, check.HasLen, 2)
}
