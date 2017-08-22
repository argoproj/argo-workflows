package tool

import (
	"applatix.io/axerror"
)

var (
	ErrToolMissingToken = axerror.ERR_API_INVALID_PARAM.NewWithMessage("token is required.")
)

type SplunkConfig struct {
	*ToolBase
	Token string `json:"token,omitempty"`
}

type SplunkModel struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Token    string `json:"token"`
}

func (t *SplunkConfig) Omit() {
}

func (t *SplunkConfig) Test() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *SplunkConfig) pre() (*axerror.AXError, int) {

	t.Category = CategoryNotification
	return nil, axerror.REST_STATUS_OK
}

func (t *SplunkConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryNotification {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if len(t.Token) < 1 {
		return ErrToolMissingToken, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeSplunk)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*SplunkConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("You can only have one splunk config."), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *SplunkConfig) PushUpdate() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *SplunkConfig) pushDelete() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}
