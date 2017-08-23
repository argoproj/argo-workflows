package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"applatix.io/axamm/application"
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"github.com/gin-gonic/gin"
)

type ApplicationsData struct {
	Data []*application.Application `json:"data"`
}

func ListApplications() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Header.Get("If-None-Match") == application.GetETag() {
			c.Status(http.StatusNotModified)
			return
		}

		params, axErr := common.GetContextParams(c,
			[]string{
				application.ApplicationID,
				application.ApplicationName,
				application.ApplicationDescription,
				application.ApplicationStatus,
				application.ApplicationEndpoints,
				application.ApplicationAppID,
			},
			[]string{},
			[]string{},
			[]string{})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		params, axErr = common.GetContextTimeParams(c, params)

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		if applications, axErr := application.GetLatestApplications(params, true); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {

			includeDetails := c.Request.URL.Query().Get("include_details")
			if strings.ToLower(includeDetails) == "true" {
				var wg sync.WaitGroup
				wg.Add(len(applications))
				for i := range applications {
					// Load the deployments in parallel, this might be expensive, we can use redis cache to
					// improve if needed.
					go applications[i].LoadDeployments(&wg)
				}
				wg.Wait()
			}

			applicationData := ApplicationsData{
				Data: applications,
			}

			c.Header("ETag", application.GetETag())
			c.JSON(axerror.REST_STATUS_OK, applicationData)
		}
	}
}

func ListApplicationHistories() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Header.Get("If-None-Match") == application.GetETag() {
			c.Status(http.StatusNotModified)
			return
		}

		params, axErr := common.GetContextParams(c,
			[]string{
				application.ApplicationID,
				application.ApplicationAppID,
				application.ApplicationName,
				application.ApplicationDescription,
				application.ApplicationStatus,
				application.ApplicationEndpoints,
			},
			[]string{},
			[]string{},
			[]string{})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		params, axErr = common.GetContextTimeParams(c, params)

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		if applications, axErr := application.GetHistoryApplications(params, true); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {

			applicationData := ApplicationsData{
				Data: applications,
			}
			c.Header("ETag", application.GetETag())
			c.JSON(axerror.REST_STATUS_OK, applicationData)
		}
	}
}

func GetApplication() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Header.Get("If-None-Match") == application.GetETag() {
			c.Status(http.StatusNotModified)
			return
		}

		id := c.Param("id")
		if app, axErr := application.GetLatestApplicationByID(id, true); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if app != nil {
				if app.Status != application.AppStateTerminated {
					axErr := app.LoadDeployments(nil)
					if axErr != nil {
						c.JSON(axerror.REST_INTERNAL_ERR, axErr)
						return
					}
				}
				c.Header("ETag", application.GetETag())
				c.JSON(axdb.RestStatusOK, app)
				return
			}
		}

		if app, axErr := application.GetHistoryApplicationByID(id, true); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if app != nil {
				c.Header("ETag", application.GetETag())
				c.JSON(axdb.RestStatusOK, app)
				return
			} else {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
				return
			}
		}
	}
}

func PostApplication() gin.HandlerFunc {
	return func(c *gin.Context) {

		var app *application.Application
		err := common.GetUnmarshalledBody(c, &app)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		application.AppLockGroup.Lock(app.Key())
		defer application.AppLockGroup.Unlock(app.Key())

		// Create the application object with init state
		app, axErr, code := app.Create()
		if axErr != nil {

			if _, axErr, _ := app.Delete(); axErr != nil {
				utils.ErrorLog.Printf("Failed to clean up the application %v: %v.\n", app.Name, axErr)
			}

			c.JSON(code, axErr)
			return
		}

		app.Ctime = app.Ctime / 1e6
		app.Mtime = app.Mtime / 1e6

		c.JSON(code, app)
		return
	}
}

func PutApplication() gin.HandlerFunc {
	return func(c *gin.Context) {

		id := c.Param("id")

		if old, axErr := application.GetLatestApplicationByID(id, false); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if old == nil {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
				return
			}

			var new *application.Application
			err := common.GetUnmarshalledBody(c, &new)
			if err != nil {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
				return
			}

			application.AppLockGroup.Lock(old.Key())
			defer application.AppLockGroup.Unlock(old.Key())

			old.Description = new.Description
			old, axErr, code := old.UpdateObject()
			if axErr != nil {
				c.JSON(code, axErr)
				return
			}

			old.Ctime = old.Ctime / 1e3
			old.Mtime = old.Mtime / 1e3

			c.JSON(code, old)
			return
		}
	}
}

func DeleteApplication() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if app, axErr := application.GetLatestApplicationByID(id, false); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if app == nil {
				c.JSON(axdb.RestStatusOK, common.NullMap)
				return
			}

			application.AppLockGroup.Lock(app.Key())
			defer application.AppLockGroup.Unlock(app.Key())

			if axErr := app.LoadDeployments(nil); axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}

			_, axErr, code := app.Delete()
			if axErr != nil {
				c.JSON(code, axErr)
				return
			}

			c.JSON(axdb.RestStatusOK, common.NullMap)
			return
		}
	}
}

func StartApplication() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if app, axErr := application.GetLatestApplicationByID(id, false); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if app == nil {
				c.JSON(axdb.RestStatusOK, common.NullMap)
				return
			}

			application.AppLockGroup.Lock(app.Key())
			defer application.AppLockGroup.Unlock(app.Key())

			if axErr := app.LoadDeployments(nil); axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}

			_, axErr, code := app.Start()
			if axErr != nil {
				c.JSON(code, axErr)
				return
			}

			c.JSON(axdb.RestStatusOK, common.NullMap)
			return
		}
	}
}

func StopApplication() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if app, axErr := application.GetLatestApplicationByID(id, false); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if app == nil {
				c.JSON(axdb.RestStatusOK, common.NullMap)
				return
			}

			application.AppLockGroup.Lock(app.Key())
			defer application.AppLockGroup.Unlock(app.Key())

			if axErr := app.LoadDeployments(nil); axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}

			_, axErr, code := app.Stop()
			if axErr != nil {
				c.JSON(code, axErr)
				return
			}

			c.JSON(axdb.RestStatusOK, common.NullMap)
			return
		}
	}
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

		ids := []string{}
		idStr := c.Request.URL.Query().Get("id")
		if idStr != "" {
			ids = strings.Split(idStr, ",")
		}

		ch, axErr := application.GetStatusServiceIdsChannel(c, ids)
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
				application.ClearStatusIdsChannel(c)
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
