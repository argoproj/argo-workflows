package main

import (
	"applatix.io/axamm/heartbeat"
	"applatix.io/axerror"
	"applatix.io/common"
	"encoding/json"
	"github.com/gin-gonic/gin"
)

func PostHeartBeat() gin.HandlerFunc {
	return func(c *gin.Context) {

		var hb *heartbeat.HeartBeat

		body, err := common.GetBodyString(c)
		if err != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error()))
			return
		}

		err = json.Unmarshal(body, &hb)
		if err != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error()))
			return
		}

		hb.Origin = body

		// Make the heart beat handling to be synchronized for now
		axErr := heartbeat.ProcessHeartBeat(hb)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		c.JSON(axerror.REST_CREATE_OK, common.NullMap)
		return
	}
}

func PingHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(axerror.REST_STATUS_OK, "pong")
		return
	}
}
