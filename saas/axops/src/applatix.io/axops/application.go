// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi Application API [/applications]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"bufio"
	"fmt"
	"github.com/gin-gonic/gin"
)

// Fake Object to make swagger happy
type ApplicationsData struct {
	Data []*Application `json:"data"`
}

// Fake Object to make swagger happy
type Application struct {
	ID            string                 `json:"id,omitempty"`
	ApplicationID string                 `json:"-"`
	Name          string                 `json:"name,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Status        string                 `json:"status,omitempty"`
	StatusDetail  map[string]interface{} `json:"status_detail,omitempty"`
	Ctime         int64                  `json:"ctime,omitempty"`
	Mtime         int64                  `json:"mtime,omitempty"`
	Deployments   []*Deployment          `json:"deployments,omitempty"`
	Endpoints     []string               `json:"endpoints,omitempty"`

	DeploymentsInit        int64 `json:"deployments_init"`
	DeploymentsWaiting     int64 `json:"deployments_waiting"`
	DeploymentsError       int64 `json:"deployments_error"`
	DeploymentsActive      int64 `json:"deployments_active"`
	DeploymentsTerminating int64 `json:"deployments_terminating"`
	DeploymentsTerminated  int64 `json:"deployments_terminated"`
	DeploymentsStopping    int64 `json:"deployments_stopping"`
	DeploymentsStopped     int64 `json:"deployments_stopped"`
}

//@Title ListApplications
//@Description List latest applications
//@Accept  json
//@Param   name  	 query   string     false       "Name."
//@Param   description	 query   string     false       "Description."
//@Param   status	 query   string     false       "Status, eg:Init,Waiting,Active,Error,Terminating,Terminated."
//@Param   include_details	 query   bool       false       "Include details, all the latest deployments will be included."
//@Param   endpoints	 query   string     false       "External endpoints."
//@Param   sort          query   string     false       "Sort, eg:sort=-name,status which is sorting by name DESC and status ASC"
//@Param   search  	 query   string     false       "The text to search for."
//@Param   search_fields query   string     false       "Search fields."
//@Param   fields	 query   string     false       "Fields, eg:fields=id,name,description,status,status_detail,commit,deployments"
//@Param   limit	 query   int 	    false       "Limit."
//@Param   offset        query   int        false       "Offset."
//@Success 200 {object} ApplicationsData
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /applications
//@Router /applications [GET]
func ListApplications() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title ListApplicationHistories
// @Description List application histories
// @Accept  json
// @Param   name  	 query   string     false       "Name."
// @Param   description	 query   string     false       "Description."
// @Param   status	 query   string     false       "Status, eg:Init,Waiting,Active,Error,Terminating,Terminated"
// @Param   endpoints	 query   string     false       "External endpoints."
// @Param   sort         query   string     false       "Sort, eg:sort=-name,status which is sorting by name DESC and status ASC"
// @Param   search  	 query   string     false       "The text to search for."
// @Param   search_fields query   string     false       "Search fields."
// @Param   fields	 query   string     false       "Fields, eg:fields=id,name,description,status,status_detail,commit,deployments"
// @Param   limit	 query   int 	    false       "Limit."
// @Param   offset       query   int        false       "Offset."
// @Param   min_time	 query   int 	    false       "Min time."
// @Param   max_time	 query   int 	    false       "Max time."
// @Success 200 {object} ApplicationsData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /applications
// @Router /application/histories [GET]
func ListApplicationHistories() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title GetApplication
// @Description Get application details
// @Accept  json
// @Param   id           path    string     true        "UUID of the application"
// @Success 200 {object} Application
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /applications
// @Router /applications/{id} [GET]
func GetApplication() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title LaunchApplication
// @Description Launch an application
// @Accept  json
// @Param   application  body    Application     true        "Application object."
// @Success 201 {object} Application
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /applications
// @Router /applications [POST]
func PostApplication() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title UpdateApplication
// @Description Update application attributes
// @Accept  json
// @Param   id           path    string     		     true        "UUID of the application"
// @Param   application  body    Application     true        "Application object."
// @Success 200 {object} Application
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /applications
// @Router /applications/{id} [PUT]
func PutApplication() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title TerminateApplication
// @Description Terminate an application
// @Accept  json
// @Param   id           path    string     true        "UUID of the application"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /applications
// @Router /applications/{id} [DELETE]
func DeleteApplication() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title StartApplication
// @Description Start an application
// @Accept  json
// @Param   id           path    string     true        "UUID of the application"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /applications
// @Router /applications/{id}/start [POST]
func StartApplication() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

// @Title StopApplication
// @Description Stop an application
// @Accept  json
// @Param   id           path    string     true        "UUID of the application"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /applications
// @Router /applications/{id}/stop [POST]
func StopApplication() gin.HandlerFunc {
	return AppMonitorMgrProxy()
}

type DeploymentEvent struct {
	Id           string            `json:"id"`
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	StatusDetail map[string]string `json:"status_detail"`
}

// @Title GetApplicationEvents
// @Description Get the application events.
// @Accept  json
// @Param   id    query   string     false       "UUID of the application."
// @Success 200 {object} DeploymentEvent
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Failure 404 {object} axerror.AXError "service not found"
// @Resource /applications
// @Router /application/events [GET]
func AppEventsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		url := fmt.Sprintf("http://axamm.axsys:8966/v1/application/events")
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
