package deployment

import (
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/template"
)

type Fixture struct {
	RootWorkflowId       string                                  `json:"root_workflow_id,omitempty"`
	ServiceId            string                                  `json:"service_id,omitempty"`
	ApplicationName      string                                  `json:"application_name,omitempty"`
	AppID                string                                  `json:"application_id,omitempty"`
	AppGeneration        string                                  `json:"application_generation,omitempty"`
	DeploymentName       string                                  `json:"deployment_name,omitempty"`
	DeploymentID         string                                  `json:"deployment_id,omitempty"`
	DeploymentGeneration string                                  `json:"deployment_generation,omitempty"`
	User                 string                                  `json:"user,omitempty"`
	Synchronous          *bool                                   `json:"synchronous,omitempty"`
	Requirements         map[string]*template.FixtureRequirement `json:"requirements,omitempty"`
	VolRequirements      template.VolumeRequirements             `json:"vol_requirements,omitempty"`
	Assignment           map[string]map[string]interface{}       `json:"assignment,omitempty"`
	VolAssignment        map[string]map[string]interface{}       `json:"vol_assignment,omitempty"`
	Requester            string                                  `json:"requester,omitempty"`
}

func (f *Fixture) reserve() (*Fixture, *axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, nil, 200
	}

	assignment := Fixture{}
	axErr, code := utils.FixMgrCl.PostWithTimeRetry("fixture/requests", nil, f, &assignment, retryConfig)
	if axErr != nil {
		return nil, axErr, code
	}
	return &assignment, nil, code
}

func (f *Fixture) release() (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	axErr, code := utils.FixMgrCl.DeleteWithTimeRetry("fixture/requests/"+f.ServiceId, nil, nil, nil, retryConfig)
	if axErr != nil {
		return axErr, code
	}
	return nil, code
}
