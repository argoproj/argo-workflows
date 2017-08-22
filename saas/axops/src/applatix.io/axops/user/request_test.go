// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package user_test

import (
	"math/rand"
	"strconv"
	"time"

	"applatix.io/axerror"
	"applatix.io/axops/user"
	"gopkg.in/check.v1"
)

func (s *S) TestCreateRequest(c *check.C) {
	r := &user.SystemRequest{
		UserID:   "aaaa",
		Username: "a@a.com",
		Target:   "b@b.com",
		Type:     user.SysReqPassReset,
	}

	r, err := r.Create(user.DEFAULT_REQUEST_DURATION)
	c.Assert(err, check.IsNil)
}

func (s *S) TestCreateRequestMissingUser(c *check.C) {
	r := &user.SystemRequest{
		Target: "b@b.com",
		Type:   user.SysReqPassReset,
	}

	r, err := r.Create(user.DEFAULT_REQUEST_DURATION)
	c.Assert(err, check.NotNil)
}

func (s *S) TestCreateRequestMissingType(c *check.C) {
	r := &user.SystemRequest{
		UserID:   "aaaa",
		Username: "a@a.com",
		Target:   "b@b.com",
	}

	r, err := r.Create(user.DEFAULT_REQUEST_DURATION)
	c.Assert(err, check.NotNil)
}

func (s *S) TestCreateRequestMissingTarget(c *check.C) {
	r := &user.SystemRequest{
		UserID:   "aaaa",
		Username: "a@a.com",
		Type:     user.SysReqPassReset,
	}

	r, err := r.Create(user.DEFAULT_REQUEST_DURATION)
	c.Assert(err, check.NotNil)
}

func (s *S) TestReloadRequest(c *check.C) {
	r := &user.SystemRequest{
		UserID:   "aaaa",
		Username: "a@a.com",
		Target:   "b@b.com",
		Type:     user.SysReqPassReset,
	}

	r, err := r.Create(user.DEFAULT_REQUEST_DURATION)
	c.Assert(err, check.IsNil)

	id := r.ID
	r = &user.SystemRequest{
		ID: id,
	}
	r, err = r.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(r.UserID, check.Equals, "aaaa")
	c.Assert(r.Username, check.Equals, "a@a.com")
	c.Assert(r.Target, check.Equals, "b@b.com")
	c.Assert(r.Type, check.Equals, int64(user.SysReqPassReset))

	r.Delete()
	c.Assert(err, check.IsNil)
	r = &user.SystemRequest{
		ID: id,
	}
	r, err = r.Reload()
	c.Assert(r, check.IsNil)
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_RESOURCE_NOT_FOUND.Code)
}

func (s *S) TestValidateRequest(c *check.C) {
	r := &user.SystemRequest{
		UserID:   "aaaa",
		Username: "a@a.com",
		Target:   "b@b.com",
		Type:     user.SysReqPassReset,
	}

	r, err := r.Create(user.DEFAULT_REQUEST_DURATION)
	c.Assert(err, check.IsNil)

	err = r.Validate()
	c.Assert(err, check.IsNil)

	r.Expiry = time.Now().Add(-1 * time.Minute).Unix()
	err = r.Validate()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_INVALID_REQ.Code)

	r, err = user.GetSysReqById(r.ID)
	c.Assert(err, check.IsNil)
	c.Assert(r, check.IsNil)
}

func (s *S) TestSendRequest(c *check.C) {
	r := &user.SystemRequest{
		UserID:   "aaaa",
		Username: "a@a.com",
		Target:   "b@b.com",
		Type:     user.SysReqPassReset,
	}

	r, err := r.Create(user.DEFAULT_REQUEST_DURATION)
	c.Assert(err, check.IsNil)

	r.Type = user.SysReqUserInvite
	err = r.SendRequest()
	c.Assert(err, check.IsNil)

	r.Type = user.SysReqUserConfirm
	err = r.SendRequest()
	c.Assert(err, check.IsNil)

	r.Type = user.SysReqPassReset
	err = r.SendRequest()
	c.Assert(err, check.IsNil)
}

func (s *S) TestGetRequestByTarget(c *check.C) {
	r := &user.SystemRequest{
		UserID:   "aaaa",
		Username: "a@a.com",
		Target:   "b@b" + strconv.Itoa(rand.Int()) + ".com",
		Type:     user.SysReqPassReset,
	}

	r, err := r.Create(user.DEFAULT_REQUEST_DURATION)
	c.Assert(err, check.IsNil)

	r, err = r.Create(user.DEFAULT_REQUEST_DURATION)
	c.Assert(err, check.IsNil)

	reqs, err := user.GetSysReqsByTarget(r.Target)
	c.Assert(err, check.IsNil)
	c.Assert(reqs, check.HasLen, 2)
}
