package notification_center

import (
	"applatix.io/axerror"
	"applatix.io/common"

	"github.com/Shopify/sarama"

	"encoding/json"
	"log"
	"sync"
	"time"
)

type EventNotificationMessage struct {
	Id         string                 `json:"event_id"`
	Timestamp  int64                  `json:"timestamp"`
	TraceId    string                 `json:"trace_id"`
	Code       string                 `json:"code"`
	Facility   string                 `json:"facility"`
	Recipients []string               `json:"recipients,omitempty"`
	Detail     map[string]interface{} `json:"detail,omitempty"`
}

type EventNotificationProducer interface {
	SendMessage(code, traceId string, recipients []string, detail map[string]interface{}) (*EventNotificationMessage, *axerror.AXError)
	Close() *axerror.AXError
}

type AXKafkaEventNotificationProducer struct {
	facility        string
	log             *log.Logger
	kafkaBrokerAddr []string
	kafkaProducer   sarama.SyncProducer
	mutex           *sync.Mutex
}

func (p AXKafkaEventNotificationProducer) produceEventNotificationMessage(code, traceId string, recipients []string, detail map[string]interface{}) *EventNotificationMessage {

	entry := &EventNotificationMessage{
		Code:       code,
		Recipients: recipients,
		Detail:     detail,
		Facility:   p.facility,
	}
	// timestamp in microseconds
	entry.Timestamp = time.Now().UnixNano() / 1000
	entry.Id = common.GenerateUUIDv1()
	if len(traceId) > 0 {
		entry.TraceId = traceId
	} else {
		entry.TraceId = entry.Id
	}

	return entry
}

func (p AXKafkaEventNotificationProducer) getKafkaProducer(brokerList []string) (sarama.SyncProducer, *axerror.AXError) {
	if p.kafkaProducer != nil {
		return p.kafkaProducer, nil
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	var producerConfig = sarama.NewConfig()
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokerList, producerConfig)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("failed to create kafka producer due to error: %v", err)
	}
	p.kafkaProducer = producer
	return producer, nil
}

func (p AXKafkaEventNotificationProducer) SendMessage(code, traceId string,
	recipients []string, detail map[string]interface{}) (*EventNotificationMessage, *axerror.AXError) {

	msg := p.produceEventNotificationMessage(code, traceId, recipients, detail)

	kafkaProducer, err := p.getKafkaProducer(p.kafkaBrokerAddr)
	if err != nil {
		return nil, err
	}
	if msgStr, err := json.Marshal(msg); err == nil {
		// send message
		_, _, err = kafkaProducer.SendMessage(&sarama.ProducerMessage{
			Topic: TopicAxnc,
			Key:   sarama.StringEncoder(msg.TraceId),
			Value: sarama.StringEncoder(msgStr),
		})
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to send event notification message:%v due to error:%v", msg, err)
		}
	} else {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("failed to json encode event notification message:%v due to error:%v", msg, err)
	}
	return msg, nil
}

func (p AXKafkaEventNotificationProducer) Close() *axerror.AXError {
	if p.kafkaProducer == nil {
		return nil
	}
	if err := p.kafkaProducer.Close(); err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Unable to close kafka producer due to error:%v", err)
	}
	return nil
}

var Producer EventNotificationProducer

func InitProducer(facility string, log *log.Logger, kafkaBrokerAddr ...string) {
	Producer = &AXKafkaEventNotificationProducer{
		facility:        facility,
		log:             log,
		kafkaBrokerAddr: kafkaBrokerAddr,
		mutex:           &sync.Mutex{},
	}
}
