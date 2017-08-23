// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Policy API [/policies]
package axops

import (
	"net/http"

	"applatix.io/axerror"
	"applatix.io/axops/policy"
	"applatix.io/axops/utils"
	"applatix.io/axops/yaml"
	"applatix.io/common"
	"applatix.io/notification_center"
	"fmt"
	"github.com/gin-gonic/gin"
	"time"
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

		if p.Status == policy.InvalidStatus {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_AX_ILLEGAL_ARGUMENT.New())
			return
		}

		// Get user who enabled the policy
		u := GetContextUser(c)
		p.Enabled = utils.NewTrue()
		current_time := time.Now()
		p.Status = fmt.Sprintf("Policy %s is enabled by %s on %s", p.Name, u.Username, current_time.Format("2006-01-02 15:04:05"))
		p, axErr = p.Update()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		// Notify job scheduler for cron
		go yaml.NotifyScheduleChange(p.Repo, p.Branch)
		// Send notification to notification center
		go sendNotificationForPolicy(p)

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

		if p.Status == policy.InvalidStatus {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_AX_ILLEGAL_ARGUMENT.New())
			return
		}

		// Get user who disables the policy
		u := GetContextUser(c)
		p.Enabled = utils.NewFalse()
		current_time := time.Now()
		p.Status = fmt.Sprintf("Policy %s is disabled by %s on %s", p.Name, u.Username, current_time.Format("2006-01-02 15:04:05"))

		p, axErr = p.Update()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		// Notify job scheduler for cron
		go yaml.NotifyScheduleChange(p.Repo, p.Branch)
		// Send notification to notification center
		go sendNotificationForPolicy(p)

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

func sendNotificationForPolicy(p *policy.Policy) {
	detail := map[string]interface{}{}
	detail["Policy Url"] = fmt.Sprintf("https://%s/app/policies/details/%s", common.GetPublicDNS(), p.ID)
	detail["Policy Name"] = p.Name
	detail["Policy Associated Template"] = p.Template
	detail["Policy Repo"] = p.Repo
	detail["Policy Branch"] = p.Branch
	detail["Message"] = p.Status
	if *(p.Enabled) == true {
		notification_center.Producer.SendMessage(notification_center.CodeEnabledPolicy, "", []string{}, detail)
	} else if *(p.Enabled) == false {
		notification_center.Producer.SendMessage(notification_center.CodeDisabledPolicy, "", []string{}, detail)
	}
}
