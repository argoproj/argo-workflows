package handler

import (
	"gopkg.in/check.v1"

	"applatix.io/axnc/dispatcher"
	"applatix.io/common"
	"applatix.io/slackcl"
	"time"
)

func (s *S) TestSlackHandlerProcessEvent(c *check.C) {

	oauthToken := "xoxp-154317291860-154971669639-159819063185-5895f4f3071a4e6147b868505b05ba67"
	handler := SlackHandler{
		oauthToken:            oauthToken,
		slackClient:           slackcl.New(oauthToken),
		oauthTokenLastRefresh: time.Now().Unix(),
	}
	var eventID = common.GenerateUUIDv1()
	var traceID = common.GenerateUUIDv1()
	var timestamp = time.Now().UnixNano() / 1000
	var recipients = []string{"general@slack"}
	var event = dispatcher.Event{
		EventID:    eventID,
		TraceID:    traceID,
		Code:       "job.failure",
		Message:    "Job failed",
		Facility:   "axops.axsys",
		Cluster:    "",
		Channel:    "job",
		Severity:   "warning",
		Timestamp:  timestamp,
		Recipients: recipients,
		Detail:     map[string]string{"key1": "value1", "key2": "value2"},
	}

	c.Logf("msg:%s", handler.ConstructMessageFromEvent(&event))
	handler.sendMessage(&event)
}
