// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package project

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

func DeleteProjectsByRepo(repo string, best bool) *axerror.AXError {
	utils.InfoLog.Printf("Delete project from %v starting\n", repo)
	projects, axErr := GetProjects(map[string]interface{}{
		ProjectRepo:            repo,
		axdb.AXDBSelectColumns: []string{ProjectBranch, ProjectName, ProjectRepo, ProjectID},
	})
	utils.InfoLog.Printf("Delete %v projects from %v\n", len(projects), repo)

	if axErr != nil {
		if best {
			utils.ErrorLog.Println("Failed to delete branches:", axErr)
		} else {
			return axErr
		}
	}

	for i, _ := range projects {
		axErr = projects[i].Delete()
		if axErr != nil {
			if best {
				utils.ErrorLog.Println("Failed to delete branches:", axErr)
			} else {
				return axErr
			}
		}
	}

	utils.InfoLog.Printf("Delete projects from %v finished\n", repo)
	UpdateETag()
	return nil
}
