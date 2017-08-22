package axops

import (
	"strconv"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/user"
	"github.com/gin-gonic/gin"
)

type RequestsData struct {
	Data []user.SystemRequest `json:"data"`
}

// @Title GetSystemRequests
// @Description Get existing system requests
// @Accept  json
// @Param   type     query    int     false        "Request type eg:Userer invitation(1), email confirmation (2), or password reset (3)"
// @Success 200 {object} user.SystemRequest
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /system_requests
// @Router /system_requests [GET]
func GetSystemRequests() gin.HandlerFunc {
	return func(c *gin.Context) {

		params := make(map[string]interface{})
		typeStr := c.Request.URL.Query().Get("type")
		if typeStr != "" {
			typeInt, err := strconv.ParseInt(typeStr, 10, 64)
			if err != nil {
				c.JSON(axdb.RestStatusInvalid, nullMap)
				return
			}
			params[user.SysReqType] = typeInt
		}

		requests, axErr := user.GetSysReqs(params)
		if axErr != nil {
			c.JSON(axerror.REST_NOT_FOUND, axErr)
			return
		}

		data := RequestsData{Data: requests}
		c.JSON(axerror.REST_STATUS_OK, data)
	}
}
