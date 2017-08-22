package main

import (
	"applatix.io/common"
	"github.com/gin-gonic/gin"
)

func GetRouterAM() *gin.Engine {
	router := gin.Default()
	router.Use(common.ValidateCache)
	router.Use(common.GetGZipHandler())

	v1 := router.Group("v1")
	{
		v1.GET("ping", PingHandler())

		deployments := v1.Group("deployments")
		{
			deployments.GET("", ListDeployments())
			deployments.GET("/:id", GetDeployment())
			deployments.POST("", PostDeployment())
			deployments.PUT("/:id", UpdateDeployment())
			deployments.DELETE("/:id", DeleteDeployment())

			deployments.POST("/:id/start", StartDeployment())
			deployments.POST("/:id/stop", StopDeployment())
			deployments.POST("/:id/scale", ScaleDeployment())
		}

		heartBeats := v1.Group("heartbeats")
		{
			heartBeats.POST("", PostHeartBeat())
		}
	}

	return router
}
