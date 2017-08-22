package service_test

import (
	"applatix.io/axops/notification"
	"applatix.io/axops/service"
	"bytes"
	"gopkg.in/check.v1"
	"text/template"
)

func (s *S) TestDefaultServiceSubjectTemplate(c *check.C) {
	c.Log("TestDefaultServiceSubjectTemplate")
	summary := service.Summary{
		Status:       "a",
		TemplateName: "a",
		Owner:        "a",
		Repo:         "a",
		Branch:       "a",
		Revision:     "aaaaaaaaaaaaaaaaaaaa",
	}

	var subject bytes.Buffer
	t, err := template.New("ServiceNotificationSubject").Parse(notification.DefaultServiceSubjectTemplate)
	c.Assert(err, check.IsNil)
	err = t.Execute(&subject, summary)
	c.Assert(err, check.IsNil)
}
