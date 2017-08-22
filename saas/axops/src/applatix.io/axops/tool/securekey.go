package tool

import (
	"applatix.io/axerror"
	"crypto/rsa"
)

var (
	ErrToolInvalidPrivateKey = axerror.ERR_API_INVALID_PARAM.NewWithMessage("private_key is invalid.")
)

type SecureKeyConfig struct {
	*ToolBase
	PrivateKey *rsa.PrivateKey `json:"private_key,omitempty"`
	KeyName    string          `json:"keyname,omitempty"`
	Version    string          `json:"version,omitempty"`
}

type SecureKeyModel struct {
	ID         string          `json:"id"`
	Category   string          `json:"category"`
	Type       string          `json:"type"`
	PrivateKey *rsa.PrivateKey `json:"private_key"`
}

func (t *SecureKeyConfig) Omit() {
	t.PrivateKey = nil
}

func (t *SecureKeyConfig) Test() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *SecureKeyConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategorySecret {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.PrivateKey == nil {
		return ErrToolMissingPrivateKey, axerror.REST_BAD_REQ
	}

	err := t.PrivateKey.Validate()
	if err != nil {
		return ErrToolInvalidPrivateKey, axerror.REST_INTERNAL_ERR
	}

	tools, axErr := GetToolsByType(TypeSecureKey)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*SecureKeyConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("You can only have one secure key."), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *SecureKeyConfig) pre() (*axerror.AXError, int) {
	//t.URL = "secret/secure_key"
	return nil, axerror.REST_STATUS_OK
}

func (t *SecureKeyConfig) PushUpdate() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *SecureKeyConfig) pushDelete() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *SecureKeyConfig) Post(old, new interface{}) (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}
