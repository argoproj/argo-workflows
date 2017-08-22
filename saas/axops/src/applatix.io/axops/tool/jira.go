package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"strings"
)

type JiraConfig struct {
	*ToolBase
	HostName string   `json:"hostname,omitempty"`
	Username string   `json:"username,omitemtpy"`
	Projects []string `json:"projects,omitemtpy"`
}

type JiraModel struct {
	ID       string   `json:"id"`
	Category string   `json:"category"`
	Type     string   `json:"type"`
	HostName string   `json:"hostname"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	URL      string   `json:"url"`
	Projects []string `json:"projects"`
}

func (t *JiraConfig) Omit() {
	t.Password = ""
}

func (t *JiraConfig) Test() (*axerror.AXError, int) {
	//TODO: add test functionaliy
	return nil, axerror.REST_STATUS_OK
}

func (t *JiraConfig) pre() (*axerror.AXError, int) {
	t.Category = CategoryIssueManagement

	if t.HostName != "" {
		t.HostName = strings.TrimSpace(strings.ToLower(t.HostName))
	}

	if t.URL != "" {
		t.URL = strings.TrimSpace(strings.ToLower(t.URL))
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *JiraConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryIssueManagement {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.Username == "" {
		return ErrToolMissingUsername, axerror.REST_BAD_REQ
	}

	if t.Password == "" {
		return ErrToolMissingPassword, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeJira)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*JiraConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("You can only have one jira config."), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *JiraConfig) getOldJira() (*JiraConfig, *axerror.AXError) {
	tools, axErr := GetToolsByType(TypeJira)
	if axErr != nil {
		return nil, axErr
	}

	if len(tools) == 0 {
		return nil, nil
	}

	for _, oldTool := range tools {
		if oldTool.(*JiraConfig).ID == t.ID {
			return oldTool.(*JiraConfig), nil
		}
	}

	return nil, nil

}

type gatewayConfig struct {
	Webhook  string   `json:"webhook,omitempty"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	URL      string   `json:"url,omitempty"`
	Projects []string `json:"projects,omitempty"`
}

func (t *JiraConfig) PushUpdate() (*axerror.AXError, int) {

	oldJira, axErr := t.getOldJira()
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}
	if oldJira == nil {
		// call gateway to create webhook
		config := gatewayConfig{
			Webhook:  "https://" + t.HostName + "/v1/webhooks/jira",
			Username: t.Username,
			Password: t.Password,
			URL:      t.URL,
			Projects: t.Projects,
		}

		utils.InfoLog.Println("calling gateway to add webhook for jira")
		if axErr, code := utils.DevopsCl.Post2("jira/webhooks", nil, config, nil); axErr != nil {
			return axErr, code
		}
	} else {
		// call update
		config := gatewayConfig{
			Projects: t.Projects,
		}
		utils.InfoLog.Println("calling gateway to update webhook for jira")
		if axErr, code := utils.DevopsCl.Put2("jira/webhooks", nil, config, nil); axErr != nil {
			return axErr, code
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *JiraConfig) pushDelete() (*axerror.AXError, int) {

	// call gateway
	utils.InfoLog.Println("calling gateway to delete webhooks for jira")
	if axErr, code := utils.DevopsCl.Delete2("jira/webhooks", nil, nil, nil); axErr != nil {
		return axErr, code
	}

	return nil, axerror.REST_STATUS_OK
}
