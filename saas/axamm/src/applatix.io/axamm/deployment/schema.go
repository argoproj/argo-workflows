package deployment

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"applatix.io/axamm/utils"
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/commit"
	"applatix.io/axops/index"
	"applatix.io/axops/notification"
	"applatix.io/axops/service"
	"applatix.io/common"
	"applatix.io/template"
	"github.com/gocql/gocql"
)

// table schema
const (
	ServiceTemplateName = "template_name"
	ServiceTemplateId   = "template_id"
	ServiceTemplateStr  = "template"
	ServiceArguments    = "arguments"
	ServiceStatus       = "status"
	ServiceName         = "name"
	ServiceDescription  = "description"
	ServiceMem          = "mem"
	ServiceCPU          = "cpu"
	ServiceCostId       = "cost_id"
	ServiceUserName     = "username"
	ServiceUserId       = "user_id"

	ServiceCost   = "cost"
	ServiceTaskId = "task_id"

	ServiceLaunchTime      = "launch_time"
	ServiceEndTime         = "end_time"
	ServiceWaitTime        = "wait_time"
	ServiceAverageWaitTime = "avg_wait_time"
	ServiceRunTime         = "run_time"

	ServiceNotifications = "notifications"
	ServiceCommit        = "commit"

	ServiceLabels       = "labels"
	ServiceAnnotations  = "annotations"
	ServiceStatusDetail = "status_detail"

	ServiceRepo       = "repo"
	ServiceBranch     = "branch"
	ServiceRevision   = "revision"
	ServiceRepoBranch = "repo_branch"

	DeploymentAppGene = "app_generation"
	DeploymentAppName = "app_name"
	DeploymentAppId   = "app_id"
	DeploymentId      = "deployment_id"

	DeploymentFixtures  = "fixtures"
	DeploymentInstances = "instances"
	DeploymentEndpoints = "endpoints"

	ServiceId   = axdb.AXDBUUIDColumnName
	ServiceTime = axdb.AXDBTimeColumnName

	ServiceTerminationPolicy = "termination_policy"
	ServiceJiraIssues        = "jira_issues"

	PreviousDeploymentId = "previous_deployment_id"
)

const (
	DeploymentLatestTable  = "deployment_latest"
	DeploymentHistoryTable = "deployment_history"
)

var deploySchema = axdb.Table{AppName: axdb.AXDBAppAMM, Name: "", Type: axdb.TableTypeTimeSeries, Columns: map[string]axdb.Column{
	ServiceName:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
	ServiceDescription: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},

	DeploymentAppName: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
	DeploymentAppId:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	DeploymentId:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	DeploymentAppGene: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},

	ServiceTemplateName: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceTemplateId:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	ServiceTemplateStr:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceArguments:    axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexStrong},

	ServiceStatus:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceStatusDetail: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},

	ServiceMem:  axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	ServiceCPU:  axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	ServiceCost: axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},

	ServiceCostId:   axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
	ServiceUserName: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceUserId:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceTaskId:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},

	ServiceLaunchTime:      axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceEndTime:         axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceWaitTime:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceAverageWaitTime: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceRunTime:         axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},

	ServiceNotifications: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},

	ServiceCommit:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceRepo:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceBranch:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceRepoBranch: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceRevision:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},

	ServiceLabels:      axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
	ServiceAnnotations: axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},

	DeploymentFixtures:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	DeploymentInstances: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	DeploymentEndpoints: axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},

	ServiceTerminationPolicy: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceJiraIssues:        axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},

	PreviousDeploymentId: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
},
	UseSearch: true,
	ExcludedIndexColumns: map[string]bool{
		DeploymentFixtures:       true,
		DeploymentInstances:      true,
		ServiceTemplateStr:       true,
		ServiceTerminationPolicy: true,
		ServiceNotifications:     true,
	},
}

func GetHistoryDeploymentSchema() axdb.Table {
	table := deploySchema.Copy()
	table.Name = DeploymentHistoryTable
	return table
}

func GetLatestDeploymentSchema() axdb.Table {
	table := deploySchema.Copy()
	table.Name = DeploymentLatestTable
	table.Type = axdb.TableTypeKeyValue
	table.Columns[ServiceId] = axdb.Column{Type: axdb.ColumnTypeTimeUUID, Index: axdb.ColumnIndexStrong}
	table.Columns[ServiceTime] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}

	return table
}

func getObjects(tableName string, params map[string]interface{}, external bool) ([]*Deployment, *axerror.AXError) {

	var fields []string
	if params != nil {
		if params[axdb.AXDBSelectColumns] != nil {
			fields = params[axdb.AXDBSelectColumns].([]string)
			fields = append(fields, axdb.AXDBUUIDColumnName)
			fields = append(fields, axdb.AXDBTimeColumnName)
			fields = append(fields, ServiceTaskId)
			fields = append(fields, ServiceLaunchTime)
			fields = append(fields, ServiceRunTime)
			fields = append(fields, ServiceWaitTime)
			fields = append(fields, ServiceEndTime)
			fields = append(fields, ServiceAverageWaitTime)
			fields = append(fields, ServiceStatus)
			fields = append(fields, ServiceStatusDetail)
			fields = common.DedupFields(fields)
			params[axdb.AXDBSelectColumns] = fields
		}
	}

	resultArray := []map[string]interface{}{}
	axErr := utils.DbCl.GetWithTimeRetry(axdb.AXDBAppAMM, tableName, params, &resultArray, retryConfig)

	if axErr != nil {
		return nil, axErr
	}

	deploymentArray := []*Deployment{}
	for _, resultMap := range resultArray {
		var dpmt Deployment
		axErr := dpmt.initFromMap(resultMap, external)
		if axErr != nil {
			utils.ErrorLog.Println("Error:", axErr)
		}
		deploymentArray = append(deploymentArray, &dpmt)
	}

	sort.Sort(DeploymentSorter(deploymentArray))

	return deploymentArray, axErr
}

var MaxDBRetryDuration time.Duration = 15 * time.Minute

func (d *Deployment) CreateObject(c *service.ServiceContext) *axerror.AXError {
	return d.createObject(c)
}

func (d *Deployment) createObject(c *service.ServiceContext) *axerror.AXError {

	srvMap, axErr := d.createDeploymentMap(c)
	if axErr != nil {
		return axErr
	}

	_, axErr = utils.DbCl.PostWithTimeRetry(axdb.AXDBAppAMM, DeploymentLatestTable, srvMap, retryConfig)

	UpdateETag()

	return axErr
}

func (d *Deployment) UpdateHistoryObject() (*Deployment, *axerror.AXError) {
	srvMap, axErr := d.createDeploymentMap(nil)
	if axErr != nil {
		return nil, axErr
	}

	_, axErr = utils.DbCl.PutWithTimeRetry(axdb.AXDBAppAMM, DeploymentHistoryTable, srvMap, retryConfig)

	UpdateETag()
	return d, axErr
}

func (d *Deployment) UpdateObject() (*Deployment, *axerror.AXError) {
	return d.updateObject()
}

func (d *Deployment) updateObject() (*Deployment, *axerror.AXError) {

	srvMap, axErr := d.createDeploymentMap(nil)
	if axErr != nil {
		return nil, axErr
	}

	_, axErr = utils.DbCl.PutWithTimeRetry(axdb.AXDBAppAMM, DeploymentLatestTable, srvMap, retryConfig)

	if axErr == nil && (d.Status != DeployStateInit) {
		err := d.sendRedisEvent()
		if err != nil {
			utils.ErrorLog.Printf("[RedisEvent]Failed to publish the redis event: %v\n", err)
		}
	}

	go SendHeartBeat(d.ApplicationName)

	UpdateETag()

	return d, axErr
}

func (d *Deployment) DeleteObject() *axerror.AXError {
	return d.deleteObject()
}

func (d *Deployment) deleteObject() *axerror.AXError {
	srvMap, axErr := d.createDeploymentMap(nil)
	if axErr != nil {
		return axErr
	}

	// delete from latest table
	_, axErr = utils.DbCl.DeleteWithTimeRetry(axdb.AXDBAppAMM, DeploymentLatestTable, []map[string]interface{}{srvMap}, retryConfig)
	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s table failed, err: %v", DeploymentLatestTable, axErr)
		return axErr
	}

	UpdateETag()

	return nil
}

func (d *Deployment) sendRedisEvent() *axerror.AXError {
	event := RedisDeploymentResult{
		Id:              d.Id,
		Name:            d.Name,
		ApplicationName: d.ApplicationName,
		Status:          d.Status,
		StatusDetail:    d.StatusDetail,
	}

	if axErr := utils.RedisSaaSCl.RPushWithTTL(RedisDeployUpdate, event.String(), time.Hour*1); axErr != nil {
		return axErr
	} else {
		common.DebugLog.Printf("[RedisEvent]Push the event: %v\n", event.String())
	}

	if d.Status == DeployStateActive {
		if event.StatusDetail != nil {
			event.StatusDetail["code"] = InfoDeployed
		}
	}

	if d.Status == DeployStateWaiting {
		if event.StatusDetail != nil {
			event.StatusDetail["code"] = InfoDeploying
		}
	}

	if d.Status == DeployStateUpgrading {
		if event.StatusDetail != nil {
			event.StatusDetail["code"] = InfoUpgrading
		}
	}

	if axErr := utils.RedisAdcCl.SetWithTTL(fmt.Sprintf(RedisDeployUpKeyTemplate, d.Id), event.String(), time.Hour*8); axErr != nil {
		return axErr
	} else {
		common.DebugLog.Printf("[RedisEvent]Push the event: %v\n", event.String())
	}

	if axErr := utils.RedisAdcCl.RPushWithTTL(fmt.Sprintf(RedisDeployUpListKeyTemplate, d.Id), event.String(), time.Hour*8); axErr != nil {
		return axErr
	} else {
		common.DebugLog.Printf("[RedisEvent]Push the event: %v\n", event.String())
	}

	return nil
}

func (d *Deployment) CopyToHistory() (*axerror.AXError, int) {
	srvMap, axErr := d.createDeploymentMap(nil)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	// move to done table
	_, axErr = utils.DbCl.Put(axdb.AXDBAppAMM, DeploymentHistoryTable, srvMap)
	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s table failed, err: %v", DeploymentHistoryTable, axErr)
		return axErr, axerror.REST_INTERNAL_ERR
	}

	UpdateETag()

	return nil, axerror.REST_STATUS_OK
}

func (d *Deployment) initFromMap(srvMap map[string]interface{}, external bool) *axerror.AXError {

	if srvMap[axdb.AXDBUUIDColumnName] != nil {
		d.Id = srvMap[axdb.AXDBUUIDColumnName].(string)
	}

	if srvMap[PreviousDeploymentId] != nil {
		d.PreviousDeploymentId = srvMap[PreviousDeploymentId].(string)
	}

	if srvMap[ServiceName] != nil {
		d.Name = srvMap[ServiceName].(string)
	}

	if srvMap[ServiceDescription] != nil {
		d.Description = srvMap[ServiceDescription].(string)
	}

	if srvMap[DeploymentAppId] != nil {
		d.ApplicationID = srvMap[DeploymentAppId].(string)
	}

	if srvMap[ServiceTaskId] != nil {
		d.TaskID = srvMap[ServiceTaskId].(string)
	}

	if srvMap[DeploymentAppGene] != nil {
		d.ApplicationGeneration = srvMap[DeploymentAppGene].(string)
	}

	if srvMap[DeploymentAppName] != nil {
		d.ApplicationName = srvMap[DeploymentAppName].(string)
	}

	if srvMap[DeploymentId] != nil {
		d.DeploymentID = srvMap[DeploymentId].(string)
	}

	if srvMap[ServiceJiraIssues] != nil {
		for _, jiraObj := range srvMap[ServiceJiraIssues].([]interface{}) {
			d.JiraIssues = append(d.JiraIssues, jiraObj.(string))
		}
	}

	if srvMap[ServiceTemplateStr] != nil {
		err := json.Unmarshal([]byte(srvMap[ServiceTemplateStr].(string)), &d.Template)
		if err != nil {
			return axerror.ERR_AXDB_INTERNAL.NewWithMessage(fmt.Sprintf("Can't unmarshal template from string: %s", srvMap[ServiceTemplateStr]))
		}
	}
	if srvMap[ServiceTemplateId] != nil {
		d.TemplateID = srvMap[ServiceTemplateId].(string)
	}

	if srvMap[ServiceArguments] != nil {
		d.Arguments = make(template.Arguments)
		args := srvMap[ServiceArguments].(map[string]interface{})
		for argName, argValIf := range args {
			argVal := argValIf.(string)
			d.Arguments[argName] = &argVal
		}
	}

	if srvMap[ServiceStatus] != nil {
		d.Status = srvMap[ServiceStatus].(string)
	}

	if statusDetailStr, ok := srvMap[ServiceStatusDetail]; ok {
		statusDetail := map[string]interface{}{}
		if statusDetailStr.(string) != "" {
			if err := json.Unmarshal([]byte(statusDetailStr.(string)), &statusDetail); err != nil {
				errMsg := fmt.Sprintf("Failed to unmarshal the status detail string in service:%v", err)
				utils.ErrorLog.Println(errMsg)
				return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
			}
		}
		d.StatusDetail = statusDetail
	}

	if srvMap[ServiceMem] != nil {
		d.Mem = srvMap[ServiceMem].(float64)
	}

	if srvMap[ServiceCPU] != nil {
		d.CPU = srvMap[ServiceCPU].(float64)
	}

	if srvMap[ServiceUserName] != nil {
		d.User = srvMap[ServiceUserName].(string)
	}

	if notificationStr, ok := srvMap[ServiceNotifications]; ok {
		notifications := []notification.Notification{}
		if notificationStr.(string) != "" {
			if err := json.Unmarshal([]byte(notificationStr.(string)), &notifications); err != nil {
				errMsg := fmt.Sprintf("Failed to unmarshal the notifications string in service:%v", err)
				utils.ErrorLog.Println(errMsg)
				return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
			}
		}
		d.Notifications = notifications
	}

	if termPolicyStr, ok := srvMap[ServiceTerminationPolicy]; ok {
		termPolicy := template.TerminationPolicy{}
		if termPolicyStr.(string) != "" {
			if err := json.Unmarshal([]byte(termPolicyStr.(string)), &termPolicy); err != nil {
				errMsg := fmt.Sprintf("Failed to unmarshal the termination policy string in service:%v", err)
				utils.ErrorLog.Println(errMsg)
				return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
			}
		}
		d.TerminationPolicy = &termPolicy
	}

	if fixtureStr, ok := srvMap[DeploymentFixtures]; ok {
		fixtures := map[string]map[string]interface{}{}
		if fixtureStr.(string) != "" {
			if err := json.Unmarshal([]byte(fixtureStr.(string)), &fixtures); err != nil {
				errMsg := fmt.Sprintf("Failed to unmarshal the fixtures string in service:%v", err)
				utils.ErrorLog.Println(errMsg)
				return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
			}
		}
		d.Fixtures = fixtures
	}

	if instanceStre, ok := srvMap[DeploymentInstances]; ok {
		instances := []*Pod{}
		if instanceStre.(string) != "" {
			if err := json.Unmarshal([]byte(instanceStre.(string)), &instances); err != nil {
				errMsg := fmt.Sprintf("Failed to unmarshal the instances string in service:%v", err)
				utils.ErrorLog.Println(errMsg)
				return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
			}
		}
		d.Instances = instances
	}

	if commitStr, ok := srvMap[ServiceCommit]; ok {
		commit := commit.ApiCommit{}
		if commitStr.(string) != "" {
			if err := json.Unmarshal([]byte(commitStr.(string)), &commit); err != nil {
				errMsg := fmt.Sprintf("Failed to unmarshal the commit string in service:%v", err)
				utils.ErrorLog.Println(errMsg)
				return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to unmarshal the commit string in service:%v", err))
			}
		}
		commit.Jobs = nil
		d.Commit = &commit
	}

	if srvMap[ServiceCostId] != nil {
		costid := srvMap[ServiceCostId].(map[string]interface{})
		d.CostId = costid
	}

	// time related
	var launchTimeUs int64 = 0
	var waitTimeUs int64 = 0
	var endTimeUs int64 = 0
	var runTimeUs int64 = 0
	var createTimeUs int64 = 0

	if srvMap[axdb.AXDBTimeColumnName] != nil {
		createTimeUs = int64(srvMap[axdb.AXDBTimeColumnName].(float64))
	}

	if srvMap[ServiceLaunchTime] != nil {
		launchTimeUs = int64(srvMap[ServiceLaunchTime].(float64))
	}

	if srvMap[ServiceEndTime] != nil {
		endTimeUs = int64(srvMap[ServiceEndTime].(float64))
	}

	switch d.Status {
	case DeployStateInit:
	case DeployStateUpgrading:
	case DeployStateWaiting:
		if launchTimeUs != 0 {
			waitTimeUs = time.Now().UnixNano()/1e3 - launchTimeUs
		}

		d.Cost = service.GetSpendingCents(d.CPU, d.Mem, float64(runTimeUs))

	case DeployStateActive, DeployStateError:
		if _, ok := srvMap[ServiceWaitTime]; ok {
			waitTimeUs = int64(srvMap[ServiceWaitTime].(float64))
		}
		if launchTimeUs != 0 {
			runTimeUs = time.Now().UnixNano()/1e3 - launchTimeUs - waitTimeUs
		}

		d.Cost = service.GetSpendingCents(d.CPU, d.Mem, float64(runTimeUs))

	case DeployStateTerminated, DeployStateTerminating, DeployStateStopping, DeployStateStopped:
		if _, ok := srvMap[ServiceWaitTime]; ok {
			waitTimeUs = int64(srvMap[ServiceWaitTime].(float64))
		}
		if _, ok := srvMap[ServiceRunTime]; ok {
			runTimeUs = int64(srvMap[ServiceRunTime].(float64))
		}

		if srvMap[ServiceCost] != nil {
			d.Cost = srvMap[ServiceCost].(float64)
		}
	default:
		utils.DebugLog.Println("Unexpected service status:", d.Status)
	}

	if external {
		// external deployment obj uses unit second
		d.CreateTime = createTimeUs / 1e6
		d.EndTime = endTimeUs / 1e6
		d.LaunchTime = launchTimeUs / 1e6
		d.RunTime = runTimeUs / 1e6
		d.WaitTime = waitTimeUs / 1e6
	} else {
		d.CreateTime = createTimeUs
		d.EndTime = endTimeUs
		d.LaunchTime = launchTimeUs
		d.RunTime = runTimeUs
		d.WaitTime = waitTimeUs
	}

	if srvMap[ServiceLabels] != nil {
		d.Labels = map[string]string{}
		labelMap := srvMap[ServiceLabels].(map[string]interface{})
		for key, value := range labelMap {
			d.Labels[key] = value.(string)
		}
	}

	if srvMap[ServiceAnnotations] != nil {
		d.Annotations = map[string]string{}
		annotationMap := srvMap[ServiceAnnotations].(map[string]interface{})
		for key, value := range annotationMap {
			d.Annotations[key] = value.(string)
		}
	}

	if srvMap[DeploymentEndpoints] != nil {
		endPoints := srvMap[DeploymentEndpoints].([]interface{})
		for _, endPoint := range endPoints {
			d.Endpoints = append(d.Endpoints, endPoint.(string))
		}
	}

	return nil
}

var deployIndexKeyList []string = []string{
	ServiceStatus,
	ServiceName,
	ServiceDescription,
	ServiceUserName,
	ServiceRepo,
	ServiceBranch,
	DeploymentAppName,
}

func (d *Deployment) createDeploymentMap(c *service.ServiceContext) (map[string]interface{}, *axerror.AXError) {
	if len(d.Id) == 0 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("service doesn't have an id")
	}
	uuid, err := gocql.ParseUUID(d.Id)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Invalid service uuid: " + d.Id)
	}

	srvMap := make(map[string]interface{})

	srvMap[DeploymentAppId] = d.ApplicationID
	srvMap[DeploymentAppGene] = d.ApplicationGeneration
	srvMap[DeploymentAppName] = d.ApplicationName
	srvMap[DeploymentId] = d.DeploymentID

	srvMap[axdb.AXDBUUIDColumnName] = d.Id
	srvMap[axdb.AXDBTimeColumnName] = uuid.Time().UnixNano() / 1e3

	srvMap[ServiceName] = d.Name
	srvMap[ServiceDescription] = d.Description

	srvMap[ServiceLaunchTime] = d.LaunchTime
	srvMap[ServiceRunTime] = d.RunTime
	srvMap[ServiceWaitTime] = d.WaitTime
	srvMap[ServiceEndTime] = d.EndTime

	if d.Template != nil {
		srvMap[ServiceTemplateName] = d.Template.Name
		srvMap[ServiceTemplateId] = d.Template.ID
		tempBytes, err := json.Marshal(d.Template)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Can't marshal service object to json")
		}
		srvMap[ServiceTemplateStr] = string(tempBytes)
	} else {
		srvMap[ServiceTemplateName] = ""
	}
	srvMap[ServiceArguments] = d.Arguments

	srvMap[ServiceStatus] = d.Status
	statusDetailBytes, err := json.Marshal(d.StatusDetail)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the status detail object: %v", err))
	}
	srvMap[ServiceStatusDetail] = string(statusDetailBytes)

	srvMap[ServiceMem] = d.Mem
	srvMap[ServiceCPU] = d.CPU
	srvMap[ServiceCost] = d.Cost

	srvMap[ServiceCostId] = d.CostId
	srvMap[ServiceUserName] = d.User
	if c != nil {
		srvMap[ServiceUserId] = c.User.ID
	}

	if d.Commit != nil {

		commitBytes, err := json.Marshal(d.Commit)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the commit object: %v", err))
		}
		srvMap[ServiceCommit] = string(commitBytes)

		srvMap[ServiceRepo] = d.Commit.Repo
		srvMap[ServiceBranch] = d.Commit.Branch
		srvMap[ServiceRevision] = d.Commit.Revision

		if d.Commit.Repo != "" && d.Commit.Branch != "" {
			srvMap[ServiceRepoBranch] = d.Commit.Repo + "_" + d.Commit.Branch
		} else {
			srvMap[ServiceRepoBranch] = ""
		}
	}

	notificationsBytes, err := json.Marshal(d.Notifications)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the notifications object: %v", err))
	}
	srvMap[ServiceNotifications] = string(notificationsBytes)

	if d.TerminationPolicy != nil {
		termPolicyBytes, err := json.Marshal(d.TerminationPolicy)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the termination policy object: %v", err))
		}
		srvMap[ServiceTerminationPolicy] = string(termPolicyBytes)
	}

	if d.Fixtures != nil {
		fixturesBytes, err := json.Marshal(d.Fixtures)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the fixtures object: %v", err))
		}
		srvMap[DeploymentFixtures] = string(fixturesBytes)
	}

	instancesBytes, err := json.Marshal(d.Instances)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the instances object: %v", err))
	}
	srvMap[DeploymentInstances] = string(instancesBytes)

	srvMap[ServiceTaskId] = d.TaskID

	if d.Labels != nil {
		srvMap[ServiceLabels] = d.Labels
	} else {
		srvMap[ServiceLabels] = map[string]string{}
	}

	if d.Annotations != nil {
		srvMap[ServiceAnnotations] = d.Annotations
	} else {
		srvMap[ServiceAnnotations] = map[string]string{}
	}

	if len(d.Endpoints) != 0 {
		srvMap[DeploymentEndpoints] = d.Endpoints
	}

	if len(d.PreviousDeploymentId) > 0 {
		srvMap[PreviousDeploymentId] = d.PreviousDeploymentId
	}

	for _, key := range deployIndexKeyList {
		if _, ok := srvMap[key]; ok {
			index.SendToSearchIndexChan("deployments", key, srvMap[key].(string))
		}
	}

	return srvMap, nil
}
