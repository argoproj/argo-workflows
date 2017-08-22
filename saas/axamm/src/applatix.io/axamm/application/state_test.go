package application_test

import (
	"applatix.io/axamm/application"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestStates(c *check.C) {

	name := TEST_PREFIX + "-" + "applicaiton-" + test.RandStr()

	get, err := application.GetLatestApplicationByName(name, true)
	c.Assert(err, check.IsNil)
	c.Assert(get, check.IsNil)

	a := &application.Application{
		Name:        name,
		Description: "This doesn't matter at all",
	}

	a, err, _ = a.Create()
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)

	a, err = application.GetLatestApplicationByID(a.ID, true)
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Status, check.Equals, application.AppStateWaiting)

	err, _ = a.MarkObjectWaiting(nil)
	a, err = application.GetLatestApplicationByID(a.ID, true)
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Status, check.Equals, application.AppStateWaiting)

	err, _ = a.MarkObjectActive(nil)
	a, err = application.GetLatestApplicationByID(a.ID, true)
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Status, check.Equals, application.AppStateActive)

	err, _ = a.MarkObjectError(nil)
	a, err = application.GetLatestApplicationByID(a.ID, true)
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Status, check.Equals, application.AppStateError)

	err, _ = a.MarkObjectStopping(nil)
	a, err = application.GetLatestApplicationByID(a.ID, true)
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Status, check.Equals, application.AppStateStopping)

	err, _ = a.MarkObjectStopped(nil)
	a, err = application.GetLatestApplicationByID(a.ID, true)
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Status, check.Equals, application.AppStateStopped)

	err, _ = a.MarkObjectTerminating(nil)
	a, err = application.GetLatestApplicationByID(a.ID, true)
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Status, check.Equals, application.AppStateTerminating)

	err, _ = a.MarkObjectTerminated(nil)
	a, err = application.GetLatestApplicationByID(a.ID, true)
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.Status, check.Equals, application.AppStateTerminated)
}
