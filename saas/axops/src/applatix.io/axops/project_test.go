// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

import (
	"encoding/json"
	"time"

	"applatix.io/axops/project"
	"applatix.io/common"
	"applatix.io/template"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func assertCatSearch(c *check.C, cat string, expectedCount int) {
	param := map[string]interface{}{
		project.ProjectCategories: cat,
	}
	data := &GeneralGetResult{}
	err := axopsClient.Get("projects", param, &data)
	dataBytes, _ := json.Marshal(data)
	c.Logf("project search api response: %v", string(dataBytes))
	c.Assert(err, check.IsNil)
	c.Assert(len(data.Data), check.Equals, expectedCount)
}

func assertLabelSearch(c *check.C, word string, expectedCount int) {
	param := map[string]interface{}{
		"search": word,
	}
	data := &GeneralGetResult{}
	err := axopsClient.Get("projects", param, &data)
	c.Assert(err, check.IsNil)
	c.Assert(len(data.Data), check.Equals, expectedCount)
}

func (s *S) TestProjectGetList(c *check.C) {
	randStr := "rand" + test.RandStr()
	p1 := &project.Project{}
	p1.Name = randStr
	p1.Description = randStr
	p1.Repo = randStr
	p1.Branch = randStr
	p1.Categories = []string{"A2", "B1"}
	p1.Labels = map[string]string{"lang": "go", "db": "axdb"}
	p1.Assets = &project.Assets{
		Icon:   &project.AssetDetail{Path: "icon.png"},
		Detail: &project.AssetDetail{Path: "d.md"},
	}
	p1.Actions = map[string]template.TemplateRef{
		"build": template.TemplateRef{
			Template:  "T1",
			Arguments: map[string]*string{"repo": test.NewString("a")},
		},
		"test": template.TemplateRef{
			Template:  "T2",
			Arguments: map[string]*string{"image": test.NewString("i1")},
		},
	}

	randStr = "rand" + test.RandStr()
	p2 := &project.Project{}
	p2.Name = randStr
	p2.Description = randStr
	p2.Repo = randStr
	p2.Branch = randStr
	p2.Categories = []string{"A1", "B1"}

	p2.Labels = map[string]string{"lang": "go"}
	p2.Assets = &project.Assets{
		Icon:   &project.AssetDetail{Path: "icon.png"},
		Detail: &project.AssetDetail{Path: "d.md"},
	}
	p2.Actions = map[string]template.TemplateRef{
		"build": template.TemplateRef{
			Template:  "T1",
			Arguments: map[string]*string{"repo": test.NewString("a")},
		},
		"test": template.TemplateRef{
			Template:  "T2",
			Arguments: map[string]*string{"image": test.NewString("i1")},
		},
	}

	_, err := p1.Insert()
	c.Assert(err, check.IsNil)

	_, err = p2.Insert()
	c.Assert(err, check.IsNil)

	time.Sleep(time.Second)

	assertCatSearch(c, "A2", 1)
	assertCatSearch(c, "B1", 2)
	assertCatSearch(c, "A1", 1)
	assertCatSearch(c, "A3", 0)

	assertLabelSearch(c, "~go", 2)
	assertLabelSearch(c, "~axdb", 1)
	assertLabelSearch(c, "~gone", 0)

	for _, field := range []string{
		project.ProjectName,
		project.ProjectDescription,
		project.ProjectRepo,
		project.ProjectBranch,
	} {
		params := map[string]interface{}{
			field: p1.Description,
		}

		// Filter by name
		data := &GeneralGetResult{}
		err = axopsClient.Get("projects", params, data)
		c.Assert(err, check.IsNil)
		c.Assert(data, check.NotNil)
		c.Assert(len(data.Data), check.Equals, 1)

		// Search by name to get one
		params = map[string]interface{}{
			field: "~" + p1.Description,
		}
		err = axopsClient.Get("projects", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data), check.Equals, 1)

		// Search by name to get two
		params = map[string]interface{}{
			field: "~rand",
		}
		err = axopsClient.Get("projects", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) >= 2, check.Equals, true)

		// Search by name to get two with limit one
		params = map[string]interface{}{
			field:   "~rand",
			"limit": 1,
		}
		err = axopsClient.Get("projects", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) == 1, check.Equals, true)

		// Search to get one
		params = map[string]interface{}{
			"search": "~" + p1.Description,
		}
		err = axopsClient.Get("projects", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) == 1, check.Equals, true)

		// Search to get more
		params = map[string]interface{}{
			"search": "~rand",
		}
		err = axopsClient.Get("projects", params, &data)
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
	err = axopsClient.Get("projects", params, &data)
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
	err = axopsClient.Get("projects", params, &data)
	c.Assert(err, check.IsNil)
	c.Assert(len(data.Data) == 2, check.Equals, true)
}
