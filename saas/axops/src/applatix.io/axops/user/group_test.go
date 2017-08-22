// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package user_test

import (
	"applatix.io/axerror"
	"applatix.io/axops/user"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestGroupCreate(c *check.C) {
	g := &user.Group{
		Name: "GroupName-" + test.RandStr(),
	}

	g, err := g.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(g.ID), check.Not(check.Equals), 0)
}

func (s *S) TestGroupCreateNameDupe(c *check.C) {
	g := &user.Group{
		Name: "GroupName-" + test.RandStr(),
	}

	g, err := g.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(g.ID), check.Not(check.Equals), 0)

	g, err = g.Create()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_DUP_GROUPNAME.Code)
}

func (s *S) TestGroupReload(c *check.C) {
	g := &user.Group{
		Name: "GroupName-" + test.RandStr(),
	}

	g, err := g.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(g.ID), check.Not(check.Equals), 0)

	id := g.ID
	g = &user.Group{
		Name: g.Name,
	}

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g.ID, check.Equals, id)
}

func (s *S) TestGroupDelete(c *check.C) {
	g := &user.Group{
		Name: "GroupName-" + test.RandStr(),
	}

	g, err := g.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(g.ID), check.Not(check.Equals), 0)

	err = g.Delete()
	c.Assert(err, check.IsNil)

	g = &user.Group{
		Name: g.Name,
	}

	g, err = g.Reload()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_RESOURCE_NOT_FOUND.Code)
}

func (s *S) TestGroupDeleteHasUsername(c *check.C) {
	g := &user.Group{
		Name: "GroupName-" + test.RandStr(),
	}

	g, err := g.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(g.ID), check.Not(check.Equals), 0)

	u := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{g.Name},
	}

	u, err = u.Create()
	c.Assert(err, check.IsNil)

	g, err = g.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(g, check.NotNil)

	err = g.Delete()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_INVALID_REQ.Code)
}

func (s *S) TestGroupGetByID(c *check.C) {
	g := &user.Group{
		Name: "GroupName-" + test.RandStr(),
	}

	g, err := g.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(g.ID), check.Not(check.Equals), 0)

	id := g.ID

	g, err = user.GetGroupById(id)
	c.Assert(err, check.IsNil)
	c.Assert(g, check.NotNil)
}

func (s *S) TestGroupGetByName(c *check.C) {
	g := &user.Group{
		Name: "GroupName-" + test.RandStr(),
	}

	g, err := g.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(g.ID), check.Not(check.Equals), 0)

	name := g.Name

	g, err = user.GetGroupByName(name)
	c.Assert(err, check.IsNil)
	c.Assert(g, check.NotNil)
}
