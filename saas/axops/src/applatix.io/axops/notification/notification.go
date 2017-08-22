// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package notification

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"applatix.io/restcl"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"strings"
)

var DefaultServiceSubjectTemplate = `[{{ .Status }}] {{ .TemplateName }} ({{ .Owner }}/{{ .ShortRepo }} {{ .Branch }} {{ .Committer }} - {{ .ShortRevision }})`

type Notification struct {
	//Type    string   `json:"type,omitempty"`
	Whom []string `json:"whom,omitempty"`
	When []string `json:"when,omitempty"`
}

func packNotificationMessage(op string, payload interface{}) string {
	data := map[string]interface{}{
		"Op": op, "Payload": payload,
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return ""
	} else {
		return string(jsonBytes[:])
	}
}

// Move notify method to user object, since the notification method should be defined by user later
func (n *Notification) Notify(subject, body *string, axeventProducer sarama.SyncProducer, gatewayCl *restcl.RestClient) {
	//TODO: Whom list contains labels, it should be tranlsated to users.
	if len(n.Whom) != 0 {
		utils.DebugLog.Println("Notification recipients:", n.Whom)
		email := NewEmail(n.Whom, subject, body)
		produceMsg := &sarama.ProducerMessage{Topic: "messages", Key: sarama.StringEncoder(utils.GenerateUUIDv1()),
			Value: sarama.StringEncoder(packNotificationMessage("email", email))}
		if _, _, err := axeventProducer.SendMessage(produceMsg); err != nil {
			utils.ErrorLog.Printf(fmt.Sprintf("Failed to send notification event:%v\n", err))
		}
	} else {
		utils.InfoLog.Println("Skip notification, no recipient found.")
	}
}

func (n *Notification) Validate() *axerror.AXError {
	for _, when := range n.When {
		if _, ok := utils.ServiceEventMap[strings.ToLower(when)]; !ok {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("'%s' is not validate option.", when))
		}
	}
	return nil
}

type Email struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Html    bool     `json:"html,omitempty"`
}

func NewEmail(to []string, subject, body *string) *Email {
	return &Email{
		To:      to,
		Subject: *subject,
		Body:    *body,
		Html:    true,
	}
}
