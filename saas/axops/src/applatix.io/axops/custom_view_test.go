package axops_test

import (
	"applatix.io/axops/custom_view"
	"applatix.io/test"
	"gopkg.in/check.v1"
	"time"
)

func (s *S) TestCustomViewCRUD(c *check.C) {

	// C
	view := &custom_view.CustomView{
		Name:     "name-" + test.RandStr(),
		Type:     "type-" + test.RandStr(),
		UserID:   "user-id-" + test.RandStr(),
		Username: "username-" + test.RandStr(),
		Info:     "info-" + test.RandStr(),
	}

	cp := &custom_view.CustomView{}
	err, _ := axopsClient.Post2("custom_views", nil, view, cp)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.ID, check.Not(check.Equals), "")
	c.Assert(cp.Ctime, check.Not(check.Equals), 0)
	c.Assert(cp.Mtime, check.Not(check.Equals), 0)
	c.Assert(cp.Name, check.Equals, view.Name)
	c.Assert(cp.Type, check.Equals, view.Type)
	c.Assert(cp.Info, check.Equals, view.Info)
	c.Assert(cp.UserID, check.Not(check.Equals), "")
	c.Assert(cp.Username, check.Not(check.Equals), "")

	view.ID = cp.ID
	view.UserID = cp.UserID
	view.Username = cp.Username

	// R
	cp = &custom_view.CustomView{}
	err = axopsClient.Get("custom_views/"+view.ID, nil, cp)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.ID, check.Not(check.Equals), "")
	c.Assert(cp.Ctime, check.Not(check.Equals), 0)
	c.Assert(cp.Mtime, check.Not(check.Equals), 0)
	c.Assert(cp.Name, check.Equals, view.Name)
	c.Assert(cp.Type, check.Equals, view.Type)
	c.Assert(cp.Info, check.Equals, view.Info)
	c.Assert(cp.UserID, check.Equals, view.UserID)
	c.Assert(cp.Username, check.Equals, view.Username)
	c.Assert(cp.ID, check.Equals, view.ID)

	// U
	view.Name = "name-" + test.RandStr()
	view.Info = "info-" + test.RandStr()
	view.Type = "type-" + test.RandStr()

	cp = &custom_view.CustomView{}
	err, _ = axopsClient.Put2("custom_views/"+view.ID, nil, view, cp)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.ID, check.Not(check.Equals), "")
	c.Assert(cp.Ctime, check.Not(check.Equals), 0)
	c.Assert(cp.Mtime, check.Not(check.Equals), 0)
	c.Assert(cp.Name, check.Equals, view.Name)
	c.Assert(cp.Type, check.Equals, view.Type)
	c.Assert(cp.Info, check.Equals, view.Info)
	c.Assert(cp.UserID, check.Equals, view.UserID)
	c.Assert(cp.Username, check.Equals, view.Username)
	c.Assert(cp.ID, check.Equals, view.ID)

	// D
	res := map[string]interface{}{}
	err, _ = axopsClient.Delete2("custom_views/"+view.ID, nil, nil, &res)
	c.Assert(err, check.IsNil)

	cp = &custom_view.CustomView{}
	err = axopsClient.Get("custom_views/"+view.ID, nil, cp)
	c.Assert(err, check.NotNil)
}

func (s *S) TestCustomViewList(c *check.C) {

	endpoint := "custom_views"

	randStr := "rand" + test.RandStr()
	p1 := &custom_view.CustomView{
		Name:     randStr,
		Type:     randStr,
		UserID:   randStr,
		Username: randStr,
		Info:     randStr,
	}

	randStr = "rand" + test.RandStr()
	p2 := &custom_view.CustomView{
		Name:     randStr,
		Type:     randStr,
		UserID:   randStr,
		Username: randStr,
		Info:     randStr,
	}

	_, err, _ := p1.Create()
	c.Assert(err, check.IsNil)

	_, err, _ = p2.Create()
	c.Assert(err, check.IsNil)

	time.Sleep(time.Second)

	for _, field := range []string{
		custom_view.CustomViewName,
		custom_view.CustomViewType,
		custom_view.CustomViewUserID,
		custom_view.CustomViewUserName,
	} {
		params := map[string]interface{}{
			field: p1.Name,
		}

		// Filter by name
		data := &GeneralGetResult{}
		err = axopsClient.Get(endpoint, params, data)
		c.Assert(err, check.IsNil)
		c.Assert(data, check.NotNil)
		c.Assert(len(data.Data), check.Equals, 1)

		// Search by name to get one
		params = map[string]interface{}{
			field: "~" + p1.Name,
		}
		err = axopsClient.Get(endpoint, params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data), check.Equals, 1)

		// Search by name to get two
		params = map[string]interface{}{
			field: "~rand",
		}
		err = axopsClient.Get(endpoint, params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) >= 2, check.Equals, true)

		// Search by name to get two with limit one
		params = map[string]interface{}{
			field:   "~rand",
			"limit": 1,
		}
		err = axopsClient.Get(endpoint, params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) == 1, check.Equals, true)

		// Search to get one
		params = map[string]interface{}{
			"search": "~" + p1.Name,
		}
		err = axopsClient.Get(endpoint, params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) == 1, check.Equals, true)

		// Search to get more
		params = map[string]interface{}{
			"search": "~rand",
		}
		err = axopsClient.Get(endpoint, params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) >= 2, check.Equals, true)
	}
}
