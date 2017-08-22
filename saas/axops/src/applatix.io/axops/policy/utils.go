// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package policy

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

func DeletePoliciesByRepo(repo string, best bool) *axerror.AXError {
	utils.InfoLog.Printf("Delete policy from %v starting\n", repo)
	policies, axErr := GetPolicies(map[string]interface{}{
		PolicyRepo:             repo,
		axdb.AXDBSelectColumns: []string{PolicyBranch, PolicyName, PolicyRepo, PolicyID},
	})
	utils.InfoLog.Printf("Delete %v policies from %v\n", len(policies), repo)

	if axErr != nil {
		if best {
			utils.ErrorLog.Println("Failed to delete branches:", axErr)
		} else {
			return axErr
		}
	}

	for i, _ := range policies {
		axErr = policies[i].Delete()
		if axErr != nil {
			if best {
				utils.ErrorLog.Println("Failed to delete branches:", axErr)
			} else {
				return axErr
			}
		}
	}

	utils.InfoLog.Printf("Delete policies from %v finished\n", repo)
	UpdateETag()
	return nil
}
