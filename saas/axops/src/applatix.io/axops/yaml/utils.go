package yaml

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

func DeleteYamlDataByBranch(repo, branch, revision string, best bool) *axerror.AXError {
	utils.InfoLog.Printf("Delete YAML data from repo - %v branch - %v starting\n", repo, branch)
	axErr := HandleYamlUpdateEvent(repo, branch, revision, []interface{}{})
	if axErr != nil {
		if best {
			utils.DebugLog.Println(axErr)
		} else {
			return axErr
		}
	}
	utils.InfoLog.Printf("Delete YAML data from repo - %v branch - %v finished\n", repo, branch)
	return nil
}
