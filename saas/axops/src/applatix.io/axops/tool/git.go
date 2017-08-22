// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"fmt"
	"strings"
)

var (
	ErrToolMissingRepo = axerror.ERR_API_INVALID_PARAM.NewWithMessage("repo is required.")
)

type GitConfig struct {
	*GitHubConfig
}

func (t *GitConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategorySCM {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if len(t.Repos) == 0 {
		return ErrToolMissingRepo, axerror.REST_BAD_REQ
	}

	repo := strings.TrimSpace(t.Repos[0])

	if strings.HasPrefix(repo, "git@") || strings.HasPrefix(repo, "ssh") {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("SSH protocol is not supported."), axerror.REST_BAD_REQ
	}

	if !(strings.HasPrefix(repo, "https") || strings.HasPrefix(repo, "http")) {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Please add the protocol to the repo. For example, https://git.company.com/example.git."), axerror.REST_BAD_REQ
	}

	if strings.Contains(repo, "@") {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Please remove the credential information in the repository address."), axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeGIT)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*GitConfig).Repos[0] == repo && oldTool.(*GitConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The repo(%v) has been used by another configuration.", repo)), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *GitConfig) pre() (*axerror.AXError, int) {

	if len(t.Repos) != 0 {
		t.URL = strings.TrimSpace(t.Repos[0])
	}

	if t.URL != "" && len(t.Repos) == 0 {
		t.Repos = append(t.Repos, t.URL)
	}

	t.Category = CategorySCM

	repo := strings.ToLower(t.URL)
	if strings.HasPrefix(repo, "https:") {
		t.Protocol = "https"
	} else if strings.HasPrefix(repo, "http:") {
		t.Protocol = "http"
	} else {
		t.Protocol = "ssh"
	}

	// Remove the credentials requirement to support public repos
	if t.Username == "" {
		t.Password = ""
	}

	if t.Password == "" {
		t.Username = ""
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *GitConfig) Post(old, new interface{}) (*axerror.AXError, int) {

	if old == nil {
		return nil, axerror.REST_STATUS_OK
	}

	oldRepos := make(map[string]bool)
	newRepos := make(map[string]bool)

	repos := old.(*GitConfig).Repos
	for _, repo := range repos {
		oldRepos[repo] = true
	}

	if new != nil {
		repos = new.(*GitConfig).Repos
		for _, repo := range repos {
			newRepos[repo] = true
		}
	}

	for repo, _ := range oldRepos {
		if _, ok := newRepos[repo]; !ok {
			if err, code := PurgeCachedBranchHeads(repo); err != nil {
				return err, code
			}
			if err, code := DeleteDataByRepo(repo); err != nil {
				return err, code
			}
		}
	}

	return nil, axerror.REST_STATUS_OK
}
