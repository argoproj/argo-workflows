package kafkacl

import (
	"applatix.io/axerror"
	"applatix.io/common"
	"applatix.io/retry"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"runtime/debug"
	"time"
)

type EventConsumer struct {
	Name            string
	consumer        *cluster.Consumer
	closed          bool
	closeChannel    chan int
	brokerAddr      string
	consumerGroupID string
	topic           string
	brokerConfig    *cluster.Config
}

func (ec *EventConsumer) ConsumeEvents(fn func(*sarama.ConsumerMessage) *axerror.AXError, retryConfig *retry.RetryConfig) *axerror.AXError {

	for !ec.closed {
		ec.consumeEventsHelper(fn, retryConfig)
		if ec.closed {
			return nil
		}
		if axErr := ec.initConsumerWithRetry(); axErr != nil {
			return axErr
		}
	}
	return nil
}

func (ec *EventConsumer) consumeEventsHelper(fn func(*sarama.ConsumerMessage) *axerror.AXError, retryConfig *retry.RetryConfig) {

	defer func() {
		if err := ec.consumer.Close(); err != nil {
			common.ErrorLog.Printf("failed to close the consumer connection, Err: %v\n", err)
		} else {
			common.InfoLog.Println("Consumer is closed")
		}
	}()

	liveTicker := time.NewTicker(20 * time.Minute)
	consumerAlive := false
	for {
		select {
		case msg := <-ec.consumer.Messages():
			consumerAlive = true
			ec.processMessage(msg, fn, retryConfig)
			ec.consumer.MarkOffset(msg, ec.Name)
			ec.consumer.CommitOffsets()
		case err := <-ec.consumer.Errors():
			common.ErrorLog.Printf("%s encountered error: %v", ec.Name, err)
			return

		case <-liveTicker.C:
			if !consumerAlive {
				return
			} else {
				consumerAlive = false
			}
		case <-ec.closeChannel:
			return
		}
	}
}

func (ec *EventConsumer) processMessage(msg *sarama.ConsumerMessage, fn func(*sarama.ConsumerMessage) *axerror.AXError, retryConfig *retry.RetryConfig) {

	defer func() {
		if r := recover(); r != nil {
			common.ErrorLog.Println("[Panic]Msg:", msg)
			debug.PrintStack()
			common.ErrorLog.Println("[Panic]Recovered:", r)
		}
	}()

	if retryConfig == nil {
		fn(msg)
	} else {
		retryConfig.Retry(
			func() *axerror.AXError {
				return fn(msg)
			})
	}
}

func (ec *EventConsumer) Close() {
	if !ec.closed {
		ec.closed = true
		ec.closeChannel <- 0
	}
}

func NewEventConsumer(name, brokerAddr, consumerGroupID, topic string, brokerConfig *cluster.Config) (*EventConsumer, *axerror.AXError) {
	eventConsumer := &EventConsumer{
		Name:            name,
		brokerAddr:      brokerAddr,
		consumerGroupID: consumerGroupID,
		topic:           topic,
		brokerConfig:    brokerConfig,
		closed:          false,
		closeChannel:    make(chan int, 100),
	}
	if axErr := eventConsumer.initConsumerWithRetry(); axErr != nil {
		return nil, axErr
	}
	return eventConsumer, nil
}

func (ec *EventConsumer) initConsumerWithRetry() *axerror.AXError {
	var retryConfig = retry.NewRetryConfig(15*60, 1, 60, 2, nil)
	return retryConfig.Retry(
		func() *axerror.AXError {
			if ec.closed {
				return nil
			}
			return ec.initConsumer()
		},
	)
}

func (ec *EventConsumer) initConsumer() *axerror.AXError {
	common.InfoLog.Printf("Creating consumer (host: %s) ...", ec.brokerAddr)
	consumerInstance, err := cluster.NewConsumer([]string{ec.brokerAddr}, ec.consumerGroupID, []string{ec.topic}, ec.brokerConfig)
	if err != nil {
		var message = fmt.Sprintf("Failed to create consumer (err: %v)", err)
		common.ErrorLog.Print(message)
		return axerror.ERR_AX_INTERNAL.NewWithMessage(message)
	} else {
		common.InfoLog.Print("Successfully created consumer")
		ec.consumer = consumerInstance
		return nil
	}
}
