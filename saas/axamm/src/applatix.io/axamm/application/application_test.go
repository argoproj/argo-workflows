package application_test

import (
	"time"

	"applatix.io/axamm/application"
	"applatix.io/axamm/heartbeat"
	"applatix.io/axamm/utils"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestCreateDeleteApplication(c *check.C) {

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

	get, err = application.GetLatestApplicationByName(name, true)
	c.Assert(err, check.IsNil)
	c.Assert(get, check.NotNil)
	c.Assert(get.Name, check.Equals, name)
	c.Assert(get.Status, check.Equals, application.AppStateWaiting)

	key := get.Key()
	c.Assert(heartbeat.GetHandler(key), check.NotNil)
	c.Assert(heartbeat.GetFreshness(key), check.Not(check.Equals), 0)

	hb := &heartbeat.HeartBeat{
		Date: time.Now().Unix(),
		Key:  key,
	}
	heartbeat.ProcessHeartBeat(hb)
	c.Assert(heartbeat.GetFreshness(key), check.Not(check.Equals), 0)

	_, err, _ = a.Delete()
	c.Assert(err, check.IsNil)

	get, err = application.GetLatestApplicationByName(name, true)
	c.Assert(err, check.IsNil)
	c.Assert(get.Status, check.Equals, application.AppStateTerminated)

	c.Assert(heartbeat.GetHandler(key), check.IsNil)
	c.Assert(heartbeat.GetFreshness(key), check.Equals, int64(0))

	apps, err := application.GetHistoryApplications(map[string]interface{}{application.ApplicationName: name}, true)
	c.Assert(err, check.IsNil)
	c.Assert(len(apps), check.Equals, 1)

	// Create again to generate history
	a, err, _ = a.Create()
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)

	get, err = application.GetLatestApplicationByName(name, true)
	c.Assert(err, check.IsNil)
	c.Assert(get, check.NotNil)
	c.Assert(get.Name, check.Equals, name)
	c.Assert(get.Status, check.Equals, application.AppStateWaiting)

	apps, err = application.GetHistoryApplications(map[string]interface{}{application.ApplicationName: name}, true)
	c.Assert(err, check.IsNil)
	c.Assert(len(apps), check.Equals, 1)

	// Create again with return the same obj
	b := &application.Application{
		Name:        name,
		Description: "This doesn't matter at all 2",
	}

	b, err, _ = b.Create()
	c.Assert(err, check.IsNil)
	c.Assert(a, check.NotNil)
	c.Assert(a.ID, check.Equals, b.ID)

	// Cannot create while previous is terminating
	err, _ = a.MarkObjectTerminating(utils.GetStatusDetail("", "", ""))
	c.Assert(err, check.IsNil)
	b, err, _ = b.Create()
	c.Assert(err, check.IsNil)
}
