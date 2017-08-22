// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"fmt"
)

type GitLabConfig struct {
	*GitHubConfig
}

func (t *GitLabConfig) pre() (*axerror.AXError, int) {
	if t.URL == "" {
		t.URL = "https://gitlab.com"
	}

	if t.UseWebhook == nil {
		useWebhook := false
		t.UseWebhook = &useWebhook
	}

	t.Protocol = "https"

	return t.GitHubConfig.pre()
}

func (t *GitLabConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategorySCM {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.Username == "" {
		return ErrToolMissingUsername, axerror.REST_BAD_REQ
	}

	if t.Password == "" {
		return ErrToolMissingPassword, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeGitLab)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*GitLabConfig).Username == t.Username && oldTool.(*GitLabConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The username(%v) has been used by another configuration.", t.Username)), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *GitLabConfig) Post(old, new interface{}) (*axerror.AXError, int) {

	if old == nil {
		return nil, axerror.REST_STATUS_OK
	}

	oldRepos := make(map[string]bool)
	newRepos := make(map[string]bool)

	repos := old.(*GitLabConfig).Repos
	for _, repo := range repos {
		oldRepos[repo] = true
	}

	if new != nil {
		repos = new.(*GitLabConfig).Repos
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

	if old.(*GitLabConfig).UseWebhook != nil && *(old.(*GitLabConfig).UseWebhook) == true && t.UseWebhook != nil && *(t.UseWebhook) == false {
		config := scmConfig{
			Category:   t.GetCategory(),
			Type:       t.GetType(),
			Protocol:   t.Protocol,
			Username:   t.Username,
			Password:   t.Password,
			UseWebhook: t.UseWebhook,
		}

		for _, repo := range t.Repos {
			config.Repo = repo
			if axErr, _ := utils.DevopsCl.Delete2("scm/webhooks", nil, config, nil); axErr != nil {
				utils.ErrorLog.Println("Delete webhook for repo ", repo, " failed:", axErr)
			}
		}
	}

	return nil, axerror.REST_STATUS_OK
}
