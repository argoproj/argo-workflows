// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"fmt"
	"sort"
)

var (
	ErrToolMissingUsername     = axerror.ERR_API_INVALID_PARAM.NewWithMessage("username is required.")
	ErrToolMissingPassword     = axerror.ERR_API_INVALID_PARAM.NewWithMessage("password or access key is required.")
	ErrInvalidDomainNameFormat = axerror.ERR_API_INVALID_PARAM.NewWithMessage("the name doesn't comply with FDQN format.")
	ErrToolMissingName         = axerror.ERR_API_INVALID_PARAM.NewWithMessage("the name is requried.")
)

type GitModel struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	Category   string   `json:"category"`
	Type       string   `json:"type"`
	Password   string   `json:"password"`
	Username   string   `json:"username"`
	Protocol   string   `json:"protocol"`
	AllRepos   []string `json:"all_repos"`
	Repos      []string `json:"repos"`
	UseWebhook *bool    `json:"use_webhook"`
}

type GitHubConfig struct {
	*ToolBase
	Username   string   `json:"username,omitempty"`
	Protocol   string   `json:"protocol,omitemtpy"`
	AllRepos   []string `json:"all_repos,omitempty"`
	Repos      []string `json:"repos,omitempty"`
	UseWebhook *bool    `json:"use_webhook,omitempty"`
}

func (t *GitHubConfig) Omit() {
	t.Password = ""
}

func (t *GitHubConfig) Test() (*axerror.AXError, int) {
	return utils.DevopsCl.Post2("scm/test", nil, t, nil)
}

func (t *GitHubConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategorySCM {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.Username == "" {
		return ErrToolMissingUsername, axerror.REST_BAD_REQ
	}

	if t.Password == "" {
		return ErrToolMissingPassword, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeGitHub)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*GitHubConfig).Username == t.Username && oldTool.(*GitHubConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The username(%v) has been used by another configuration.", t.Username)), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *GitHubConfig) pre() (*axerror.AXError, int) {

	if t.URL == "" {
		t.URL = "https://api.github.com"
	}

	if t.UseWebhook == nil {
		useWebhook := false
		t.UseWebhook = &useWebhook
	}

	t.Category = CategorySCM

	if t.Protocol == "" {
		t.Protocol = "https"
	}

	results := map[string]interface{}{}
	axErr, code := utils.DevopsCl.Post2("scm/test", nil, t, &results)
	if axErr != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to connect to the SCM: %v", axErr)), code
	}

	allRepoStrs := []string{}
	allRepoMap := make(map[string]string)
	// TODO: This twisted payload needs to be changed
	if repos, ok := results["repos"]; ok {
		repoMap := repos.(map[string]interface{})
		for _, v := range repoMap {
			allRepoStrs = append(allRepoStrs, v.(string))
			allRepoMap[v.(string)] = v.(string)
		}
		sort.Strings(allRepoStrs)
		t.AllRepos = allRepoStrs
	}

	activeRepoMap := make(map[string]string)
	for _, repo := range t.Repos {
		if _, ok := allRepoMap[repo]; ok {
			activeRepoMap[repo] = repo
		}
	}

	activeRepos := []string{}
	for k, _ := range activeRepoMap {
		activeRepos = append(activeRepos, k)
	}

	sort.Strings(activeRepos)

	for _, repo := range activeRepos {
		if toolId, ok := ActiveRepos[repo]; ok {
			if toolId != t.GetID() {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The repository(%v) has been connected via another configuration. Connectting via two configurations is not allowed.", repo)), axerror.REST_BAD_REQ
			}
		}
	}

	t.Repos = activeRepos

	return nil, axerror.REST_STATUS_OK
}

func (t *GitHubConfig) PushUpdate() (*axerror.AXError, int) {

	AddActiveRepos(t.Repos, t.GetID())

	if t.UseWebhook != nil && *(t.UseWebhook) == true {
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
			//if axErr, code := utils.DevopsCl.Post2("scm/pull_commit", nil, config, nil); axErr != nil {
			//	return axErr, code
			//}
			if axErr, code := utils.DevopsCl.Post2("scm/webhooks", nil, config, nil); axErr != nil {
				return axErr, code
			}
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *GitHubConfig) pushDelete() (*axerror.AXError, int) {

	if t.UseWebhook != nil && *t.UseWebhook == true {
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
			if axErr, code := utils.DevopsCl.Delete2("scm/webhooks", nil, config, nil); axErr != nil {
				return axErr, code
			}
		}
	}

	DeleteActiveRepos(t.Repos)

	return nil, axerror.REST_STATUS_OK
}

type scmConfig struct {
	Category   string `json:"category,omitempty"`
	Type       string `json:"type,omitempty"`
	Repo       string `json:"repo,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	UseWebhook *bool  `json:"use_webhook,omitempty"`
}

func (t *GitHubConfig) Post(old, new interface{}) (*axerror.AXError, int) {

	if old == nil {
		return nil, axerror.REST_STATUS_OK
	}

	oldRepos := make(map[string]bool)
	newRepos := make(map[string]bool)

	repos := old.(*GitHubConfig).Repos
	for _, repo := range repos {
		oldRepos[repo] = true
	}

	if new != nil {
		repos = new.(*GitHubConfig).Repos
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

	if old.(*GitHubConfig).UseWebhook != nil && *(old.(*GitHubConfig).UseWebhook) == true && t.UseWebhook != nil && *(t.UseWebhook) == false {
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
