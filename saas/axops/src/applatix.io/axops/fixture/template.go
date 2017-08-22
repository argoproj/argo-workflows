package fixture

import (
	"encoding/json"
	"fmt"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"applatix.io/template"
)

type templateDB struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Attributes  string `json:"attributes"`
	Repo        string `json:"repo"`
	Branch      string `json:"branch"`
	Revision    string `json:"revision"`
	Actions     string `json:"actions"`
	RepoBranch  string `json:"repo_branch"`
}

func (c *templateDB) Template() (*template.FixtureTemplate, *axerror.AXError) {
	tmpl := template.FixtureTemplate{}
	tmpl.ID = c.ID
	tmpl.Repo = c.Repo
	tmpl.Branch = c.Branch
	tmpl.Revision = c.Revision
	tmpl.Name = c.Name
	tmpl.Description = c.Description
	// unmarshall attributes
	if len(c.Attributes) > 0 {
		if err := json.Unmarshal([]byte(c.Attributes), &tmpl.Attributes); err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to unmarshal the fixture attributes string in fixture template:%v", err))
		}
	}
	// unmarshall actions
	if len(c.Actions) > 0 {
		err := json.Unmarshal([]byte(c.Actions), &tmpl.Actions)
		if err != nil {
			utils.InfoLog.Printf("Can't unmarshal Actions with  ERR: %v", err)
			return nil, axerror.ERR_AXDB_INTERNAL.NewWithMessage(fmt.Sprintf("Can't unmarshal actions from string: %s", c.Actions))
		}
	}
	return &tmpl, nil
}

func (c *templateDB) insert() *axerror.AXError {
	if _, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, TemplateTableName, c); axErr != nil {
		return axErr
	}
	return nil
}

func (c *templateDB) update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, TemplateTableName, c); axErr != nil {
		return axErr
	}
	return nil
}

func (c *templateDB) delete() *axerror.AXError {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, TemplateTableName, []*templateDB{c})
	if axErr != nil {
		return axErr
	}
	return nil
}

func ToTemplateDB(tmpl *template.FixtureTemplate) (*templateDB, *axerror.AXError) {
	tmplDB := templateDB{
		ID:          tmpl.ID,
		Name:        tmpl.Name,
		Description: tmpl.Description,
		Repo:        tmpl.Repo,
		Branch:      tmpl.Branch,
		Revision:    tmpl.Revision,
	}
	if tmpl.Attributes != nil {
		attributesBytes, err := json.Marshal(tmpl.Attributes)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the fixture attributes in fixture template: %v", err))
		}
		tmplDB.Attributes = string(attributesBytes)
	}
	if tmpl.Actions != nil {
		actionBytes, err := json.Marshal(tmpl.Actions)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the fixture actions in fixture template: %v", err))
		}
		tmplDB.Actions = string(actionBytes)
	}
	if tmpl.Branch != "" && tmpl.Repo != "" {
		tmplDB.RepoBranch = fmt.Sprintf("%s_%s", tmpl.Repo, tmpl.Branch)
	}
	return &tmplDB, nil
}

func UpsertFixtureTemplate(tmpl *template.FixtureTemplate) *axerror.AXError {
	UpdateETag()
	templateDB, axErr := ToTemplateDB(tmpl)
	if axErr != nil {
		return axErr
	}
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, TemplateTableName, templateDB); axErr != nil {
		return axErr
	}
	UpdateETag()
	return nil
}

func DeleteFixtureTemplateByID(id string) *axerror.AXError {
	UpdateETag()
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, TemplateTableName, []map[string]interface{}{{TemplateID: id}})
	return axErr
}

func GetFixtureTemplates(params map[string]interface{}) ([]template.FixtureTemplate, *axerror.AXError) {
	templateDBs := []templateDB{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, TemplateTableName, params, &templateDBs)
	if axErr != nil {
		return nil, axErr
	}

	templates := make([]template.FixtureTemplate, len(templateDBs))
	for i := range templateDBs {
		template, axErr := templateDBs[i].Template()
		if axErr != nil {
			return templates, axErr
		}
		templates[i] = *template
	}

	return templates, nil
}

func GetFixtureTemplateByID(id string) (*template.FixtureTemplate, *axerror.AXError) {
	templates, axErr := GetFixtureTemplates(map[string]interface{}{
		TemplateID: id,
	})
	if axErr != nil {
		return nil, axErr
	}
	if len(templates) == 0 {
		return nil, nil
	}
	return &templates[0], nil
}
