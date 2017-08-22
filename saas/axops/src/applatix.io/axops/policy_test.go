// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

import (
	"applatix.io/axops/label"
	"applatix.io/axops/notification"
	"applatix.io/axops/policy"
	"applatix.io/common"
	"applatix.io/test"
	"encoding/json"
	"gopkg.in/check.v1"
	"time"
)

func (s *S) TestPolicyGetList(c *check.C) {

	randStr := "rand" + test.RandStr()
	p1 := &policy.Policy{
		Name:        randStr,
		Description: randStr,
		Repo:        randStr,
		Branch:      randStr,
		Template:    randStr,
		Enabled:     test.NewTrue(),
		Notifications: []notification.Notification{
			notification.Notification{
				When: []string{"on_change"},
				Whom: []string{label.UserLabelSubmitter},
			},
		},
		When: []policy.When{
			policy.When{
				Event:          "on_push",
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

	randStr = "rand" + test.RandStr()
	p2 := &policy.Policy{
		Name:          randStr,
		Description:   randStr,
		Repo:          randStr,
		Branch:        randStr,
		Template:      randStr,
		Enabled:       test.NewFalse(),
		Notifications: []notification.Notification{},
		When: []policy.When{
			policy.When{
				Event:          "on_push",
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

	_, err := p1.Insert()
	c.Assert(err, check.IsNil)

	_, err = p2.Insert()
	c.Assert(err, check.IsNil)

	time.Sleep(time.Second)

	for _, field := range []string{
		policy.PolicyName,
		policy.PolicyDescription,
		policy.PolicyRepo,
		policy.PolicyBranch,
		policy.PolicyTemplate,
	} {
		params := map[string]interface{}{
			field: p1.Description,
		}

		// Filter by name
		data := &GeneralGetResult{}
		err = axopsClient.Get("policies", params, data)
		c.Assert(err, check.IsNil)
		c.Assert(data, check.NotNil)
		c.Assert(len(data.Data), check.Equals, 1)

		// Search by name to get one
		params = map[string]interface{}{
			field: "~" + p1.Description,
		}
		err = axopsClient.Get("policies", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data), check.Equals, 1)

		// Search by name to get two
		params = map[string]interface{}{
			field: "~rand",
		}
		err = axopsClient.Get("policies", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) >= 2, check.Equals, true)

		// Search by name to get two with limit one
		params = map[string]interface{}{
			field:   "~rand",
			"limit": 1,
		}
		err = axopsClient.Get("policies", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) == 1, check.Equals, true)

		// Search to get one
		params = map[string]interface{}{
			"search": "~" + p1.Description,
		}
		err = axopsClient.Get("policies", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) == 1, check.Equals, true)

		// Search to get more
		params = map[string]interface{}{
			"search": "~rand",
		}
		err = axopsClient.Get("policies", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) >= 2, check.Equals, true)
	}

	branches := []common.RepoBranch{
		common.RepoBranch{
			Repo:   p1.Repo,
			Branch: p1.Branch,
		},
	}

	branchesBytes, _ := json.Marshal(branches)
	params := map[string]interface{}{
		"repo_branch": string(branchesBytes),
	}
	data := &GeneralGetResult{}
	err = axopsClient.Get("policies", params, &data)
	c.Assert(err, check.IsNil)
	c.Assert(len(data.Data) == 1, check.Equals, true)

	branches = []common.RepoBranch{
		common.RepoBranch{
			Repo:   p1.Repo,
			Branch: p1.Branch,
		},
		common.RepoBranch{
			Repo:   p2.Repo,
			Branch: p2.Branch,
		},
	}

	branchesBytes, _ = json.Marshal(branches)
	params = map[string]interface{}{
		"repo_branch": string(branchesBytes),
	}
	data = &GeneralGetResult{}
	err = axopsClient.Get("policies", params, &data)
	c.Assert(err, check.IsNil)
	c.Assert(len(data.Data) == 2, check.Equals, true)
}

func (s *S) TestPolicyEnableDisable(c *check.C) {

	randStr := "rand" + test.RandStr()
	p1 := &policy.Policy{
		Name:        randStr,
		Description: randStr,
		Repo:        randStr,
		Branch:      randStr,
		Template:    randStr,
		Enabled:     test.NewFalse(),
		Notifications: []notification.Notification{
			notification.Notification{
				When: []string{"on_change"},
				Whom: []string{label.UserLabelSubmitter},
			},
		},
		When: []policy.When{
			policy.When{
				Event:          "on_push",
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

	p1, err := p1.Insert()
	c.Assert(err, check.IsNil)

	p, err := policy.GetPolicyByID(p1.ID)
	c.Assert(err, check.IsNil)
	c.Assert(p, check.NotNil)
	c.Assert(p.ID, check.Equals, p1.ID)

	_, err = axopsClient.Put("policies/"+p.ID+"/enable", nil)
	c.Assert(err, check.IsNil)

	p, err = policy.GetPolicyByID(p1.ID)
	c.Assert(err, check.IsNil)
	c.Assert(p, check.NotNil)
	c.Assert(p.ID, check.Equals, p1.ID)
	c.Assert(*p.Enabled, check.Equals, true)

	_, err = axopsClient.Put("policies/"+p.ID+"/disable", nil)
	c.Assert(err, check.IsNil)

	p, err = policy.GetPolicyByID(p1.ID)
	c.Assert(err, check.IsNil)
	c.Assert(p, check.NotNil)
	c.Assert(p.ID, check.Equals, p1.ID)
	c.Assert(*p.Enabled, check.Equals, false)
}
