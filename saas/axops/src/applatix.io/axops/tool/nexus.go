// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"fmt"
	"strings"
)

type NexusConfig struct {
	*ToolBase
	HostName string `json:"hostname,omitempty"`
	Port     *int   `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
}

type NexusModel struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Type     string `json:"type"`
	HostName string `json:"hostname"`
	Port     *int   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (t *NexusConfig) Omit() {
	t.Password = ""
}

func (t *NexusConfig) Test() (*axerror.AXError, int) {
	return utils.DevopsCl.Put2("results/test_nexus_credential", nil, t, nil)
}

func (t *NexusConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryArtifactManagement {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.HostName == "" {
		return ErrToolMissingHostname, axerror.REST_BAD_REQ
	}

	hostname := strings.TrimSpace(t.HostName)

	if !(strings.HasPrefix(hostname, "https") || strings.HasPrefix(hostname, "http")) {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Please add the protocol(http/https) to the hostname."), axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeNexus)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*NexusConfig).HostName == t.HostName && oldTool.(*NexusConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The hostname(%v) has been used by another configuration.", t.HostName)), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *NexusConfig) pre() (*axerror.AXError, int) {

	t.Category = CategoryArtifactManagement

	if t.HostName != "" {
		t.URL = strings.TrimSpace(strings.ToLower(t.HostName))
	}

	if t.URL != "" {
		t.HostName = strings.TrimSpace(strings.ToLower(t.URL))
	}

	if t.Port == nil {
		port := 587
		t.Port = &port
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *NexusConfig) PushUpdate() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *NexusConfig) pushDelete() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}
