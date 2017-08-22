package handler

import (
	"applatix.io/axerror"
	"applatix.io/axnc"
	"applatix.io/axnc/dispatcher"
	"applatix.io/common"
	"applatix.io/kafkacl"
	"applatix.io/notification_center"
	"applatix.io/restcl"
	"applatix.io/slackcl"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"

	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	OauthTokenCacheRefreshIntervalInSec = 120
)

type SlackHandler struct {
	oauthToken            string
	oauthTokenLastRefresh int64
	kafkaAddr             string
	axopsClient           *restcl.RestClient
	slackClient           *slackcl.SlackClient
	KafkaConsumer         *kafkacl.EventConsumer
}

func NewSlackHandler(kafkaAddr, axopsAddr string) (*SlackHandler, *axerror.AXError) {
	var consumerConfig = cluster.NewConfig()
	consumerConfig.Consumer.Fetch.Max = 100
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Return.Errors = true
	kafkaConsumer, axErr := kafkacl.NewEventConsumer(axnc.NameSlack, kafkaAddr, axnc.ConsumerGroupSlack, axnc.TopicSlack, consumerConfig)
	if axErr != nil {
		common.ErrorLog.Printf("Failed to create event consumer (err: %v)", axErr)
		return nil, axErr
	}

	var slackHandler = &SlackHandler{
		axopsClient:   restcl.NewRestClientWithTimeout(axopsAddr, 30*time.Minute),
		kafkaAddr:     kafkaAddr,
		KafkaConsumer: kafkaConsumer,
	}

	return slackHandler, nil
}

func (handler *SlackHandler) getOauthToken() (string, *axerror.AXError) {

	type GeneralGetResult struct {
		Data []map[string]interface{} `json:"data,omitempty"`
	}
	var tools GeneralGetResult

	params := map[string]interface{}{"type": "slack", "category": "notification"}
	axErr := handler.axopsClient.Get("tools", params, &tools)
	if axErr != nil {
		return "", axErr
	}
	if len(tools.Data) > 0 {
		v, exists := tools.Data[0]["oauth_token"]
		if exists {
			return v.(string), nil
		}
	}
	return "", nil
}

func (handler *SlackHandler) checkOauthToken() *axerror.AXError {
	if (len(handler.oauthToken) < 1) || (handler.oauthTokenLastRefresh+OauthTokenCacheRefreshIntervalInSec <= time.Now().Unix()) {
		tokenFromDB, axErr := handler.getOauthToken()
		if axErr != nil {
			return axErr
		}
		if len(tokenFromDB) < 1 {
			var traceID = common.GenerateUUIDv1()
			var detail = map[string]interface{}{"error": "Oauth token for slack is not configured"}
			_, axErr := notification_center.Producer.SendMessage(
				notification_center.CodeConfigurationNotificationInvalidSlack, traceID, []string{}, detail)
			if axErr != nil {
				common.ErrorLog.Printf("Failed to create event (err: %v)", axErr)
			}
			return axerror.ERR_AX_INTERNAL.NewWithMessage("Oauth token for slack is not configured")
		}
		if tokenFromDB != handler.oauthToken {
			handler.oauthToken = tokenFromDB
			handler.slackClient = slackcl.New(tokenFromDB)
		}
		handler.oauthTokenLastRefresh = time.Now().Unix()
	}
	return nil
}

func (handler *SlackHandler) ConstructMessageFromEvent(event *dispatcher.Event) string {
	var body = fmt.Sprintf(">*Event:* %s\n", event.Message)
	body += fmt.Sprintf(">*Severity:* %s\n", strings.ToUpper(event.Severity))
	body += fmt.Sprintf(">*Channel:* %s\n", event.Channel)
	body += fmt.Sprintf(">*Timestamp:* %v\n", time.Unix(event.Timestamp/1e6, (event.Timestamp%1e6)*1000))
	body += fmt.Sprintf(">*Cluster:* %s\n", event.Cluster)

	if len(event.Detail) > 0 {
		body += fmt.Sprintln(">>>*Details:*  ")
		var keys = []string{}
		for k := range event.Detail {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			body += fmt.Sprintf("    *%s:* %s\n", strings.Title(k), event.Detail[k])
		}

	}
	return body
}

func (handler *SlackHandler) sendMessage(event *dispatcher.Event) *axerror.AXError {
	msg := handler.ConstructMessageFromEvent(event)

	axErr := handler.checkOauthToken()
	if axErr != nil {
		common.ErrorLog.Printf("Failed to check OAuth token (err: %v)", axErr)
		return axErr
	}
	for _, recipient := range event.Recipients {
		if strings.HasSuffix(recipient, "@slack") {
			channel := strings.Split(recipient, "@slack")[0]
			if len(channel) > 0 {
				axErr = handler.slackClient.PostMessageToChannel(channel, msg)
				if axErr != nil {
					common.ErrorLog.Printf("Error while posting event to slack channel: %v. error: %v", channel, axErr)
				}
			}
		} else {
			user, axErr := handler.slackClient.GetUserForEmail(recipient, true)
			if axErr != nil {
				common.ErrorLog.Printf("Error while retrieving slack user for email: %v. error: %v", recipient, axErr)
			} else if len(user) > 0 {
				axErr = handler.slackClient.PostDirectMessage(user, msg)
				if axErr != nil {
					common.ErrorLog.Printf("Error while posting direct message to slack user: %v. error: %v", user, axErr)
				}
			} else {
				common.InfoLog.Printf("Unable to post direct messasge. User with email: %v not found in slack", recipient)
			}
		}

	}
	return nil
}

func (handler *SlackHandler) ProcessEvent(msg *sarama.ConsumerMessage) *axerror.AXError {
	var event *dispatcher.Event = &dispatcher.Event{}

	common.InfoLog.Println("Unmarshaling event body ...")
	if json.Unmarshal(msg.Value, event) != nil {
		var message = "Failed to unmarshal event body, skip"
		common.ErrorLog.Println(message)
		return nil
	}
	return handler.sendMessage(event)
}
