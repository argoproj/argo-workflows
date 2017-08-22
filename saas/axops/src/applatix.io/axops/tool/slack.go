package tool

import (
	"applatix.io/axerror"
	"applatix.io/slackcl"
)

var (
	ErrToolMissingOauthToken = axerror.ERR_API_INVALID_PARAM.NewWithMessage("oauth token is required.")
)

type SlackConfig struct {
	*ToolBase
	OauthToken string `json:"oauth_token,omitempty"`
}

type SlackModel struct {
	ID         string `json:"id"`
	Category   string `json:"category"`
	Type       string `json:"type"`
	OauthToken string `json:"oauth_token"`
}

func (t *SlackConfig) Omit() {
	t.OauthToken = ""
}

func (t *SlackConfig) Test() (*axerror.AXError, int) {

	// use the API to get channels and users
	// TODO: also post a test message?
	client := slackcl.New(t.OauthToken)
	_, err := client.GetChannels()
	if err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}
	_, err = client.GetUsers()
	if err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}
	return nil, axerror.REST_STATUS_OK
}

func (t *SlackConfig) pre() (*axerror.AXError, int) {

	if t.URL == "" {
		t.URL = "https://api.slack.com"
	}

	t.Category = CategoryNotification
	return nil, axerror.REST_STATUS_OK
}

func (t *SlackConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryNotification {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if len(t.OauthToken) < 1 {
		return ErrToolMissingOauthToken, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeSlack)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*SlackConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("You can only have one slack config."), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *SlackConfig) PushUpdate() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *SlackConfig) pushDelete() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}
