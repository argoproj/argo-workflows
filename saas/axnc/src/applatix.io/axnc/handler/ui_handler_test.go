package handler

import (
	"applatix.io/axnc/dispatcher"
	"applatix.io/common"

	"gopkg.in/check.v1"

	"time"
)

func (s *S) TestUiHandlerProcessEvent(c *check.C) {
	var eventID = common.GenerateUUIDv1()
	var traceID = common.GenerateUUIDv1()
	var timestamp = time.Now().Unix()
	var recipients = []string{"admin@internal"}

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
		Detail:     map[string]string{},
	}

	var handler = uiHandler{}

	payload := handler.constructPayloadFromEvent(&event)

	c.Assert(payload["event_id"].(string), check.Equals, eventID)
	c.Assert(payload["timestamp"].(int64), check.Equals, timestamp)
}

func (s *S) TestUiHandlerNoRecipients(c *check.C) {
	var eventID = common.GenerateUUIDv1()
	var traceID = common.GenerateUUIDv1()
	var timestamp = time.Now().Unix()
	var recipients = []string{}

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
		Detail:     map[string]string{},
	}

	var handler = uiHandler{}

	payload := handler.constructPayloadFromEvent(&event)

	c.Assert(payload["event_id"].(string), check.Equals, eventID)
	c.Assert(payload["timestamp"].(int64), check.Equals, timestamp)
	c.Assert(payload["recipients"].([]string), check.DeepEquals, []string{""})
}
