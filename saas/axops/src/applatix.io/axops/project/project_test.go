// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package project_test

import (
	"applatix.io/axops/label"
	"applatix.io/axops/project"
	"applatix.io/axops/utils"
	"applatix.io/axops/yaml"
	"applatix.io/template"
	"gopkg.in/check.v1"
)

func assertLabels(c *check.C, count int) {
	param := map[string]interface{}{
		label.LabelType: label.LabelTypeProject,
	}
	labelList, err := label.GetLabels(param)
	c.Assert(err, check.IsNil)
	c.Assert(len(labelList), check.Equals, count)
}

func (s *S) TestProject(c *check.C) {
	c.Log("TestProject")
	p := &project.Project{}
	p.Name = "test"
	p.Description = "testD"
	p.Categories = []string{"A", "B"}
	p.Labels = map[string]string{"lang": "go", "db": "axdb"}
	p.Assets = &project.Assets{
		Icon:   &project.AssetDetail{Path: "icon.png"},
		Detail: &project.AssetDetail{Path: "d.md"},
	}
	p.Actions = map[string]template.TemplateRef{
		"build": {
			Template: "T1",
			Arguments: map[string]*string{
				"commit": utils.NewString("a"),
			},
		},
		"test": {
			Template: "T2",
			Arguments: map[string]*string{
				"image": utils.NewString("i1"),
			},
		},
	}
	p.Repo = "R1"
	p.Branch = "B1"
	p1, err := p.Insert()
	c.Assert(err, check.IsNil)

	assertLabels(c, 2)

	p2, err := project.GetProjectByID(p1.ID)
	c.Assert(p1.ID, check.Equals, p2.ID)

	p.Description = "updated"

	_, err = p.Update()
	c.Assert(err, check.IsNil)
	p3, err := project.GetProjectByID(p1.ID)
	c.Assert(p3.Description, check.Equals, p.Description)

	err = p.Delete()
	c.Assert(err, check.IsNil)
	p4, err := project.GetProjectByID(p1.ID)
	c.Assert(p4 == nil, check.Equals, true)
	yaml.GarbageCollectLabels()
	assertLabels(c, 0)
}
