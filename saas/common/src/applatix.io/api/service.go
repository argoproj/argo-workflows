package api

import (
	"fmt"
	"strings"
	"time"

	"applatix.io/axerror"
	"applatix.io/axops"
	"applatix.io/axops/service"
	"applatix.io/template"
)

type ServiceCreateParams struct {
	Name       string                     `json:"string,omitempty"`
	Template   service.EmbeddedTemplateIf `json:"template,omitempty"`
	TemplateID string                     `json:"template_id,omitempty"`
	Arguments  template.Arguments         `json:"arguments,omitempty"`
	User       string                     `json:"user,omitempty"`

	// Submit options
	DryRun bool `json:"-"`
}

type ServiceListParams struct {
	MinTime      time.Time
	MaxTime      time.Time
	IsActive     *bool
	Fields       []string
	Username     string
	Limit        int64
	StatusString string
}

// ServiceList returns a list of services based on supplied filters
func (c *ArgoClient) ServiceList(params ServiceListParams) ([]*service.Service, *axerror.AXError) {
	queryArgs := []string{}
	if !params.MinTime.IsZero() {
		queryArgs = append(queryArgs, fmt.Sprintf("min_time=%d", params.MinTime.Unix()))
	}
	if !params.MaxTime.IsZero() {
		queryArgs = append(queryArgs, fmt.Sprintf("max_time=%d", params.MaxTime.Unix()))
	}
	if params.IsActive != nil {
		queryArgs = append(queryArgs, fmt.Sprintf("is_active=%v", *params.IsActive))
	}
	if params.Username != "" {
		queryArgs = append(queryArgs, fmt.Sprintf("username=%s", params.Username))
	}
	if params.Limit != 0 {
		queryArgs = append(queryArgs, fmt.Sprintf("limit=%d", params.Limit))
	}
	if params.StatusString != "" {
		queryArgs = append(queryArgs, fmt.Sprintf("status_string=%s", params.StatusString))
	}
	if len(params.Fields) > 0 {
		queryArgs = append(queryArgs, fmt.Sprintf("fields=%s", strings.Join(params.Fields, ",")))
	}
	url := fmt.Sprintf("services")
	if len(queryArgs) > 0 {
		url += fmt.Sprintf("?%s", strings.Join(queryArgs, "&"))
	}
	var servicesData axops.ServicesData
	axErr := c.get(url, &servicesData)
	if axErr != nil {
		return nil, axErr
	}
	return servicesData.Data, nil
}

// ServiceCreate creates a service
func (c *ArgoClient) ServiceCreate(params ServiceCreateParams) (*service.Service, *axerror.AXError) {
	queryArgs := []string{}
	if params.DryRun {
		queryArgs = append(queryArgs, fmt.Sprintf("dry_run=%v", params.DryRun))
	}
	url := "services"
	if len(queryArgs) > 0 {
		url += fmt.Sprintf("?%s", strings.Join(queryArgs, "&"))
	}
	var createdSvc service.Service
	axErr := c.post(url, params, &createdSvc)
	if axErr != nil {
		return nil, axErr
	}
	return &createdSvc, nil
}

// ServiceGet gets a service by ID
func (c *ArgoClient) ServiceGet(id string) (*service.Service, *axerror.AXError) {
	url := fmt.Sprintf("services/%s", id)
	var svc service.Service
	axErr := c.get(url, &svc)
	if axErr != nil {
		return nil, axErr
	}
	return &svc, nil
}

// ServiceDelete terminates a service by ID
func (c *ArgoClient) ServiceDelete(id string) *axerror.AXError {
	url := fmt.Sprintf("services/%s", id)
	return c.delete(url, nil)
}
