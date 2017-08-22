// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi Deployment API [/deployments]
package axops

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"applatix.io/axamm/deployment"
	"applatix.io/axerror"
	"applatix.io/axops/commit"
	"applatix.io/axops/notification"
	"applatix.io/axops/utils"
	"applatix.io/template"
	"github.com/gin-gonic/gin"
	"github.com/yhat/wsutil"
)

// Fake Object to make swagger happy
type Deployment struct {
	Id            string                      `json:"id,omitempty"`
	Name          string                      `json:"name,omitempty"`
	Description   string                      `json:"description,omitempty"`
	CostId        map[string]interface{}      `json:"costid,omitempty"`
	Template      MapType                     `json:"template,omitempty"`
	TemplateID    string                      `json:"template_id,omitempty"`
	Parameters    map[string]interface{}      `json:"parameters,omitempty"`
	Status        string                      `json:"status"`
	StatusDetail  map[string]interface{}      `json:"status_detail,omitempty"`
	Mem           float64                     `json:"mem,omitempty"`
	CPU           float64                     `json:"cpu,omitempty"`
	User          string                      `json:"user,omitempty"`
	Notifications []notification.Notification `json:"notifications,ommitempty"`
	CreateTime    int64                       `json:"create_time"`
	LaunchTime    int64                       `json:"launch_time"`
	EndTime       int64                       `json:"end_time"`
	WaitTime      int64                       `json:"wait_time"`
	RunTime       int64                       `json:"run_time"`
	Commit        *commit.ApiCommit           `json:"commit,ommitempty"`
	Labels        map[string]string           `json:"labels"`
	Annotations   map[string]string           `json:"annotations"`

	TaskID                string `json:"task_id,omitempty"`
	ApplicationGeneration string `json:"app_generation,omitempty"`
	ApplicationID         string `json:"app_id,omitempty"`
	ApplicationName       string `json:"app_name,omitempty"`
	DeploymentID          string `json:"deployment_id,omitempty"`

	Fixtures  map[string]map[string]interface{} `json:"fixtures,omitempty"`
	Instances []*deployment.Pod                 `json:"instances,omitempty"`
	Endpoints []string                          `json:"endpoints,omitempty"`

	TerminationPolicy *template.TerminationPolicy `json:"termination_policy,omitempty"`
}

// Fake Object to make swagger happy
type DeploymentsData struct {
	Data []Deployment `json:"data"`
}

//@Title ListDeployments
//@Description List latest deployments
//@Accept  json
//@Param   name  	 	query   string     false       "Name."
//@Param   description	 	query   string     false       "Description."
//@Param   status	 	query   string     false       "Status, eg:Init,Waiting,Active,Error,Terminating,Terminated"
//@Param   template_id	 	query   string     false       "Template ID."
//@Param   task_id	 	query   string     false       "Task ID. The job that created this deployment."
//@Param   app_generation	query   string     false       "App Generation ID."
//@Param   app_id	 	query   string     false       "App ID."
//@Param   app_name	 	query   string     false       "App Name."
//@Param   endpoints	 	query   string     false       "External endpoints."
//@Param   sort          	query   string     false       "Sort, eg:sort=-name,status which is sorting by name DESC and status ASC"
//@Param   search  	 	query   string     false       "The text to search for."
//@Param   search_fields query   string     false       "Search fields."
//@Param   fields	 	query   string     false       "Fields, eg:fields=id,name,description,status,status_detail,commit, app_generation, app_id..."
//@Param   limit	 	query   int 	   false       "Limit."
//@Param   offset        	query   int        false       "Offset."
//@Success 200 {object} DeploymentsData
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /deployments
//@Router /deployments [GET]
func ListDeployments() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

//@Title ListDeploymentHistories
//@Description List deployment histories
//@Accept  json
//@Param   name  	 	query   string     false       "Name."
//@Param   description	 	query   string     false       "Description."
//@Param   status	 	query   string     false       "Status, eg:Init,Waiting,Active,Error,Terminating,Terminated"
//@Param   template_id	 	query   string     false       "Template ID."
//@Param   task_id	 	query   string     false       "Task ID. The job that created this deployment."
//@Param   app_generation	query   string     false       "App Generation ID."
//@Param   app_id	 	query   string     false       "App ID."
//@Param   app_name	 	query   string     false       "App Name."
//@Param   endpoints	 	query   string     false       "External endpoints."
//@Param   sort          	query   string     false       "Sort, eg:sort=-name,status which is sorting by name DESC and status ASC"
//@Param   search  	 	query   string     false       "The text to search for."
//@Param   search_fields query   string     false       "Search fields."
//@Param   fields	 	query   string     false       "Fields, eg:fields=id,name,description,status,status_detail,commit, app_generation, app_id..."
//@Param   limit	 	query   int 	    false      "Limit."
//@Param   offset        	query   int        false       "Offset."
//@Param   min_time	 	query   int 	    false      "Min time."
//@Param   max_time	 	query   int 	    false      "Max time."
//@Success 200 {object} DeploymentsData
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /deployments
//@Router /deployment/histories [GET]
func ListDeploymentsHistory() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title LaunchDeployment
// @Description Launch a deployment
// @Accept  json
// @Param   deployment   body    Deployment     true        "Deployment object"
// @Success 201 {object} Deployment
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /deployments
// @Router /deployments [POST]
func PostDeployment() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title GetDeployment
// @Description Get deployment details
// @Accept  json
// @Param   id           path    string     true        "UUID of the deployment"
// @Success 200 {object} Deployment
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /deployments
// @Router /deployments/{id} [GET]
func GetDeployment() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title TerminateDeployment
// @Description Terminate a deployment
// @Accept  json
// @Param   id           path    string     true        "UUID of the deployment"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /deployments
// @Router /deployments/{id} [DELETE]
func DeleteDeployment() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title StartDeployment
// @Description Start a deployment
// @Accept  json
// @Param   id           path    string     true        "UUID of the deployment"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /deployments
// @Router /deployments/{id}/start [POST]
func StartDeployment() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title StopDeployment
// @Description Stop a deployment
// @Accept  json
// @Param   id           path    string     true        "UUID of the deployment"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /deployments
// @Router /deployments/{id}/stop [POST]
func StopDeployment() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

type Scale struct {
	Min int64 `json:"min"`
}

// @Title ScaleDeployment
// @Description Scale a deployment
// @Accept  json
// @Param   id           path    string     true        "UUID of the deployment"
// @Param   spec         body    Scale      true        "Scale specification."
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /deployments
// @Router /deployments/{id}/scale [POST]
func ScaleDeployment() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title UpdateDeployment
// @Description Update a deployment
// @Accept  json
// @Param   id           path    string     	 true        "UUID of the deployment"
// @Param   deployment   body    Deployment      true        "Deployment object"
// @Success 200 {object} Deployment
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /deployments
// @Router /deployments/{id} [PUT]
func UpdateDeployment() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title GetDeploymentContainerLog
// @Description Get deployment container live log
// @Accept  json
// @Param   id    	 	path     string     true       "UUID of deployment."
// @Param   instance    	query    string     true       "Name of instance."
// @Param   container    	query    string     true       "Name of container."
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Failure 404 {object} axerror.AXError "log not found"
// @Resource /deployments
// @Router /deployments/{id}/livelog [GET]
func GetDeploymentLiveLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		d := deployment.Deployment{}
		axErr := utils.AxammCl.Get("deployments/"+c.Param("id"), nil, &d)
		if axErr != nil {
			if axErr.Code == axerror.ERR_API_RESOURCE_NOT_FOUND.Code {
				c.JSON(axerror.REST_NOT_FOUND, axErr)
				return
			}
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		instance := c.Request.URL.Query().Get("instance")
		container := c.Request.URL.Query().Get("container")

		if instance == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Instance is required field."))
			return
		}

		if container == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Container is required field."))
			return
		}

		header := c.Writer.Header()
		header["Content-Type"] = []string{"text/event-stream"}
		header["Transfer-Encoding"] = []string{"chunked"}
		header["X-Content-Type-Options"] = []string{"nosniff"}

		logURLStr := "http://localhost:8001" + fmt.Sprintf("/api/v1/namespaces/%v/pods/%v/log?container=%v", d.ApplicationName, instance, container)
		logUrl, err := url.Parse(logURLStr)
		if err != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, err)
			return
		}
		query := logUrl.Query()
		query.Add("follow", "true")
		logUrl.RawQuery = query.Encode()
		resp, err := http.DefaultClient.Get(logUrl.String())
		if err != nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("log not found at: "+logURLStr))
			return
		}
		c.Status(resp.StatusCode)
		scanner := bufio.NewScanner(resp.Body)
		const maxCapacity = 1024 * 1024
		buf := make([]byte, maxCapacity)
		scanner.Buffer(buf, maxCapacity)
		for scanner.Scan() {
			log, _ := json.Marshal(LogEntry{Log: scanner.Text() + "\n"})
			logStr := string(log)
			c.Writer.WriteString("data:" + logStr + "\n\n")
			c.Writer.Flush()

			select {
			case <-c.Writer.CloseNotify():
				fmt.Println("client closed")
				return
			default:
			}
		}
		if scanner.Err() != nil {
			utils.ErrorLog.Println("Scanner error:", scanner.Err())
		}
	}
}

// @Title GetDeploymentContainerConsole
// @Description Get deployment container console (websocket)
// @Accept  json
// @Param   id    	  path   string    true       "UUID of the deployment"
// @Param   instance  	 query   string    true       "Name of instance."
// @Param   container  	 query   string    true       "Name of container."
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Failure 404 {object} axerror.AXError "log not found"
// @Resource /deployments
// @Router /deployments/{id}/exec [GET]
func DeploymentConsoleWebSocket() gin.HandlerFunc {
	consoleURL := &url.URL{
		Scheme: "ws://",
		Host:   "axconsole:81",
	}
	proxy := wsutil.NewSingleHostReverseProxy(consoleURL)
	return gin.WrapH(proxy)
}

var axammClient *http.Client = http.DefaultClient

// @Title GetDeploymentEvents
// @Description Get the deployment events.
// @Accept  json
// @Param   id    query   string     false       "UUID of the deployment."
// @Success 200 {object} DeploymentEvent
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Failure 404 {object} axerror.AXError "service not found"
// @Resource /deployments
// @Router /deployment/events [GET]
func DeploymentEventsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		url := fmt.Sprintf("http://axamm.axsys:8966/v1/deployment/events")
		if idStr := c.Request.URL.Query().Get("id"); idStr != "" {
			url += "?id=" + idStr
		}
		utils.InfoLog.Printf("[STREAM] Forward request to %s", url)
		resp, err := axammClient.Get(url)
		if err != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_AX_INTERNAL.NewWithMessage(err.Error()))
			return
		}

		header := c.Writer.Header()
		header["Content-Type"] = []string{"text/event-stream"}
		header["Transfer-Encoding"] = []string{"chunked"}
		header["X-Content-Type-Options"] = []string{"nosniff"}

		c.Status(axerror.REST_STATUS_OK)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		const maxCapacity = 1024 * 1024
		buf := make([]byte, maxCapacity)
		scanner.Buffer(buf, maxCapacity)
		for scanner.Scan() {
			c.Writer.WriteString(scanner.Text() + "\n\n")
			c.Writer.Flush()

			select {
			case <-c.Writer.CloseNotify():
				utils.DebugLog.Println("[STREAM] client closed:", err.Error())
				return
			default:
			}
		}
		if scanner.Err() != nil {
			utils.ErrorLog.Println("[STREAM] Scanner error:", scanner.Err())
		}
	}
}
