package application

import (
	"encoding/json"
	"fmt"
	"time"

	"applatix.io/axamm/deployment"
	"applatix.io/axamm/utils"
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/index"
	"applatix.io/common"
	"applatix.io/restcl"
	"github.com/gocql/gocql"
)

const (
	ApplicationID           = axdb.AXDBUUIDColumnName
	ApplicationAppID        = "application_id"
	ApplicationName         = "name"
	ApplicationDescription  = "description"
	ApplicationStatus       = "status"
	ApplicationStatusDetail = "status_detail"
	ApplicationCtime        = axdb.AXDBTimeColumnName
	ApplicationMtime        = "mtime"
	ApplicationDeployments  = "deployments"
	ApplicationEndpoints    = "endpoints"
	ApplicationJiraIssues   = "jira_issues"

	DeploymentsInit        = "deployments_init"
	DeploymentsWaiting     = "deployments_waiting"
	DeploymentsError       = "deployments_error"
	DeploymentsActive      = "deployments_active"
	DeploymentsTerminating = "deployments_terminating"
	DeploymentsTerminated  = "deployments_terminated"
	DeploymentsStopping    = "deployments_stopping"
	DeploymentsStopped     = "deployments_stopped"
)

const (
	ApplicationLatestTable  = "application_latest"
	ApplicationHistoryTable = "application_history"
)

var appSchema = axdb.Table{
	AppName: axdb.AXDBAppAMM,
	Name:    "",
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ApplicationName:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ApplicationAppID:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ApplicationDescription:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ApplicationStatus:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		ApplicationStatusDetail: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ApplicationMtime:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		ApplicationDeployments:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ApplicationEndpoints:    axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},
		ApplicationJiraIssues:   axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},

		DeploymentsInit:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		DeploymentsWaiting:     axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		DeploymentsError:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		DeploymentsActive:      axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		DeploymentsTerminating: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		DeploymentsTerminated:  axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		DeploymentsStopping:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		DeploymentsStopped:     axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
	ExcludedIndexColumns: map[string]bool{
		ApplicationDeployments: true,
	},
}

func GetLatestApplicationSchema() axdb.Table {
	table := appSchema.Copy()
	table.Name = ApplicationLatestTable
	table.Type = axdb.TableTypeKeyValue
	table.Columns[ApplicationID] = axdb.Column{Type: axdb.ColumnTypeTimeUUID, Index: axdb.ColumnIndexStrong}
	table.Columns[ApplicationCtime] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}

	return table
}

func GetHistoryApplicationSchema() axdb.Table {
	table := appSchema.Copy()
	table.Name = ApplicationHistoryTable
	table.Type = axdb.TableTypeTimeSeries

	return table
}

func (a *Application) CreateObject() (*Application, *axerror.AXError, int) {
	if !common.ValidateKubeObjName(a.Name) {
		return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("The application name %v is invalid: expect the format ^([a-z0-9]([-a-z0-9]*[a-z0-9])?)$.", a.Name), axerror.REST_BAD_REQ
	}

	if len(a.Name) > 63 {
		return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("The application name %v can not be more than 63 characters.", a.Name), axerror.REST_BAD_REQ
	}

	a.ID = common.GenerateUUIDv1()
	a.ApplicationID = common.GenerateUUIDv5(a.Name)
	a.Status = AppStateInit
	a.StatusDetail = map[string]interface{}{}

	a.Ctime = time.Now().UnixNano() / 1e3
	a.Mtime = time.Now().UnixNano() / 1e3
	if axErr := a.createObject(ApplicationLatestTable); axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	utils.InfoLog.Printf("Initialized new %s", a)
	return a, nil, axerror.REST_CREATE_OK
}

func (a *Application) UpdateObject() (*Application, *axerror.AXError, int) {
	a.Mtime = int64(time.Now().UnixNano() / 1e3)

	if axErr := a.updateObject(ApplicationLatestTable); axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	return a, nil, axerror.REST_STATUS_OK
}

func (a *Application) CopyToHistory() (*axerror.AXError, int) {

	if axErr := a.createObject(ApplicationHistoryTable); axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_CREATE_OK
}

func (a *Application) DeleteObject() (*axerror.AXError, int) {

	if axErr := a.deleteObject(ApplicationLatestTable); axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}
	utils.InfoLog.Printf("Deleted %s", a)
	return nil, axerror.REST_CREATE_OK
}

var MaxRetryDuration time.Duration = 15 * time.Minute
var retryConfig *restcl.RetryConfig = &restcl.RetryConfig{
	Timeout:      MaxRetryDuration,
	TriableCodes: []string{axerror.ERR_AXDB_INTERNAL.Code},
}

func (d *Application) updateObject(table string) *axerror.AXError {

	srvMap, axErr := d.createAppMap()
	if axErr != nil {
		return axErr
	}

	_, axErr = utils.DbCl.PutWithTimeRetry(axdb.AXDBAppAMM, table, srvMap, retryConfig)

	utils.DebugLog.Printf("[APP] %v(%v) is updated to state %v %v.\n", d.Name, d.ID, d.Status, d.StatusDetail)

	UpdateETag()

	PostStatusEvent(d.ID, d.Name, d.Status, d.StatusDetail)

	return axErr
}

func (d *Application) createObject(table string) *axerror.AXError {

	srvMap, axErr := d.createAppMap()
	if axErr != nil {
		return axErr
	}

	_, axErr = utils.DbCl.PostWithTimeRetry(axdb.AXDBAppAMM, table, srvMap, retryConfig)

	UpdateETag()

	PostStatusEvent(d.ID, d.Name, d.Status, d.StatusDetail)

	return axErr
}

func (d *Application) deleteObject(table string) *axerror.AXError {

	appMap, axErr := d.createAppMap()
	if axErr != nil {
		return axErr
	}

	_, axErr = utils.DbCl.DeleteWithTimeRetry(axdb.AXDBAppAMM, table, []map[string]interface{}{appMap}, retryConfig)
	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s table failed, err: %v", table, axErr)
		return axErr
	}

	UpdateETag()

	PostStatusEvent(d.ID, d.Name, d.Status, d.StatusDetail)

	return nil
}

func (d *Application) initFromMap(appMap map[string]interface{}, external bool) *axerror.AXError {

	if appMap[axdb.AXDBUUIDColumnName] != nil {
		d.ID = appMap[axdb.AXDBUUIDColumnName].(string)
	}

	if appMap[ApplicationAppID] != nil {
		d.ApplicationID = appMap[ApplicationAppID].(string)
	}

	if appMap[ApplicationName] != nil {
		d.Name = appMap[ApplicationName].(string)
	}

	if appMap[ApplicationDescription] != nil {
		d.Description = appMap[ApplicationDescription].(string)
	}

	if appMap[ApplicationStatus] != nil {
		d.Status = appMap[ApplicationStatus].(string)
	}

	if appMap[ApplicationJiraIssues] != nil {
		for _, jiraObj := range appMap[ApplicationJiraIssues].([]interface{}) {
			d.JiraIssues = append(d.JiraIssues, jiraObj.(string))
		}
	}

	if statusDetailStr, ok := appMap[ApplicationStatusDetail]; ok {
		statusDetail := map[string]interface{}{}
		if statusDetailStr.(string) != "" {
			if err := json.Unmarshal([]byte(statusDetailStr.(string)), &statusDetail); err != nil {
				errMsg := fmt.Sprintf("Failed to unmarshal the status detail string in application:%v", err)
				utils.ErrorLog.Println(errMsg)
				return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
			}
		}
		d.StatusDetail = statusDetail
	}

	if deploymentsStr, ok := appMap[ApplicationDeployments]; ok {
		deployments := []*deployment.Deployment{}
		if deploymentsStr.(string) != "" {
			if err := json.Unmarshal([]byte(deploymentsStr.(string)), &deployments); err != nil {
				errMsg := fmt.Sprintf("Failed to unmarshal the deployments string in application:%v", err)
				utils.ErrorLog.Println(errMsg)
				return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
			}
		}
		d.Deployments = deployments
	}

	if appMap[ApplicationEndpoints] != nil {
		endPoints := appMap[ApplicationEndpoints].([]interface{})
		for _, endPoint := range endPoints {
			d.Endpoints = append(d.Endpoints, endPoint.(string))
		}
	}

	if appMap[axdb.AXDBTimeColumnName] != nil {
		d.Ctime = int64(appMap[axdb.AXDBTimeColumnName].(float64))
	}

	if appMap[ApplicationMtime] != nil {
		d.Mtime = int64(appMap[ApplicationMtime].(float64))
	}

	if appMap[DeploymentsInit] != nil {
		d.DeploymentsInit = int64(appMap[DeploymentsInit].(float64))
	}

	if appMap[DeploymentsWaiting] != nil {
		d.DeploymentsWaiting = int64(appMap[DeploymentsWaiting].(float64))
	}

	if appMap[DeploymentsActive] != nil {
		d.DeploymentsActive = int64(appMap[DeploymentsActive].(float64))
	}

	if appMap[DeploymentsError] != nil {
		d.DeploymentsError = int64(appMap[DeploymentsError].(float64))
	}

	if appMap[DeploymentsStopping] != nil {
		d.DeploymentsStopping = int64(appMap[DeploymentsStopping].(float64))
	}

	if appMap[DeploymentsStopped] != nil {
		d.DeploymentsStopped = int64(appMap[DeploymentsStopped].(float64))
	}

	if appMap[DeploymentsTerminating] != nil {
		d.DeploymentsTerminating = int64(appMap[DeploymentsTerminating].(float64))
	}

	if appMap[DeploymentsTerminated] != nil {
		d.DeploymentsTerminated = int64(appMap[DeploymentsTerminated].(float64))
	}

	if external {
		d.Ctime = d.Ctime / 1e6
		d.Mtime = d.Mtime / 1e6
	}

	return nil
}

var appIndexKeyList []string = []string{
	ApplicationName,
	ApplicationDescription,
	ApplicationStatus,
}

func (d *Application) createAppMap() (map[string]interface{}, *axerror.AXError) {
	if len(d.ID) == 0 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("application doesn't have an id")
	}
	uuid, err := gocql.ParseUUID(d.ID)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Invalid application uuid: " + d.ID)
	}

	srvMap := make(map[string]interface{})

	srvMap[ApplicationID] = d.ID
	srvMap[ApplicationCtime] = uuid.Time().UnixNano() / 1e3

	srvMap[ApplicationAppID] = d.ApplicationID
	srvMap[ApplicationName] = d.Name
	srvMap[ApplicationDescription] = d.Description

	srvMap[ApplicationStatus] = d.Status
	statusDetailBytes, err := json.Marshal(d.StatusDetail)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the status detail object: %v", err))
	}
	srvMap[ApplicationStatusDetail] = string(statusDetailBytes)

	srvMap[ApplicationMtime] = int64(time.Now().UnixNano() / 1e3)

	if d.Deployments != nil && len(d.Deployments) != 0 {
		deploymentsBytes, err := json.Marshal(d.Deployments)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the status detail object: %v", err))
		}
		srvMap[ApplicationDeployments] = string(deploymentsBytes)
	}

	srvMap[ApplicationEndpoints] = d.Endpoints

	srvMap[DeploymentsInit] = d.DeploymentsInit
	srvMap[DeploymentsWaiting] = d.DeploymentsWaiting
	srvMap[DeploymentsActive] = d.DeploymentsActive
	srvMap[DeploymentsError] = d.DeploymentsError
	srvMap[DeploymentsStopping] = d.DeploymentsStopping
	srvMap[DeploymentsStopped] = d.DeploymentsStopped
	srvMap[DeploymentsTerminating] = d.DeploymentsTerminating
	srvMap[DeploymentsTerminated] = d.DeploymentsTerminated

	for _, key := range appIndexKeyList {
		if _, ok := srvMap[key]; ok {
			index.SendToSearchIndexChan("applications", key, srvMap[key].(string))
		}
	}

	return srvMap, nil
}
