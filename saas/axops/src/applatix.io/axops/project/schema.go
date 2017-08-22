// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package project

import (
	"encoding/json"
	"fmt"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"applatix.io/template"
)

const (
	ProjectTableName    = "project"
	ProjectID           = "id"
	ProjectName         = "name"
	ProjectDescription  = "description"
	ProjectVersion      = "version"
	ProjectRepo         = "repo"
	ProjectBranch       = "branch"
	ProjectRevision     = "revision"
	ProjectActions      = "actions"
	ProjectCategories   = "categories"
	ProjectAssets       = "assets"
	ProjectLabels       = "labels"
	ProjectAnnotations  = "annotations"
	ProjectRepoBranch   = "repo_branch"
	ProjectPublished    = "published"
	ProjectLabelsKeys   = "project_labels_keys"
	ProjectLabelsValues = "project_labels_values"
)

var ProjectSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    ProjectTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ProjectID:           axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		ProjectName:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
		ProjectDescription:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ProjectVersion:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ProjectRepo:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ProjectBranch:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
		ProjectRevision:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ProjectActions:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ProjectCategories:   axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexWeak},
		ProjectAssets:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ProjectLabels:       axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
		ProjectRepoBranch:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		ProjectPublished:    axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexWeak},
		ProjectLabelsKeys:   axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexNone},
		ProjectLabelsValues: axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}

type projectDB struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Version      string            `json:"version"`
	Repo         string            `json:"repo"`
	Branch       string            `json:"branch"`
	Revision     string            `json:"revision"`
	Actions      string            `json:"actions"`
	Assets       string            `json:"assets"`
	Labels       map[string]string `json:"labels"`
	Categories   []string          `json:"categories"`
	RepoBranch   string            `json:"repo_branch"`
	Published    bool              `json:"published,omitempty"`
	LabelsKeys   []string          `json:"project_labels_keys"`
	LabelsValues []string          `json:"project_labels_values"`
}

func (p *projectDB) project() (*Project, *axerror.AXError) {
	project := &Project{}
	project.ID = p.ID
	project.Name = p.Name
	project.Description = p.Description
	project.Version = template.NumberOrString(p.Version)
	project.Repo = p.Repo
	project.Branch = p.Branch
	project.Revision = p.Revision
	project.Labels = p.Labels
	project.Categories = p.Categories
	project.Published = p.Published

	// unmarshall actions and assets
	if len(p.Actions) != 0 {
		err := json.Unmarshal([]byte(p.Actions), &project.Actions)
		if err != nil {
			utils.InfoLog.Printf("Can't unmarshal Actions with  ERR: %v", err)
			return nil, axerror.ERR_AXDB_INTERNAL.NewWithMessage(fmt.Sprintf("Can't unmarshal actions from string: %s", p.Actions))
		}
	}
	if len(p.Assets) != 0 {
		err := json.Unmarshal([]byte(p.Assets), &project.Assets)
		if err != nil {
			utils.InfoLog.Printf("Can't unmarshal Assets with  ERR: %v", err)
			return nil, axerror.ERR_AXDB_INTERNAL.NewWithMessage(fmt.Sprintf("Can't unmarshal assets from string: %s", p.Assets))
		}
	}

	return project, nil
}

func (p *projectDB) update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, ProjectTableName, p); axErr != nil {
		return axErr
	}
	return nil
}

func (p *projectDB) insert() *axerror.AXError {
	if _, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, ProjectTableName, p); axErr != nil {
		return axErr
	}
	return nil
}

func (p *projectDB) delete() *axerror.AXError {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, ProjectTableName, []*projectDB{p})
	if axErr != nil {
		return axErr
	}
	return nil
}

func getProjectDBs(params map[string]interface{}) ([]projectDB, *axerror.AXError) {
	var projects []projectDB
	dbErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, ProjectTableName, params, &projects)

	if dbErr != nil {
		return nil, dbErr
	}
	return projects, nil
}
