// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"encoding/json"
)

const (
	ToolTableName = "tools"
	ToolID        = "id"
	ToolURL       = "url"
	ToolCategory  = "category"
	ToolType      = "type"
	ToolConfig    = "config"
)

var ToolSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    ToolTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ToolID:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ToolCategory: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexWeak},
		ToolURL:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexWeak},
		ToolType:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexWeak},
		ToolConfig:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
}

type ToolDB struct {
	ID       string `json:"id,omitempty"`
	URL      string `json:"url,omitempty"`
	Category string `json:"category,omitempty"`
	Type     string `json:"type,omitempty"`
	Config   string `json:"config,omitempty"`
}

func (t *ToolDB) ToTool() (Tool, *axerror.AXError) {
	config := t.Config
	switch t.Type {
	case TypeGitHub:
		tool := &GitHubConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeBitBucket:
		tool := &BitbucketConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeGitLab:
		tool := &GitLabConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeCodeCommit:
		tool := &CodeCommitConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeGIT:
		tool := &GitConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeSMTP:
		tool := &SMTPConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeSlack:
		tool := &SlackConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeSplunk:
		tool := &SplunkConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeSAML:
		tool := &SAMLConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeServer:
		tool := &ServerCertConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeDockerHub:
		tool := &DockerHubConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypePrivateRegistry:
		tool := &PrivateRegistryConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeRoute53:
		tool := &DomainConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeNexus:
		tool := &NexusConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeSecureKey:
		tool := &SecureKeyConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	case TypeJira:
		tool := &JiraConfig{}
		err := json.Unmarshal([]byte(config), tool)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the configuration.")
		}
		return tool, nil
	default:
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("The %v is not supported type.", t.Type)
	}
}

func (t *ToolDB) update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, ToolTableName, t); axErr != nil {
		return axErr
	}
	return nil
}

func (t *ToolDB) delete() *axerror.AXError {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, ToolTableName, []*ToolDB{t})
	if axErr != nil {
		return axErr
	}
	return nil
}

func GetToolByID(id string) (Tool, *axerror.AXError) {
	schemes, axErr := GetTools(map[string]interface{}{
		ToolID: id,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(schemes) == 0 {
		return nil, nil
	}

	return schemes[0], nil
}

func GetToolsByType(toolType string) ([]Tool, *axerror.AXError) {
	return GetTools(map[string]interface{}{
		ToolType: toolType,
	})
}

func GetToolsByCategory(category string) ([]Tool, *axerror.AXError) {
	return GetTools(map[string]interface{}{
		ToolCategory: category,
	})
}

func GetTools(params map[string]interface{}) ([]Tool, *axerror.AXError) {

	toolDBs := []ToolDB{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, ToolTableName, params, &toolDBs)
	if axErr != nil {
		return nil, axErr
	}

	tools := []Tool{}
	for i, _ := range toolDBs {
		tool, axErr := toolDBs[i].ToTool()
		if axErr != nil {
			return tools, axErr
		}
		tools = append(tools, tool)
	}

	return tools, nil
}
