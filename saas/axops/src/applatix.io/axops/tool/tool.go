// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"encoding/json"
	"fmt"
	"sync"
)

var IsSystemInitialized bool = false
var mutex = &sync.Mutex{}

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

func (t *ToolBase) GetPassword() string {
	return t.Password
}

func (t *ToolBase) GenerateUUID() {
	t.ID = utils.GenerateUUIDv1()
}

func Create(t Tool) (*axerror.AXError, int) {
	t.GenerateUUID()
	mutex.Lock()
	defer mutex.Unlock()

	if err, code := t.pre(); err != nil {
		return err, code
	}

	if err, code := validate(t); err != nil {
		return err, code
	}

	if err, code := t.validate(); err != nil {
		return err, code
	}

	if err, code := t.PushUpdate(); err != nil {
		return err, code
	}

	if err, code := save(t); err != nil {
		return err, code
	}

	return nil, axerror.REST_CREATE_OK
}

func Update(t Tool) (*axerror.AXError, int) {

	mutex.Lock()
	defer mutex.Unlock()

	if err, code := t.pre(); err != nil {
		return err, code
	}

	if err, code := validate(t); err != nil {
		return err, code
	}

	if err, code := t.validate(); err != nil {
		return err, code
	}

	if err, code := t.PushUpdate(); err != nil {
		return err, code
	}

	old, err := GetToolByID(t.GetID())
	if err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	if old == nil {
		return axerror.ERR_API_RESOURCE_NOT_FOUND.New(), axerror.REST_NOT_FOUND
	}

	oldTool := old.(Tool)

	if err, code := t.Post(oldTool, t); err != nil {
		return err, code
	}

	if err, code := save(t); err != nil {
		return err, code
	}

	return nil, axerror.REST_STATUS_OK
}

func Delete(t Tool) (*axerror.AXError, int) {

	mutex.Lock()
	defer mutex.Unlock()

	if err, code := t.pushDelete(); err != nil {
		return err, code
	}

	old, err := GetToolByID(t.GetID())
	if err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	if old == nil {
		return axerror.ERR_API_RESOURCE_NOT_FOUND.New(), axerror.REST_NOT_FOUND
	}

	oldTool := old.(Tool)

	if err, code := t.Post(oldTool, nil); err != nil {
		return err, code
	}

	if err, code := toolDelete(t); err != nil {
		return err, code
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *ToolBase) Post(old, new interface{}) (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func toolDelete(t Tool) (*axerror.AXError, int) {

	tool := &ToolDB{
		ID: t.GetID(),
	}

	if err := tool.delete(); err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_STATUS_OK
}

func save(t Tool) (*axerror.AXError, int) {
	tool := &ToolDB{
		ID:       t.GetID(),
		URL:      t.GetURL(),
		Category: t.GetCategory(),
		Type:     t.GetType(),
	}

	config, err, code := getConfig(t)
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

func validate(t Tool) (*axerror.AXError, int) {

	if t.GetCategory() == "" {
		return ErrToolMissingCategory, axerror.REST_BAD_REQ
	}

	if t.GetType() == "" {
		return ErrToolMissingType, axerror.REST_BAD_REQ
	}

	if t.GetURL() == "" {
		return ErrToolMissingUrl, axerror.REST_BAD_REQ
	}

	if t.GetID() == "" {
		return ErrToolMissingID, axerror.REST_BAD_REQ
	}

	return nil, axerror.REST_STATUS_OK
}

func getConfig(t Tool) (string, *axerror.AXError, int) {
	configStr, err := json.Marshal(t)
	if err != nil {
		return "", axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the tool object: %v", err)), axerror.REST_INTERNAL_ERR
	}
	return string(configStr), nil, axerror.REST_STATUS_OK
}
