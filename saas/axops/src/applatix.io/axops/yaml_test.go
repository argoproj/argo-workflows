// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

/*
import (
	"strings"
	"time"

	"applatix.io/axops"
	"applatix.io/axops/event"
	"applatix.io/axops/fixture"
	"applatix.io/axops/label"
	"applatix.io/axops/policy"
	"applatix.io/axops/project"
	"applatix.io/axops/service"
	"applatix.io/axops/tool"
	"applatix.io/axops/utils"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestTemplateYAML(c *check.C) {
	containerYaml, err := utils.ReadFromFile("testdata/yaml/container.yaml")
	c.Assert(err, check.IsNil)

	workflowYaml, err := utils.ReadFromFile("testdata/yaml/workflow.yaml")
	c.Assert(err, check.IsNil)

	nestedYaml, err := utils.ReadFromFile("testdata/yaml/nested.yaml")
	c.Assert(err, check.IsNil)

	wrongWorkflowYaml, err := utils.ReadFromFile("testdata/yaml/bad_workflow.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	tool.AddActiveRepos([]string{"bitbucket.org"}, "id")

	payload := map[string]interface{}{
		"Revision": "revision" + test.RandStr(),
		"Content":  []string{string(containerYaml), string(workflowYaml), string(nestedYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, "bitbucket.org$$$$master", "update", payload)

	time.Sleep(6 * time.Second)

	// look for templates that are compatible with commits, we should find workflow, checkout, but not axdb_build
	var tempResult TemplateListResult
	axErr := axopsClient.Get("templates?repo=bitbucket.org&branch=master", nil, &tempResult)
	checkError(c, axErr)
	c.Assert(axErr, check.IsNil)

	found := 0
	expected := map[string]int{
		"nested":                               1,
		"checkout_axdbbuild":                   1,
		"buildbuildaxdb":                       1,
		"axcheckout":                           1,
		"axbuild2":                             1,
		"mongodb":                              1,
		"test against db":                      1,
		"checkout build and test with fixture": 1,
	}
	for _, temp := range tempResult.Data {
		if _, ok := expected[temp.Name]; ok {
			found++
		} else {
			c.Logf("template -%s- is not supposed to be there", temp.Name)
			fail(c)
		}
	}
	if found != 8 {
		c.Logf("Expecting to find 8 templates, found only %d", found)
		fail(c)
	}

	// Now post again, this time with one extra workflow that doesn't work
	payload = map[string]interface{}{
		"Revision": "revision" + test.RandStr(),
		"Content":  []string{string(containerYaml), string(workflowYaml), string(nestedYaml), string(wrongWorkflowYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, "bitbucket.org$$$$master", "update", payload)

	time.Sleep(6 * time.Second)
	// verify again, we should still just find 8. The wrong one would be discarded.
	tempResult = TemplateListResult{}
	axErr = axopsClient.Get("templates?repo=bitbucket.org&branch=master", nil, &tempResult)
	checkError(c, axErr)

	found = 0
	for _, temp := range tempResult.Data {
		if _, ok := expected[temp.Name]; ok {
			found++
		} else {
			c.Logf("template -%s- is not supposed to be there", temp.Name)
			fail(c)
		}
	}

	c.Check(found, check.Equals, 8)

	// Post again, this time without the nested workflow and with the bad yaml. It should still take effect
	// with 7 overall number of templates.
	payload = map[string]interface{}{
		"Revision": "revision" + test.RandStr(),
		"Content":  []string{string(containerYaml), string(workflowYaml), string(wrongWorkflowYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, "bitbucket.org$$$$master", "update", payload)

	time.Sleep(6 * time.Second)
	// verify again, we should still just find 8. The wrong one would be discarded.
	tempResult = TemplateListResult{}
	axErr = axopsClient.Get("templates?repo=bitbucket.org&branch=master", nil, &tempResult)
	checkError(c, axErr)

	found = 0
	for _, temp := range tempResult.Data {
		if _, ok := expected[temp.Name]; ok {
			found++
		} else {
			c.Logf("template -%s- is not supposed to be there", temp.Name)
			fail(c)
		}
	}
	c.Check(found, check.Equals, 7)
}

func (s *S) TestPolicyYAML(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	containerPolicyYaml, err := utils.ReadFromFile("testdata/yaml/policy/container_policy.yaml")
	c.Assert(err, check.IsNil)

	workflowPolicyYaml, err := utils.ReadFromFile("testdata/yaml/policy/workflow_policy.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(containerPolicyYaml), string(workflowPolicyYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)
	policies, err := policy.GetPolicies(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(policies), check.Equals, 2)

	payload = map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(containerPolicyYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)
	policies, err = policy.GetPolicies(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(policies), check.Equals, 1)

	payload = map[string]interface{}{
		"Revision": revision,
		"Content":  []string{},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)
	time.Sleep(6 * time.Second)
	policies, err = policy.GetPolicies(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(policies), check.Equals, 0)
}

func (s *S) TestPolicyBadYAML(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	badPolicyYaml, err := utils.ReadFromFile("testdata/yaml/policy/bad_policy.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(badPolicyYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)
	policies, err := policy.GetPolicies(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(policies), check.Equals, 1)
}

func clearProjects(c *check.C, repo string, branch string) {
	payload := map[string]interface{}{
		"Revision": "revision" + test.RandStr(),
		"Content":  []string{},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)
	time.Sleep(6 * time.Second)
	projects, err := project.GetProjects(map[string]interface{}{
		project.ProjectRepo:   repo,
		project.ProjectBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(projects), check.Equals, 0)

}

func (s *S) TestProjectYAML(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	clearProjects(c, repo, branch)

	containerProjectYaml, err := utils.ReadFromFile("testdata/yaml/project/container_project.yaml")
	c.Assert(err, check.IsNil)

	workflowProjectYaml, err := utils.ReadFromFile("testdata/yaml/project/workflow_project.yaml")
	c.Assert(err, check.IsNil)

	c.Logf("containerProjectYAML:%v", containerProjectYaml)
	c.Logf("workflowProjectYAML:%v", workflowProjectYaml)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(containerProjectYaml), string(workflowProjectYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)
	projects, err := project.GetProjects(map[string]interface{}{
		project.ProjectRepo:   repo,
		project.ProjectBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(projects), check.Equals, 2)

	// assert 1 app is published
	projects, err = project.GetProjects(map[string]interface{}{
		project.ProjectRepo:      repo,
		project.ProjectBranch:    branch,
		project.ProjectPublished: true,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(projects), check.Equals, 1)

	payload = map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(containerProjectYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)
	time.Sleep(6 * time.Second)
	projects, err = project.GetProjects(map[string]interface{}{
		project.ProjectRepo:   repo,
		project.ProjectBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(projects), check.Equals, 1)

	clearProjects(c, repo, branch)

}

func (s *S) TestProjectBadYAML(c *check.C) {
	c.Log("TestProjectBadYAML")
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	clearProjects(c, repo, branch)

	badPolicyYaml, err := utils.ReadFromFile("testdata/yaml/project/bad_project.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(badPolicyYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)
	projects, err := project.GetProjects(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(projects), check.Equals, 1)

	clearProjects(c, repo, branch)

}

func (s *S) TestFixtureBadYAML(c *check.C) {
	c.Log("TestFixtureBadYAML")
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	badFixtureYaml, err := utils.ReadFromFile("testdata/yaml/fixture/bad_fixture.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(badFixtureYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)
	time.Sleep(6 * time.Second)
	fixtures, err := fixture.GetTemplates(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(fixtures), check.Equals, 1)
}

func (s *S) TestYAMLsPost(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	containerPolicyYaml, err := utils.ReadFromFile("testdata/yaml/policy/container_policy.yaml")
	c.Assert(err, check.IsNil)

	workflowPolicyYaml, err := utils.ReadFromFile("testdata/yaml/policy/workflow_policy.yaml")
	c.Assert(err, check.IsNil)

	yamls := axops.YAMLs{
		Repo:     repo,
		Branch:   branch,
		Revision: revision,
		Files: []axops.File{
			axops.File{
				Path:    "testdata/yaml/policy/container_policy.yaml",
				Content: containerPolicyYaml,
			},
			axops.File{
				Path:    "testdata/yaml/policy/workflow_policy.yaml",
				Content: workflowPolicyYaml,
			},
		},
	}

	_, err = axopsClient.Post("yamls", yamls)
	c.Assert(err, check.IsNil)

	policies, err := policy.GetPolicies(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(policies), check.Equals, 2)

	yamls = axops.YAMLs{
		Repo:     repo,
		Branch:   branch,
		Revision: revision,
		Files: []axops.File{
			axops.File{
				Path:    "testdata/yaml/policy/container_policy.yaml",
				Content: containerPolicyYaml,
			},
		},
	}

	_, err = axopsClient.Post("yamls", yamls)
	c.Assert(err, check.IsNil)

	policies, err = policy.GetPolicies(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(policies), check.Equals, 1)

	yamls = axops.YAMLs{
		Repo:     repo,
		Branch:   branch,
		Revision: revision,
		Files:    []axops.File{},
	}

	_, err = axopsClient.Post("yamls", yamls)
	c.Assert(err, check.IsNil)

	policies, err = policy.GetPolicies(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(policies), check.Equals, 0)
}

func (s *S) TestYAMLLabel(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	workflowPolicyYaml, err := utils.ReadFromFile("testdata/yaml/policy/workflow_policy.yaml")
	c.Assert(err, check.IsNil)

	containerPolicyYaml, err := utils.ReadFromFile("testdata/yaml/policy/container_policy.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(containerPolicyYaml), string(workflowPolicyYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)
	policies, err := policy.GetPolicies(map[string]interface{}{
		policy.PolicyRepo:   repo,
		policy.PolicyBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(policies), check.Equals, 2)
	c.Assert(len(policies[0].Labels), check.Equals, 3)
	c.Assert(len(policies[1].Labels), check.Equals, 3)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 4)
	c.Assert(len(templates[0].Labels), check.Equals, 3)
	c.Assert(len(templates[1].Labels), check.Equals, 3)
	c.Assert(len(templates[2].Labels), check.Equals, 3)
	c.Assert(len(templates[3].Labels), check.Equals, 3)

	labels, err := label.GetLabels(map[string]interface{}{
		label.LabelType: label.LabelTypeService,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(labels) >= 3, check.Equals, true)

	labels, err = label.GetLabels(map[string]interface{}{
		label.LabelType: label.LabelTypePolicy,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(labels) >= 3, check.Equals, true)
}

func (s *S) TestYAMLGood(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_good.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 4)
}

func (s *S) TestYAMLBadMissingInputParams(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_miss_input_param1.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)

	repo = "repo" + test.RandStr()
	branch = "branch" + test.RandStr()

	yaml, err = utils.ReadFromFile("testdata/yaml/template/template_bad_miss_input_param2.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload = map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)

	templates, err = service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)
}

func (s *S) TestYAMLBadParamFormat(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_wrong_quotes.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)
}

func (s *S) TestYAMLSecretInputGood(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_good_secret_input1.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 6)

	repo = "repo" + test.RandStr()
	branch = "branch" + test.RandStr()
	revision = "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err = utils.ReadFromFile("testdata/yaml/template/template_good_secret_input2.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload = map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err = service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 4)

}

func (s *S) TestYAMLArtifactOputputGood(c *check.C) {

	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_good_artifact_output.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 3)
}

func (s *S) TestYAMLArtifactOputputBadFormat(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_miss_artifact_output.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}

	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 3)
}

func (s *S) TestYAMLBADMissingLabelValue(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_miss_label_value.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)
}

func (s *S) TestYAMLBadResource(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_container_wrong_resource.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 1)
}

func (s *S) TestYAMLBadDeploymentLoop(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_deployment_loop.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 4)
}

func (s *S) TestYAMLBadWorkflowLoop(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_workflow_loop.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 2)
}

func (s *S) TestYAMLGoodDeploymentWithVolume(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_good_deployment_with_vol.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 4)
}

func (s *S) TestYAMLGoodDeploymentWithBadNamedVolume(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_deployment_with_bad_named_vol.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)
}

func (s *S) TestYAMLGoodDeploymentWithBadAnonymousVolume(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_deployment_with_bad_anonymous_vol.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)
}

func (s *S) TestYAMLGoodDeploymentWithBadMount(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_deployment_with_bad_mount.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)
}

func (s *S) TestYAMLGoodDeploymentWithBadReference(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_deployment_with_bad_vol_reference.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)
}

func (s *S) TestYAMLBadDeploymentWithBadArtReference(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_deployment_with_bad_art_reference.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)
}

func (s *S) TestBadYAMLWithTabs(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	badYaml, err := utils.ReadFromFile("testdata/yaml/yamlwithtabs.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(badYaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(6 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 0)
}

func (s *S) TestYAMLBadContainerArgsCommands(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_container_args_command_commands.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 2)

	for _, t := range templates {
		c.Assert(strings.Contains(t.Name, "mongodb-good"), check.Equals, true)
	}
}

func (s *S) TestYAMLBadContainerEnvVariables(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_env_variables.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 1)

	for _, t := range templates {
		c.Assert(strings.Contains(t.Name, "mongodb-good-env"), check.Equals, true)
	}
}

func (s *S) TestYAMLBadDeploymentWithBadVolRefName(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_bad_deployment_with_bad_ref_name.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})

	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), 4)
}

func WorkFlowWithVolumesHelper(c *check.C, testFileName string, numTemplates int) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile(testFileName)
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})

	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Not(check.Equals), numTemplates)
}

func (s *S) TestYAMLWorkflowSingleStepWithVolume(c *check.C) {
	WorkFlowWithVolumesHelper(c, "testdata/yaml/template/template_volumes_with_workflows.yaml", 1)
}

func (s *S) TestYAMLWorkflowSingleStepWithVolumeBad(c *check.C) {
	WorkFlowWithVolumesHelper(c, "testdata/yaml/template/template_volumes_with_workflows_bad.yaml", 0)
}

func (s *S) TestYAMLGoodDeploymentWithInlinedContainer(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_good_deployment_inlined_container.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 1)
}

func (s *S) TestYAMLDeploymentWithVisibility(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_deployment_with_visibility.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 3)

	for _, t := range templates {
		c.Assert(strings.Contains(t.Name, "good-visibility"), check.Equals, true)
	}
}

func (s *S) TestYAMLDeploymentWithUpdateStrategy(c *check.C) {
	repo := "repo" + test.RandStr()
	branch := "branch" + test.RandStr()
	revision := "revision" + test.RandStr()

	tool.AddActiveRepos([]string{repo}, "id")

	yaml, err := utils.ReadFromFile("testdata/yaml/template/template_deployment_with_update_strategy.yaml")
	c.Assert(err, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTemplates, event.GetDevopsTemplateHandler())

	payload := map[string]interface{}{
		"Revision": revision,
		"Content":  []string{string(yaml)},
	}
	PostOneEvent(c, event.TopicDevopsTemplates, repo+"$$$$"+branch, "update", payload)

	time.Sleep(3 * time.Second)

	templates, err := service.GetTemplates(map[string]interface{}{
		service.TemplateRepo:   repo,
		service.TemplateBranch: branch,
	})
	c.Assert(err, check.IsNil)
	c.Assert(len(templates), check.Equals, 3)
}
*/
