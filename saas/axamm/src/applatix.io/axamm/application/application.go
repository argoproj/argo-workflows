package application

import (
	"fmt"
	"sync"
	"time"

	"applatix.io/axamm/deployment"
	"applatix.io/axamm/utils"
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/common"
	"applatix.io/restcl"
)

type Application struct {
	ID            string                   `json:"id,omitempty"`
	ApplicationID string                   `json:"application_id,omitempty"`
	Name          string                   `json:"name,omitempty"`
	Description   string                   `json:"description,omitempty"`
	Status        string                   `json:"status,omitempty"`
	StatusDetail  map[string]interface{}   `json:"status_detail,omitempty"`
	Ctime         int64                    `json:"ctime,omitempty"`
	Mtime         int64                    `json:"mtime,omitempty"`
	Deployments   []*deployment.Deployment `json:"deployments,omitempty"`
	Endpoints     []string                 `json:"endpoints,omitempty"`
	JiraIssues    []string                 `json:"jira_issues"`

	DeploymentsInit        int64 `json:"deployments_init"`
	DeploymentsWaiting     int64 `json:"deployments_waiting"`
	DeploymentsError       int64 `json:"deployments_error"`
	DeploymentsActive      int64 `json:"deployments_active"`
	DeploymentsTerminating int64 `json:"deployments_terminating"`
	DeploymentsTerminated  int64 `json:"deployments_terminated"`
	DeploymentsStopping    int64 `json:"deployments_stopping"`
	DeploymentsStopped     int64 `json:"deployments_stopped"`
	DeploymentsUpgrading   int64 `json:"deployments_upgrading"`
}

// String is the string representation of the application
func (a *Application) String() string {
	return fmt.Sprintf("Application %s (ID: %s) (ApplicationID: %s)", a.Name, a.ID, a.ApplicationID)
}

// Reachable quickly pings the AM to see if it is alive and responding
func (a *Application) Reachable() bool {
	amClient := restcl.NewRestClientWithTimeout(fmt.Sprintf("http://axam.%v:8968/v1", a.Name), time.Second*3)
	data := map[string]interface{}{}
	retryConfig := restcl.RetryConfig{
		Timeout: 7 * time.Second,
	}
	axErr, _ := amClient.GetWithTimeRetry("ping", nil, &data, &retryConfig)
	return axErr == nil
}

// AgeSeconds returns the age of the app in seconds (based on Ctime)
func (a *Application) AgeSeconds() int64 {
	now := time.Now().Unix()
	return now - int64(a.Ctime/1e6)
}

func (a *Application) Key() string {
	return a.Name
}

func GetLatestApplications(params map[string]interface{}, external bool) ([]*Application, *axerror.AXError) {
	apps, axErr := getApplications(ApplicationLatestTable, params, external)
	if axErr != nil {
		return nil, axErr
	}

	return apps, nil
}

func GetHistoryApplications(params map[string]interface{}, external bool) ([]*Application, *axerror.AXError) {
	apps, axErr := getApplications(ApplicationHistoryTable, params, external)
	if axErr != nil {
		return nil, axErr
	}

	return apps, nil
}

func getApplications(tableName string, params map[string]interface{}, external bool) ([]*Application, *axerror.AXError) {

	var fields []string
	if params != nil {
		if params[axdb.AXDBSelectColumns] != nil {
			fields = params[axdb.AXDBSelectColumns].([]string)
			fields = append(fields, axdb.AXDBUUIDColumnName)
			fields = append(fields, axdb.AXDBTimeColumnName)
			fields = common.DedupFields(fields)
			params[axdb.AXDBSelectColumns] = fields
		}
	}

	resultArray := []map[string]interface{}{}
	axErr := utils.DbCl.GetWithTimeRetry(axdb.AXDBAppAMM, tableName, params, &resultArray, retryConfig)

	if axErr != nil {
		return nil, axErr
	}

	appArr := []*Application{}
	for _, resultMap := range resultArray {
		var app Application
		app.initFromMap(resultMap, external)
		appArr = append(appArr, &app)
	}

	return appArr, axErr
}

func GetLatestApplicationByName(name string, external bool) (*Application, *axerror.AXError) {

	applications, axErr := GetLatestApplications(map[string]interface{}{
		ApplicationName: name,
	}, external)

	if axErr != nil {
		return nil, axErr
	}

	if len(applications) == 0 {
		return nil, nil
	}

	application := applications[0]
	return application, nil
}

func GetLatestApplicationByID(id string, external bool) (*Application, *axerror.AXError) {

	applications, axErr := GetLatestApplications(map[string]interface{}{
		ApplicationID: id,
	}, external)

	if axErr != nil {
		return nil, axErr
	}

	if len(applications) == 0 {
		return nil, nil
	}

	application := applications[0]
	return application, nil
}

func GetLatestApplicationByAppID(id string, external bool) (*Application, *axerror.AXError) {

	applications, axErr := GetLatestApplications(map[string]interface{}{
		ApplicationAppID: id,
	}, external)

	if axErr != nil {
		return nil, axErr
	}

	if len(applications) == 0 {
		return nil, nil
	}

	application := applications[0]
	return application, nil
}

func GetHistoryApplicationByID(id string, external bool) (*Application, *axerror.AXError) {

	applications, axErr := GetHistoryApplications(map[string]interface{}{
		ApplicationID: id,
	}, external)

	if axErr != nil {
		return nil, axErr
	}

	if len(applications) == 0 {
		return nil, nil
	}

	application := applications[0]
	return application, nil
}

func (a *Application) LoadDeployments(wg *sync.WaitGroup) *axerror.AXError {
	deployments, axErr := deployment.GetLatestDeploymentsByApplication(a.Name, true)
	if axErr != nil {
		if wg != nil {
			wg.Done()
		}
		return axErr
	}
	a.Deployments = deployments

	if wg != nil {
		wg.Done()
	}
	return nil
}
