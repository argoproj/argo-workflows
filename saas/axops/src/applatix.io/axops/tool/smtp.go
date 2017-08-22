// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"fmt"
)

var (
	ErrToolMissingHostname     = axerror.ERR_API_INVALID_PARAM.NewWithMessage("hostname is required.")
	ErrToolMissingAdminAddress = axerror.ERR_API_INVALID_PARAM.NewWithMessage("admin_address is required.")
)

type SMTPConfig struct {
	*ToolBase
	NickName     string `json:"nickname,omitempty"`
	HostName     string `json:"hostname,omitempty"`
	AdminAddress string `json:"admin_address,omitempty"`
	Port         *int   `json:"port,omitempty"`
	Timeout      *int   `json:"timeout,omitempty"`
	UseTLS       *bool  `json:"use_tls"`
	Username     string `json:"username,omitempty"`
}

type SMTPModel struct {
	ID           string `json:"id"`
	Category     string `json:"category"`
	Type         string `json:"type"`
	NickName     string `json:"nickname"`
	HostName     string `json:"hostname"`
	AdminAddress string `json:"admin_address"`
	Port         *int   `json:"port"`
	Timeout      *int   `json:"timeout"`
	UseTLS       *bool  `json:"use_tls"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

func (t *SMTPConfig) Omit() {
	t.Password = ""
}

func (t *SMTPConfig) Test() (*axerror.AXError, int) {
	return utils.AxNotifierCl.Post2("configurations/test", nil, t, nil)
}

func (t *SMTPConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryNotification {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.HostName == "" {
		return ErrToolMissingHostname, axerror.REST_BAD_REQ
	}

	if t.AdminAddress == "" {
		return ErrToolMissingAdminAddress, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeSMTP)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*SMTPConfig).HostName == t.HostName && oldTool.(*SMTPConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The hostname(%v) has been used by another configuration.", t.HostName)), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *SMTPConfig) pre() (*axerror.AXError, int) {

	t.Category = CategoryNotification

	if t.HostName != "" {
		t.URL = t.HostName
	}

	if t.URL != "" {
		t.HostName = t.URL
	}

	if t.Timeout == nil {
		timeout := 30
		t.Timeout = &timeout
	}

	if t.Port == nil {
		port := 587
		t.Port = &port
	}

	if t.UseTLS == nil {
		useTLS := true
		t.UseTLS = &useTLS
	}

	if t.NickName == "" {
		t.NickName = "default"
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *SMTPConfig) PushUpdate() (*axerror.AXError, int) {
	return utils.AxNotifierCl.Put2("configurations", nil, t, nil)
}

func (t *SMTPConfig) pushDelete() (*axerror.AXError, int) {
	return utils.AxNotifierCl.Delete2("configurations/"+t.ID, nil, nil, nil)
}
