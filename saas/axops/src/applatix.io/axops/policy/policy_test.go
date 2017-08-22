// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package policy_test

import (
	"applatix.io/axops/notification"
	"applatix.io/axops/policy"
	"applatix.io/axops/utils"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestPolicyInsertUpdate(c *check.C) {
	randStr := test.RandStr()
	p := &policy.Policy{
		ID:          utils.GenerateUUIDv1(),
		Name:        randStr,
		Description: randStr,
		Repo:        randStr,
		Branch:      randStr,
		Template:    randStr,
		Enabled:     test.NewTrue(),
		Notifications: []notification.Notification{
			notification.Notification{
				Whom: []string{"a", "b"},
				When: []string{"c", "d"},
			},
		},
		When: []policy.When{
			policy.When{
				Event:          policy.EventOnPullRequest,
				TargetBranches: []string{"master"},
			},
		},
		Parameters: map[string]string{
			"a": "a",
			"b": "1",
			"c": "1.0",
			"d": "true",
		},
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
	c.Assert(*copy.Enabled, check.Equals, *p.Enabled)
	c.Assert(len(copy.Notifications), check.Equals, len(p.Notifications))
	c.Assert(len(copy.When), check.Equals, len(p.When))
	c.Assert(len(copy.Parameters), check.Equals, len(p.Parameters))

	p.Enabled = test.NewFalse()
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
	c.Assert(*copy.Enabled, check.Equals, *p.Enabled)
	c.Assert(len(copy.Notifications), check.Equals, len(p.Notifications))
	c.Assert(len(copy.When), check.Equals, len(p.When))
	c.Assert(len(copy.Parameters), check.Equals, len(p.Parameters))

	err = p.Delete()
	c.Assert(err, check.IsNil)

	copy, err = policy.GetPolicyByID(p.ID)
	c.Assert(err, check.IsNil)
	c.Assert(copy, check.IsNil)
}
