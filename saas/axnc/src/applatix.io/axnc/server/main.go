package main

import (
	"applatix.io/axerror"
	"applatix.io/axnc"
	"applatix.io/axnc/dispatcher"
	"applatix.io/axnc/handler"
	"applatix.io/common"
	"applatix.io/notification_center"
	"applatix.io/retry"

	"flag"
	"fmt"
	"os"
	"sync"
)

func main() {
	common.InitLoggers("AXNC")

	role := flag.String("role", "", "Role of the server: dispatcher | ui_handler | email_handler | slack_handler")
	axdbAddr := flag.String("axdb", "", "URL of AXDB")
	kafkaAddr := flag.String("kafka", "", "URL of Kafka")
	notificationServiceAddr := flag.String("notification", "", "URL of notification service")
	axopsAddr := flag.String("axops", "", "URL of axops")
	eventSkeletonFile := flag.String("skeleton", "", "Absolute path to event skeleton file")
	refreshInterval := flag.Int("interval", 30, "Interval to refresh metadata")
	concurrency := flag.Int("concurrency", 1, "Number of worker threads")

	flag.Parse()

	switch *role {
	case axnc.RoleDispatcher:
		if *axdbAddr == "" {
			fmt.Println("Usage: missing URL of AXDB")
			os.Exit(1)
		}

		if *kafkaAddr == "" {
			fmt.Println("Usage: missing URL of Kafka")
			os.Exit(1)
		}

		if *eventSkeletonFile == "" {
			fmt.Println("Usage: missing path to event skeleton file")
			os.Exit(1)
		}

		runAsDispatcher(*axdbAddr, *kafkaAddr, *eventSkeletonFile, *refreshInterval)
	case axnc.RoleEmailHandler:
		if *kafkaAddr == "" {
			fmt.Println("Usage: missing URL of Kafka")
			os.Exit(1)
		}

		if *notificationServiceAddr == "" {
			fmt.Println("Usage: missing URL of notification service")
			os.Exit(1)
		}

		var wg sync.WaitGroup
		wg.Add(*concurrency)
		for i := 0; i < *concurrency; i++ {
			go runAsEmailHandler(*kafkaAddr, *notificationServiceAddr, &wg)
		}
		wg.Wait()
	case axnc.RoleSlackHandler:
		if *kafkaAddr == "" {
			fmt.Println("Usage: missing URL of Kafka")
			os.Exit(1)
		}

		if *axopsAddr == "" {
			fmt.Println("Usage: missing URL of axops")
			os.Exit(1)
		}

		notification_center.InitProducer(notification_center.FacilityAxNotificationCenter, common.DebugLog, *kafkaAddr)

		var wg sync.WaitGroup
		wg.Add(*concurrency)
		for i := 0; i < *concurrency; i++ {
			go runAsSlackHandler(*kafkaAddr, *axopsAddr, &wg)
		}
		wg.Wait()
	case axnc.RoleUiHandler:
		if *axdbAddr == "" {
			fmt.Println("Usage: missing URL of AXDB")
			os.Exit(1)
		}

		if *kafkaAddr == "" {
			fmt.Println("Usage: missing URL of Kafka")
			os.Exit(1)
		}

		var wg sync.WaitGroup
		wg.Add(*concurrency)
		for i := 0; i < *concurrency; i++ {
			go runAsUiHandler(*axdbAddr, *kafkaAddr, &wg)
		}
		wg.Wait()
	case axnc.RoleAxSupportHandler:
		if *kafkaAddr == "" {
			fmt.Println("Usage: missing URL of Kafka")
			os.Exit(1)
		}

		runAsAxSupportHandler(*kafkaAddr)
	default:
		fmt.Printf("Usage: unrecognized role (%s), please select from [dispatcher | ui_handler | email_handler | slack_handler]", *role)
		os.Exit(1)
	}
}

func runAsDispatcher(axdbAddr, kafkaAddr, eventSkeletonFile string, refreshInterval int) {
	axncDispatcher, axErr := dispatcher.NewDispatcher(axdbAddr, kafkaAddr, eventSkeletonFile)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Unable to start dispatcher: %v", axErr))
		panic(axErr)
	}

	go axncDispatcher.RefreshMetaData(refreshInterval)

	defer func() {
		axncDispatcher.KafkaConsumer.Close()
		axncDispatcher.KafkaProducer.Producer.Close()
	}()

	common.InfoLog.Printf("Starting dispatcher ...")
	axErr = axncDispatcher.KafkaConsumer.ConsumeEvents(axncDispatcher.ProcessEvent, nil)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Kafka consumer failed: %v", axErr))
		panic(axErr)
	}

}

func runAsUiHandler(axdbAddr, kafkaAddr string, wg *sync.WaitGroup) {
	axncUiHandler, axErr := handler.NewUiHandler(axdbAddr, kafkaAddr)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Unable to start ui handler: %v", axErr))
		panic(axErr)
	}

	defer func() {
		axncUiHandler.KafkaConsumer.Close()
		wg.Done()
	}()

	common.InfoLog.Printf("Starting ui handler ...")
	var retryConfig = retry.NewRetryConfig(1*60, 1, 60, 2, nil)
	axErr = axncUiHandler.KafkaConsumer.ConsumeEvents(axncUiHandler.ProcessEvent, retryConfig)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Kafka consumer failed: %v", axErr))
		panic(axErr)
	}
}

func runAsEmailHandler(kafkaAddr, notificationServiceAddr string, wg *sync.WaitGroup) {
	axncEmailHandler, axErr := handler.NewEmailHandler(kafkaAddr, notificationServiceAddr)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Unable to start email handler: %v", axErr))
		panic(axErr)
	}

	defer func() {
		axncEmailHandler.KafkaConsumer.Close()
		wg.Done()
	}()

	common.InfoLog.Printf("Starting email handler ...")
	var retryConfig = retry.NewRetryConfig(1*60, 1, 60, 2, map[string]bool{axerror.ERR_AX_HTTP_CONNECTION.Code: true})
	axErr = axncEmailHandler.KafkaConsumer.ConsumeEvents(axncEmailHandler.ProcessEvent, retryConfig)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Kafka consumer failed: %v", axErr))
		panic(axErr)
	}

}

func runAsSlackHandler(kafkaAddr, axopsAddr string, wg *sync.WaitGroup) {
	axncSlackHandler, axErr := handler.NewSlackHandler(kafkaAddr, axopsAddr)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Unable to start slack handler: %v", axErr))
		panic(axErr)
	}

	defer func() {
		axncSlackHandler.KafkaConsumer.Close()
		wg.Done()
	}()

	common.InfoLog.Printf("Starting slack handler ...")
	var retryConfig = retry.NewRetryConfig(1*60, 1, 60, 2, map[string]bool{axerror.ERR_AX_HTTP_CONNECTION.Code: true})
	axErr = axncSlackHandler.KafkaConsumer.ConsumeEvents(axncSlackHandler.ProcessEvent, retryConfig)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Kafka consumer failed: %v", axErr))
		panic(axErr)
	}
}

func runAsAxSupportHandler(kafkaAddr string) {
	axSupportHandler, axErr := handler.NewAxSupportHandler(kafkaAddr)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Unable to start ax support handler: %v", axErr))
		panic(axErr)
	}

	defer func() {
		axSupportHandler.KafkaConsumer.Close()
	}()

	common.InfoLog.Printf("Starting ax support handler ...")
	axErr = axSupportHandler.KafkaConsumer.ConsumeEvents(axSupportHandler.ProcessEvent, nil)
	if axErr != nil {
		common.ErrorLog.Printf(fmt.Sprintf("Kafka consumer failed: %v", axErr))
		panic(axErr)
	}

}
