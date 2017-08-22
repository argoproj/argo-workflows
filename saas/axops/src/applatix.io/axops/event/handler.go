// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package event

import (
	"encoding/json"
	"fmt"
	"strings"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/container"
	"applatix.io/axops/service"
	"applatix.io/axops/tool"
	"applatix.io/axops/utils"
	"applatix.io/axops/yaml"
	"github.com/Shopify/sarama"
)

func GetContainerUsageHandler() AXEventHandler {
	return func(event *AXEvent) *axerror.AXError {
		var err *axerror.AXError
		_, err = utils.Dbcl.Post(axdb.AXDBAppAXOPS, axdb.AXDBTableContainerUsage, event.Data)
		if err != nil {
			utils.ErrorLog.Printf(err.String())
			return err
		}
		return nil
	}
}

func GetHostUsageHandler() AXEventHandler {
	return func(event *AXEvent) *axerror.AXError {
		var err *axerror.AXError
		hostUsg, ok := event.Data.(map[string]interface{})
		if !ok {
			utils.ErrorLog.Printf("Bad format for host usage: %v", event.Data)
			return nil
		}
		delete(hostUsg, "model")
		_, err = utils.Dbcl.Post(axdb.AXDBAppAXOPS, axdb.AXDBTableHostUsage, hostUsg)
		if err != nil {
			utils.ErrorLog.Printf(err.String())
			return err
		}
		return nil
	}
}

func GetHostHandler() AXEventHandler {
	return func(event *AXEvent) *axerror.AXError {
		var err *axerror.AXError
		if event.Op == "create" {
			_, err = utils.Dbcl.Post(axdb.AXDBAppAXOPS, axdb.AXDBTableHost, event.Data)
		} else if event.Op == "update" {
			_, err = utils.Dbcl.Put(axdb.AXDBAppAXOPS, axdb.AXDBTableHost, event.Data)
		} else if event.Op == "delete" {
			_, err = utils.Dbcl.Delete(axdb.AXDBAppAXOPS, axdb.AXDBTableHost, []interface{}{map[string]interface{}{"id": event.Key}})
		} else {
			utils.ErrorLog.Printf("Event dropped, cannot handle it")
		}
		if err != nil {
			utils.ErrorLog.Printf(err.String())
			return err
		}
		return nil
	}
}

func GetContainerHandler() AXEventHandler {
	processUpdate := func(event *AXEvent) *axerror.AXError {
		ctn, ok := event.Data.(map[string]interface{})
		if !ok {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Bad format for container info: %v", event.Data)
		}
		var serviceID string
		if serviceIDIf, ok := ctn[container.ContainerServiceId]; ok {
			serviceID, _ = serviceIDIf.(string)
		} else {
			utils.DebugLog.Printf("Event did not contain a service ID: %v", ctn)
		}
		var containerID string
		if ctnIDIf, ok := ctn[container.ContainerId]; ok {
			containerID, _ = ctnIDIf.(string)
		} else {
			utils.DebugLog.Printf("Event did not contain a container ID: %v", ctn)
		}

		if msg, axErr := utils.RedisCacheCl.GetString(fmt.Sprintf(service.RedisServiceCtnKey, serviceID, containerID)); axErr == nil {
			if msg == "processed" {
				utils.DebugLog.Printf("[Cache] cache hit for service with id %v, skip update the container %v information to service.\n", serviceID, containerID)
				return nil
			}
		}

		_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, axdb.AXDBTableContainer, container.EventDataToContainer(ctn))
		if axErr != nil {
			return axErr
		}

		return service.HandleServiceContainerInfoUpdate(ctn)
	}

	return func(event *AXEvent) *axerror.AXError {
		var err *axerror.AXError
		if event.Op == "create" || event.Op == "update" {
			err = processUpdate(event)
		} else if event.Op == "delete" {
			_, err = utils.Dbcl.Delete(axdb.AXDBAppAXOPS, axdb.AXDBTableContainer, []interface{}{map[string]interface{}{"id": event.Key}})
		} else {
			utils.ErrorLog.Printf("Event dropped, cannot handle it")
		}
		if err != nil {
			utils.ErrorLog.Printf(err.String())
			return err
		}

		return nil
	}
}

func GetDevopsTaskHandler() AXEventHandler {
	return func(event *AXEvent) *axerror.AXError {
		var err *axerror.AXError
		if event.Op == "status" {
			dataMap := event.Data.(map[string]interface{})
			if origin := dataMap["origin"]; origin != nil && origin.(string) == "axdevops" {
				// private axdevops events, ignore
				return nil
			}

			if dataMap["status"] != nil {
				statusCode := utils.ServiceStatusFailed
				status := dataMap["status"].(string)
				switch status {
				case "COMPLETE":
					if dataMap["result"] != nil {
						result := dataMap["result"].(string)
						if result == "SUCCESS" {
							statusCode = utils.ServiceStatusSuccess
						} else if result == "CANCELLED" {
							statusCode = utils.ServiceStatusCancelled
						} else {
							statusCode = utils.ServiceStatusFailed
						}
					}
				case "SUCCESS":
					statusCode = utils.ServiceStatusSuccess
				case "FAILURE":
					statusCode = utils.ServiceStatusFailed
				case "CANCELLED":
					statusCode = utils.ServiceStatusCancelled
				case "SKIPPED":
					statusCode = utils.ServiceStatusSkipped
				case "WAITING":
					statusCode = utils.ServiceStatusWaiting
				case "RUNNING":
					statusCode = utils.ServiceStatusRunning
				case "RETRY":
					// TODO handle this better later. Now just keep it in running state to avoid messing up the stats
					statusCode = utils.ServiceStatusRunning
				case "PRELIM":
					statusCode = utils.ServiceStatusWaiting
				}

				statusPayload := map[string]interface{}{}
				if dataMap["status_detail"] != nil {
					statusPayload["status_detail"] = dataMap["status_detail"].(map[string]interface{})
				}

				// for static fixture attribute information
				if dataMap["static_fixture_parameter"] != nil {
					statusPayload["static_fixture_parameter"] = dataMap["static_fixture_parameter"].(map[string]interface{})
				}
				if serviceId, ok := dataMap["service_id"]; !ok {
					utils.ErrorLog.Printf("serviceID isn't found in this message, bad format.")
				} else {
					err = service.HandleServiceUpdate(serviceId.(string), statusCode, statusPayload, AxEventProducer, utils.DevopsCl)
				}

			}
		}
		if err != nil {
			utils.ErrorLog.Printf(err.String())
			return err
		}
		return nil
	}
}

func GetMessageHandler() AXEventHandler {
	return func(event *AXEvent) *axerror.AXError {
		if event.Op == "email" {
			utils.InfoLog.Printf("Sending email message to Notifier")
			if _, axErr := utils.AxNotifierCl.Post("email", event.Data); axErr != nil {
				utils.ErrorLog.Printf(axErr.String())
				return axErr
			}
		} else {
			utils.InfoLog.Printf("Unexpected event: %v", event)
		}

		return nil
	}
}

func GetDevopsTemplateHandler() AXEventHandler {
	return func(event *AXEvent) *axerror.AXError {
		utils.InfoLog.Printf("Recieving devops event.Op: %s", event.Op)

		if event.Op == "ci" {
			utils.InfoLog.Printf("Recieving ci event, will redirect to the devops_ci_event topic: %v", event.Topic)
			payload := event.Data.(map[string]interface{})
			for key := range []string{"repo", "branch", "commit"} {
				if _, ok := payload["repo"]; !ok {
					utils.InfoLog.Printf("Key %v is missing for devops_ci_event event, unexpected event: %v", key, event)
					return nil
				}
			}
			key := fmt.Sprintf("%s_%s_%s", payload["repo"], payload["branch"], payload["commit"])

			value, err := json.Marshal(payload)
			if err != nil {
				utils.ErrorLog.Printf("Failed to marshal event before sending devops_ci_event topic: %v", err)
				return axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to marshal event before sending devops_ci_event topic: %v", err)
			}
			_, _, err = AxEventProducer.SendMessage(&sarama.ProducerMessage{
				Topic: TopicDevopsCI,
				Key:   sarama.StringEncoder(key),
				Value: sarama.StringEncoder(value),
			})
			if err != nil {
				utils.ErrorLog.Printf("Failed to redirect event to devops_ci_event topic: %v", err)
				return axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to redirect event to devops_ci_event topic: %v", err)
			}
		} else if event.Op == "update" {
			fmt.Println("event.Key", event.Key)
			slice := strings.Split(event.Key, "$$$$")
			if len(slice) != 2 {
				utils.ErrorLog.Printf("devops_template event's key is not in repo:branch format: %s", event.Key)
				return nil
			} else {
				fmt.Println("slice", slice)
			}
			tool.SCMRWMutex.RLock()
			defer tool.SCMRWMutex.RUnlock()
			if _, ok := tool.ActiveRepos[slice[0]]; ok {
				payload := event.Data.(map[string]interface{})
				if revision, ok := payload["Revision"]; ok {
					axErr := yaml.HandleYamlUpdateEvent(slice[0], slice[1], revision.(string), payload["Content"].([]interface{}))
					if axErr != nil {
						utils.ErrorLog.Printf("Failed to parse devops_templates event, error: %v", axErr)
						return axErr
					}
				} else {
					utils.InfoLog.Printf("Revision is missing, unexpected event: %v", event)
				}

			} else {
				utils.InfoLog.Printf("The repository %s is deleted, drop the event.", slice[0])
			}
		} else {
			utils.InfoLog.Printf("Unexpected event: %v", event)
		}

		return nil
	}
}

func GetNullHandler() AXEventHandler {
	return func(event *AXEvent) *axerror.AXError {
		return nil
	}
}

func GetRepoGCHandler() AXEventHandler {
	return func(event *AXEvent) *axerror.AXError {
		utils.InfoLog.Printf("Recieving garbage collection event: %v", event)
		yaml.GarbageCollectTemplatePolicyProjectFixture()
		return nil
	}
}
