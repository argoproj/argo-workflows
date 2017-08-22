// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Policy API [/policies]
package axops

import (
	"net/http"

	"applatix.io/axerror"
	"applatix.io/axops/policy"
	"applatix.io/axops/utils"
	"applatix.io/axops/yaml"
	"github.com/gin-gonic/gin"
)

type PoliciesData struct {
	Data []policy.Policy `json:"data"`
}

// @Title GetPolicies
// @Description List policies
// @Accept  json
// @Param   id  	 query   string     false       "ID."
// @Param   name  	 query   string     false       "Name."
// @Param   description	 query   string     false       "Description."
// @Param   repo	 query   string     false       "Repo."
// @Param   branch	 query   string     false       "Branch."
// @Param   repo_branch	 query   string     false       "Repo_Branch, eg:repo_branch=https://bitbucket.org/example.git_master or [{"repo":"https://bitbucket.org/example.git","branch":"master"},{"repo":"https://bitbucket.org/example.git","branch":"test"}] (encode needed)"
// @Param   template	 query   string     false       "Template name."
// @Param   enabled	 query   bool       false       "Enabled."
// @Param   labels	 query   string     false       "Labels, eg:labels=k1:v1,v2;k2:v1"
// @Param   search	 query   string     false       "Search."
// @Param   search_fields query   string     false       "Search fields."
// @Param   fields	 query   string     false       "Fields, eg:fields=id,name,repo,branch,description"
// @Param   limit	 query   int 	    false       "Limit."
// @Param   offset       query   int        false       "Offset."
// @Param   sort         query   string     false       "Sort, eg:sort=-name,repo which is sorting by name DESC and repo ASC"
// @Param   status       query   string     false       "Status."
// @ Success 200 {object} PoliciesData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /policies
// @Router /policies [GET]
func ListPolicies() gin.HandlerFunc {
	return func(c *gin.Context) {

		if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && policy.GetETag() == etag {
			c.Status(http.StatusNotModified)
			return
		}

		params, axErr := GetContextParams(c,
			[]string{
				policy.PolicyID,
				policy.PolicyName,
				policy.PolicyDescription,
				policy.PolicyRepo,
				policy.PolicyBranch,
				policy.PolicyTemplate,
				policy.PolicyRepoBranch,
				policy.PolicyStatus,
			},
			[]string{policy.PolicyEnabled},
			[]string{},
			[]string{policy.PolicyLabels})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		if policies, axErr := policy.GetPolicies(params); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			resultMap := map[string]interface{}{utils.RestData: policies}
			c.Header("ETag", policy.GetETag())
			c.JSON(axerror.REST_STATUS_OK, resultMap)
		}
	}
}

// @Title GetPolicyByID
// @Description Get policy by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of policy"
// @ Success 200 {object} policy.Policy
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /policies
// @Router /policies/{id} [GET]
func GetPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {

		if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && policy.GetETag() == etag {
			c.Status(http.StatusNotModified)
			return
		}

		id := c.Param("id")
		p, axErr := policy.GetPolicyByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if p == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		c.Header("ETag", policy.GetETag())
		c.JSON(axerror.REST_STATUS_OK, p)
		return
	}
}

// @Title EnablePolicyByID
// @Description Enable policy by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of policy"
// @ Success 200 {object} policy.Policy
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /policies
// @Router /policies/{id}/enable [PUT]
func EnablePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		p, axErr := policy.GetPolicyByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if p == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		p.Enabled = utils.NewTrue()
		p, axErr = p.Update()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		go yaml.NotifyScheduleChange(p.Repo, p.Branch)

		c.JSON(axerror.REST_STATUS_OK, p)
		return
	}
}

// @Title DisablePolicyByID
// @Description Enable policy by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of policy"
// @ Success 200 {object} policy.Policy
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /policies
// @Router /policies/{id}/disable [PUT]
func DisablePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		p, axErr := policy.GetPolicyByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if p == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		p.Enabled = utils.NewFalse()
		p, axErr = p.Update()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		go yaml.NotifyScheduleChange(p.Repo, p.Branch)

		c.JSON(axerror.REST_STATUS_OK, p)
		return
	}
}

// @Title DeleteInvalidPolicyByID
// @Description Delete a policy by ID. The policy has to be invalid to be deleted.
// @Accept  json
// @Param   id     	 path    string     true        "ID of policy"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /policies
// @Router /policies/{id} [DELETE]
func DeletePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		p, axErr := policy.GetPolicyByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		if p == nil {
			c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
			return
		}
		if p.Status != "invalid" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("The associated policy is not an invalid policy"))
			return
		}
		axErr = p.Delete()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}
