// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"fmt"
)

type CodeCommitConfig struct {
	*GitHubConfig
}

func (t *CodeCommitConfig) pre() (*axerror.AXError, int) {
	if t.URL == "" {
		t.URL = "https://codecommit.us-east-1.amazonaws.com"
	}

	useWebhook := false
	t.UseWebhook = &useWebhook
	t.Protocol = "https"

	return t.GitHubConfig.pre()
}

func (t *CodeCommitConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategorySCM {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.Username == "" {
		return ErrToolMissingUsername, axerror.REST_BAD_REQ
	}

	if t.Password == "" {
		return ErrToolMissingPassword, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeCodeCommit)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*CodeCommitConfig).Username == t.Username && oldTool.(*CodeCommitConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The username(%v) has been used by another configuration.", t.Username)), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *CodeCommitConfig) Post(old, new interface{}) (*axerror.AXError, int) {

	if old == nil {
		return nil, axerror.REST_STATUS_OK
	}

	oldRepos := make(map[string]bool)
	newRepos := make(map[string]bool)

	repos := old.(*CodeCommitConfig).Repos
	for _, repo := range repos {
		oldRepos[repo] = true
	}

	if new != nil {
		repos = new.(*CodeCommitConfig).Repos
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
