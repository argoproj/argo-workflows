// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"fmt"
)

type PrivateRegistryConfig struct {
	*ToolBase
	Username string `json:"username,omitempty"`
	HostName string `json:"hostname,omitemtpy"`
}

type PrivateRegistryModel struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Type     string `json:"type"`
	Password string `json:"password"`
	Username string `json:"username"`
	HostName string `json:"hostname"`
}

func (t *PrivateRegistryConfig) Omit() {
	t.Password = ""
}

func (t *PrivateRegistryConfig) Test() (*axerror.AXError, int) {

	if t.Username == "" {
		return ErrToolMissingUsername, axerror.REST_BAD_REQ
	}

	if t.Password == "" {
		return ErrToolMissingPassword, axerror.REST_BAD_REQ
	}

	axErr, code := utils.AxmonCl.Post2("registry/test", nil, t, nil)
	if axErr != nil {
		return axErr, code
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *PrivateRegistryConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryRegistry {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.Username == "" {
		return ErrToolMissingUsername, axerror.REST_BAD_REQ
	}

	if t.Password == "" {
		return ErrToolMissingPassword, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypePrivateRegistry)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*PrivateRegistryConfig).HostName == t.HostName && oldTool.(*PrivateRegistryConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The hoastname(%v) has been used by another configuration.", t.HostName)), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *PrivateRegistryConfig) pre() (*axerror.AXError, int) {

	if t.HostName != "" {
		t.URL = t.HostName
	}

	if t.URL != "" {
		t.HostName = t.URL
	}

	t.Category = CategoryRegistry

	return nil, axerror.REST_STATUS_OK
}

func (t *PrivateRegistryConfig) PushUpdate() (*axerror.AXError, int) {
	return utils.AxmonCl.Put2("registry", nil, t, nil)
}

func (t *PrivateRegistryConfig) pushDelete() (*axerror.AXError, int) {
	return utils.AxmonCl.Delete2("registry/"+t.HostName, nil, nil, nil)
}

func (t *PrivateRegistryConfig) Post(old, new interface{}) (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}
