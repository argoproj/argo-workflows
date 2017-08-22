// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi Slack Config API [/slack]
package axops

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/tool"
	"applatix.io/slackcl"
	"github.com/gin-gonic/gin"
)

type ChannelsData struct {
	Data []string `json:"data"`
}

// @Title GetSlackChannels
// @Description List slack channels
// @Accept  json
// @Success 200 {object} ChannelsData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /slack
// @Router /slack/channels [GET]
func GetSlackChannels() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, axErr := getToken()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		if len(token) == 0 {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}
		slackapi := slackcl.New(token)
		channels, axErr := slackapi.GetChannels()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		resultMap := map[string]interface{}{RestData: channels}
		c.JSON(axdb.RestStatusOK, resultMap)
	}
}

func getToken() (string, *axerror.AXError) {
	tools, axErr := tool.GetToolsByType(tool.TypeSlack)
	if axErr != nil {
		return "", axErr
	}
	if len(tools) == 0 {
		return "", nil
	}
	return tools[0].(*tool.SlackConfig).OauthToken, nil
}
