package kafkacl

import (
	"applatix.io/axerror"
	"applatix.io/common"
	"applatix.io/retry"

	"github.com/Shopify/sarama"

	"fmt"
)

type EventProducer struct {
	Name     string
	Producer sarama.SyncProducer
}

func (ep *EventProducer) Close() {
	ep.Producer.Close()
}

func NewEventProducer(name, brokerAddr string, brokerConfig *sarama.Config) (*EventProducer, *axerror.AXError) {
	var retryConfig = retry.NewRetryConfig(15*60, 1, 60, 2, nil)
	var producer sarama.SyncProducer

	axErr := retryConfig.Retry(
		func() *axerror.AXError {
			return initProducer(brokerAddr, brokerConfig, &producer)
		},
	)

	if axErr != nil {
		return nil, axErr
	}

	return &EventProducer{
		Name:     name,
		Producer: producer,
	}, nil
}

func initProducer(brokerAddr string, brokerConfig *sarama.Config, producer *sarama.SyncProducer) *axerror.AXError {
	common.InfoLog.Printf("Creating producer (host: %s) ...", brokerAddr)
	producerInstance, err := sarama.NewSyncProducer([]string{brokerAddr}, brokerConfig)
	if err != nil {
		var message = fmt.Sprintf("Failed to create producer (err: %v)", err)
		common.ErrorLog.Print(message)
		return axerror.ERR_AX_INTERNAL.NewWithMessage(message)
	} else {
		common.InfoLog.Printf("Successfully created producer")
		*producer = producerInstance
		return nil
	}
}
