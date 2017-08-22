package event_test

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/container"
	"applatix.io/axops/event"
	"applatix.io/axops/host"
	"applatix.io/axops/service"
	"applatix.io/axops/usage"
	"encoding/json"
	"fmt"
	"github.com/gocql/gocql"
	"gopkg.in/check.v1"
	"time"
)

const successCode = "ERR_OK"

// returns whether it's successful
func processDelete(c *check.C, app string, table string, payload interface{}) (string, map[string]interface{}) {
	if verbose {
		payloadJson, _ := json.Marshal(payload)
		c.Logf("===> DELETE %s/%s %s", app, table, string(payloadJson))
	}

	resMap, err := axdbClient.Delete(app, table, payload)
	effectiveStatus := successCode
	if err != nil {
		effectiveStatus = err.Code
		c.Logf("%s: %s", err.Code, err.Message)
	}

	if verbose {
		resJson, _ := json.Marshal(resMap)
		c.Logf("<=== %s DELETE %s/%s %s", effectiveStatus, app, table, string(resJson))
	}

	return effectiveStatus, resMap
}

func deleteTable(c *check.C, appName, tableName string) {
	status, _ := processDelete(c, appName, tableName, nil)
	if status != successCode && status != axerror.ERR_AXDB_TABLE_NOT_FOUND.Code {
		c.Logf("delete table got status %s", status)
		// fail(t) don't fail the test for now. There is a cassandra timeout that happens every now and then
		time.Sleep(5)
	}
}

type TestHandlerContext struct {
	key      string
	op       string
	expected int64
}

// For the same event, processing is sequential for the same object
func (s *S) TestAsyncSequential(c *check.C) {

	roughCount := 0
	keyCount := 2
	cm := make(map[string]*TestHandlerContext)
	hc := make([]TestHandlerContext, keyCount)
	for i := 0; i < keyCount; i++ {
		hc[i].key = fmt.Sprintf("key-%d", i)
		hc[i].op = "create"
		hc[i].expected = 0
		cm[hc[i].key] = &hc[i]
	}

	handler := func(event *event.AXEvent) *axerror.AXError {
		v, err := event.Data.(json.Number).Int64()
		c.Assert(err, check.IsNil)

		context := cm[event.Key]
		c.Assert(v, check.Equals, context.expected)
		context.expected++
		roughCount++

		return nil
	}
	// register a handler
	event.RegisterEventHandler("topic", handler)

	// post events for one object and one topic
	eventPerObject := 5
	postOneObject := func(topic string, key string, op string) {
		for j := 0; j < eventPerObject; j++ {
			PostOneEvent(c, topic, key, op, j)
		}
	}

	// sleep 5 second to ensure consumer is ready before messages are produced
	// by default the consumer take the message from broker with the newest offset.
	time.Sleep(5 * time.Second)

	// post events to all the topics
	for _, c := range hc {
		go postOneObject("topic", c.key, c.op)
	}

	current := 0
	totalCount := eventPerObject * keyCount
	for {
		if current == totalCount {
			break
		}
		current = roughCount
		time.Sleep(1 * time.Second)
		continue
	}

	// wait until all are done
	for _, cxt := range hc {
		c.Assert(cxt.expected, check.Equals, int64(eventPerObject))
	}
}

func (s *S) TestHostHandle(c *check.C) {
	deleteTable(c, axdb.AXDBAppAXOPS, axdb.AXDBTableHost)
	_, axErr := axdbClient.Post(axdb.AXDBAppAXDB, axdb.AXDBOpCreateTable, host.HostSchema)
	c.Assert(axErr, check.IsNil)

	h := host.Host{
		ID:        "aaa",
		Name:      "hostname",
		Status:    1,
		PrivateIP: []string{"1.1.1.1"},
		PublicIP:  []string{"2.2.2.2"},
		Mem:       1024.0,
		CPU:       0.5,
		ECU:       0.5,
		Disk:      100101.1,
		Network:   1029.1,
		Model:     "xlarge.x1",
	}

	PostOneEvent(c, event.TopicHost, h.ID, "create", h)
	var hosts []host.Host
	for {
		time.Sleep(1 * time.Second)
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableHost, nil, &hosts)
		c.Assert(axErr, check.IsNil)
		if len(hosts) == 1 {
			c.Assert(hosts[0].Status, check.Equals, int64(1))
			break
		}
	}

	h.Status = 2
	PostOneEvent(c, event.TopicHost, h.ID, "update", h)
	for {
		time.Sleep(1 * time.Second)
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableHost, nil, &hosts)
		c.Assert(axErr, check.IsNil)
		c.Assert(len(hosts), check.Equals, 1)
		if hosts[0].Status == int64(2) {
			break
		}
	}

	PostOneEvent(c, event.TopicHost, h.ID, "delete", nil)
	for {
		time.Sleep(1 * time.Second)
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableHost, nil, &hosts)
		c.Assert(axErr, check.IsNil)
		if len(hosts) == 0 {
			break
		}
	}
	c.Log("TestHostHandle done")
}

func (s *S) TestHostUsageHandle(c *check.C) {
	c.Log("TestHostUsageHandle")
	deleteTable(c, axdb.AXDBAppAXOPS, axdb.AXDBTableHostUsage)
	_, axErr := axdbClient.Post(axdb.AXDBAppAXDB, axdb.AXDBOpCreateTable, usage.HostUsageSchema)
	c.Assert(axErr, check.IsNil)

	hostUsage := usage.HostUsage{
		HostId:         "aaaa",
		HostName:       "ec2-aaaa",
		CPU:            10.0,
		CPUUsed:        10.0,
		CPUTotal:       10.0,
		CPUPercent:     10.0,
		Mem:            10.0,
		MemPercent:     10.0,
		CPURequest:     10.0,
		CPURequestUsed: 10.0,
		MemRequest:     10.0,
	}

	PostOneEvent(c, event.TopicHostUsage, hostUsage.HostId, "blahblah", hostUsage)
	var hostUsages []usage.HostUsage
	for {
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableHostUsage, nil, &hostUsages)
		c.Assert(axErr, check.IsNil)
		if len(hostUsages) == 1 {
			break
		} else {
			time.Sleep(1 * time.Second)
		}
	}
	c.Log("TestHostUsageHandle finished")
}

func (s *S) TestContainerUsageHandle(c *check.C) {
	c.Log("TestContainerUsageHandle")
	deleteTable(c, axdb.AXDBAppAXOPS, axdb.AXDBTableContainerUsage)
	_, axErr := axdbClient.Post(axdb.AXDBAppAXDB, axdb.AXDBOpCreateTable, usage.ContainerUsageSchema)
	c.Assert(axErr, check.IsNil)
	containerUsage := usage.ContainerUsage{
		CostId: map[string]string{
			"App":  "A",
			"Proj": "a",
		},
		HostId:         "aaaa",
		ContainerId:    "bbbb",
		ContainerName:  "container-bbbb",
		CPU:            10.0,
		CPUUsed:        10.0,
		CPUTotal:       10.0,
		CPUPercent:     10.0,
		Mem:            10.0,
		MemPercent:     10.0,
		CPURequest:     10.0,
		CPURequestUsed: 10.0,
		MemRequest:     10.0,
	}

	PostOneEvent(c, event.TopicContainerUsage, containerUsage.ContainerId, "blahblah", containerUsage)
	var containerUsages []usage.ContainerUsage
	for {
		time.Sleep(1 * time.Second)
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableContainerUsage, nil, &containerUsages)
		c.Assert(axErr, check.IsNil)
		if len(containerUsages) == 1 {
			break
		}
	}

	c.Log("TestContainerUsageHandle done")
}

func (s *S) TestContainerHandler(c *check.C) {
	c.Log("TestContainerHandler")

	deleteTable(c, axdb.AXDBAppAXOPS, axdb.AXDBTableContainer)
	_, axErr := axdbClient.Post(axdb.AXDBAppAXDB, axdb.AXDBOpCreateTable, container.ContainerSchema)
	c.Assert(axErr, check.IsNil)

	ctn := container.Container{
		ID:        "aaa",
		Name:      "bbb",
		ServiceId: "0220f0d3-be6a-11e6-8393-0a580af4001d",
		HostId:    "ddd",
		HostName:  "eee",
		CostId: map[string]string{
			"app":  "a",
			"proj": "b",
		},
		Mem: 0.5,
		CPU: 500.5,
	}

	PostOneEvent(c, event.TopicContainer, ctn.ID, "create", ctn)
	var containers []container.Container
	for {
		time.Sleep(1 * time.Second)
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableContainer, nil, &containers)
		c.Assert(axErr, check.IsNil)
		if len(containers) == 1 {
			c.Assert(containers[0].Name, check.Equals, "bbb")
			break
		}
	}

	ctn.Name = "zzz"
	PostOneEvent(c, event.TopicContainer, ctn.ID, "update", ctn)
	for {
		time.Sleep(1 * time.Second)
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableContainer, nil, &containers)
		c.Assert(axErr, check.IsNil)
		c.Assert(len(containers), check.Equals, 1)
		if containers[0].Name == "zzz" {
			break
		}
	}

	PostOneEvent(c, event.TopicContainer, ctn.ID, "delete", nil)
	for {
		time.Sleep(1 * time.Second)
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableContainer, nil, &containers)
		c.Assert(axErr, check.IsNil)
		if len(containers) == 0 {
			break
		}
	}
	c.Log("TestContainerHandler done")
}

var workflowTemplateStr string = `
{
  "type": "service_template",
  "subtype": "workflow",
  "version": "v1",
  "name": "checkout_axdbbuild",
  "dns_name": "",
  "description": "",
  "cost": 0,
  "inputs": {
    "artifacts": null,
    "parameters": {
      "build_service": {
        "description": "hidden",
        "default": "%%service.axdb_build.id%%"
      },
      "commit": {
        "description": "this is the commit",
        "default": "%%session.commit%%"
      },
      "repo": {
        "description": "this is the repo",
        "default": "%%session.repo%%"
      }
    }
  },
  "outputs": {
    "artifacts": {
      "build_result": {
        "service_id": "%%build_service%%",
        "name": "output2"
      }
    }
  },
  "steps": [
    {
      "checkout": {
        "template": {
          "type": "service_template",
          "subtype": "checkout",
          "version": "v1",
          "name": "axcheckout",
          "dns_name": "axcheckout",
          "description": "axcheckout",
          "cost": 0,
          "container": {
            "resources": {
              "mem_mib": 500,
              "cpu_cores": 0.5,
              "disk_gb": 0
            },
            "image": "docker.example.io/%%name_space%%/axcheckout:%%version%%",
            "docker_options": "-v /var/run/docker.sock:/var/run/docker.sock --add-host docker.example.io:52.35.230.80 --add-host docker.local:$(MASTER_IP) -e JENKINS_OPTS=--httpPort=$PORT0 -p $PORT0:$PORT0",
            "command": "",
            "expand": true,
            "once": true
          },
          "inputs": {
            "parameters": {
              "branch": {
                "description": "The branch that we checkout",
                "default": "%%session.branch%%"
              },
              "commit": {
                "description": "The commit revision that we checkout",
                "default": "%%session.commit%%"
              },
              "name_space": {
                "description": "The namespace we get this image from",
                "default": "staging"
              },
              "repo": {
                "description": "The repo that we checkout",
                "default": "%%session.repo%%"
              },
              "scripts": {
                "description": "The checkout scripts that we want to run, separated by semicolons, e.g. dir1/checkout.sh; dir2/checkout.sh",
                "default": "./checkout.sh demo; sleep 1"
              },
              "version": {
                "description": "The version of the docker image",
                "default": "latest"
              }
            }
          },
          "outputs": {
            "artifacts": {
              "logs": {
                "path": "/outputs/logs",
                "archive_mode": "tar",
                "storage_method": "blob"
              },
              "results": {
                "path": "/outputs/artifacts",
                "archive_mode": "tar",
                "storage_method": "blob"
              },
              "source": {
                "path": "/tmp",
                "archive_mode": "tar",
                "storage_method": "blob"
              }
            }
          }
        },
        "parameters": {
          "branch": "",
          "commit": "",
          "name_space": "dogfood",
          "password": "",
          "repo": "",
          "scripts": "echo starting",
          "username": "",
          "version": ""
        },
        "status": 0,
        "cost": 0,
        "launch_time": 0,
        "run_time": 0,
        "average_runtime": 0
      }
    },
    {
      "axdb_build": {
        "template": {
          "type": "service_template",
          "subtype": "custom",
          "version": "v1",
          "name": "buildbuildaxdb",
          "dns_name": "buildbuildaxdb",
          "description": "",
          "cost": 0,
          "container": {
            "resources": {
              "mem_mib": 50,
              "cpu_cores": 0.5,
              "disk_gb": 1
            },
            "image": "docker.example.io/axdb-dev:latest",
            "docker_options": "",
            "command": "/src/saas/axdb/build.sh",
            "expand": false,
            "once": false
          },
          "inputs": {
            "artifacts": [
              {
                "service_id": "%%input_service%%",
                "name": "results",
                "path": "/src"
              }
            ],
            "parameters": {
              "input_service": {
                "description": "The service that checks out the source code into \"results\" output"
              }
            }
          },
          "outputs": {
            "artifacts": {
              "output2": {
                "path": "/src/saas/axdb",
                "excludes": [
                  ""
                ],
                "index": [
                  ""
                ],
                "meta_data": [
                  ""
                ],
                "storage_method": "blob"
              }
            }
          }
        },
        "parameters": {
          "input_service": "%%service.checkout.id%%"
        },
        "status": 0,
        "cost": 0,
        "launch_time": 0,
        "run_time": 0,
        "average_runtime": 0
      }
    }
  ]
}`

func (s *S) TestServiceContainerHandler(c *check.C) {
	c.Log("TestServiceContainerHandler")

	deleteTable(c, axdb.AXDBAppAXOPS, axdb.AXDBTableContainer)
	deleteTable(c, axdb.AXDBAppAXOPS, service.RunningServiceTable)
	_, axErr := axdbClient.Post(axdb.AXDBAppAXDB, axdb.AXDBOpCreateTable, container.ContainerSchema)
	c.Assert(axErr, check.IsNil)

	currentTime := time.Now()
	serviceId := gocql.UUIDFromTime(currentTime).String()

	liveServiceTable := service.ServiceSchema
	liveServiceTable.Name = service.RunningServiceTable
	_, axErr = axdbClient.Post(axdb.AXDBAppAXDB, axdb.AXDBOpCreateTable, liveServiceTable)
	c.Assert(axErr, check.IsNil)

	srvMap := map[string]interface{}{
		axdb.AXDBUUIDColumnName:     serviceId,
		service.ServiceTemplateName: "testTemplateName",
		service.ServiceUserId:       "aaa",
	}

	_, axErr = axdbClient.Put(axdb.AXDBAppAXOPS, service.RunningServiceTable, srvMap)
	c.Assert(axErr, check.IsNil)

	var resultArrray []map[string]interface{}
	axErr = axdbClient.Get(axdb.AXDBAppAXOPS, service.RunningServiceTable, nil, &resultArrray)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(resultArrray), check.Equals, 1)

	ctn := container.Container{
		ID:        "aaa",
		Name:      "bbb",
		ServiceId: serviceId,
		HostId:    "ddd",
		HostName:  "eee",
		CostId: map[string]string{
			"app":  "a",
			"proj": "b",
		},
		Mem: 0.5,
		CPU: 500.5,
	}

	PostOneEvent(c, event.TopicContainer, ctn.ID, "create", ctn)
	var containers []container.Container
	for {
		time.Sleep(1 * time.Second)
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableContainer, nil, &containers)
		c.Assert(axErr, check.IsNil)
		if len(containers) == 1 {
			c.Assert(containers[0].Name, check.Equals, "bbb")
			break
		}
	}

	time.Sleep(5 * time.Second)
	axErr = axdbClient.Get(axdb.AXDBAppAXOPS, service.RunningServiceTable, nil, &resultArrray)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(resultArrray), check.Equals, 1)
	srvMap = resultArrray[0]
	c.Assert(srvMap[service.ServiceContainerId], check.Equals, "aaa")
	c.Assert(srvMap[service.ServiceContainerName], check.Equals, "bbb")
	c.Assert(srvMap[service.ServiceHostId], check.Equals, "ddd")
	c.Assert(srvMap[service.ServiceHostName], check.Equals, "eee")

	currentTime = time.Now()
	serviceId = gocql.UUIDFromTime(currentTime).String()

	srvMap = map[string]interface{}{
		axdb.AXDBUUIDColumnName:     serviceId,
		service.ServiceTemplateName: "testTemplateName",
		service.ServiceCost:         0.6,
		service.ServiceUserId:       "",
	}

	_, axErr = axdbClient.Put(axdb.AXDBAppAXOPS, service.RunningServiceTable, srvMap)
	c.Assert(axErr, check.IsNil)

	params := map[string]interface{}{
		axdb.AXDBUUIDColumnName: serviceId,
	}

	axErr = axdbClient.Get(axdb.AXDBAppAXOPS, service.RunningServiceTable, params, &resultArrray)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(resultArrray), check.Equals, 1)

	ctn = container.Container{
		ID:        "mmm",
		Name:      "nnn",
		ServiceId: serviceId,
		HostId:    "xxx",
		HostName:  "yyy",
		CostId: map[string]string{
			"app":  "a",
			"proj": "b",
		},
		Mem: 0.6,
		CPU: 500.6,
	}

	PostOneEvent(c, event.TopicContainer, ctn.ID, "create", ctn)
	for {
		time.Sleep(1 * time.Second)
		axErr = axdbClient.Get(axdb.AXDBAppAXOPS, service.RunningServiceTable, params, &resultArrray)
		c.Assert(axErr, check.IsNil)
		c.Log(resultArrray)
		if len(resultArrray) == 0 {
			break
		}
	}
	c.Log("TestServiceContainerHandler done")
}
