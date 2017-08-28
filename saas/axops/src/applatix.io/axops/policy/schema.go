// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package policy

import (
	"encoding/json"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

const (
	PolicyTableName   = "policy"
	PolicyID          = "id"
	PolicyName        = "name"
	PolicyDescription = "description"
	PolicyRepo        = "repo"
	PolicyBranch      = "branch"
	PolicyRevision    = "revision"
	PolicyTemplate    = "template"
	PolicyEnabled     = "enabled"
	PolicyBody        = "body"
	PolicyLabels      = "labels"
	PolicyRepoBranch  = "repo_branch"
	PolicyStatus      = "status"
)

var PolicySchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    PolicyTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		PolicyID:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		PolicyName:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
		PolicyDescription: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		PolicyRepo:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		PolicyBranch:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
		PolicyRevision:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		PolicyTemplate:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		PolicyEnabled:     axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
		PolicyBody:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		PolicyLabels:      axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
		PolicyRepoBranch:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		PolicyStatus:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}

type policyDB struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Repo        string            `json:"repo"`
	Branch      string            `json:"branch"`
	Revision    string            `json:"revision"`
	Template    string            `json:"template"`
	Enabled     bool              `json:"enabled"`
	Body        string            `json:"body"`
	Labels      map[string]string `json:"labels"`
	RepoBranch  string            `json:"repo_branch"`
	Status      string            `json:"status"`
}

func (p *policyDB) policy() (*Policy, *axerror.AXError) {
	policy := &Policy{}
	if p.Body != "" {
		err := json.Unmarshal([]byte(p.Body), policy)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not unmarshal the policy body:" + err.Error())
		}
	} else {
		policy.ID = p.ID
		policy.Name = p.Name
		policy.Description = p.Description
		policy.Repo = p.Repo
		policy.Branch = p.Branch
		policy.Revision = p.Revision
		policy.Template = p.Template
		policy.Labels = p.Labels
		policy.Enabled = p.Enabled
		policy.Status = p.Status
	}
	return policy, nil
}

func (p *policyDB) update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, PolicyTableName, p); axErr != nil {
		return axErr
	}
	return nil
}

func (p *policyDB) insert() *axerror.AXError {
	if _, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, PolicyTableName, p); axErr != nil {
		return axErr
	}
	return nil
}

func (p *policyDB) delete() *axerror.AXError {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, PolicyTableName, []*policyDB{p})
	if axErr != nil {
		return axErr
	}
	return nil
}

func getPolicyDBs(params map[string]interface{}) ([]policyDB, *axerror.AXError) {
	var policies []policyDB
	dbErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, PolicyTableName, params, &policies)

	if dbErr != nil {
		return nil, dbErr
	}
	return policies, nil
}
