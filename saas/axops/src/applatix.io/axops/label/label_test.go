// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package label_test

import (
	"applatix.io/axerror"
	"applatix.io/axops/label"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestLabelCreate(c *check.C) {
	l := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}

	l, err := l.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(l.ID), check.Not(check.Equals), 0)
}

func (s *S) TestLabelCreateNameDupe(c *check.C) {
	l := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}

	l, err := l.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(l.ID), check.Not(check.Equals), 0)

	l, err = l.Create()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_DUP_LABEL.Code)
}

func (s *S) TestLabelReload(c *check.C) {
	l := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}

	l, err := l.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(l.ID), check.Not(check.Equals), 0)

	id := l.ID
	l = &label.Label{
		Type: label.LabelTypeUser,
		Key:  l.Key,
	}

	l, err = l.Reload()
	c.Assert(err, check.IsNil)
	c.Assert(l.ID, check.Equals, id)
}

func (s *S) TestLabelDelete(c *check.C) {
	l := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}

	l, err := l.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(l.ID), check.Not(check.Equals), 0)

	err = l.Delete()
	c.Assert(err, check.IsNil)

	l = &label.Label{
		Type: label.LabelTypeUser,
		Key:  l.Key,
	}

	l, err = l.Reload()
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_RESOURCE_NOT_FOUND.Code)
}

func (s *S) TestLabelGet(c *check.C) {
	l := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}

	l, err := l.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(l.ID), check.Not(check.Equals), 0)

	name := l.Key

	l, err = label.GetLabel(label.LabelTypeUser, name, "")
	c.Assert(err, check.IsNil)
	c.Assert(l, check.NotNil)
}

func (s *S) TestLabelGetByID(c *check.C) {
	l := &label.Label{
		Type: label.LabelTypeUser,
		Key:  "Label-" + test.RandStr(),
	}

	l, err := l.Create()
	c.Assert(err, check.IsNil)
	c.Assert(len(l.ID), check.Not(check.Equals), 0)

	id := l.ID

	l, err = label.GetLabelByID(id)
	c.Assert(err, check.IsNil)
	c.Assert(l, check.NotNil)
}
