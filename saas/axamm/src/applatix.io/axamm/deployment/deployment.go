package deployment

import (
	"fmt"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/commit"
	"applatix.io/axops/notification"
	"applatix.io/axops/service"
	"applatix.io/template"
)

type Deployment struct {
	Base

	TaskID                string   `json:"task_id,omitempty"`
	ApplicationGeneration string   `json:"app_generation,omitempty"`
	ApplicationID         string   `json:"app_id,omitempty"`
	ApplicationName       string   `json:"app_name,omitempty"`
	DeploymentID          string   `json:"deployment_id,omitempty"`
	JiraIssues            []string `json:"jira_issues"`

	PreviousDeploymentId string                            `json:"previous_deployment_id,omitempty"`
	Fixtures             map[string]map[string]interface{} `json:"fixtures,omitempty"`
	Instances            []*Pod                            `json:"instances"`
	Endpoints            []string                          `json:"endpoints"`
}

type Base struct {
	Id                string                              `json:"id,omitempty"`
	Name              string                              `json:"name,omitempty"`
	Description       string                              `json:"description,omitempty"`
	CostId            map[string]interface{}              `json:"costid,omitempty"`
	Template          *service.EmbeddedDeploymentTemplate `json:"template,omitempty"`
	TemplateID        string                              `json:"template_id,omitempty"`
	Arguments         template.Arguments                  `json:"arguments,omitempty"`
	Status            string                              `json:"status"`
	StatusDetail      map[string]interface{}              `json:"status_detail,omitempty"`
	Mem               float64                             `json:"mem,omitempty"`
	CPU               float64                             `json:"cpu,omitempty"`
	Cost              float64                             `json:"cost"`
	User              string                              `json:"user,omitempty"`
	Notifications     []notification.Notification         `json:"notifications,ommitempty"`
	CreateTime        int64                               `json:"create_time"`
	LaunchTime        int64                               `json:"launch_time"`
	EndTime           int64                               `json:"end_time"`
	WaitTime          int64                               `json:"wait_time"`
	RunTime           int64                               `json:"run_time"`
	Commit            *commit.ApiCommit                   `json:"commit,ommitempty"`
	Labels            map[string]string                   `json:"labels"`
	Annotations       map[string]string                   `json:"annotations"`
	TerminationPolicy *template.TerminationPolicy         `json:"termination_policy,omitempty"`
}

// String is the string representation of the deployment
func (d *Deployment) String() string {
	return fmt.Sprintf("Deployment %s (ID: %s) (DeploymentID: %s)", d.Name, d.Id, d.DeploymentID)
}

func (d *Deployment) Key() string {
	return d.DeploymentID
}

func (d *Deployment) HeartBeatKey() string {
	return d.Id
}

func GetLatestDeployments(params map[string]interface{}, external bool) ([]*Deployment, *axerror.AXError) {
	return getObjects(DeploymentLatestTable, params, external)
}

func GetHistoryDeployments(params map[string]interface{}, external bool) ([]*Deployment, *axerror.AXError) {
	return getObjects(DeploymentHistoryTable, params, external)
}

func GetLatestDeploymentByID(id string, external bool) (*Deployment, *axerror.AXError) {
	deployments, axErr := getObjects(
		DeploymentLatestTable,
		map[string]interface{}{
			axdb.AXDBUUIDColumnName: id,
		},
		external,
	)

	if axErr != nil {
		return nil, axErr
	}

	if len(deployments) == 0 {
		return nil, nil
	}

	d := deployments[0]
	return d, nil
}

func GetLatestDeploymentByName(appName, name string, external bool) (*Deployment, *axerror.AXError) {
	deployments, axErr := getObjects(
		DeploymentLatestTable,
		map[string]interface{}{
			ServiceName:       name,
			DeploymentAppName: appName,
		},
		external,
	)

	if axErr != nil {
		return nil, axErr
	}

	if len(deployments) == 0 {
		return nil, nil
	}

	d := deployments[0]
	return d, nil
}

func GetHistoryDeploymentByID(id string, external bool) (*Deployment, *axerror.AXError) {
	deployments, axErr := getObjects(
		DeploymentHistoryTable,
		map[string]interface{}{
			axdb.AXDBUUIDColumnName: id,
		},
		external,
	)

	if axErr != nil {
		return nil, axErr
	}

	if len(deployments) == 0 {
		return nil, nil
	}

	d := deployments[0]
	return d, nil
}

func GetDeploymentByID(id string, external bool) (*Deployment, *axerror.AXError) {
	d, axErr := GetLatestDeploymentByID(id, external)
	if axErr != nil {
		return nil, axErr
	}

	if d != nil {
		return d, nil
	}

	return GetHistoryDeploymentByID(id, external)
}

func GetLatestDeploymentsByApplication(app string, external bool) ([]*Deployment, *axerror.AXError) {
	deployments, axErr := getObjects(
		DeploymentLatestTable,
		map[string]interface{}{
			DeploymentAppName: app,
		},
		external,
	)

	if axErr != nil {
		return nil, axErr
	}

	return deployments, nil
}
