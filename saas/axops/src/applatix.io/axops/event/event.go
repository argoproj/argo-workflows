// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package event

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"applatix.io/promcl"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"github.com/gocql/gocql"
	"io/ioutil"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

type AXEvent struct {
	Id    string      `json:"ax_uuid,omitempty"`
	Topic string      `json:"topic,omitempty"`
	Key   string      `json:"key,omitempty"`
	Op    string      `json:"op,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

type AXEventHandler func(*AXEvent) *axerror.AXError

var AxEventProducer sarama.SyncProducer
var died chan int = make(chan int, 100)
var promchannel chan *AXEvent = make(chan *AXEvent)

var handlerMap = map[string]AXEventHandler{
	TopicContainerUsage:  GetContainerUsageHandler(),
	TopicHostUsage:       GetHostUsageHandler(),
	TopicHost:            GetHostHandler(),
	TopicContainer:       GetContainerHandler(),
	TopicDevopsTasks:     GetDevopsTaskHandler(),
	TopicMessage:         GetMessageHandler(),
	TopicDevopsTemplates: GetDevopsTemplateHandler(),
	TopicRepoGC:          GetRepoGCHandler(),

	//TopicBitbucketBranch: GetNullHandler(),
	//TopicBitBucketCommit: GetNullHandler(),
}

func GetHandlerByTopic(topic string) AXEventHandler {
	return handlerMap[topic]
}

// needs to decide whether we really needs UseNumber. If not using bytes and unmarshal
func unmarshalBody(buffer *bytes.Buffer) (interface{}, *axerror.AXError) {
	var data interface{}
	decoder := json.NewDecoder(buffer)
	decoder.UseNumber()
	err := decoder.Decode(&data)
	if err != nil {
		return nil, axerror.ERR_EVENT_INVALID.NewWithMessage(fmt.Sprintf("Can't decode data %s into json, decoder error: %v", buffer.String(), err))
	}
	return data, nil
}

func ProcessConsumerMessage(msg *sarama.ConsumerMessage) {

	topic := msg.Topic
	key := string(msg.Key[:])
	key = strings.TrimRight(strings.TrimLeft(key, "\""), "\"")
	// kafka key has quote at beginning and end of the key
	if len(key) == 0 {
		utils.ErrorLog.Println(fmt.Sprintf("dropping event with empty key: topic = %s, partition = %d, offset = %d ", msg.Topic, msg.Partition, msg.Offset))
		return
	}

	//valBuffer.Read(msg.Value)
	valBuffer := bytes.NewBuffer(msg.Value)
	data, axErr := unmarshalBody(valBuffer)
	if axErr != nil || data == nil {
		utils.InfoLog.Println(fmt.Sprintf("The message body is malformated: topic = %s, partition = %d, offset = %d value = %v", msg.Topic, msg.Partition, msg.Offset, valBuffer))
		//TODO: fail axops or just ignore this illegal message? ignore it at this moment
		return
	}

	uuid := gocql.UUIDFromTime(msg.Timestamp)

	kafkaMsgBody := data.(map[string]interface{})
	if kafkaMsgBody["Op"] == nil {
		utils.InfoLog.Println(fmt.Sprintf("dropping event with empty op: topic = %s, partition = %d, offset = %d", msg.Topic, msg.Partition, msg.Offset))
		return
	}

	op := kafkaMsgBody["Op"].(string)
	eventData := kafkaMsgBody["Payload"]

	// TODO: need to make sure if the eventData is required to Marshalled again here
	// TODO: now we don't.
	event := &AXEvent{Topic: topic, Key: key, Op: op, Data: eventData}
	event.Id = uuid.String()
	utils.InfoLog.Println(fmt.Sprintf("topic %s at partition %d with offset %d received = %v", msg.Topic, msg.Partition, msg.Offset, event))
	defer func() {
		if r := recover(); r != nil {
			utils.ErrorLog.Println("[Panic]Event:", event)
			debug.PrintStack()
			utils.ErrorLog.Println("[Panic]Recovered:", r)
		}
	}()
	//if the topic doesn't exist, just skip it.
	if h, exist := handlerMap[topic]; exist {
		retryCount := 0
		for {
			retryCount++
			err := h(event)
			if err != nil {
				//retry 20 times
				if retryCount <= 20 {
					utils.ErrorLog.Printf("Event Handler for topic %s failed, retry count %d, err: %v", topic, retryCount, err)
					time.Sleep(1 * time.Second)
					continue
				} else {
					break
				}
			} else {
				break
			}
		}
	} else {
		utils.InfoLog.Println(fmt.Sprintf("No event handler defined for topic %s, just skip it.", topic))
	}

	// This is to add the devops events from WFE to another channel
	// This is mainly for Prometheus to actively delete the metrics of user workflows
	if topic == TopicDevopsTasks {
		promchannel <- event
	}
}

// registers a handler.
func RegisterEventHandler(topic string, handler AXEventHandler) {
	handlerMap[topic] = handler
}

func initTable(table axdb.Table) {
	//initialize the processed event table
	count := 0
	for {
		count++
		_, dbErr := utils.Dbcl.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, table)
		if dbErr == nil {
			break
		} else {
			utils.InfoLog.Printf(fmt.Sprintf("failed to create table %s .... tried count %d, error %v", table.Name, count, dbErr))
		}

		//retry 100 times
		if count == 100 {
			panic(fmt.Sprintf("failed to create table %s after retrying 100 times", table.Name))
		}
		time.Sleep(1 * time.Second)
	}
}

func Init(brokerAddr ...string) {
	utils.InfoLog.Printf("Start up new Axevent service")

	var brokerAddrs []string
	if len(brokerAddr) == 0 {
		brokerAddrs = []string{KafkaServiceName}
	} else {
		brokerAddrs = []string{brokerAddr[0]}
	}

	var topics []string
	for k, _ := range handlerMap {
		topics = append(topics, k)
	}

	sarama.Logger = utils.DebugLog
	//initialize producer for notification
	InitProducer(brokerAddrs)
	//initialize consumer group for axevent
	InitConsumerGroup(brokerAddrs, ConsumerGroupID, topics, NumOfConsumers)

	//Start a health check background thread
	go StartHealthCheck(brokerAddrs, ConsumerGroupID, topics)

	//initialize consumer group for prometheus agent
	go ConsumePromchannel()
}

func StartHealthCheck(addrs []string, groupId string, topics []string) {
	utils.InfoLog.Printf("Consumer Health Check started.\n")
	for {
		select {
		case id := <-died:
			utils.InfoLog.Printf("Consumer %d was found to be killed, restart it ...", id)
			go RunConsumerGroupInstance(addrs, groupId, topics, id)
		}
	}
}

func InitConsumerGroup(addrs []string, groupId string, topics []string, numConsumers int) {
	//c := make(chan int, numConsumers*10)
	for i := 0; i < numConsumers; i++ {
		//go RunConsumerGroupInstance(addrs, groupId, topics, i, c)
		go RunConsumerGroupInstance(addrs, groupId, topics, i)
	}

	// wait for all consumers to be ready before we move on
	//for i := 0; i < numConsumers; i++ {
	//	<-c
	//}
	//utils.InfoLog.Printf("All consumers are ready, move on.")
}

func InitProducer(brokerAddrs []string) {
	var err error
	retryCount := 0
	sarama.Logger = utils.DebugLog
	for {
		retryCount++
		AxEventProducer, err = sarama.NewSyncProducer(brokerAddrs, nil)
		if err == nil {
			break
		} else {
			if retryCount < 300 {
				utils.InfoLog.Printf(fmt.Sprintf("failed to create a kafka producer, retrying %d ...", retryCount))
				time.Sleep(1 * time.Second)
			} else {
				utils.ErrorLog.Printf(fmt.Sprintf("failed to create a kafka producer for notification, Err: %v", err))
				os.Exit(1)
			}
		}
	}
}

//func RunConsumerGroupInstance(addrs []string, groupId string, topics []string, id int, c chan int) {
func RunConsumerGroupInstance(addrs []string, groupId string, topics []string, id int) {
	var eventCounter int64 = 0
	var consumer *cluster.Consumer = nil
	var err error
	var countMap map[string][]int64 = make(map[string][]int64)

	for _, topic := range topics {
		countMap[topic] = []int64{0, 0}
	}

	config := cluster.NewConfig()
	// always consumes message from beginning if there is no committed offset found;
	// otherwise Initial will be ignore and consumption begins with the last committed offset.
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true
	retryCount := 0
	for {
		retryCount++
		consumer, err = cluster.NewConsumer(addrs, groupId, topics, config)

		if err != nil {
			if retryCount < 300 {
				utils.InfoLog.Printf(fmt.Sprintf("failed to create a kafka consumer, retrying %d ...", retryCount))
				time.Sleep(1 * time.Second)
			} else {
				utils.ErrorLog.Printf(fmt.Sprintf("failed to create a kafka consumer, Err: %v", err))
				os.Exit(1)
			}
		} else {
			utils.InfoLog.Printf("consumer %d is created successfully!", id)
			break
		}
		//} else {
		//	consumer, err = cluster.NewConsumerFromClient(client, groupId, topics)
		//	if err != nil {
		//		if retryCount < 300 {
		//			utils.InfoLog.Printf(fmt.Sprintf("failed to create a kafka consumer, retrying %d ...", retryCount))
		//			time.Sleep(1 * time.Second)
		//		} else {
		//			utils.ErrorLog.Printf(fmt.Sprintf("failed to create a kafka consumer, Err: %v", err))
		//			os.Exit(1)
		//		}
		//	} else {
		//		utils.InfoLog.Printf("consumer group is created successfully!")
		//		break
		//	}
		//}
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			utils.ErrorLog.Printf(fmt.Sprintf("failed to close the consumer connection, Err: %v", err))
		} else {
			utils.InfoLog.Printf(fmt.Sprintf("Consumer %d is closed, it processed %d messages", id, eventCounter))
		}
	}()

	ticker := time.NewTicker(5 * time.Minute)
	liveticker := time.NewTicker(20 * time.Minute)
	// start the long-running loop to consumer the messages from Kafka
	for {
		select {
		case msg := <-consumer.Messages():
			utils.InfoLog.Printf(fmt.Sprintf("Consumer %d Received message with topic = %s, partition = %d, offset = %d", id, msg.Topic, msg.Partition, msg.Offset))
			ProcessConsumerMessage(msg)
			// need to commit the offset in order to avoid consume the same message multiple time.
			// TODO: if performance is a concern, we can commit once per 10 messages, but each message should be marked.
			//utils.InfoLog.Printf(fmt.Sprintf("Consumer %d prepare to commit the offset = %d for topic %s, partition = %d", id, msg.Offset, msg.Topic, msg.Partition))
			consumer.MarkOffset(msg, "metadata")
			consumer.CommitOffsets()
			//utils.InfoLog.Printf(fmt.Sprintf("Consumer %d finished committing the offset for topic %s, partition = %d", id, msg.Topic, msg.Partition))
			eventCounter++
			if countMap[msg.Topic] != nil {
				countMap[msg.Topic][1]++
			}
		case <-ticker.C:
			utils.InfoLog.Printf(fmt.Sprintf("Consumer %d is still alive", id))
		case <-liveticker.C:
			for t, counts := range countMap {
				if counts[0] == counts[1] {
					utils.InfoLog.Printf(fmt.Sprintf("We think consumer %d is blocked for a while, killing it.", id))
					died <- id
					return
				} else {
					counts[0] = counts[1]
					countMap[t] = counts
				}
			}
			//if eventCounter == preEventCount {
			//	utils.InfoLog.Printf(fmt.Sprintf("Consumer %d is blocked for a while, killing it.", id))
			//	died <- id
			//	return
			//} else {
			//	preEventCount = eventCounter
			//}
		case err := <-consumer.Errors():
			utils.InfoLog.Printf(fmt.Sprintf("Consumer %d runs into err: %v, killing it.", id, err))
			died <- id
			return
		}
	}
}

func ConsumePromchannel() {
	utils.InfoLog.Println("[Prom GC] Start to actively garbage collection of prometheus metrics!!")
	for event := range promchannel {
		if event != nil && event.Op == "status" {
			dataMap := event.Data.(map[string]interface{})
			if origin := dataMap["origin"]; origin != nil && origin.(string) == "axdevops" {
				// private axdevops events, ignore
				continue
			}

			serviceId, ok := dataMap["service_id"]
			if !ok {
				utils.ErrorLog.Printf("[Prom GC] serviceID isn't found in this message, bad format. Event ID: %v", event.Id)
				continue
			}

			rootId, ok := dataMap["root_id"]
			if !ok {
				utils.ErrorLog.Printf("[Prom GC] rootID isn't found in this message, bad format. Event ID: %v", event.Id)
				continue
			}

			// if rootId equals serviceId, this is a status update for workflow not leaf node
			if rootId == serviceId {
				statusCode, ok := dataMap["status"]
				if !ok {
					utils.ErrorLog.Printf("[Prom GC] status isn't found in this message, bad format. Event ID: %v", event.Id)
					continue
				}
				status := statusCode.(string)
				if status == "COMPLETE" || status == "SUCCESS" || status == "FAILURE" || status == "CANCELLED" || status == "SKIPPED" {
					utils.InfoLog.Printf("[Prom GC] Find done workflow %s, start deleting Prometheus metrics", rootId)

					r, axErr := promcl.DeleteVolumeMetric(rootId.(string))
					if axErr != nil {
						utils.ErrorLog.Printf("[Prom GC] Error: %v", axErr)
						continue
					}

					if r.StatusCode == 200 { // OK
						bodyBytes, err2 := ioutil.ReadAll(r.Body)
						if err2 != nil {
							utils.ErrorLog.Printf("[Prom GC] Failed to read delete call return: %v, error: %v", rootId, err2)
							continue
						}
						bodyString := string(bodyBytes)
						utils.InfoLog.Printf("[Prom GC] delete workflows for rootId: %s, return payload: %s", rootId, bodyString)
						r.Body.Close()
					}
				}
			}
		}
	}
}
