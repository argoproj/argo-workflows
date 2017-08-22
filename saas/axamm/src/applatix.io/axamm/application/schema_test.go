package application_test

import (
	"applatix.io/axamm/application"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestCreateApplicationObject(c *check.C) {

	name := TEST_PREFIX + "-" + "applicaiton-" + test.RandStr()

	a := &application.Application{
		Name:        name,
		Description: "This doesn't matter at all",
	}

	a, err, _ := a.CreateObject()
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Name, check.Equals, name)
	c.Assert(len(a.Description), check.Not(check.Equals), 0)
	c.Assert(len(a.ID), check.Not(check.Equals), 0)
	c.Assert(a.Ctime, check.Not(check.Equals), 0)
	c.Assert(a.Mtime, check.Not(check.Equals), 0)
	c.Assert(a.Status, check.Equals, application.AppStateInit)

	get, err := application.GetLatestApplicationByName(name, true)
	c.Assert(err, check.IsNil)
	c.Assert(get, check.NotNil)
	c.Assert(get.ID, check.Equals, a.ID)

	get, err = application.GetLatestApplicationByID(a.ID, true)
	c.Assert(err, check.IsNil)
	c.Assert(get, check.NotNil)
	c.Assert(get.Name, check.Equals, a.Name)

	// Cannot create with the same name
	b := &application.Application{
		Name:        name,
		Description: "This doesn't matter at all",
	}
	_, err, _ = b.CreateObject()
	c.Assert(err, check.NotNil)
}

func (s *S) TestUpdateApplicationObject(c *check.C) {

	name := TEST_PREFIX + "-" + "applicaiton-" + test.RandStr()

	a := &application.Application{
		Name:        name,
		Description: "This doesn't matter at all",
	}

	a, err, _ := a.CreateObject()
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Name, check.Equals, name)
	c.Assert(len(a.Description), check.Not(check.Equals), 0)
	c.Assert(len(a.ID), check.Not(check.Equals), 0)
	c.Assert(a.Ctime, check.Not(check.Equals), 0)
	c.Assert(a.Mtime, check.Not(check.Equals), 0)
	c.Assert(a.Status, check.Equals, application.AppStateInit)

	a.Status = application.AppStateTerminating
	a, err, _ = a.UpdateObject()
	c.Assert(err, check.IsNil)
	c.Assert(a.Name, check.Equals, name)
	c.Assert(a.Status, check.Equals, application.AppStateTerminating)

	apps, err := application.GetLatestApplications(nil, true)
	c.Assert(err, check.IsNil)
	c.Assert(apps, check.NotNil)
	c.Assert(len(apps), check.Not(check.Equals), 0)
}

func (s *S) TestDeleteApplicationObject(c *check.C) {

	name := TEST_PREFIX + "-" + "applicaiton-" + test.RandStr()

	a := &application.Application{
		Name:        name,
		Description: "This doesn't matter at all",
	}

	a, err, _ := a.CreateObject()
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)

	get, err := application.GetLatestApplicationByName(name, true)
	c.Assert(err, check.IsNil)
	c.Assert(get, check.NotNil)
	c.Assert(get.ID, check.Equals, a.ID)

	err, _ = a.MarkObjectTerminated(nil)
	c.Assert(err, check.IsNil)

	get, err = application.GetLatestApplicationByName(name, true)
	c.Assert(err, check.IsNil)
	c.Assert(get.Status, check.Equals, application.AppStateTerminated)
}
