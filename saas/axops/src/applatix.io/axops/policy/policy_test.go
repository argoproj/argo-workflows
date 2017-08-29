// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package policy_test

import (
	"applatix.io/axops/policy"
	"applatix.io/axops/utils"
	"applatix.io/template"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestPolicyInsertUpdate(c *check.C) {
	randStr := test.RandStr()
	p := &policy.Policy{}
	p.ID = utils.GenerateUUIDv1()
	p.Name = randStr
	p.Description = randStr
	p.Repo = randStr
	p.Branch = randStr
	p.Template = randStr
	p.Enabled = true
	p.Notifications = []template.Notification{
		template.Notification{
			Whom: []string{"a", "b"},
			When: []string{"c", "d"},
		},
	}
	p.When = []template.When{
		template.When{
			Event: template.EventOnPullRequest,
		},
	}
	p.Arguments = map[string]*string{
		"a": test.NewString("a"),
		"b": test.NewString("1"),
		"c": test.NewString("1.0"),
		"d": test.NewString("true"),
	}

	p, err := p.Insert()
	c.Assert(err, check.IsNil)
	c.Assert(p, check.NotNil)

	copy, err := policy.GetPolicyByID(p.ID)
	c.Assert(err, check.IsNil)
	c.Assert(copy, check.NotNil)

	c.Assert(copy.ID, check.Equals, p.ID)
	c.Assert(copy.Name, check.Equals, p.Name)
	c.Assert(copy.Description, check.Equals, p.Description)
	c.Assert(copy.Repo, check.Equals, p.Repo)
	c.Assert(copy.Branch, check.Equals, p.Branch)
	c.Assert(copy.Template, check.Equals, p.Template)
	c.Assert(copy.Enabled, check.Equals, p.Enabled)
	c.Assert(len(copy.Notifications), check.Equals, len(p.Notifications))
	c.Assert(len(copy.When), check.Equals, len(p.When))
	c.Assert(len(copy.Arguments), check.Equals, len(p.Arguments))

	p.Enabled = false
	p.Description = "changed"
	p, err = p.Update()
	c.Assert(err, check.IsNil)
	c.Assert(p, check.NotNil)

	copy, err = policy.GetPolicyByID(p.ID)
	c.Assert(err, check.IsNil)
	c.Assert(copy, check.NotNil)

	c.Assert(copy.ID, check.Equals, p.ID)
	c.Assert(copy.Name, check.Equals, p.Name)
	c.Assert(copy.Description, check.Equals, p.Description)
	c.Assert(copy.Repo, check.Equals, p.Repo)
	c.Assert(copy.Branch, check.Equals, p.Branch)
	c.Assert(copy.Template, check.Equals, p.Template)
	c.Assert(copy.Enabled, check.Equals, p.Enabled)
	c.Assert(len(copy.Notifications), check.Equals, len(p.Notifications))
	c.Assert(len(copy.When), check.Equals, len(p.When))
	c.Assert(len(copy.Arguments), check.Equals, len(p.Arguments))

	err = p.Delete()
	c.Assert(err, check.IsNil)

	copy, err = policy.GetPolicyByID(p.ID)
	c.Assert(err, check.IsNil)
	c.Assert(copy, check.IsNil)
}
