// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi Sandbox API [/sandbox]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/sandbox"
	"github.com/gin-gonic/gin"
)

type SandboxStatus struct {
	Enabled bool `json:"enabled"`
}

// @Title GetSandboxStatus
// @Description Get sandbox status for the cluster
// @Accept  json
// @Success 200 {object} SandboxStatus
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /sandbox
// @Router /sandbox/status [GET]
func getSandboxStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(axerror.REST_STATUS_OK, SandboxStatus{Enabled: sandbox.IsSandboxEnabled()})
		return
	}
}
