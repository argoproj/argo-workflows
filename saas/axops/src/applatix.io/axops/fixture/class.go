// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package fixture

import (
	"encoding/json"
	"fmt"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"applatix.io/template"
)

type TypeMap map[string]interface{}

type Class struct {
	ID              string                                `json:"id,omitempty"`
	Name            string                                `json:"name,omitempty"`
	Description     string                                `json:"description,omitempty"`
	Attributes      TypeMap                               `json:"attributes,omitempty"`
	Actions         map[string]template.FixtureAction     `json:"actions,omitempty"`
	Repo            string                                `json:"repo,omitempty"`
	Branch          string                                `json:"branch,omitempty"`
	Revision        string                                `json:"revision,omitempty"`
	RepoBranch      string                                `json:"repo_branch,omitempty"`
	Status          string                                `json:"status,omitempty"`
	StatusDetail    TypeMap                               `json:"status_detail,omitempty"`
	ActionTemplates map[string]service.EmbeddedTemplateIf `json:"action_templates,omitempty"`
}

type classDB struct {
	ID              string `json:"id,omitempty"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	Attributes      string `json:"attributes,omitempty"`
	Actions         string `json:"actions,omitempty"`
	Repo            string `json:"repo,omitempty"`
	Branch          string `json:"branch,omitempty"`
	Revision        string `json:"revision,omitempty"`
	RepoBranch      string `json:"repo_branch,omitempty"`
	Status          string `json:"status,omitempty"`
	StatusDetail    string `json:"status_detail,omitempty"`
	ActionTemplates string `json:"action_templates,omitempty"`
}

type Attribute struct {
	Name    string        `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name"`
	Type    string        `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type"`
	Flags   string        `json:"flags,omitempty" yaml:"flags,omitempty" mapstructure:"flags"`
	Options []interface{} `json:"options,omitempty" yaml:"options,omitempty" mapstructure:"options"`
	Default interface{}   `json:"default,omitempty" yaml:"default,omitempty" mapstructure:"default"`
}

func (c *classDB) Class() (*Class, *axerror.AXError) {
	class := &Class{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Repo:        c.Repo,
		Branch:      c.Branch,
		Revision:    c.Revision,
		RepoBranch:  c.RepoBranch,
		Status:      c.Status,
	}
	// unmarshall attributes
	if len(c.Attributes) > 0 {
		if err := json.Unmarshal([]byte(c.Attributes), &class.Attributes); err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to unmarshal the fixture attributes string in fixture template:%v", err))
		}
	}
	// unmarshall actions
	if len(c.Actions) > 0 {
		err := json.Unmarshal([]byte(c.Actions), &class.Actions)
		if err != nil {
			utils.InfoLog.Printf("Can't unmarshal actions: %v", err)
			return nil, axerror.ERR_AXDB_INTERNAL.NewWithMessage(fmt.Sprintf("Can't unmarshal actions from string: %s", c.Actions))
		}
	}
	// unmarshall action templates
	if len(c.ActionTemplates) > 0 {
		var jsonRaw map[string]*json.RawMessage
		err := json.Unmarshal([]byte(c.ActionTemplates), &jsonRaw)
		if err != nil {
			utils.InfoLog.Printf("Can't unmarshal actions templates: %v", err)
			return nil, axerror.ERR_AXDB_INTERNAL.NewWithMessagef("Can't unmarshal actions templates from string: %s", c.ActionTemplates)
		}
		actionTemplates := make(map[string]service.EmbeddedTemplateIf)
		for action, raw := range jsonRaw {
			tmpl, axErr := service.UnmarshalEmbeddedTemplate([]byte(*raw))
			if axErr != nil {
				utils.InfoLog.Printf("Can't unmarshal actions templates: %v", axErr)
				return nil, axerror.ERR_AXDB_INTERNAL.NewWithMessagef("Can't unmarshal actions templates from string: %s", c.Actions)
			}
			actionTemplates[action] = tmpl
		}
		class.ActionTemplates = actionTemplates
	}
	// unmarshall status detail
	if len(c.StatusDetail) > 0 {
		err := json.Unmarshal([]byte(c.StatusDetail), &class.StatusDetail)
		if err != nil {
			utils.InfoLog.Printf("Can't unmarshal status detail: %v", err)
			return nil, axerror.ERR_AXDB_INTERNAL.NewWithMessage(fmt.Sprintf("Can't unmarshal status detail from string: %s", c.StatusDetail))
		}
	}
	return class, nil
}

func GetClassByID(id string) (*Class, *axerror.AXError) {
	classes, axErr := GetClasses(map[string]interface{}{
		ClassID: id,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(classes) == 0 {
		return nil, nil
	}

	c := classes[0]
	return &c, nil
}

func GetClasses(params map[string]interface{}) ([]Class, *axerror.AXError) {
	classDBs := []classDB{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, ClassTableName, params, &classDBs)
	if axErr != nil {
		return nil, axErr
	}

	classes := []Class{}
	for _, classDB := range classDBs {
		class, axErr := classDB.Class()
		if axErr != nil {
			return classes, axErr
		}
		classes = append(classes, *class)
	}

	return classes, nil
}
