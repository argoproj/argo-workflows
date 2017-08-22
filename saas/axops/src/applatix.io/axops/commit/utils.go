// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package commit

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

func DeleteCommitsByRepo(repo string, best bool) *axerror.AXError {
	utils.InfoLog.Printf("Delete commits from repo - %v starting\n", repo)
	utils.InfoLog.Printf("Delete commits from repo - %v finished\n", repo)
	return nil
}
