// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Commit API [/commits]
package axops

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/commit"
	"applatix.io/axops/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type CommitsData struct {
	Data []commit.ApiCommit `json:"data"`
}

// @Title GetCommits
// @Description List commits
// @Accept  json
// @Param   repo	 query   string     false       "Repo."
// @Param   revision	 query   string     false       "Revision."
// @Param   branch	 query   string     false       "Branch."
// @Param   repo_branch	 query   string     false       "Repo_Branch, eg:[{"repo":"https://bitbucket.org/example.git","branch":"master"},{"repo":"https://bitbucket.org/example.git","branch":"test"}] (encode needed)"
// @Param   committer	 query   string     false       "Committer."
// @Param   author	 query   string     false       "Author."
// @Param   description	 query   string     false       "Description."
// @Param   min_time	 query   int 	    false       "Min time."
// @Param   max_time	 query   int 	    false       "Max time."
// @Param   limit	 query   int 	    false       "Limit."
// @Param   search	 query   string     false       "Search."
// @Success 200 {object} CommitsData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /commits
// @Router /commits [GET]
func GetCommitList() gin.HandlerFunc {
	return func(c *gin.Context) {

		if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && commit.GetETag() == etag {
			c.Status(http.StatusNotModified)
			return
		}

		utils.DebugLog.Println("Getting Parameters")

		//params, axErr := getContextParams(c,
		//	[]string{commit.CommitRepo, commit.CommitRevision, commit.CommitBranch, commit.CommitCommitter, commit.CommitAuthor, commit.CommitDescription},
		//	[]string{},
		//	[]string{},
		//	[]string{})

		params, axErr := getContextRawParams(c)

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		//params, axErr = getContextTimeParams(c, params)
		//
		//if axErr != nil {
		//	c.JSON(axerror.REST_BAD_REQ, axErr)
		//	return
		//}

		commits, dbErr := commit.GetAPICommits(params)
		if dbErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, dbErr)
			return
		}

		utils.DebugLog.Println("Adapting API commits")

		branch := c.Request.URL.Query().Get(commit.CommitBranch)
		if branch != "" && !strings.HasPrefix(branch, "~") {
			for i, _ := range commits {
				commits[i].Branch = branch
			}
		}

		resultMap := CommitsData{
			Data: commits,
		}

		utils.DebugLog.Println("Returning API commits")

		c.Header("ETag", commit.GetETag())
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}

// @Title GetCommitByRevision
// @Description Get commit by revision
// @Accept  json
// @Param   repo	 query   string     false       "Repo. This will help the backend to return the commit in a more efficient way."
// @Param   revision     path    string     true        "Commit revision"
// @Success 200 {object} commit.ApiCommit
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /commits
// @Router /commits/{revision} [GET]
func GetCommit() gin.HandlerFunc {
	return func(c *gin.Context) {

		if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && commit.GetETag() == etag {
			c.Status(http.StatusNotModified)
			return
		}

		revision := c.Param("revision")
		repo := c.Request.URL.Query().Get("repo")

		cmt, axErr := commit.GetAPICommitByRevision(revision, repo)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if cmt == nil {
			c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		c.Header("ETag", commit.GetETag())
		c.JSON(axerror.REST_STATUS_OK, cmt)
	}
}

//func GatewaySCMCommitsList() gin.HandlerFunc {
//	commitsUrl := DevopsCl.GetRootUrl() + "/scm/commits"
//	url, err := url.Parse(commitsUrl)
//	if err != nil {
//		panic(fmt.Sprintf("Can not parse the gateway commits url: %v", commitsUrl))
//	}
//	fmt.Println(commitsUrl)
//	return gin.WrapH(NewSingleHostReverseProxyWebhook(url))
//}
