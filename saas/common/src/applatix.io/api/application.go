package api

import (
	"fmt"
	"strings"

	"applatix.io/axamm/application"
	"applatix.io/axerror"
)

type ApplicationListParams struct {
	Fields   []string
	Limit    int64
	Statuses []string
	Name     string
}

// RunningApplicationStates is the list of non-terminated application states
var RunningApplicationStates = []string{
	application.AppStateInit,
	application.AppStateWaiting,
	application.AppStateError,
	application.AppStateActive,
	application.AppStateTerminating,
	application.AppStateStopping,
	application.AppStateStopped,
	application.AppStateUpgrading,
}

// ApplicationList returns a list of applications based on supplied filters
func (c *ArgoClient) ApplicationList(params ApplicationListParams) ([]*application.Application, *axerror.AXError) {
	queryArgs := []string{}
	if len(params.Statuses) > 0 {
		queryArgs = append(queryArgs, fmt.Sprintf("status=%s", strings.Join(params.Statuses, ",")))
	}
	if len(params.Fields) > 0 {
		queryArgs = append(queryArgs, fmt.Sprintf("fields=%s", strings.Join(params.Fields, ",")))
	}
	if params.Name != "" {
		queryArgs = append(queryArgs, fmt.Sprintf("name=%s", params.Name))
	}
	if params.Limit != 0 {
		queryArgs = append(queryArgs, fmt.Sprintf("limit=%d", params.Limit))
	}
	url := fmt.Sprintf("applications")
	if len(queryArgs) > 0 {
		url += fmt.Sprintf("?%s", strings.Join(queryArgs, "&"))
	}
	type ApplicationsData struct {
		Data []*application.Application `json:"data"`
	}
	var appsData ApplicationsData
	axErr := c.get(url, &appsData)
	if axErr != nil {
		return nil, axErr
	}
	return appsData.Data, nil
}

// ApplicationGet retrieves an application by its ID
func (c *ArgoClient) ApplicationGet(id string) (*application.Application, *axerror.AXError) {
	url := fmt.Sprintf("applications/%s", id)
	var app application.Application
	axErr := c.get(url, &app)
	if axErr != nil {
		return nil, axErr
	}
	return &app, nil
}

// ApplicationGetByName retrieves an application by its name
func (c *ArgoClient) ApplicationGetByName(name string) (*application.Application, *axerror.AXError) {
	apps, axErr := c.ApplicationList(ApplicationListParams{
		Fields: []string{application.ApplicationID},
		Name:   name,
	})
	if axErr != nil {
		return nil, axErr
	}
	if len(apps) == 0 {
		return nil, nil
	}
	if len(apps) > 1 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Found %d applications with name '%s'", len(apps), name)
	}
	return c.ApplicationGet(apps[0].ID)
}
