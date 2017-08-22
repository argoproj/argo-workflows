// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package auth_test

import (
	"applatix.io/axops/auth"
	"gopkg.in/check.v1"
	"math/rand"
	"strconv"
	"time"
)

func (s *S) TestAuthRequestCreateWithID(c *check.C) {

	request := &auth.AuthRequest{
		ID: "TestAuthRequestCreateWithID" + strconv.Itoa(rand.Int()),
	}

	request, err := request.CreateWithID()
	c.Assert(err, check.IsNil)
	c.Assert(request, check.NotNil)
	c.Assert(request.Ctime, check.Not(check.Equals), 0)
	c.Assert(request.Expiry, check.Not(check.Equals), 0)
}

func (s *S) TestAuthRequestValidate(c *check.C) {

	request := &auth.AuthRequest{
		ID: "TestAuthRequestValidate" + strconv.Itoa(rand.Int()),
	}

	request, err := request.CreateWithID()

	id := request.ID

	c.Assert(err, check.IsNil)
	c.Assert(request, check.NotNil)
	c.Assert(request.Ctime, check.Not(check.Equals), 0)
	c.Assert(request.Expiry, check.Not(check.Equals), 0)

	err = request.Validate()
	c.Assert(err, check.IsNil)

	request.Expiry = time.Now().Add(-1 * time.Minute).Unix()
	err = request.Save()
	c.Assert(err, check.IsNil)

	err = request.Validate()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, auth.ErrAuthRequestNotFound.Code)

	request, err = auth.GetAuthRequestById(id)
	c.Assert(err, check.IsNil)
	c.Assert(request, check.IsNil)
}
