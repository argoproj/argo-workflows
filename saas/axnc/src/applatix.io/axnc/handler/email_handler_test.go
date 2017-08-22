package handler

import (
	"applatix.io/axnc/dispatcher"
	"applatix.io/common"

	"gopkg.in/check.v1"

	"fmt"
	"strings"
	"time"
)

func (s *S) TestConstructEmailFromEvent(c *check.C) {
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

	var handler = emailHandler{}

	email := handler.constructEmailFromEvent(&event)
	c.Assert(email.To, check.DeepEquals, recipients)
	c.Assert(email.Subject, check.Equals, fmt.Sprintf("[%s] %s", strings.ToUpper(event.Severity), strings.Title(event.Message)))
}
