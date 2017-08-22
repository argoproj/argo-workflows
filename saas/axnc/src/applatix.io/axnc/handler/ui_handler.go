package handler

import (
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axerror"
	"applatix.io/axnc"
	"applatix.io/axnc/dispatcher"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/kafkacl"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"

	"encoding/json"
	"fmt"
	"time"
)

type uiHandler struct {
	KafkaConsumer *kafkacl.EventConsumer
}

func NewUiHandler(axdbAddr, kafkaAddr string) (*uiHandler, *axerror.AXError) {
	var consumerConfig = cluster.NewConfig()
	consumerConfig.Consumer.Fetch.Max = 100
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Return.Errors = true
	kafkaConsumer, axErr := kafkacl.NewEventConsumer(axnc.NameUI, kafkaAddr, axnc.ConsumerGroupUI, axnc.TopicUI, consumerConfig)
	if axErr != nil {
		common.ErrorLog.Printf("Failed to create event consumer (err: %v)", axErr)
		return nil, axErr
	}

	utils.Dbcl = axdbcl.NewAXDBClientWithTimeout(axdbAddr, 30*time.Minute)

	var uiHandler = &uiHandler{
		KafkaConsumer: kafkaConsumer,
	}

	return uiHandler, nil
}

func (handler *uiHandler) constructPayloadFromEvent(event *dispatcher.Event) map[string]interface{} {
	var payload = map[string]interface{}{}
	payload[axnc.Channel] = event.Channel
	payload[axnc.Code] = event.Code
	payload[axnc.Detail] = event.Detail
	payload[axnc.EventID] = event.EventID
	payload[axnc.Facility] = event.Facility
	payload[axnc.Cluster] = event.Cluster
	payload[axnc.Message] = event.Message
	// workaround for cassandra bug where it fails if a column of type set has null value
	if len(event.Recipients) == 0 {
		payload[axnc.Recipients] = []string{""}
	} else {
		payload[axnc.Recipients] = event.Recipients
	}
	payload[axnc.Severity] = event.Severity
	payload[axnc.Timestamp] = event.Timestamp
	payload[axnc.TraceID] = event.TraceID
	payload[axdb.AXDBTimeColumnName] = event.Timestamp
	return payload
}

func (handler *uiHandler) ProcessEvent(msg *sarama.ConsumerMessage) *axerror.AXError {
	var event *dispatcher.Event = &dispatcher.Event{}

	common.InfoLog.Printf("Unmarshaling event body ...")
	if json.Unmarshal(msg.Value, event) != nil {
		common.ErrorLog.Printf("Failed to unmarshal event body, skip")
		return nil
	}

	common.InfoLog.Printf("Preparing event payload ...")
	payload := handler.constructPayloadFromEvent(event)

	common.InfoLog.Printf("Posting event (event_id: %s) to AXDB ...", event.EventID)
	_, axErr := utils.Dbcl.Post(axdb.AXDBAppAXNC, axnc.EventTableName, payload)
	if axErr != nil {
		var message = fmt.Sprintf("Failed to post event (event_id: %s) to AXDB (err: %v)", event.EventID, axErr)
		common.ErrorLog.Print(message)
		return axerror.ERR_AX_INTERNAL.NewWithMessage(message)
	}
	common.InfoLog.Printf("Successfully posted event (event_id: %s) to AXDB ...", event.EventID)

	return nil
}
