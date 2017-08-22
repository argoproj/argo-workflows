package main

import (
	"applatix.io/axamm/heartbeat"
	"applatix.io/axerror"
	"applatix.io/common"
	"github.com/gin-gonic/gin"
)

func PostHeartBeat() gin.HandlerFunc {
	return func(c *gin.Context) {

		var hb *heartbeat.HeartBeat
		err := common.GetUnmarshalledBody(c, &hb)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		// Make the heart beat handling to be synchronized for now
		axErr := heartbeat.ProcessHeartBeat(hb)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		}

		c.JSON(axerror.REST_CREATE_OK, common.NullMap)
		return
	}
}
