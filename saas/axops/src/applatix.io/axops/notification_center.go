// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi Notification Center API [/notification_center]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axnc"
	"applatix.io/axops/notification_center"
	"applatix.io/axops/utils"

	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
)

type ChannelData struct {
	Data []string `json:"data"`
}

// @Title GetChannels
// @Description Get channels of notification center
// @Accept  json
// @Success 200 {object} ChannelData
// @Resource /notification_center
// @Router /notification_center/channels [GET]
func ChannelListHandler(c *gin.Context) {
	c.JSON(axerror.REST_STATUS_OK, map[string]interface{}{RestData: notification_center.GetChannels()})
}

type SeverityData struct {
	Data []string `json:"data"`
}

// @Title GetSeverities
// @Description Get severities of notification center
// @Accept  json
// @Success 200 {object} SeverityData
// @Resource /notification_center
// @Router /notification_center/severities [GET]
func SeverityListHandler(c *gin.Context) {
	c.JSON(axerror.REST_STATUS_OK, map[string]interface{}{RestData: notification_center.GetSeverities()})
}

type RuleData struct {
	Data []axnc.Rule `json:"data"`
}

// @Title GetRules
// @Description Get rules of notification center
// @Accept  json
// @Param   enabled  	 query   string     false       "Enabled"
// @Success 200 {object} RuleData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /notification_center
// @Router /notification_center/rules [GET]
func RuleListHandler(c *gin.Context) {

	params, axErr := GetContextParams(c,
		[]string{
			"enabled",
		},
		[]string{},
		[]string{},
		[]string{})
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	rules, axErr := notification_center.ListRules(params)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
	} else {
		c.JSON(axerror.REST_STATUS_OK, map[string]interface{}{RestData: rules})
	}
}

// @Title CreateRule
// @Description Create a rule for notification center
// @Accept  json
// @Param   rule	body    axnc.Rule		true        "Rule object"
// @Success 201 {object} axnc.Rule
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /notification_center
// @Router /notification_center/rules [POST]
func RuleCreateHandler(c *gin.Context) {
	var rule notification_center.Rule
	err := utils.GetUnmarshalledBody(c, &rule)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Request body does not include a valid rule object (err: %v)", err)))
		return
	}

	rule.RuleID = ""

	axErr := validateRulePayload(&rule)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	axErr = notification_center.UpdateRule(&rule)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	c.JSON(axerror.REST_CREATE_OK, rule)
	return
}

// @Title UpdateRule
// @Description Update a rule for notification center
// @Accept  json
// @Param   rule	body    axnc.Rule			true        "Rule object"
// @Param   id		path    string     			true        "Rule ID."
// @Success 200 {object} axnc.Rule
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /notification_center
// @Router /notification_center/rules/{id} [PUT]
func RuleUpdateHandler(c *gin.Context, ruleID string) {
	var rule notification_center.Rule
	err := utils.GetUnmarshalledBody(c, &rule)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Request body does not include a valid rule object (err: %v)", err)))
		return
	}

	rule.RuleID = ruleID

	axErr := validateRuleID(rule.RuleID)
	if axErr != nil {
		if axErr.Code == axerror.ERR_API_INVALID_PARAM.Code {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		} else {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
	}

	axErr = validateRulePayload(&rule)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	axErr = notification_center.UpdateRule(&rule)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	c.JSON(axerror.REST_STATUS_OK, rule)
	return
}

// @Title DeleteRule
// @Description Delete a rule for notification center
// @Accept  json
// @Param   id		path    string	true        "Rule id"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /notification_center
// @Router /notification_center/rules/{id} [DELETE]
func RuleDeleteHandler(c *gin.Context, ruleID string) {
	axErr := validateRuleID(ruleID)
	if axErr != nil {
		if axErr.Code == axerror.ERR_API_INVALID_PARAM.Code {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		} else {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
	}

	axErr = notification_center.DeleteRule(ruleID)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	c.JSON(axerror.REST_STATUS_OK, map[string]string{})
	return
}

func validateRulePayload(rule *notification_center.Rule) *axerror.AXError {
	if len(rule.Channels) == 0 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Must specify at least one channel")
	} else {
		var channelMap = map[string]bool{}
		for _, channel := range notification_center.GetChannels() {
			channelMap[channel] = true
		}
		for _, channel := range rule.Channels {
			if _, ok := channelMap[channel]; !ok {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Invalid channel supplied (%s)", channel))
			}
		}
	}

	if len(rule.Severities) == 0 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Must specify at least one severity")
	} else {
		var severityMap = map[string]bool{}
		for _, severity := range notification_center.GetSeverities() {
			severityMap[severity] = true
		}
		for _, severity := range rule.Severities {
			if _, ok := severityMap[severity]; !ok {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Invalid severity supplied (%s)", severity))
			}
		}
	}

	if len(rule.Recipients) == 0 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Must specify at least one recipient")
	}

	if len(rule.Codes) > 0 {
		var codeMap = map[string]bool{}
		codes, axErr := notification_center.GetCodes()
		if axErr != nil {
			return axErr
		}

		for _, code := range codes {
			codeMap[code] = true
		}
		for _, code := range rule.Codes {
			if _, ok := codeMap[code]; !ok {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Invalid code supplied (%s)", code))
			}
		}
	}

	return nil
}

func validateRuleID(ruleID string) *axerror.AXError {
	r, axErr := notification_center.GetRule(ruleID)
	if axErr != nil {
		return axErr
	}

	if r == nil {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Unable to find rule with given ID (%s)", ruleID))
	}

	return nil
}

type EventData struct {
	Data []axnc.EventDetail `json:"data"`
}

// @Title GetEvents
// @Description Get events of notification center
// @Accept  json
// @Param   channel	query	string	false	"Channel"
// @Param   facility	query	string	false	"Facility"
// @Param   severity	query	string	false	"Severity"
// @Param   trace_id	query	string	false	"Trace ID"
// @Param   recipient	query	string	false	"Recipient"
// @Param   ordering	query	string	false	"Ordering"
// @Param   order_by	query	string	false	"Order By"
// @Param   min_time	query	int	false	"Min Time"
// @Param   max_time	query	int	false	"Max Time"
// @Param   limit	query	int	false	"Limit"
// @Param   offset      query   int     false   "Offset."
// @Success 200 {object} EventData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /notification_center
// @Router /notification_center/events [GET]
func EventListHandler(c *gin.Context) {

	var channel = c.Request.URL.Query().Get("channel")
	var facility = c.Request.URL.Query().Get("facility")
	var severity = c.Request.URL.Query().Get("severity")
	var traceID = c.Request.URL.Query().Get("trace_id")
	var recipient = c.Request.URL.Query().Get("recipient")
	var ordering = c.Request.URL.Query().Get("ordering")
	var orderBy = c.Request.URL.Query().Get("order_by")
	var minTimeString = c.Request.URL.Query().Get("min_time")
	var maxTimeString = c.Request.URL.Query().Get("max_time")
	var limitString = c.Request.URL.Query().Get("limit")
	var offsetString = c.Request.URL.Query().Get("offset")

	if orderBy != "" && orderBy != axnc.Timestamp {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Ordering by fields other than 'timestamp' is currently NOT supported"))
		return
	}
	if ordering != "" && ordering != notification_center.Ascending && ordering != notification_center.Descending {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(
			fmt.Sprintf("Value of ordering must be either '%s' or '%s'", notification_center.Ascending, notification_center.Descending)))
		return
	}

	var minTime, maxTime, limit, offset int64
	var err error

	if minTimeString != "" {
		minTime, err = strconv.ParseInt(minTimeString, 10, 64)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Value of min_time must be integer"))
			return
		}
		if minTime < 0 {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Value of min_time must be positive"))
			return
		}
	}

	if maxTimeString != "" {
		maxTime, err = strconv.ParseInt(maxTimeString, 10, 64)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Value of max_time must be integer"))
			return
		}
		if maxTime < 0 {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Value of max_time must be positive"))
			return
		}
	}

	if limitString != "" {
		limit, err = strconv.ParseInt(limitString, 10, 64)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Value of limit must be integer"))
			return
		}
		if limit < 0 {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Value of limit must be positive"))
			return
		}
	}

	if offsetString != "" {
		offset, err = strconv.ParseInt(offsetString, 10, 64)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Value of offset must be integer"))
			return
		}
		if offset < 0 {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Value of offset must be positive"))
			return
		}
	}

	events, axErr := notification_center.GetEvents(channel, facility, severity, traceID, recipient, ordering, orderBy, minTime*1e6, maxTime*1e6, limit, offset)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	} else {
		c.JSON(axerror.REST_STATUS_OK, map[string]interface{}{RestData: events})
		return
	}
}

// @Title MarkEventAsRead
// @Description Mark an event as read
// @Accept  json
// @Param   id     	 path    string     true        "ID of event"
// @Success 200 {object} axnc.EventDetail
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /notification_center
// @Router /notification_center/events/{id}/read [PUT]
func MarkEventAsRead(c *gin.Context) {

	id := c.Param("id")
	e, axErr := notification_center.GetEvent(id)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}
	if e == nil {
		c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
		return
	}

	e.AcknowledgedTime = time.Now().Unix()
	u := GetContextUser(c)
	e.AcknowledgedBy = u.Username

	axErr = notification_center.UpdateEvent(e)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	c.JSON(axerror.REST_STATUS_OK, e)
	return

}
