// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package branch

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

//var branchCols = []string{BranchID, BranchName, BranchRepo, BranchProject, BranchRevision}

type Branch struct {
	Name     string `json:"name,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Revision string `json:"revision,omitempty"`
}

type BranchData struct {
	Data []Branch `json:"data"`
}

func GetBranches(params map[string]interface{}) ([]Branch, *axerror.AXError) {
	data := BranchData{
		Data: []Branch{},
	}

	if params == nil {
		params = map[string]interface{}{}
	}

	axErr := utils.DevopsCl.Get("scm/branches", params, &data)
	if axErr != nil {
		return nil, axErr
	}
	return data.Data, nil
}
