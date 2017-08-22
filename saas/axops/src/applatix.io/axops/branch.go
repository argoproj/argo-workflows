// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Branch API [/branches]
// @SubApi Repo API [/repos]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/branch"
	"applatix.io/axops/commit"
	"applatix.io/axops/tool"
	"github.com/gin-gonic/gin"
	"net/http"
)

type BranchesData struct {
	Data []branch.Branch `json:"data"`
}

// @Title GetBranches
// @Description List branches
// @Accept  json
// @Param   name  	 query   string     false       "Name."
// @Param   repo	 query   string     false       "Repo."
// @Param   search	 query   string     false       "Search."
// @Success 200 {object} BranchesData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /branches
// @Router /branches [GET]
func GetBranchList() gin.HandlerFunc {
	return func(c *gin.Context) {

		if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && commit.GetETag() == etag {
			c.Status(http.StatusNotModified)
			return
		}

		params, axErr := getContextRawParams(c)

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		branches, axErr := branch.GetBranches(params)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		}

		resultMap := BranchesData{
			Data: branches,
		}

		c.Header("ETag", commit.GetETag())
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}

//func GatewaySCMBranchesList() gin.HandlerFunc {
//	branchesUrl := DevopsCl.GetRootUrl() + "/scm/branches"
//	url, err := url.Parse(branchesUrl)
//	if err != nil {
//		panic(fmt.Sprintf("Can not parse the gateway branches url: %v", branchesUrl))
//	}
//	fmt.Println(branchesUrl)
//	return gin.WrapH(NewSingleHostReverseProxy(url))
//}

type ReposData struct {
	Data []string `json:"data"`
}

// @Title ListRepos
// @Description List repos
// @Accept  json
// @Success 200 {object} ReposData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /repos
// @Router /repos [GET]
func GetRepoList() gin.HandlerFunc {
	return func(c *gin.Context) {

		if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && commit.GetETag() == etag {
			c.Status(http.StatusNotModified)
			return
		}

		tools, axErr := tool.GetToolsByCategory(tool.CategorySCM)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		repos := []string{}
		for _, t := range tools {
			tt, ok := t.(tool.Tool)
			if ok {
				switch tt.GetType() {
				case tool.TypeGIT, tool.TypeGitHub, tool.TypeGitLab, tool.TypeBitBucket, tool.TypeCodeCommit:
					if tt.GetType() == tool.TypeGIT {
						repos = append(repos, tt.(*tool.GitConfig).Repos...)
					}
					if tt.GetType() == tool.TypeGitHub {
						repos = append(repos, tt.(*tool.GitHubConfig).Repos...)
					}
					if tt.GetType() == tool.TypeGitLab {
						repos = append(repos, tt.(*tool.GitLabConfig).Repos...)
					}
					if tt.GetType() == tool.TypeBitBucket {
						repos = append(repos, tt.(*tool.BitbucketConfig).Repos...)
					}
					if tt.GetType() == tool.TypeCodeCommit {
						repos = append(repos, tt.(*tool.CodeCommitConfig).Repos...)
					}
				}
			}
		}

		resultMap := ReposData{
			Data: repos,
		}

		c.Header("ETag", commit.GetETag())
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}
