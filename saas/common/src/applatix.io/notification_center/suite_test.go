package notification_center

import (
	"applatix.io/axerror"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"gopkg.in/check.v1"

	"encoding/json"
	"flag"
	"log"
	"os"
	"os/exec"
	"runtime/debug"
	"testing"
	"time"
)

const (
	kafkaUrl = "localhost:9092"
	facility = "eventnotification"
)

var consumer *cluster.Consumer = nil

type S struct{}

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&S{})

func startKafka(c *check.C) {
	//start up zookeeper first
	c.Logf("starting zookeeper")
	exec.Command("export KAFKA_HEAP_OPTS=-Xmx128M -Xms64M").Run()
	cmd := exec.Command("/usr/bin/zookeeper-server-start", "-daemon", "/etc/kafka/zookeeper.properties")
	err := cmd.Run()
	cmd.Start()
	if err != nil {
		fail(c)
	}
	c.Logf("started zookeeper")

	// start up kafka server
	c.Logf("starting kafka")
	exec.Command("export KAFKA_HEAP_OPTS=-Xmx375M -Xms256M").Run()
	cmd = exec.Command("/usr/bin/kafka-server-start", "-daemon", "/etc/kafka/server.properties")
	err = cmd.Run()
	if err != nil {
		fail(c)
	}
	c.Logf("started kafka")
	time.Sleep(10 * time.Second)
}

func (s *S) SetUpSuite(c *check.C) {
	flag.Parse()
	startKafka(c)
	startConsumer(c)

	logger := log.New(os.Stdout, "[eventnotification_test-debug] ", log.Ldate|log.Ltime|log.Lshortfile)
	var err *axerror.AXError
	InitProducer(facility, logger, kafkaUrl)
	c.Assert(err, check.IsNil)
}

func fail(c *check.C) {
	debug.PrintStack()
	c.FailNow()
}

func startConsumer(c *check.C) {
	config := cluster.NewConfig()
	// always consumes message from beginning if there is no committed offset found;
	// otherwise Initial will be ignore and consumption begins with the last committed offset.
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true

	var err error
	consumer, err = cluster.NewConsumer([]string{kafkaUrl}, "test", []string{TopicAxnc}, config)
	if err != nil {
		c.Skip("problem setting up kafka consumer")
	}
}

func AssertReceiveEventNotification(c *check.C, message *EventNotificationMessage) {
	ticker := time.NewTicker(2 * time.Second)
	select {
	case msg := <-consumer.Messages():
		c.Logf("received message:%v", msg)
		received := &EventNotificationMessage{}
		json.Unmarshal(msg.Value, received)
		c.Assert(message, check.DeepEquals, received)
	case <-ticker.C:
		c.Log("no message in 2 secs")
		c.FailNow()
	case err := <-consumer.Errors():
		c.Logf("error in consumer:%v", err)
		c.Skip("problem setting up kafka consumer")
	}
}
