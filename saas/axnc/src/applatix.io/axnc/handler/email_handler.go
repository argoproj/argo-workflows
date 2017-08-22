package handler

import (
	"applatix.io/axerror"
	"applatix.io/axnc"
	"applatix.io/axnc/dispatcher"
	"applatix.io/axops/email"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/kafkacl"
	"applatix.io/restcl"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"

	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type emailHandler struct {
	KafkaConsumer *kafkacl.EventConsumer
}

func NewEmailHandler(kafkaAddr, notificationServiceAddr string) (*emailHandler, *axerror.AXError) {
	var consumerConfig = cluster.NewConfig()
	consumerConfig.Consumer.Fetch.Max = 100
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Return.Errors = true
	kafkaConsumer, axErr := kafkacl.NewEventConsumer(axnc.NameEmail, kafkaAddr, axnc.ConsumerGroupEmail, axnc.TopicEmail, consumerConfig)
	if axErr != nil {
		common.ErrorLog.Printf("Failed to create event consumer (err: %v)", axErr)
		return nil, axErr
	}
	utils.AxNotifierCl = restcl.NewRestClientWithTimeout(notificationServiceAddr, 30*time.Minute)

	var emailHandler = &emailHandler{
		KafkaConsumer: kafkaConsumer,
	}

	return emailHandler, nil
}

func (handler *emailHandler) constructEmailFromEvent(event *dispatcher.Event) *email.Email {
	var mailSubject = fmt.Sprintf("[%s] %s", strings.ToUpper(event.Severity), strings.Title(event.Message))

	var mailBody = ""
	mailBody += emailTop
	mailBody += fmt.Sprintf(emailBodyListTemplate, "Event", event.Message)
	mailBody += fmt.Sprintf(emailBodyListTemplate, "Severity", strings.ToUpper(event.Severity))
	mailBody += fmt.Sprintf(emailBodyListTemplate, "Channel", event.Channel)
	mailBody += fmt.Sprintf(emailBodyListTemplate, "Timestamp", time.Unix(int64(event.Timestamp/1e6), 0))
	mailBody += fmt.Sprintf(emailBodyListTemplate, "Cluster", event.Cluster)
	mailBody += fmt.Sprintf(emailBodyListTemplate, "Detail", "")
	mailBody += emailMiddle
	var keys = []string{}
	for k := range event.Detail {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		var v = event.Detail[k]
		// Construct url
		if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
			v = fmt.Sprintf("<a href='%s'>%s</a>", v, v)
		}
		mailBody += fmt.Sprintf(emailBodyTableTemplate, strings.Title(k), v)
	}
	mailBody += emailBottom

	return &email.Email{
		To:      event.Recipients,
		Subject: mailSubject,
		Html:    true,
		Body:    mailBody,
	}
}

func (handler *emailHandler) ProcessEvent(msg *sarama.ConsumerMessage) *axerror.AXError {
	var event *dispatcher.Event = &dispatcher.Event{}

	common.InfoLog.Printf("Unmarshaling event body ...")
	if json.Unmarshal(msg.Value, event) != nil {
		common.ErrorLog.Printf("Failed to unmarshal event body, skip")
		return nil
	}

	common.InfoLog.Printf("Preparing email ...")
	mail := handler.constructEmailFromEvent(event)

	common.InfoLog.Printf("Sending email (event_id: %s) ...", event.EventID)
	axErr := mail.Send()
	if axErr != nil {
		var message = fmt.Sprintf(fmt.Sprintf("Failed to send event (event_id: %s) via email (err: %v)", event.EventID, axErr))
		common.ErrorLog.Print(message)
		return axerror.ERR_AX_INTERNAL.NewWithMessage(message)
	}
	common.InfoLog.Printf("Successfully sent email (event_id: %s)", event.EventID)

	return nil
}
