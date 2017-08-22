// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

import (
	"applatix.io/axops/label"
	"applatix.io/axops/user"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestLabelList(c *check.C) {
	var labels GeneralGetResult
	err := axopsClient.Get("labels", nil, &labels)
	c.Assert(err, check.IsNil)
	c.Assert(len(labels.Data) >= 3, check.Equals, true)
}

func (s *S) TestLabelCreate(c *check.C) {
	label := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}
	err, _ := axopsClient.Post2("labels", nil, label, label)
	c.Assert(err, check.IsNil)
	c.Assert(len(label.ID), check.Not(check.Equals), 0)
	c.Assert(len(label.Key), check.Not(check.Equals), 0)
	c.Assert(label.Ctime, check.Not(check.Equals), 0)
	c.Assert(label.Reserved, check.Equals, false)

	err = axopsClient.Get("labels/"+label.ID, nil, label)
	c.Assert(err, check.IsNil)
}

func (s *S) TestLabelDelete(c *check.C) {
	label := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}
	err, _ := axopsClient.Post2("labels", nil, label, label)
	c.Assert(err, check.IsNil)
	c.Assert(len(label.ID), check.Not(check.Equals), 0)
	c.Assert(len(label.Key), check.Not(check.Equals), 0)
	c.Assert(label.Ctime, check.Not(check.Equals), 0)
	c.Assert(label.Reserved, check.Equals, false)

	err = axopsClient.Get("labels/"+label.ID, nil, label)
	c.Assert(err, check.IsNil)

	_, err = axopsClient.Delete("labels/"+label.ID, nil)
	c.Assert(err, check.IsNil)

	err = axopsClient.Get("labels/"+label.ID, nil, label)
	c.Assert(err, check.NotNil)
}

func (s *S) TestLabelUserCreate(c *check.C) {
	l1 := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}

	l1, err := l1.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(l1.ID), check.Not(check.Equals), 0)

	l2 := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}

	l2, err = l2.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(l2.ID), check.Not(check.Equals), 0)

	u1 := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
		Labels:      []string{l1.Key, l2.Key},
	}

	u2 := &user.User{
		LastName:    "Wang",
		FirstName:   "Hong",
		Username:    "TestCreateUser@" + test.RandStr() + ".com",
		Password:    "Test@test100",
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
		Labels:      []string{l1.Key, l2.Key},
	}

	_, err = axopsClient.Post("users", u1)
	c.Assert(err, check.IsNil)
	u1, err = u1.Reload()
	c.Assert(len(u1.Labels), check.Equals, 2)

	_, err = axopsClient.Post("users", u2)
	c.Assert(err, check.IsNil)
	u2, err = u2.Reload()
	c.Assert(len(u2.Labels), check.Equals, 2)

	_, err = axopsClient.Delete("labels/"+l1.ID, nil)
	c.Assert(err, check.IsNil)

	u1, err = u1.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(len(u1.Labels), check.Equals, 1)

	u2, err = u2.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(len(u2.Labels), check.Equals, 1)

	_, err = axopsClient.Delete("labels/"+l2.ID, nil)
	c.Assert(err, check.IsNil)

	u1, err = u1.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(len(u1.Labels), check.Equals, 0)

	u2, err = u2.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(len(u2.Labels), check.Equals, 0)
}
