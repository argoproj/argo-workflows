package handler

import (
	"applatix.io/axerror"
	"applatix.io/axnc"
	"applatix.io/axnc/dispatcher"
	"applatix.io/common"
	"applatix.io/kafkacl"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"

	"encoding/json"
	"fmt"
	"time"
)

type axSupportHandler struct {
	KafkaConsumer *kafkacl.EventConsumer
}

func NewAxSupportHandler(kafkaAddr string) (*axSupportHandler, *axerror.AXError) {
	var consumerConfig = cluster.NewConfig()
	consumerConfig.Consumer.Fetch.Max = 100
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Return.Errors = true
	kafkaConsumer, axErr := kafkacl.NewEventConsumer(axnc.NameAxSupport, kafkaAddr, axnc.ConsumerGroupAxSupport, axnc.TopicAxSupport, consumerConfig)

	if axErr != nil {
		common.ErrorLog.Printf("Failed to create event consumer (err: %v)", axErr)
		return nil, axErr
	}

	var axSupportHandler = &axSupportHandler{
		KafkaConsumer: kafkaConsumer,
	}

	return axSupportHandler, nil
}

func (handler *axSupportHandler) ProcessEvent(msg *sarama.ConsumerMessage) *axerror.AXError {
	var event *dispatcher.Event = &dispatcher.Event{}

	if json.Unmarshal(msg.Value, event) != nil {
		return nil
	}

	var output string = "Received the following event:\n"
	output += fmt.Sprintf("Event ID: %s\n", event.EventID)
	output += fmt.Sprintf("Trace ID: %s\n", event.TraceID)
	output += fmt.Sprintf("Channel: %s\n", event.Channel)
	output += fmt.Sprintf("Severity: %s\n", event.Severity)
	output += fmt.Sprintf("Timestamp: %s\n", time.Unix(int64(event.Timestamp/1e6), 0))
	output += fmt.Sprintf("Cluster: %s\n", event.Cluster)
	output += fmt.Sprintf("Facility: %s\n", event.Facility)
	output += fmt.Sprintf("Code: %s\n", event.Code)
	output += fmt.Sprintf("Message: %s\n", event.Message)
	output += fmt.Sprint("Detail:\n")
	for k, v := range event.Detail {
		output += fmt.Sprintf("    %s: %s\n", k, v)
	}

	common.InfoLog.Print(output)

	return nil
}
