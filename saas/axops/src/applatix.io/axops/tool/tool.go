// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"encoding/json"
	"fmt"
)

var IsSystemInitialized bool = false

var (
	ErrToolMissingID            = axerror.ERR_API_INVALID_PARAM.NewWithMessage("id is required.")
	ErrToolMissingCategory      = axerror.ERR_API_INVALID_PARAM.NewWithMessage("category is required.")
	ErrToolMissingType          = axerror.ERR_API_INVALID_PARAM.NewWithMessage("type is required.")
	ErrToolMissingUrl           = axerror.ERR_API_INVALID_PARAM.NewWithMessage("url is required.")
	ErrToolCategoryNotMatchType = axerror.ERR_API_INVALID_PARAM.NewWithMessage("category is not compatible with type.")
)

type ToolBase struct {
	ID       string `json:"id,omitempty"`
	URL      string `json:"url,omitempty"`
	Category string `json:"category,omitempty"`
	Type     string `json:"type,omitempty"`
	Password string `json:"password,omitempty"`
	Real     Tool   `json:"-"`
}

func (t *ToolBase) GetID() string {
	return t.ID
}

func (t *ToolBase) GetURL() string {
	return t.URL
}

func (t *ToolBase) GetCategory() string {
	return t.Category
}

func (t *ToolBase) GetType() string {
	return t.Type
}

func (t *ToolBase) GetConfig() (string, *axerror.AXError, int) {
	configStr, err := json.Marshal(t.Real)
	if err != nil {
		return "", axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the tool object: %v", err)), axerror.REST_INTERNAL_ERR
	}
	return string(configStr), nil, axerror.REST_STATUS_OK
}

func (t *ToolBase) GetPassword() string {
	return t.Password
}

func (t *ToolBase) Create() (Tool, *axerror.AXError, int) {

	t.ID = utils.GenerateUUIDv1()

	if err, code := t.Real.pre(); err != nil {
		return nil, err, code
	}

	if err, code := t.validate(); err != nil {
		return nil, err, code
	}

	if err, code := t.Real.validate(); err != nil {
		return nil, err, code
	}

	if err, code := t.Real.PushUpdate(); err != nil {
		return nil, err, code
	}

	if err, code := t.save(); err != nil {
		return nil, err, code
	}

	return t.Real, nil, axerror.REST_CREATE_OK
}

func (t *ToolBase) Update() (Tool, *axerror.AXError, int) {

	if err, code := t.Real.pre(); err != nil {
		return nil, err, code
	}

	if err, code := t.validate(); err != nil {
		return nil, err, code
	}

	if err, code := t.Real.validate(); err != nil {
		return nil, err, code
	}

	if err, code := t.Real.PushUpdate(); err != nil {
		return nil, err, code
	}

	old, err := GetToolByID(t.ID)
	if err != nil {
		return nil, err, axerror.REST_INTERNAL_ERR
	}

	if old == nil {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.New(), axerror.REST_NOT_FOUND
	}

	oldTool := old.(Tool)

	if err, code := t.Real.Post(oldTool, t.Real); err != nil {
		return nil, err, code
	}

	if err, code := t.save(); err != nil {
		return nil, err, code
	}

	return t.Real, nil, axerror.REST_STATUS_OK
}

func (t *ToolBase) Delete() (*axerror.AXError, int) {

	if err, code := t.Real.pushDelete(); err != nil {
		return err, code
	}

	old, err := GetToolByID(t.ID)
	if err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	if old == nil {
		return axerror.ERR_API_RESOURCE_NOT_FOUND.New(), axerror.REST_NOT_FOUND
	}

	oldTool := old.(Tool)

	if err, code := t.Real.Post(oldTool, nil); err != nil {
		return err, code
	}

	if err, code := t.delete(); err != nil {
		return err, code
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *ToolBase) delete() (*axerror.AXError, int) {

	tool := &ToolDB{
		ID: t.GetID(),
	}

	if err := tool.delete(); err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *ToolBase) save() (*axerror.AXError, int) {

	tool := &ToolDB{
		ID:       t.GetID(),
		URL:      t.GetURL(),
		Category: t.GetCategory(),
		Type:     t.GetType(),
	}

	config, err, code := t.GetConfig()
	if err != nil {
		return err, code
	}
	tool.Config = config

	err = tool.update()
	if err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *ToolBase) validate() (*axerror.AXError, int) {

	if t.Category == "" {
		return ErrToolMissingCategory, axerror.REST_BAD_REQ
	}

	if t.Type == "" {
		return ErrToolMissingType, axerror.REST_BAD_REQ
	}

	if t.URL == "" {
		return ErrToolMissingUrl, axerror.REST_BAD_REQ
	}

	if t.ID == "" {
		return ErrToolMissingID, axerror.REST_BAD_REQ
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *ToolBase) Post(old, new interface{}) (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}
