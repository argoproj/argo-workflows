package notification_center

import (
	"gopkg.in/check.v1"
)

const (
	testCode = "test"
)

func (s *S) TestProduceSuccessfully(c *check.C) {
	traceId := "traceId"
	recipients := []string{"user@example.com"}
	detail := map[string]interface{}{"k": "v"}
	msg, err := Producer.SendMessage(testCode, traceId, recipients, detail)
	if err != nil {
		c.Logf("message failed with error:%v", err)
	}
	c.Assert(err, check.IsNil)
	c.Assert(string(msg.Code), check.Equals, testCode)
	c.Assert(msg.TraceId, check.Equals, traceId)
	c.Assert(msg.Recipients, check.DeepEquals, recipients)
	c.Assert(msg.Detail, check.DeepEquals, detail)
	c.Assert(msg.Facility, check.Equals, facility)
	c.Assert(err, check.IsNil)
	AssertReceiveEventNotification(c, msg)
}
