// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi JIRA API [/jira]
package axops

import (
	//"applatix.io/axdb"
	//"applatix.io/axerror"
	//"applatix.io/axops/service"
	//"applatix.io/axops/utils"
	//"bytes"
	//"encoding/json"
	"github.com/gin-gonic/gin"
	//"io"
	//"net/http"
	//"strings"
	//"time"
	"applatix.io/axerror"
	"applatix.io/axops/jira"
	"applatix.io/axops/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type JiraProjectData struct {
	Data []*JiraProject `json:"data"`
}

// Fake Object to make swagger happy
type JiraProject struct {
	Name           string `json:"name,omitempty"`
	Key            string `json:"key,omitempty"`
	Id             string `json:"id,omitempty"`
	ProjectTypeKey string `json:"projectTypeKey,omitempty"`
}

type JiraUserData struct {
	Data []*JiraUser `json:"data"`
}

// Fake Object to make swagger happy
type JiraUser struct {
	Key      string `json:"key,omitempty"`
	Fullname string `json:"fullname,omitempty"`
	Email    string `json:"email,omitempty"`
	Active   bool   `json:"active,omitempty"`
}

type IssueComment struct {
	User    string `json:"user,omitempty"`
	Comment string `json:"comment,omitempty"`
}

type JiraIssueBody struct {
	Project string `json:"project,omitempty"`
	//JiraId      string `json:"jiraid,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Issuetype   string `json:"issuetype,omitempty"`
	Reporter    string `json:"reporter,omitempty"`
	Description string `json:"description,omitempty"`
}

type JiraStatus struct {
	Name string `json:"name,omitempty"`
	Id   string `json:"id,omitempty"`
}

type JiraIssueBodyFromJira struct {
	Project     JiraProject `json:"project,omitempty"`
	JiraId      string      `json:"key,omitempty"`
	Summary     string      `json:"summary,omitempty"`
	Status      JiraStatus  `json:"status,omitempty"`
	Description string      `json:"description,omitempty"`
}

//@Title ListJiraProjects
//@Description List Jira Projects satisfying some query condition
//@Accept  json
//@Param   name           query   string     false       "Project name"
//@Param   key            query   string     false       "Project key"
//@Param   projectTypeKey query   string     false       "Project type"
//@Success 200 {object} JiraProjectData
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/projects [GET]
func ListJiraProjects() gin.HandlerFunc {
	return JiraProcessor()
}

//@Title ListJiraIssues
//@Description List Jira issues satisfying some query condition
//@Accept  json
//@Param   ids                query    string    false    "Issue list, will ignore other filters"
//@Param   project            query    string    false    "Project name"
//@Param   status             query    string    false    "Issue status"
//@Param   component          query    string    false    "Issue component"
//@Param   labels             query    string    false    "Issue labels"
//@Param   issuetype          query    string    false    "Issue type"
//@Param   priority           query    string    false    "Issue priority"
//@Param   creator            query    string    false    "Issue creator"
//@Param   assignee           query    string    false    "Issue assignee"
//@Param   reporter           query    string    false    "Issue reporter"
//@Param   fixversion         query    string    false    "Issue fixversion"
//@Param   affectedversion    query    string    false    "Issue affectedversion"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues [GET]
func ListJiraIssues() gin.HandlerFunc {
	return JiraProcessor()
}

//@Title CreateJiraIssue
//@Description Create Jira issue
//@Accept  json
//@Param   jira  body    JiraIssueBody   true        "Jira Issue body"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues [POST]
func CreateJiraIssue(c *gin.Context) {
	timeout := time.Duration(60 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("POST", "http://gateway:8889/v1/jira/issues", c.Request.Body)
	if err != nil {
		utils.ErrorLog.Printf("Failed to create a new request to gateway.")
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessagef(fmt.Sprintf("Request body does not include a valid Jira body object, or the gatewary address is incorrect. (err: %v)", err)))
		return
	}

	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["Accept"] = []string{"application/json"}
	utils.InfoLog.Printf("new request was successfully created.")
	res, err := client.Do(req)

	if err != nil {
		utils.ErrorLog.Printf("Failed to execute a forwarding request to gateway.")
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessagef(fmt.Sprintf("Failed to execute a forwarding request to gateway. (err: %v)", err)))
		return
	}

	utils.InfoLog.Printf("new request return status code %d, %s.", res.StatusCode, res.Status)
	var jiraBody jira.JiraIssue
	var jiraBodyFromJira JiraIssueBodyFromJira
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	if err != nil {
		utils.ErrorLog.Printf("Failed to read response body returned from Jira service.")
		c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Failed to read response body returned from Jira service. (err %v)", err))
		return
	}

	err = json.Unmarshal(body, &jiraBodyFromJira)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessagef(fmt.Sprintf("Request body does not include a valid Jira body object (err: %v)", err)))
		return
	}

	jiraBody.JiraId = jiraBodyFromJira.JiraId
	jiraBody.Status = jiraBodyFromJira.Status.Name
	jiraBody.Description = jiraBodyFromJira.Description
	jiraBody.Summary = jiraBodyFromJira.Summary
	jiraBody.Project = jiraBodyFromJira.Project.Key

	utils.InfoLog.Printf("jira body: %v", jiraBody)
	axErr := jira.CreateJiraIssue(jiraBody)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef(fmt.Sprintf("Failed to create jira issue %v (err: %v)", jiraBody, axErr)))
	} else {
		c.JSON(axerror.REST_STATUS_OK, jiraBodyFromJira)
		//c.JSON(axerror.REST_STATUS_OK, res.Body)
	}
}

/*
//@Title DeleteJiraIssue
//@Description Delete Jira issue
//@Accept  json
//@Param   id  path    string   true        "Jira ID"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues/{id} [DELETE]
*/
func DeleteJiraIssue(c *gin.Context, id string) {
	//var jiraBody jira.JiraIssue
	//err := utils.GetUnmarshalledBody(c, &jiraBody)
	//if err != nil {
	//	c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Request body does not include a valid Jira body object (err: %v)", err)))
	//	return
	//}

	axErr := jira.DeleteJiraIssue(id)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(fmt.Sprintf("Failed to delete jira issue %s (err: %v)", id, axErr)))
	} else {
		c.JSON(axerror.REST_STATUS_OK, nullMap)
	}

}

//@Title UpdateJiraIssue
//@Description Update Jira issue
//@Accept  json
//@Param   Jira           body    jira.JiraIssue      true   "Jira Body"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues [PUT]
func UpdateJiraIssue(c *gin.Context) {
	var jiraBody jira.JiraIssue
	err := utils.GetUnmarshalledBody(c, &jiraBody)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Request body does not include a valid Jira body object (err: %v)", err)))
		return
	}

	utils.InfoLog.Printf("The payload received from Jira system is %v", jiraBody)
	var axErr *axerror.AXError = nil
	// it's a jira moved from other project
	if len(jiraBody.OldJiraId) != 0 {
		axErr = jira.MoveJiraIssue(jiraBody)
	} else {
		axErr = jira.UpdateJiraIssue(jiraBody)
	}

	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(fmt.Sprintf("Failed to update jira issue %v (err: %v)", jiraBody, axErr)))
	} else {
		c.JSON(axerror.REST_STATUS_OK, nullMap)
	}
}

/*
//@Title AssociateJiraService
//@Description Associate Jira issue with a service object
//@Accept  json
//@Param   jiraId          path    string      true   "Jira ID"
//@Param   serviceId       path    string      true   "Service ID"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues/{jiraId}/service/{serviceId} [PUT]
*/
func JiraServiceHandler(c *gin.Context, sid string, jid string) {
	//first check if the jid exits in DB or not
	jiraMap, axErr := jira.GetJiraBodyByID(jid)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}
	if jiraMap == nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The jira(%s) doesn't exist in DB", jid)))
		return
	}

	axErr = jira.AttachJiraToService(sid, jid)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	// add service id to jira table
	axErr = jira.AttachServiceToJira(sid, jiraMap)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
	} else {
		c.JSON(axerror.REST_STATUS_OK, nullMap)
	}
}

/*
//@Title AssociateJiraApplication
//@Description Associate Jira issue with an application object
//@Accept  json
//@Param   jiraId          path    string      true   "Jira ID"
//@Param   appId           path    string      true   "Application ID"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues/{jiraId}/application/{appId} [PUT]
*/
func JiraApplicationHandler(c *gin.Context, aid string, jid string) {
	//first check if the jid exits in DB or not
	jiraMap, axErr := jira.GetJiraBodyByID(jid)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}
	if jiraMap == nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The jira(%s) doesn't exist in DB", jid)))
		return
	}

	axErr = jira.AttachJiraToApplication(aid, jid)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	// add application id to jira table
	axErr = jira.AttachApplicationToJira(aid, jiraMap)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
	} else {
		c.JSON(axerror.REST_STATUS_OK, nullMap)
	}
}

/*
//@Title AssociateJiraDeployment
//@Description Associate Jira issue with a deployment object
//@Accept  json
//@Param   jiraId          path    string      true   "Jira ID"
//@Param   deployId        path    string      true   "Deployment ID"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues/{jiraId}/deployment/{deployId} [PUT]
*/
func JiraDeploymentHandler(c *gin.Context, did string, jid string) {
	//first check if the jid exits in DB or not
	jiraMap, axErr := jira.GetJiraBodyByID(jid)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}
	if jiraMap == nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The jira(%s) doesn't exist in DB", jid)))
		return
	}

	axErr = jira.AttachJiraToDeployment(did, jid)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	// add application id to jira table
	axErr = jira.AttachDeploymentToJira(did, jiraMap)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
	} else {
		c.JSON(axerror.REST_STATUS_OK, nullMap)
	}
}

//@Title ListJiraUsers
//@Description List Jira users satisfying some query condition
//@Accept  json
//@Param   email            query    string    false    "User email address"
//@Param   key              query    string    false    "Username"
//@Param   fullname         query    string    false    "User fullname"
//@Param   active           query    bool      false    "User activity"
//@Success 200 {object} JiraUserData
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/users [GET]
func ListJiraUsers() gin.HandlerFunc {
	return JiraProcessor()
}

//@Title ListJiraIssueTypes
//@Description List Jira issue types
//@Accept  json
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issuetypes [GET]
func ListJiraIssueTypes() gin.HandlerFunc {
	return JiraProcessor()
}

//@Title GetJiraIssueType
//@Description Get Jira issue type with specified name
//@Accept  json
//@Param   key   path   string     true       "Issue type name"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issuetypes/{key} [GET]
func GetJiraIssueType() gin.HandlerFunc {
	return JiraProcessor()
}

//@Title GetJiraIssueComments
//@Description Get Jira issue comments
//@Accept  json
//@Param   key   path   string     false       "Jira issue ID"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues/{key}/getcomments [GET]
func GetJiraIssueComments() gin.HandlerFunc {
	return JiraProcessor()
}

//@Title AddJiraIssueComment
//@Description Add a comment to a Jira issue
//@Accept  json
//@Param   key   path    string         false       "Jira issue ID"
//@Param   body  body    IssueComment   true        "Comment details"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues/{key}/addcomment [POST]
func AddJiraIssueComment() gin.HandlerFunc {
	return JiraProcessor()
}

//@Title GetJiraProject
//@Description Get Jira project with specified project key
//@Accept  json
//@Param   key   path   string     true       "Project key"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/projects/{key} [GET]
func GetJiraProject() gin.HandlerFunc {
	return JiraProcessor()
}

//@Title GetJiraIssue
//@Description Get Jira issue with specified issue ID
//@Accept  json
//@Param   key   path   string     true       "Issue ID"
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /jira
//@Router /jira/issues/{key} [GET]
func GetJiraIssue() gin.HandlerFunc {
	return JiraProcessor()
}
