package main

import (
	"applatix.io/axerror"
	"applatix.io/common"
	"github.com/gin-gonic/gin"
)

func GetRouterAMM() *gin.Engine {
	router := gin.Default()
	router.Use(common.ValidateCache)
	router.Use(common.GetGZipHandler())

	v1 := router.Group("v1")
	{
		v1.GET("ping", func(c *gin.Context) {
			c.JSON(axerror.REST_STATUS_OK, "pong")
		})

		applications := v1.Group("applications")
		{
			applications.GET("", ListApplications())
			applications.GET("/:id", GetApplication())
			applications.POST("", PostApplication())
			applications.PUT("/:id", PutApplication())
			applications.DELETE("/:id", DeleteApplication())
			applications.POST("/:id/start", StartApplication())
			applications.POST("/:id/stop", StopApplication())
		}

		application := v1.Group("application")
		{
			application.GET("/histories", ListApplicationHistories())
			application.GET("/events", AppEventsHandler())
		}

		deployments := v1.Group("deployments")
		{
			deployments.GET("", ListDeployments())
			deployments.GET("/:id", GetDeployment())

			deployments.GET("/:id/histories", ListDeploymentsHistory())

			// Should forward the call to the AM
			deployments.POST("", PostDeployment())
			deployments.PUT("/:id", UpdateDeployment())
			deployments.DELETE("/:id", DeleteDeployment())

			deployments.POST("/:id/start", StartDeployment())
			deployments.POST("/:id/stop", StopDeployment())
			deployments.POST("/:id/scale", ScaleDeployment())
		}

		deployment := v1.Group("deployment")
		{
			deployment.GET("/histories", ListDeploymentsHistory())
			deployment.GET("/events", DeploymentEventsHandler())
		}

		heartBeats := v1.Group("heartbeats")
		{
			heartBeats.POST("", PostHeartBeat())
		}

	}

	return router
}

func DoNothing() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(axerror.REST_STATUS_OK, common.NullMap)
	}
}
