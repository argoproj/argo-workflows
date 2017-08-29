// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

import (
	"encoding/json"
	"time"

	"applatix.io/axops/label"
	"applatix.io/axops/policy"
	"applatix.io/common"
	"applatix.io/template"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestPolicyGetList(c *check.C) {

	randStr := "rand" + test.RandStr()
	p1 := &policy.Policy{}
	p1.Name = randStr
	p1.Description = randStr
	p1.Repo = randStr
	p1.Branch = randStr
	p1.Template = randStr
	p1.Enabled = true
	p1.Notifications = []template.Notification{
		template.Notification{
			When: []string{"on_change"},
			Whom: []string{label.UserLabelSubmitter},
		},
	}
	p1.When = []template.When{
		template.When{
			Event: "on_push",
		},
	}
	p1.Arguments = map[string]*string{
		"a": test.NewString("a"),
		"b": test.NewString("1"),
		"c": test.NewString("1.0"),
		"d": test.NewString("true"),
	}

	randStr = "rand" + test.RandStr()
	p2 := &policy.Policy{}
	p2.Name = randStr
	p2.Description = randStr
	p2.Repo = randStr
	p2.Branch = randStr
	p2.Template = randStr
	p2.Enabled = false
	p2.Notifications = []template.Notification{}
	p2.When = []template.When{
		template.When{
			Event: "on_push",
		},
	}
	p2.Arguments = map[string]*string{
		"a": test.NewString("a"),
		"b": test.NewString("1"),
		"c": test.NewString("1.0"),
		"d": test.NewString("true"),
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
	p1 := &policy.Policy{}
	p1.Name = randStr
	p1.Description = randStr
	p1.Repo = randStr
	p1.Branch = randStr
	p1.Template = randStr
	p1.Enabled = false
	p1.Notifications = []template.Notification{
		template.Notification{
			When: []string{"on_change"},
			Whom: []string{label.UserLabelSubmitter},
		},
	}
	p1.When = []template.When{
		template.When{
			Event: "on_push",
		},
	}
	p1.Arguments = map[string]*string{
		"a": test.NewString("a"),
		"b": test.NewString("1"),
		"c": test.NewString("1.0"),
		"d": test.NewString("true"),
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
	c.Assert(p.Enabled, check.Equals, true)

	_, err = axopsClient.Put("policies/"+p.ID+"/disable", nil)
	c.Assert(err, check.IsNil)

	p, err = policy.GetPolicyByID(p1.ID)
	c.Assert(err, check.IsNil)
	c.Assert(p, check.NotNil)
	c.Assert(p.ID, check.Equals, p1.ID)
	c.Assert(p.Enabled, check.Equals, false)
}
