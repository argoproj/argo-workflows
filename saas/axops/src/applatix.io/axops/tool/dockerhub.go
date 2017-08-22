// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"fmt"
)

type DockerHubConfig struct {
	*PrivateRegistryConfig
}

type DockerHubModel struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Type     string `json:"type"`
	Password string `json:"password"`
	Username string `json:"username"`
}

func (t *DockerHubConfig) pre() (*axerror.AXError, int) {
	t.URL = "docker.io"
	t.HostName = "docker.io"

	return t.PrivateRegistryConfig.pre()
}

func (t *DockerHubConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryRegistry {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.Username == "" {
		return ErrToolMissingUsername, axerror.REST_BAD_REQ
	}

	if t.Password == "" {
		return ErrToolMissingPassword, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeDockerHub)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*DockerHubConfig).Username == t.Username && oldTool.(*DockerHubConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The username(%v) has been used by another configuration.", t.Username)), axerror.REST_BAD_REQ
		} else if oldTool.(*DockerHubConfig).Username != t.Username && oldTool.(*DockerHubConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("The system can support only one DockerHub configuration, please delete the old one first."), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *DockerHubConfig) Post(old, new interface{}) (*axerror.AXError, int) {

	return nil, axerror.REST_STATUS_OK
}
