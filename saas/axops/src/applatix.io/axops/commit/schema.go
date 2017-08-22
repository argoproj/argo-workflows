// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package commit

import "applatix.io/axdb"

const (
	CommitTable       = "commits"
	CommitRevision    = "revision"
	CommitRepo        = "repo"
	CommitBranch      = "branch"
	CommitAuthor      = "author"
	CommitCommitter   = "committer"
	CommitDescription = "description"
	CommitDate        = "date"
	CommitBranches    = "branches"
	CommitJobs        = "jobs"
	CommitJobsInit    = "jobs_init"
	CommitJobsWait    = "jobs_wait"
	CommitJobsRun     = "jobs_run"
	CommitJobsFail    = "jobs_fail"
	CommitJobsSuccess = "jobs_success"
)

var CommitSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    CommitTable,
	Type:    axdb.TableTypeTimedKeyValue,
	Columns: map[string]axdb.Column{
		CommitRepo:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		CommitRevision: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		//CommitBranch:      axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexWeak},
		//CommitAuthor:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexWeak},
		//CommitCommitter:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexWeak},
		//CommitDescription: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		CommitDate:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		CommitJobs:        axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},
		CommitJobsInit:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		CommitJobsWait:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		CommitJobsRun:     axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		CommitJobsFail:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		CommitJobsSuccess: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
	//UseSearch: true,
}
