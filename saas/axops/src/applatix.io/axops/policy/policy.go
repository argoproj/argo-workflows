// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package policy

import (
	"encoding/json"
	"fmt"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/index"
	"applatix.io/axops/label"
	"applatix.io/axops/utils"
	"applatix.io/template"
)

const (
	InvalidStatus = "invalid"
)

type Policy struct {
	template.PolicyTemplate
	Enabled bool   `json:"enabled,omitempty"`
	Status  string `json:"status,omitempty"`
}

func (p *Policy) String() string {
	return fmt.Sprintf("%s (ID: %s, repo: %s, branch: %s, rev: %s)", p.Name, p.ID, p.Repo, p.Branch, p.Revision)
}

func (p *Policy) PolicyDB() (*policyDB, *axerror.AXError) {
	policyDB := &policyDB{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Repo:        p.Repo,
		Branch:      p.Branch,
		Revision:    p.Revision,
		Template:    p.Template,
		Enabled:     p.Enabled,
		Labels:      p.Labels,
		RepoBranch:  p.Repo + "_" + p.Branch,
		Status:      p.Status,
	}

	if policyDB.Labels == nil {
		policyDB.Labels = map[string]string{}
	}

	body, err := json.Marshal(p)
	if err != nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Can not marshal the policy body:" + err.Error())
	}

	policyDB.Body = string(body)

	index.SendToSearchIndexChan("policies", "name", p.Name)
	index.SendToSearchIndexChan("policies", "description", p.Description)
	index.SendToSearchIndexChan("policies", "repo", p.Repo)
	index.SendToSearchIndexChan("policies", "branch", p.Branch)
	index.SendToSearchIndexChan("policies", "template", p.Template)

	return policyDB, nil
}

func (p *Policy) Insert() (*Policy, *axerror.AXError) {

	p.ID = utils.GenerateUUIDv5(fmt.Sprintf("%s:%s:%s", p.Repo, p.Branch, p.Name))

	policyDB, axErr := p.PolicyDB()
	if axErr != nil {
		return nil, axErr
	}

	axErr = policyDB.insert()
	if axErr != nil {
		return nil, axErr
	}

	for key, value := range policyDB.Labels {
		lb := label.Label{
			Type:  label.LabelTypePolicy,
			Key:   key,
			Value: value,
		}

		if _, axErr := lb.Create(); axErr != nil {
			if axErr.Code == axerror.ERR_API_DUP_LABEL.Code {
				continue
			} else {
				return nil, axErr
			}
		}
	}

	UpdateETag()
	return p, nil
}

func (p *Policy) Update() (*Policy, *axerror.AXError) {
	if p.ID == "" {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Policy ID is missing")
	}

	policyDB, axErr := p.PolicyDB()
	if axErr != nil {
		return nil, axErr
	}

	axErr = policyDB.update()
	if axErr != nil {
		return nil, axErr
	}

	for key, value := range policyDB.Labels {
		lb := label.Label{
			Type:  label.LabelTypePolicy,
			Key:   key,
			Value: value,
		}

		if _, axErr := lb.Create(); axErr != nil {
			if axErr.Code == axerror.ERR_API_DUP_LABEL.Code {
				continue
			} else {
				return nil, axErr
			}
		}
	}

	UpdateETag()
	return p, nil
}

func (p *Policy) Delete() *axerror.AXError {
	policyDB, axErr := p.PolicyDB()
	if axErr != nil {
		return axErr
	}

	axErr = policyDB.delete()
	if axErr != nil {
		return axErr
	}

	UpdateETag()
	return nil
}

func GetPolicyByID(id string) (*Policy, *axerror.AXError) {
	policies, axErr := GetPolicies(map[string]interface{}{
		PolicyID: id,
	})
	if axErr != nil {
		return nil, axErr
	}

	if len(policies) == 0 {
		return nil, nil
	}

	return &policies[0], nil
}

func GetPolicies(params map[string]interface{}) ([]Policy, *axerror.AXError) {

	if params != nil && params[axdb.AXDBSelectColumns] != nil {
		fields := params[axdb.AXDBSelectColumns].([]string)
		fields = append(fields, PolicyID)
		fields = utils.DedupStringList(fields)
		params[axdb.AXDBSelectColumns] = fields
	}

	policies := []Policy{}
	policyDBs, axErr := getPolicyDBs(params)
	if axErr != nil {
		return nil, axErr
	}

	for i, _ := range policyDBs {
		policy, axErr := policyDBs[i].policy()
		if axErr != nil {
			utils.ErrorLog.Printf("Cannot turn the policy DB oject(%v %v %v) into policy object: %v.\n", policyDBs[i].Name, policyDBs[i].Repo, policyDBs[i].Branch, axErr)
			continue
		}
		policies = append(policies, *policy)
	}
	return policies, nil
}
