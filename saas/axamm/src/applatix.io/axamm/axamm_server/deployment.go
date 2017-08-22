package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"applatix.io/axamm/application"
	"applatix.io/axamm/axam"
	"applatix.io/axamm/deployment"
	"applatix.io/axamm/utils"
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/common"
	"applatix.io/template"
	"github.com/gin-gonic/gin"
)

type DeploymentsData struct {
	Data []*deployment.Deployment `json:"data"`
}

func ListDeployments() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Header.Get("If-None-Match") == deployment.GetETag()+application.GetETag() {
			c.Status(http.StatusNotModified)
			return
		}

		params, axErr := common.GetContextParams(c,
			[]string{
				deployment.ServiceTaskId,
				deployment.ServiceName,
				deployment.ServiceTemplateName,
				deployment.ServiceTemplateId,
				deployment.ServiceDescription,
				deployment.ServiceUserName,
				deployment.ServiceUserId,
				deployment.ServiceRepo,
				deployment.ServiceBranch,
				deployment.ServiceRepoBranch,
				deployment.ServiceRevision,
				deployment.ServiceStatus,

				deployment.DeploymentAppId,
				deployment.DeploymentAppName,
				deployment.DeploymentAppGene,
				deployment.DeploymentId,

				deployment.DeploymentEndpoints,
			},
			[]string{},
			[]string{},
			[]string{deployment.ServiceLabels, deployment.ServiceAnnotations})

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		params, axErr = common.GetContextTimeParams(c, params)

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		if deployments, axErr := deployment.GetLatestDeployments(params, true); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {

			deploymentData := DeploymentsData{
				Data: deployments,
			}

			c.Header("ETag", deployment.GetETag()+application.GetETag())
			c.JSON(axerror.REST_STATUS_OK, deploymentData)
		}
	}
}

func ListDeploymentsHistory() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Header.Get("If-None-Match") == deployment.GetETag()+application.GetETag() {
			c.Status(http.StatusNotModified)
			return
		}

		params, axErr := common.GetContextParams(c,
			[]string{
				deployment.ServiceTaskId,
				deployment.ServiceName,
				deployment.ServiceTemplateName,
				deployment.ServiceTemplateId,
				deployment.ServiceDescription,
				deployment.ServiceUserName,
				deployment.ServiceUserId,
				deployment.ServiceRepo,
				deployment.ServiceBranch,
				deployment.ServiceRepoBranch,
				deployment.ServiceRevision,
				deployment.ServiceStatus,

				deployment.DeploymentAppId,
				deployment.DeploymentAppName,
				deployment.DeploymentAppGene,
				deployment.DeploymentId,
				deployment.ServiceTaskId,

				deployment.DeploymentEndpoints,
			},
			[]string{},
			[]string{},
			[]string{deployment.ServiceLabels, deployment.ServiceAnnotations})

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		params, axErr = common.GetContextTimeParams(c, params)

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		if deployments, axErr := deployment.GetHistoryDeployments(params, true); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {

			deploymentData := DeploymentsData{
				Data: deployments,
			}

			c.Header("ETag", deployment.GetETag()+application.GetETag())
			c.JSON(axerror.REST_STATUS_OK, deploymentData)
		}
	}
}

func PostDeployment() gin.HandlerFunc {
	return func(c *gin.Context) {
		var d *deployment.Deployment
		r := &deployment.Deployment{}

		err := common.GetUnmarshalledBody(c, &d)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage(err.Error()))
			return
		}

		// AMM is responsible for the deployment object validation, so the AM doesn't need to worry about it.
		if d.Template == nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid service object - template is not specified"))
			return
		}

		if d.Template.Containers == nil || len(d.Template.Containers) == 0 {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain valid containers object - containers are not specified"))
			return
		}

		if d.Template.ApplicationName == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain valid application information  - application is not specified"))
			return
		}

		if d.Template.DeploymentName == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain valid deployment information  - deployment is not specified"))
			return
		}

		// Init the deployment obj
		axErr := d.PreProcess()
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		// Substitute all the parameters
		d, axErr = d.Substitute()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		// Populate deployment parameters
		axErr = d.PostProcess()
		if axErr != nil {
			common.ErrorLog.Println(axErr)
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}
		dBytes, _ := json.Marshal(d)
		utils.DebugLog.Printf("%s after postprocesing:\n%s", d, string(dBytes))

		deployment.DeployLockGroup.Lock(d.Key())
		defer deployment.DeployLockGroup.Unlock(d.Key())

		app := &application.Application{
			Name: d.ApplicationName,
		}

		app, axErr = CreateApplication(app)
		if axErr != nil {
			common.ErrorLog.Println(axErr)
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		old, axErr := deployment.GetLatestDeploymentByName(d.ApplicationName, d.Name, false)
		if axErr != nil {
			common.ErrorLog.Println(axErr)
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		d.ApplicationGeneration = app.ID

		// remove old terminated deployment
		if old != nil && old.Status == deployment.DeployStateTerminated {
			axErr = old.DeleteObject()
			if axErr != nil {
				common.ErrorLog.Println(axErr)
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}
			old = nil
		}

		// check if duplicate create request
		if old != nil && d.Id == old.Id && len(old.PreviousDeploymentId) == 0 {
			c.JSON(axerror.REST_STATUS_OK, d)
			return
		}

		// check for first create request
		if old == nil {
			// create request
			axErr = d.CreateObject(nil)
			if axErr != nil {
				common.ErrorLog.Println(axErr)
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}
			// Ping the AM to see if it has come up
			axErr = axam.PingAM(d.ApplicationName)
			if axErr != nil {
				// it's ok;monitor thread will create the deployment later
				c.JSON(axerror.REST_STATUS_OK, d)
				return
			}
		}

		// call AM to process the create or update request
		if axErr, code := axam.PostAmDeployment(d, r); axErr != nil {
			common.ErrorLog.Println(axErr)
			c.JSON(code, axErr)
			return
		}

		c.JSON(axerror.REST_STATUS_OK, r)
		return

	}
}

func CreateApplication(app *application.Application) (*application.Application, *axerror.AXError) {

	application.AppLockGroup.Lock(app.Key())
	defer application.AppLockGroup.Unlock(app.Key())

	app, axErr, _ := app.Create()
	if axErr != nil {
		return nil, axErr
	}

	return app, nil
}

func GetDeployment() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Header.Get("If-None-Match") == deployment.GetETag()+application.GetETag() {
			c.Status(http.StatusNotModified)
			return
		}

		id := c.Param("id")
		if d, axErr := deployment.GetDeploymentByID(id, true); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if d != nil {
				c.Header("ETag", deployment.GetETag()+application.GetETag())
				c.JSON(axdb.RestStatusOK, d)
				return
			} else {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
				return
			}
		}
	}
}

func DeleteDeployment() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if d, axErr := deployment.GetLatestDeploymentByID(id, false); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if d == nil {
				c.JSON(axdb.RestStatusOK, common.NullMap)
				return
			}

			deployment.DeployLockGroup.Lock(d.Key())
			defer deployment.DeployLockGroup.Unlock(d.Key())

			if axErr, code := axam.DeleteAmDeployment(d); axErr != nil {
				c.JSON(code, axErr)
				return
			}

			c.JSON(axdb.RestStatusOK, common.NullMap)
			return
		}
	}
}

func TerminateDeployment(d *deployment.Deployment) *axerror.AXError {
	if d.Status == deployment.DeployStateTerminated {
		return d.DeleteObject()
	}

	if d.Status != deployment.DeployStateTerminating {
		if axErr, _ := d.MarkTerminating(utils.GetStatusDetail("TERMINATING", "Deployment will be terminated shortly.", "")); axErr != nil {
			return axErr
		}
	}

	if axErr, _ := axam.DeleteAmDeployment(d); axErr != nil {
		return axErr
	}

	d, axErr := deployment.GetLatestDeploymentByName(d.ApplicationName, d.Name, false)
	if axErr != nil {
		return axErr
	}

	if d != nil {
		return d.DeleteObject()
	} else {
		return nil
	}
}

func StartDeployment() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if d, axErr := deployment.GetLatestDeploymentByID(id, false); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if d == nil {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND)
				return
			}

			deployment.DeployLockGroup.Lock(d.Key())
			defer deployment.DeployLockGroup.Unlock(d.Key())

			if axErr, code := axam.StartAmDeployment(d); axErr != nil {
				c.JSON(code, axErr)
				return
			}

			c.JSON(axdb.RestStatusOK, common.NullMap)
			return
		}
	}
}

func StopDeployment() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if d, axErr := deployment.GetLatestDeploymentByID(id, false); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if d == nil {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND)
				return
			}

			deployment.DeployLockGroup.Lock(d.Key())
			defer deployment.DeployLockGroup.Unlock(d.Key())

			if axErr, code := axam.StopAmDeployment(d); axErr != nil {
				c.JSON(code, axErr)
				return
			}

			c.JSON(axdb.RestStatusOK, common.NullMap)
			return
		}
	}
}

func ScaleDeployment() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if d, axErr := deployment.GetLatestDeploymentByID(id, false); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if d == nil {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND)
				return
			}

			var scale template.Scale
			err := common.GetUnmarshalledBody(c, &scale)
			if err != nil {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage(err.Error()))
				return
			}

			if scale.Min <= 0 {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Can not scale the deployment to %v.", scale.Min))
				return
			}

			deployment.DeployLockGroup.Lock(d.Key())
			defer deployment.DeployLockGroup.Unlock(d.Key())

			if axErr, code := axam.ScaleAmDeployment(d, &scale); axErr != nil {
				c.JSON(code, axErr)
				return
			}

			c.JSON(axdb.RestStatusOK, common.NullMap)
			return
		}
	}
}

func UpdateDeployment() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if old, axErr := deployment.GetLatestDeploymentByID(id, false); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if old == nil {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND)
				return
			}

			var new deployment.Deployment
			err := common.GetUnmarshalledBody(c, &new)
			if err != nil {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage(err.Error()))
				return
			}

			old.TerminationPolicy = new.TerminationPolicy

			deployment.DeployLockGroup.Lock(old.Key())
			defer deployment.DeployLockGroup.Unlock(old.Key())

			if d, axErr, code := axam.UpdateAmDeployment(old); axErr != nil {
				c.JSON(code, axErr)
				return
			} else {
				c.JSON(code, d)
				return
			}
		}
	}
}

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

		ids := []string{}
		idStr := c.Request.URL.Query().Get("id")
		if idStr != "" {
			ids = strings.Split(idStr, ",")
		}

		ch, axErr := deployment.GetStatusServiceIdsChannel(c, ids)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		if ch == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("service not found"))
			return
		}

		header := c.Writer.Header()
		header["Content-Type"] = []string{"text/event-stream"}
		header["Transfer-Encoding"] = []string{"chunked"}
		header["X-Content-Type-Options"] = []string{"nosniff"}

		for {
			select {
			case <-c.Writer.CloseNotify():
				fmt.Println("[STREAM] client closed")
				deployment.ClearStatusIdsChannel(c)
				return
			case event := <-ch:
				eventStr, _ := json.Marshal(event)
				utils.DebugLog.Printf("[STREAM] writing to context %v event %s", c, string(eventStr))
				code, err := c.Writer.WriteString("data:" + string(eventStr) + "\n\n")
				utils.DebugLog.Printf("[STREAM] Writer error: %v %v", err, code)
				c.Writer.Flush()
			}
		}
	}
}
