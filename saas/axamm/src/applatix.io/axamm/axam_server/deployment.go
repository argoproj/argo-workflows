package main

import (
	"net/http"

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

		if c.Request.Header.Get("If-None-Match") == deployment.GetETag() {
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
			},
			[]string{},
			[]string{},
			[]string{deployment.ServiceLabels, deployment.ServiceAnnotations})

		// Filter out other applications
		params[deployment.DeploymentAppName] = utils.APPLICATION_NAME

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

			c.Header("ETag", deployment.GetETag())
			c.JSON(axerror.REST_STATUS_OK, deploymentData)
		}
	}
}

func PostDeployment() gin.HandlerFunc {
	return func(c *gin.Context) {
		var d *deployment.Deployment
		err := common.GetUnmarshalledBody(c, &d)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage(err.Error()))
			return
		}

		if d.Id == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid deployment id - id is not specified."))
			return
		}

		deployment.DeployLockGroup.Lock(d.Key())
		defer deployment.DeployLockGroup.Unlock(d.Key())

		if d.ApplicationName != utils.APPLICATION_NAME {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Can not handle the deployment for other applications."))
			return
		}

		old, axErr := deployment.GetLatestDeploymentByName(d.ApplicationName, d.Name, false)
		if axErr != nil {
			common.ErrorLog.Println(axErr)
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if old == nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Can not find deployment %v/%v.", d.ApplicationName, d.Name))
			return
		}

		// create case
		if d.Id == old.Id && len(old.PreviousDeploymentId) == 0 {
			if old.Status != deployment.DeployStateInit {
				// work has already started
				// do nothing
				c.JSON(axerror.REST_CREATE_OK, old)
				return
			}
			if axErr, code := old.Create(); axErr != nil {
				c.JSON(code, axErr)
				return
			}
			c.JSON(axerror.REST_CREATE_OK, old)
			return
		}
		// update case
		common.InfoLog.Printf("upgrade request for %v/%v.\n", d.ApplicationName, d.Name)
		d, axErr, code := d.Upgrade(old)
		if axErr != nil {
			c.JSON(code, axErr)
			return
		}
		c.JSON(axerror.REST_CREATE_OK, d)
		return
	}
}

func GetDeployment() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Header.Get("If-None-Match") == deployment.GetETag() {
			c.Status(http.StatusNotModified)
			return
		}

		id := c.Param("id")
		if d, axErr := deployment.GetDeploymentByID(id, true); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if d != nil && d.Template.ApplicationName == utils.APPLICATION_NAME {
				c.Header("ETag", deployment.GetETag())
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

			if d.Template.ApplicationName == utils.APPLICATION_NAME {

				deployment.DeployLockGroup.Lock(d.Key())
				defer deployment.DeployLockGroup.Unlock(d.Key())

				axErr, code := d.Delete(nil)
				if axErr != nil {
					c.JSON(axerror.REST_INTERNAL_ERR, code)
					return
				}
			}

			c.JSON(axdb.RestStatusOK, common.NullMap)
			return
		}
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
				c.JSON(axdb.RestStatusOK, common.NullMap)
				return
			}

			if d.Template.ApplicationName == utils.APPLICATION_NAME {

				deployment.DeployLockGroup.Lock(d.Key())
				defer deployment.DeployLockGroup.Unlock(d.Key())

				axErr, code := d.Start()
				if axErr != nil {
					c.JSON(code, axErr)
					return
				}
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
				c.JSON(axdb.RestStatusOK, common.NullMap)
				return
			}

			if d.Template.ApplicationName == utils.APPLICATION_NAME {

				deployment.DeployLockGroup.Lock(d.Key())
				defer deployment.DeployLockGroup.Unlock(d.Key())

				axErr, code := d.Stop()
				if axErr != nil {
					c.JSON(code, axErr)
					return
				}
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
				c.JSON(axdb.RestStatusOK, common.NullMap)
				return
			}

			if d.Template.ApplicationName == utils.APPLICATION_NAME {

				var scale template.Scale
				err := common.GetUnmarshalledBody(c, &scale)
				if err != nil {
					c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage(err.Error()))
					return
				}

				deployment.DeployLockGroup.Lock(d.Key())
				defer deployment.DeployLockGroup.Unlock(d.Key())

				axErr, code := d.Scale(&scale)
				if axErr != nil {
					c.JSON(code, axErr)
					return
				}
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
				c.JSON(axdb.RestStatusOK, common.NullMap)
				return
			}

			if old.Template.ApplicationName == utils.APPLICATION_NAME {

				var new deployment.Deployment
				err := common.GetUnmarshalledBody(c, &new)
				if err != nil {
					c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage(err.Error()))
					return
				}

				old.TerminationPolicy = new.TerminationPolicy

				deployment.DeployLockGroup.Lock(old.Key())
				defer deployment.DeployLockGroup.Unlock(old.Key())

				old, axErr = old.UpdateObject()
				if axErr != nil {
					c.JSON(axerror.REST_INTERNAL_ERR, axErr)
					return
				}
			}

			c.JSON(axdb.RestStatusOK, old)
			return
		}
	}
}
