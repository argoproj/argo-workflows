// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package user_test

import (
	"applatix.io/axerror"
	"applatix.io/axops/user"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestCreateUser(c *check.C) {
	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)

	g := &user.Group{
		Name: "admin",
	}

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.HasUser(u.Username), check.Equals, true)
}

func (s *S) TestCreateUserInvalidGroup(c *check.C) {
	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin", "unknown"},
	}

	_, err := u.Create()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_RESOURCE_NOT_FOUND.Code)

	g := &user.Group{
		Name: "admin",
	}

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.HasUser(u.Username), check.Equals, false)
}

func (s *S) TestCreateUserNameDupe(c *check.C) {
	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestCreateUserNameDupe@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)

	g := &user.Group{
		Name: "admin",
	}

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.HasUser(u.Username), check.Equals, true)

	err = u.Active()
	c.Assert(err, check.IsNil)

	u, err = u.Create()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_DUP_USERNAME.Code)
}

func (s *S) TestCreateUserMissScheme(c *check.C) {
	u := &user.User{
		LastName:  "Wang",
		FirstName: "Hong",
		Username:  "TestCreateUserMissScheme@" + test.RandStr() + ".com",
		Password:  "applatix",
		Groups:    []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_AX_INTERNAL.Code)
}

func (s *S) TestCreateUserMissGroup(c *check.C) {
	u := &user.User{
		LastName:  "Wang",
		FirstName: "Hong",
		Username:  "TestCreateUserMissScheme@" + test.RandStr() + ".com",
		Password:  "applatix",
	}

	u, err := u.Create()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_AX_INTERNAL.Code)
}

func (s *S) TestCreateUserWeakPassword(c *check.C) {
	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestCreateUserWeakPassword@" + test.RandStr() + ".com",
		Password:    "0000",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_WEAK_PASSWORD.Code)
}

func (s *S) TestCreateUserInvalidUsername(c *check.C) {
	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "admin",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_INVALID_USERNAME.Code)
}

func (s *S) TestBanUnBanUser(c *check.C) {
	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestBanUnBanUser@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	u, err = user.GetUserByName(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateInit)

	err = u.Active()
	c.Assert(err, check.IsNil)
	u, err = user.GetUserByName(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateActive)

	err = u.Ban()
	c.Assert(err, check.IsNil)
	u, err = user.GetUserByName(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateBanned)

	err = u.Active()
	c.Assert(err, check.IsNil)
	u, err = user.GetUserByName(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateActive)
}

func (s *S) TestActiveUser(c *check.C) {
	u := &user.User{
		Username:    "TestActiveUser@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	u, err = user.GetUserByName(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateInit)

	g := &user.Group{
		Name: "admin",
	}

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.HasUser(u.Username), check.Equals, true)

	err = u.Active()
	c.Assert(err, check.IsNil)
	u, err = user.GetUserByName(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateActive)
}

func (s *S) TestActiveUserInvalid(c *check.C) {
	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestActiveUserInvalid@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)
	u, err = user.GetUserByName(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateInit)

	g := &user.Group{
		Name: "admin",
	}

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.HasUser(u.Username), check.Equals, true)

	err = u.Active()
	c.Assert(err, check.IsNil)
	u, err = user.GetUserByName(u.Username)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateActive)

	err = u.Delete()
	c.Assert(err, check.IsNil)

	err = u.Active()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_INVALID_REQ.Code)
}

func (s *S) TestUserDelete(c *check.C) {
	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)

	g := &user.Group{
		Name: "admin",
	}

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.HasUser(u.Username), check.Equals, true)

	err = u.Delete()
	c.Assert(err, check.IsNil)

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.HasUser(u.Username), check.Equals, false)
}

func (s *S) TestUserDeleteMissingGroup(c *check.C) {
	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	u, err := u.Create()
	c.Assert(err, check.IsNil)

	g := &user.Group{
		Name: "admin",
	}

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.HasUser(u.Username), check.Equals, true)

	u.Groups = []string{"admin", "unknown"}
	err = u.Update()
	c.Assert(err, check.NotNil)

	err = u.Delete()
	c.Assert(err, check.IsNil)

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.HasUser(u.Username), check.Equals, false)
}

// TestUserPasswordChange verifies we reject password changes if user is a portal user, or if auth scheme is *only* saml.
// This is because passwords should be updated from the identity provider instead.
func (s *S) TestUserPasswordChange(c *check.C) {
	originalPW := "originalPassword1!"
	newPW := "newPassword1!"
	samlOnlyUser := &user.User{
		LastName:    "Only",
		FirstName:   "SAML",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		AuthSchemes: []string{"saml"},
		Groups:      []string{"admin"},
	}

	samlAndLocallUser := &user.User{
		LastName:    "Local",
		FirstName:   "SAML",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		Password:    originalPW,
		AuthSchemes: []string{"native", "saml"},
		Groups:      []string{"admin"},
	}

	localOnly := &user.User{
		LastName:    "Only",
		FirstName:   "Local",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		Password:    originalPW,
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
	}

	allowed := []*user.User{samlAndLocallUser, localOnly}
	disallowed := []*user.User{samlOnlyUser}

	for _, u := range allowed {
		_, err := u.Create()
		c.Assert(err, check.IsNil)
		err = u.ChangePassword(originalPW, newPW)
		c.Assert(err, check.IsNil)
		u, err = user.GetUserByName(u.Username)
		c.Assert(err, check.IsNil)
		c.Assert(u.CheckPassword(newPW), check.Equals, true)
	}

	for _, u := range disallowed {
		_, err := u.Create()
		c.Assert(err, check.IsNil)
		err = u.ChangePassword(originalPW, newPW)
		c.Assert(err, check.NotNil)
		u, err = user.GetUserByName(u.Username)
		c.Assert(err, check.IsNil)
		if len(u.AuthSchemes) == 1 && u.AuthSchemes[0] == "saml" {
			// CheckPassword doesn't work for SAML only user in unit test context
			continue
		}
		c.Assert(u.CheckPassword(originalPW), check.Equals, true)
	}
}
