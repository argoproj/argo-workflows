// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

import (
	"applatix.io/axerror"
	"applatix.io/axops/tool"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestToolSMTP(c *check.C) {
	var tools GeneralGetResult

	// Create SMTP
	smtp := map[string]interface{}{
		"nickname":      "Default",
		"url":           "smtp.example.com" + test.RandStr(),
		"category":      "notification",
		"type":          "smtp",
		"admin_address": "admin@example.com",
		"port":          587,
		"timeout":       30,
		"use_tls":       true,
		"username":      "username@example.com",
		"password":      "123",
	}

	smtp, axErr := axopsClient.Post("tools", smtp)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(smtp["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", smtp)
	c.Assert(axErr, check.NotNil)

	// Update SMTP
	id := smtp["id"].(string)
	smtp["username"] = "username@example.com"
	smtp["port"] = 111

	smtp, axErr = axopsClient.Put("tools"+"/"+id, smtp)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &smtp)
	c.Assert(axErr, check.IsNil)
	c.Assert(smtp["id"].(string), check.Equals, id)
	c.Assert(int(smtp["port"].(float64)), check.Equals, 111)
	c.Assert(smtp["username"].(string), check.Equals, "username@example.com")

	// Delete STMP
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &smtp)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolGit(c *check.C) {
	var tools GeneralGetResult

	// Create Git
	git := map[string]interface{}{
		"category": "scm",
		"type":     "git",
		"url":      "https://example.com/example" + test.RandStr() + ".git",
		"username": "username@example.com",
		"password": "123",
	}

	git, axErr := axopsClient.Post("tools", git)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(git["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", git)
	c.Assert(axErr, check.NotNil)

	// Update Git
	id := git["id"].(string)
	git["username"] = "username@example.com"
	git["password"] = "123456"

	git, axErr = axopsClient.Put("tools"+"/"+id, git)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.IsNil)
	c.Assert(git["id"].(string), check.Equals, id)
	c.Assert(git["username"].(string), check.Equals, "username@example.com")
	c.Assert(git["password"].(string), check.Equals, "123456")

	// Delete Git
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolPublicGit(c *check.C) {
	var tools GeneralGetResult

	// Create Git
	git := map[string]interface{}{
		"category": "scm",
		"type":     "git",
		"url":      "https://example.com/example" + test.RandStr() + ".git",
	}

	git, axErr := axopsClient.Post("tools", git)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(git["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", git)
	c.Assert(axErr, check.NotNil)

	id := git["id"].(string)

	// Delete Git
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolGitHub(c *check.C) {
	var tools GeneralGetResult

	// Create GitHub
	git := map[string]interface{}{
		"category": "scm",
		"type":     "github",
		"username": "username@example.com" + test.RandStr(),
		"password": "123",
	}

	git, axErr := axopsClient.Post("tools", git)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(git["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", git)
	c.Assert(axErr, check.NotNil)

	// Update GitHub
	id := git["id"].(string)
	git["password"] = "123456"

	git, axErr = axopsClient.Put("tools"+"/"+id, git)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.IsNil)
	c.Assert(git["id"].(string), check.Equals, id)
	c.Assert(git["password"].(string), check.Equals, "123456")

	// Delete GitHub
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolBitbucket(c *check.C) {
	var tools GeneralGetResult

	// Create Bitbucket
	git := map[string]interface{}{
		"category": "scm",
		"type":     "bitbucket",
		"username": "username@example.com" + test.RandStr(),
		"password": "123",
	}

	git, axErr := axopsClient.Post("tools", git)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(git["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", git)
	c.Assert(axErr, check.NotNil)

	// Update Bitbucket
	id := git["id"].(string)
	git["password"] = "123456"

	git, axErr = axopsClient.Put("tools"+"/"+id, git)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.IsNil)
	c.Assert(git["id"].(string), check.Equals, id)
	c.Assert(git["password"].(string), check.Equals, "123456")

	// Delete Bitbucket
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolGitLab(c *check.C) {
	var tools GeneralGetResult

	// Create GitLab
	git := map[string]interface{}{
		"category": "scm",
		"type":     "gitlab",
		"username": "username@example.com" + test.RandStr(),
		"password": "123",
	}

	git, axErr := axopsClient.Post("tools", git)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(git["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", git)
	c.Assert(axErr, check.NotNil)

	// Update GitLab
	id := git["id"].(string)
	git["password"] = "123456"

	git, axErr = axopsClient.Put("tools"+"/"+id, git)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.IsNil)
	c.Assert(git["id"].(string), check.Equals, id)
	c.Assert(git["password"].(string), check.Equals, "123456")

	// Delete GitLab
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolCodeCommit(c *check.C) {
	var tools GeneralGetResult

	// Create CodeCommit
	git := map[string]interface{}{
		"category": "scm",
		"type":     "codecommit",
		"username": "username@example.com" + test.RandStr(),
		"password": "123",
	}

	git, axErr := axopsClient.Post("tools", git)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(git["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", git)
	c.Assert(axErr, check.NotNil)

	// Update CodeCommit
	id := git["id"].(string)
	git["password"] = "123456"

	git, axErr = axopsClient.Put("tools"+"/"+id, git)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.IsNil)
	c.Assert(git["id"].(string), check.Equals, id)
	c.Assert(git["password"].(string), check.Equals, "123456")

	// Delete CodeCommit
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &git)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolDockerHub(c *check.C) {
	var tools GeneralGetResult
	var axErr *axerror.AXError

	// Create DockerHub
	dockerhub := map[string]interface{}{
		"type":     "dockerhub",
		"username": "username@example.com" + test.RandStr(),
		"password": "123",
	}

	dockerhub, axErr = axopsClient.Post("tools", dockerhub)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(dockerhub["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", dockerhub)
	c.Assert(axErr, check.NotNil)

	// Update DockerHub
	id := dockerhub["id"].(string)
	dockerhub["password"] = "123456"

	dockerhub, axErr = axopsClient.Put("tools"+"/"+id, dockerhub)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &dockerhub)
	c.Assert(axErr, check.IsNil)
	c.Assert(dockerhub["id"].(string), check.Equals, id)
	c.Assert(dockerhub["password"].(string), check.Equals, "123456")

	// Delete DockerHub
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &dockerhub)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolPrivateRegistry(c *check.C) {
	var tools GeneralGetResult
	var axErr *axerror.AXError

	// Create Private Registry
	registry := map[string]interface{}{
		"type":     "private_registry",
		"hostname": "example.com",
		"username": "username@example.com" + test.RandStr(),
		"password": "123",
	}

	registry, axErr = axopsClient.Post("tools", registry)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(registry["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", registry)
	c.Assert(axErr, check.NotNil)

	// Update Private Registry
	id := registry["id"].(string)
	registry["password"] = "123456"

	registry, axErr = axopsClient.Put("tools"+"/"+id, registry)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &registry)
	c.Assert(axErr, check.IsNil)
	c.Assert(registry["id"].(string), check.Equals, id)
	c.Assert(registry["password"].(string), check.Equals, "123456")

	// Delete Private Registry
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &registry)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolDomain(c *check.C) {
	var tools GeneralGetResult
	var axErr *axerror.AXError

	domain := map[string]interface{}{
		"type": tool.TypeRoute53,
	}

	domain, axErr = axopsClient.Post("tools", domain)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(domain["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", domain)
	c.Assert(axErr, check.NotNil)

	id := domain["id"].(string)
	domain["domains"] = []interface{}{
		map[string]string{
			"name": "a.b.c.",
		},
		map[string]string{
			"name": "b.c.",
		},
	}

	domain, axErr = axopsClient.Put("tools"+"/"+id, domain)
	c.Assert(axErr, check.NotNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &domain)
	c.Assert(axErr, check.IsNil)

	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &domain)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolSplunk(c *check.C) {
	var tools GeneralGetResult

	splunk := map[string]interface{}{
		"token":    "12345",
		"url":      "splunk-" + test.RandStr(),
		"category": "notification",
		"type":     "splunk",
	}

	splunk, axErr := axopsClient.Post("tools", splunk)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(splunk["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", splunk)
	c.Assert(axErr, check.NotNil)

	// Update Splunk
	id := splunk["id"].(string)
	splunk["token"] = "6789"

	splunk, axErr = axopsClient.Put("tools"+"/"+id, splunk)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &splunk)
	c.Assert(axErr, check.IsNil)
	c.Assert(splunk["id"].(string), check.Equals, id)
	c.Assert(splunk["token"].(string), check.Equals, "6789")

	// Delete Splunk
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &splunk)
	c.Assert(axErr, check.NotNil)
}

func (s *S) TestToolSlack(c *check.C) {
	var tools GeneralGetResult

	slack := map[string]interface{}{
		"oauth_token": "12345",
		"url":         "slack-" + test.RandStr(),
		"category":    "notification",
		"type":        "slack",
	}

	slack, axErr := axopsClient.Post("tools", slack)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(slack["id"].(string)), check.Not(check.Equals), 0)

	axErr = axopsClient.Get("tools", nil, &tools)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(tools.Data), check.Not(check.Equals), 0)

	// Create duplicate
	_, axErr = axopsClient.Post("tools", slack)
	c.Assert(axErr, check.NotNil)

	// Update Slack
	id := slack["id"].(string)
	slack["oauth_token"] = "6789"

	slack, axErr = axopsClient.Put("tools"+"/"+id, slack)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &slack)
	c.Assert(axErr, check.IsNil)
	c.Assert(slack["id"].(string), check.Equals, id)
	c.Assert(slack["oauth_token"].(string), check.Equals, "6789")

	// Delete Slack
	_, axErr = axopsClient.Delete("tools"+"/"+id, nil)
	c.Assert(axErr, check.IsNil)

	axErr = axopsClient.Get("tools"+"/"+id, nil, &slack)
	c.Assert(axErr, check.NotNil)
}
