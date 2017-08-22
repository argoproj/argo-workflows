package service

import (
	"encoding/json"
	"sort"
	"time"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/commit"
	"applatix.io/axops/utils"
	"applatix.io/restcl"
)

var fixMgrCl = restcl.NewRestClientWithTimeout("http://fixturemanager.axsys:8912", 5*time.Minute)

const RedisServiceCtnKey = "service-%v-ctn-%v"

func DeleteTemplatesByRepo(repo string, best bool) *axerror.AXError {
	utils.InfoLog.Printf("Delete templates from %v starting\n", repo)
	templates, axErr := GetTemplates(map[string]interface{}{
		TemplateRepo:           repo,
		axdb.AXDBSelectColumns: []string{TemplateBranch, TemplateName, TemplateRepo, TemplateId},
	})
	utils.InfoLog.Printf("Delete %v templates from %v\n", len(templates), repo)

	if axErr != nil {
		utils.ErrorLog.Printf("Failed to retrieve templates: %v", axErr)
		if !best {
			return axErr
		}
	}

	for _, tmpl := range templates {
		axErr = DeleteTemplateById(tmpl.GetID())
		if axErr != nil {
			utils.ErrorLog.Printf("Failed to delete %s: %v", tmpl, axErr)
			if !best {
				return axErr
			}
		}
	}

	utils.InfoLog.Printf("Delete templates from %v finished\n", repo)
	UpdateTemplateETag()
	return nil
}

// This implementation can lead to inconsistent stats under the race condition of two services completing at the same time.
// The proper implementation should use channels hashed on commit id. Will do later.
func UpdateCommitJobHistory(revision string, repo string) *axerror.AXError {

	params := map[string]interface{}{
		ServiceRevision:        revision,
		axdb.AXDBSelectColumns: []string{ServiceName},
		ServiceIsTask:          true,
	}

	servicesLive, axErr := GetServicesFromTable(RunningServiceTable, false, params)
	if axErr != nil {
		return axErr
	}

	servicesDone, axErr := GetServicesFromTable(DoneServiceTable, false, params)
	if axErr != nil {
		return axErr
	}

	services := []*Service{}
	services = append(services, servicesLive...)
	services = append(services, servicesDone...)
	sort.Sort(SerivceSorter(services))

	commitDB, axErr := commit.GetCommitDBByRevision(revision, repo)
	if axErr != nil {
		return axErr
	}

	if commitDB == nil {
		newCommit, axErr := commit.GetCommitByRevision(revision, repo)
		if axErr != nil {
			return axErr
		}

		if newCommit == nil {
			return nil
		}

		commitDB = newCommit.ToCommitDB()
	}

	commitDB.JobsInit = 0
	commitDB.JobsFail = 0
	commitDB.JobsRun = 0
	commitDB.JobsSuccess = 0
	commitDB.JobsWait = 0
	commitDB.Jobs = []string{}

	for i, s := range services {
		if i < 10 {
			jobBytes, err := json.Marshal(s.JobSummary())
			if err != nil {
				return axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to marshal object %v", s.JobSummary())
			}
			commitDB.Jobs = append(commitDB.Jobs, string(jobBytes))
		}
		switch s.Status {
		case utils.ServiceStatusInitiating:
			commitDB.JobsInit++
		case utils.ServiceStatusWaiting:
			commitDB.JobsWait++
		case utils.ServiceStatusRunning, utils.ServiceStatusCanceling:
			commitDB.JobsRun++
		case utils.ServiceStatusFailed, utils.ServiceStatusCancelled:
			commitDB.JobsFail++
		default:
			commitDB.JobsSuccess++
		}
	}
	return commitDB.Update()
}
