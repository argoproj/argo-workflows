package custom_view_test

import (
	"applatix.io/axerror"
	"applatix.io/axops/custom_view"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestCustomViewCreate(c *check.C) {
	view := &custom_view.CustomView{
		Name:     "name-" + test.RandStr(),
		Type:     "this is type",
		UserID:   "user-id-" + test.RandStr(),
		Username: "username-" + test.RandStr(),
		Info:     "info-" + test.RandStr(),
	}

	view, err, code := view.Create()
	c.Assert(err, check.IsNil)
	c.Assert(view, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_CREATE_OK)
	c.Assert(view.ID, check.Not(check.Equals), "")
	c.Assert(view.Ctime, check.Not(check.Equals), 0)
	c.Assert(view.Mtime, check.Not(check.Equals), 0)

	cp, err := custom_view.GetCustomViewById(view.ID)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.ID, check.Equals, view.ID)
	c.Assert(cp.Name, check.Equals, view.Name)
}

func (s *S) TestCustomViewUpdate(c *check.C) {
	view := &custom_view.CustomView{
		Name:     "name-" + test.RandStr(),
		Type:     "this is type",
		UserID:   "user-id-" + test.RandStr(),
		Username: "username-" + test.RandStr(),
		Info:     "info-" + test.RandStr(),
	}

	view, err, code := view.Create()
	c.Assert(err, check.IsNil)
	c.Assert(view, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_CREATE_OK)
	c.Assert(view.ID, check.Not(check.Equals), "")
	c.Assert(view.Ctime, check.Not(check.Equals), 0)
	c.Assert(view.Mtime, check.Not(check.Equals), 0)

	view.Name = "name-" + test.RandStr()
	view.Info = "info-" + test.RandStr()
	view, err, code = view.Update()
	c.Assert(err, check.IsNil)
	c.Assert(view, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_STATUS_OK)

	cp, err := custom_view.GetCustomViewById(view.ID)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.ID, check.Equals, view.ID)
	c.Assert(cp.Name, check.Equals, view.Name)
	c.Assert(cp.Info, check.Equals, view.Info)
}

func (s *S) TestCustomViewDelete(c *check.C) {
	view := &custom_view.CustomView{
		Name:     "name-" + test.RandStr(),
		Type:     "this is type",
		UserID:   "user-id-" + test.RandStr(),
		Username: "username-" + test.RandStr(),
		Info:     "info-" + test.RandStr(),
	}

	view, err, code := view.Create()
	c.Assert(err, check.IsNil)
	c.Assert(view, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_CREATE_OK)
	c.Assert(view.ID, check.Not(check.Equals), "")
	c.Assert(view.Ctime, check.Not(check.Equals), 0)
	c.Assert(view.Mtime, check.Not(check.Equals), 0)

	cp, err := custom_view.GetCustomViewById(view.ID)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.ID, check.Equals, view.ID)
	c.Assert(cp.Name, check.Equals, view.Name)
	c.Assert(cp.Info, check.Equals, view.Info)

	err, code = view.Delete()
	c.Assert(err, check.IsNil)
	c.Assert(code, check.Equals, axerror.REST_STATUS_OK)

	cp, err = custom_view.GetCustomViewById(view.ID)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.IsNil)
}
