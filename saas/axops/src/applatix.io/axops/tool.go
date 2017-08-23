// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Tool API [/tools]
package axops

import (
	"encoding/json"
	"fmt"
	"time"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/commit"
	"applatix.io/axops/fixture"
	"applatix.io/axops/policy"
	"applatix.io/axops/project"
	"applatix.io/axops/service"
	"applatix.io/axops/tool"
	"applatix.io/axops/utils"
	"github.com/gin-gonic/gin"
)

// @Title ToolConfiguration
// @Description Configure Tool
// @Accept  json
// @Param   config	body    tool.ToolBase			true        "Configuration object"
// @Success 201 {object} tool.ToolBase
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools [POST]
func CreateTool() gin.HandlerFunc {
	return func(c *gin.Context) {
		params := map[string]interface{}{}
		err := utils.GetUnmarshalledBody(c, &params)
		if err != nil {
			fmt.Println(1)
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err))
			return
		}

		toolType, ok := params["type"]
		if !ok {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("type field is required."))
			return
		}
		typeStr := toolType.(string)
		c.Set("tool_config", params)

		CreateToolWithType(typeStr)(c)

		service.UpdateServiceETag()
		service.UpdateTemplateETag()
		policy.UpdateETag()
		commit.UpdateETag()
		project.UpdateETag()
		fixture.UpdateETag()
	}
}

func CreateToolWithType(toolType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := map[string]interface{}{}
		err := utils.GetUnmarshalledBody(c, &params)
		if err != nil {
			if _, ok := c.Get("tool_config"); !ok {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err))
				return
			}
			params = c.MustGet("tool_config").(map[string]interface{})
		}

		params["type"] = toolType

		configBytes, err := json.Marshal(params)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err))
			return
		}

		t, axErr := unmarshalTool(toolType, configBytes)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		axErr, code := tool.Create(t.(tool.Tool))
		if axErr != nil {
			c.JSON(code, axErr)
			return
		}
		t.(tool.Tool).Omit()
		c.JSON(code, t)

		service.UpdateServiceETag()
		service.UpdateTemplateETag()
		policy.UpdateETag()
		commit.UpdateETag()
		project.UpdateETag()
		fixture.UpdateETag()
	}
}

// @Title ConfigureGit
// @Description Configure Git
// @Accept  json
// @Param   config	body    tool.GitModel	true        "Configuration object"
// @Success 201 {object} tool.GitModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/scm/git [POST]
func CreateScmGit() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeGIT)
}

// @Title ConfigureGitHub
// @Description Configure GitHub
// @Accept  json
// @Param   config	body    tool.GitModel		true        "Configuration object"
// @Success 201 {object} tool.GitModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/scm/github [POST]
func CreateScmGitHub() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeGitHub)
}

// @Title ConfigureBitbucket
// @Description Configure Bitbucket
// @Accept  json
// @Param   config	body    tool.GitModel		true        "Configuration object"
// @Success 201 {object} tool.GitModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/scm/bitbucket [POST]
func CreateScmBitbucket() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeBitBucket)
}

// @Title ConfigureGitLab
// @Description Configure GitLab
// @Accept  json
// @Param   config	body    tool.GitModel		true        "Configuration object"
// @Success 201 {object} tool.GitModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/scm/gitlab [POST]
func CreateGitLab() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeGitLab)
}

// @Title ConfigureCodeCommit
// @Description Configure CodeCommit
// @Accept  json
// @Param   config	body    tool.GitModel		true        "Configuration object"
// @Success 201 {object} tool.GitModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/scm/codecommit [POST]
func CreateScmCodeCommit() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeCodeCommit)
}

// @Title ConfigureSMPT
// @Description Configure SMTP
// @Accept  json
// @Param   config	body    tool.SMTPModel		true        "Configuration object"
// @Success 201 {object} tool.SMTPModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/notification/smtp [POST]
func CreateNotificationSMTP() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeSMTP)
}

// @Title ConfigureSlack
// @Description Configure Slack
// @Accept  json
// @Param   config	body    tool.SlackModel		true        "Configuration object"
// @Success 201 {object} tool.SlackModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/notification/slack [POST]
func CreateNotificationSlack() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeSlack)
}

// @Title ConfigureSplunk
// @Description Configure Splunk
// @Accept  json
// @Param   config	body    tool.SplunkModel		true        "Configuration object"
// @Success 201 {object} tool.SplunkModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/notification/splunk [POST]
func CreateNotificationSplunk() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeSplunk)
}

// @Title ConfigureSAML
// @Description Configure SAML
// @Accept  json
// @Param   config	body    tool.SAMLModel		true        "Configuration object"
// @Success 201 {object} tool.SAMLModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/authentication/saml [POST]
func CreateAuthenticationSAML() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeSAML)
}

// @Title ConfigureServerCertificate
// @Description Configure Server Certificate
// @Accept  json
// @Param   config	body    tool.CertificateModel	true        "Configuration object"
// @Success 201 {object} tool.CertificateModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/certificate/server [POST]
func CreateCertificateServer() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeServer)
}

// @Title ConfigureDockerHub
// @Description Configure DockerHub
// @Accept  json
// @Param   config	body    tool.DockerHubModel	true        "Configuration object"
// @Success 201 {object} tool.DockerHubModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/registry/dockerhub [POST]
func CreateRegistryDockerHub() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeDockerHub)
}

// @Title ConfigurePrivateRegistry
// @Description Configure Private Registry
// @Accept  json
// @Param   config	body    tool.PrivateRegistryModel	true        "Configuration object"
// @Success 201 {object} tool.PrivateRegistryModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/registry/private_registry [POST]
func CreateRegistryPrivate() gin.HandlerFunc {
	return CreateToolWithType(tool.TypePrivateRegistry)
}

// @Title ConfigureDomainManagement
// @Description Configure Domain Management
// @Accept  json
// @Param   config	body    tool.DomainModel	true        "Configuration object"
// @Success 201 {object} tool.DomainModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/domain_management/route53 [POST]
func CreateDomain() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeRoute53)
}

// @Title ConfigureNexus
// @Description Configure Nexus Artifact Management
// @Accept  json
// @Param   config	body    tool.NexusModel			true        "Configuration object"
// @Success 201 {object} tool.NexusModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/artifact_management/nexus [POST]
func CreateNexus() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeNexus)
}

// @Title ConfigureJira
// @Description Configure Jira Issue Management
// @Accept  json
// @Param   config	body    tool.JiraModel			true        "Configuration object"
// @Success 201 {object} tool.JiraModel
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/issue_management/jira [POST]
func CreateJira() gin.HandlerFunc {
	return CreateToolWithType(tool.TypeJira)
}

func unmarshalTool(toolType string, configBytes []byte) (tool.Tool, *axerror.AXError) {
	var t tool.Tool
	var err error
	switch toolType {
	case tool.TypeGitHub:
		github := &tool.GitHubConfig{}
		err = json.Unmarshal(configBytes, github)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = github
	case tool.TypeBitBucket:
		bitbucket := &tool.BitbucketConfig{}
		err = json.Unmarshal(configBytes, bitbucket)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = bitbucket
	case tool.TypeGitLab:
		gitlab := &tool.GitLabConfig{}
		err = json.Unmarshal(configBytes, gitlab)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = gitlab
	case tool.TypeCodeCommit:
		codecommit := &tool.CodeCommitConfig{}
		err = json.Unmarshal(configBytes, codecommit)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = codecommit
	case tool.TypeGIT:
		git := &tool.GitConfig{}
		err = json.Unmarshal(configBytes, git)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = git
	case tool.TypeSMTP:
		smtp := &tool.SMTPConfig{}
		err = json.Unmarshal(configBytes, smtp)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = smtp
	case tool.TypeSlack:
		slack := &tool.SlackConfig{}
		err = json.Unmarshal(configBytes, slack)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = slack
	case tool.TypeSplunk:
		splunk := &tool.SplunkConfig{}
		err = json.Unmarshal(configBytes, splunk)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = splunk
	case tool.TypeSAML:
		saml := &tool.SAMLConfig{}
		err = json.Unmarshal(configBytes, saml)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = saml
	case tool.TypeServer:
		cert := &tool.ServerCertConfig{}
		err = json.Unmarshal(configBytes, cert)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = cert
	case tool.TypeDockerHub:
		dockerhub := &tool.DockerHubConfig{}
		err = json.Unmarshal(configBytes, dockerhub)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = dockerhub
	case tool.TypePrivateRegistry:
		private := &tool.PrivateRegistryConfig{}
		err = json.Unmarshal(configBytes, private)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = private
	case tool.TypeRoute53:
		domain := &tool.DomainConfig{}
		err = json.Unmarshal(configBytes, domain)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = domain
	case tool.TypeNexus:
		nexus := &tool.NexusConfig{}
		err = json.Unmarshal(configBytes, nexus)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = nexus
	case tool.TypeSecureKey:
		securekey := &tool.SecureKeyConfig{}
		err = json.Unmarshal(configBytes, securekey)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = securekey
	case tool.TypeJira:
		jira := &tool.JiraConfig{}
		err = json.Unmarshal(configBytes, jira)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err)
		}
		t = jira
	default:
		return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("The %v is not supported type.", toolType)

	}
	return t, nil
}

type ToolsData struct {
	Data []tool.ToolBase `json:"data"`
}

// @Title ListTools
// @Description List tools
// @Accept  json
// @Param   category	 query   string     false       "Category."
// @Param   type	 query   string     false       "Type."
// @Success 200 {object} ToolsData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools [GET]
func GetToolList(internal bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Request.URL.Query().Get("category")
		toolType := c.Request.URL.Query().Get("type")

		params := map[string]interface{}{}

		if category != "" {
			params["category"] = category
		}

		if toolType != "" {
			params["type"] = toolType
		}

		tools, axErr := tool.GetTools(params)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if !internal {
			for i, _ := range tools {
				tools[i].(tool.Tool).Omit()
			}
		}

		resultMap := map[string]interface{}{RestData: tools}
		c.JSON(axdb.RestStatusOK, resultMap)
	}
}

// @Title GetTool
// @Description Get tool by ID
// @Accept  json
// @Param   id		path    string     true        "Tool ID."
// @Success 200 {object} tool.ToolBase
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/{id} [GET]
func GetTool(internal bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		t, axErr := tool.GetToolByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("%v", axErr)))
			return
		}

		if t == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND)
			return
		}

		if !internal {
			t.(tool.Tool).Omit()
		}
		c.JSON(axerror.REST_STATUS_OK, t)
	}
}

// @Title UpdateConfiguration
// @Description Update Configuration
// @Accept  json
// @Param   config	body    tool.ToolBase			true        "Configuration object"
// @Param   id		path    string     			true        "Tool ID."
// @Success 200 {object} tool.ToolBase
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/{id} [PUT]
func PutTool(toolType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		old, axErr := tool.GetToolByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if old == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND)
			return
		}

		oldTool := old.(tool.Tool)

		if toolType != "" {
			if oldTool.GetType() != toolType {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("This end point can not handle type %v.", oldTool.GetType()))
				return
			}
		}

		new := make(map[string]interface{})
		err := utils.GetUnmarshalledBody(c, &new)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ)
			return
		}

		// Copy fields from old tool
		new["type"] = oldTool.GetType()
		new["category"] = oldTool.GetCategory()
		new["url"] = oldTool.GetURL()
		new["id"] = oldTool.GetID()
		if _, ok := new["password"]; !ok {
			new["password"] = oldTool.GetPassword()
		} else {
			password, ok := new["password"].(string)
			if ok {
				if password == "" {
					new["password"] = oldTool.GetPassword()
				}
			} else {
				new["password"] = oldTool.GetPassword()
			}
		}

		configBytes, err := json.Marshal(new)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err))
			return
		}

		newTool, axErr := unmarshalTool(oldTool.GetType(), configBytes)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		axErr, code := tool.Update(newTool.(tool.Tool))
		if axErr != nil {
			c.JSON(code, axErr)
			return
		} else {
			newTool.(tool.Tool).Omit()
			c.JSON(code, newTool)

			service.UpdateServiceETag()
			service.UpdateTemplateETag()
			policy.UpdateETag()
			commit.UpdateETag()
			project.UpdateETag()
			fixture.UpdateETag()
			return
		}
	}
}

// @Title DeleteConfiguration
// @Description Configure Deletion
// @Accept  json
// @Param   id		path    string	true        "Tool id"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/{id} [DELETE]
func DeleteTool(toolType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		t, axErr := tool.GetToolByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if t == nil {
			c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
			return
		}

		if toolType != "" {
			if t.(tool.Tool).GetType() != toolType {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("This end point can not handle type %v.", t.(tool.Tool).GetType()))
				return
			}
		}

		if axErr, code := tool.Delete(t.(tool.Tool)); axErr != nil {
			c.JSON(code, axErr)
			return
		} else {
			resultMap := map[string]interface{}{}
			c.JSON(code, resultMap)

			service.UpdateServiceETag()
			service.UpdateTemplateETag()
			policy.UpdateETag()
			commit.UpdateETag()
			project.UpdateETag()
			fixture.UpdateETag()
			return
		}
	}
}

func PushScmTools() {
	tools, axErr := tool.GetToolsByCategory(tool.CategorySCM)
	if axErr != nil {
		panic(fmt.Sprintf("Failed to load existing tools from DB: %v", axErr))
	}

	if len(tools) != 0 {
		for _, t := range tools {
			if axErr, _ := t.(tool.Tool).PushUpdate(); axErr != nil {
				utils.ErrorLog.Printf("Init: Push tool %v to devops failed: %v, will retry later", t.(tool.Tool).GetID(), axErr)
				go PushScmTool(t.(tool.Tool).GetID())
			} else {
				utils.InfoLog.Printf("Pushed the SCM configurration successfully: %v\n", t.(tool.Tool).GetURL())
			}
		}
	}
}

func PushScmTool(id string) {
	for {
		time.Sleep(5 * time.Minute)

		t, axErr := tool.GetToolByID(id)
		if axErr != nil {
			utils.ErrorLog.Printf("Failed to load existing tools from DB with id %v: %v\n", id, axErr)
			continue
		}

		if t != nil {
			if axErr, _ := t.(tool.Tool).PushUpdate(); axErr != nil {
				utils.ErrorLog.Printf("Init: Push tool %v %v %v to devops failed: %v\n", t.(tool.Tool).GetID(), t.(tool.Tool).GetType(), t.(tool.Tool).GetURL(), axErr)
				continue
			} else {
				utils.InfoLog.Printf("Pushed the SCM configuration successfully: %v", t.(tool.Tool).GetURL())
			}
		}

		break
	}
}

func UpdateScmTools() {
	tools, axErr := tool.GetToolsByCategory(tool.CategorySCM)
	if axErr != nil {
		panic(fmt.Sprintf("[SCM] Failed to load existing tools from DB: %v", axErr))
	}

	if len(tools) != 0 {
		for _, t := range tools {
			UpdateScmTool(t.GetID())
		}
	}
}

func UpdateScmTool(id string) {
	t, axErr := tool.GetToolByID(id)
	if axErr != nil {
		utils.ErrorLog.Printf("[SCM] Failed to load existing tools from DB with id %v: %v\n", id, axErr)
		return
	}

	if t != nil {
		if axErr, _ := tool.Update(t); axErr != nil {
			utils.ErrorLog.Printf("[SCM] Update tool %v %v %v to devops failed: %v\n", t.GetID(), t.GetType(), t.GetURL(), axErr)
		} else {
			utils.InfoLog.Printf("[SCM] Updated the SCM configurration: %v", t.GetURL())
		}
	}
}

func PushNotificationConfig() {
	tools, axErr := tool.GetToolsByCategory(tool.CategoryNotification)
	if axErr != nil {
		panic(fmt.Sprintf("Failed to load existing notification configurations from DB: %v", axErr))
	}

	if len(tools) != 0 {
		for _, t := range tools {
			if axErr, _ = t.PushUpdate(); axErr != nil {
				panic(fmt.Sprintf("Init: Push tool %v to axnotification failed: %v", t.GetID(), axErr))
			} else {
				utils.InfoLog.Printf("Pushed the notification configuration successfully: %v", t.GetURL())
			}
		}
	}
}

func ApplyAuthenticationConfig() {
	tools, axErr := tool.GetToolsByCategory(tool.CategoryAuthentication)
	if axErr != nil {
		panic(fmt.Sprintf("Failed to load existing authentication configurations from DB: %v", axErr))
	}

	if len(tools) != 0 {
		for _, t := range tools {
			if axErr, _ = t.PushUpdate(); axErr != nil {
				panic(fmt.Sprintf("Init: Load authentication configuration %v failed: %v", t.GetID(), axErr))
			} else {
				utils.InfoLog.Printf("Loaded the authentication configurration successfully: %v", t.GetURL())
			}
		}
	}
}

// @Title TestConfiguration
// @Description Configure Test
// @Accept  json
// @Param   config	body    tool.ToolBase			true        "Configuration object"
// @Success 201 {object} tool.ToolBase
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /tools
// @Router /tools/test [POST]
func TestTool() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := make(map[string]interface{})
		err := utils.GetUnmarshalledBody(c, &t)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ)
			return
		}

		if toolType, ok := t[tool.ToolType]; !ok || toolType.(string) == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("type field is required."))
		}

		if id, ok := t[tool.ToolID]; ok && id.(string) != "" {
			old, axErr := tool.GetToolByID(id.(string))
			if axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}

			if old != nil {
				oldTool := old

				// Copy fields from old tool
				t["type"] = oldTool.GetType()
				t["category"] = oldTool.GetCategory()
				t["url"] = oldTool.GetURL()
				t["id"] = oldTool.GetID()
				if _, ok := t["password"]; !ok {
					t["password"] = oldTool.GetPassword()
				} else {
					password, ok := t["password"].(string)
					if ok {
						if password == "" {
							t["password"] = oldTool.GetPassword()
						}
					} else {
						t["password"] = oldTool.GetPassword()
					}
				}
			}
		}

		configBytes, err := json.Marshal(t)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err))
			return
		}

		toolObj, axErr := unmarshalTool(t[tool.ToolType].(string), configBytes)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		axErr, code := toolObj.Test()
		if axErr != nil {
			if code == 401 {
				// UI has some special logic for 401, work around in the backend for now
				code = 400
			}
			c.JSON(code, axErr)
			return
		}

		c.JSON(code, nullMap)
	}
}
