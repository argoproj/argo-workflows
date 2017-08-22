// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package commit

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"encoding/json"
	"sort"
)

// API object
type ApiCommit struct {
	Date        int64    `json:"date,omitempty"`
	AuthorDate  int64    `json:"author_data,omitempty"`
	CommitDate  int64    `json:"commit_data,omitempty"`
	Author      string   `json:"author,omitempty"`
	Description string   `json:"description,omitempty"`
	Parents     []string `json:"parents,omitempty"`
	Revision    string   `json:"revision,omitempty"`
	Repo        string   `json:"repo,omitempty"`
	Branches    []string `json:"branches,omitempty"`
	Branch      string   `json:"branch,omitempty"`
	Committer   string   `json:"committer,omitempty"`

	Jobs        []*JobSummary `json:"jobs,omitempty"`
	JobsInit    int64         `json:"jobs_init"`
	JobsWait    int64         `json:"jobs_wait"`
	JobsRun     int64         `json:"jobs_run"`
	JobsFail    int64         `json:"jobs_fail"`
	JobsSuccess int64         `json:"jobs_success"`
}

type Commit struct {
	Date        int64    `json:"date,omitempty"`
	AuthorDate  int64    `json:"author_data,omitempty"`
	CommitDate  int64    `json:"commit_data,omitempty"`
	Author      string   `json:"author,omitempty"`
	Description string   `json:"description,omitempty"`
	Parents     []string `json:"parents,omitempty"`
	Revision    string   `json:"revision,omitempty"`
	Repo        string   `json:"repo,omitempty"`
	Branches    []string `json:"branches,omitempty"`
	Committer   string   `json:"committer,omitempty"`

	Jobs        []string `json:"jobs,omitempty"`
	JobsInit    int64    `json:"jobs_init,omitempty"`
	JobsWait    int64    `json:"jobs_wait,omitempty"`
	JobsRun     int64    `json:"jobs_run,omitempty"`
	JobsFail    int64    `json:"jobs_fail,omitempty"`
	JobsSuccess int64    `json:"jobs_success,omitempty"`
}

type CommitDB struct {
	AxTime   int64  `json:"ax_time,omitempty"`
	Revision string `json:"revision,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Date     int64  `json:"date,omitempty"`

	Jobs        []string `json:"jobs"`
	JobsInit    int64    `json:"jobs_init"`
	JobsWait    int64    `json:"jobs_wait"`
	JobsRun     int64    `json:"jobs_run,"`
	JobsFail    int64    `json:"jobs_fail"`
	JobsSuccess int64    `json:"jobs_success"`
}

type JobSummary struct {
	Name           string `json:"name"`
	Status         int    `json:"status"`
	StartTime      int64  `json:"ax_time"`
	RunTime        int64  `json:"run_time"`
	AverageRunTime int64  `json:"average_runtime"`
	CreateTime     int64  `json:"create_time"`
	LaunchTime     int64  `json:"launch_time"`
	EndTime        int64  `json:"end_time"`
	WaitTime       int64  `json:"wait_time"`
}

type CommitMap map[string]interface{}

func (p *Commit) NewApiCommit() *ApiCommit {
	apiCommit := ApiCommit{
		Date:        p.Date,
		AuthorDate:  p.AuthorDate,
		CommitDate:  p.CommitDate,
		Author:      p.Author,
		Description: p.Description,
		Parents:     p.Parents,
		Revision:    p.Revision,
		Repo:        p.Repo,
		Branches:    p.Branches,
		Committer:   p.Committer,

		JobsInit:    p.JobsInit,
		JobsWait:    p.JobsWait,
		JobsRun:     p.JobsRun,
		JobsFail:    p.JobsFail,
		JobsSuccess: p.JobsSuccess,
	}

	if len(p.Branches) != 0 {
		masterIndex := sort.SearchStrings(p.Branches, "master")
		if masterIndex == len(p.Branches) || p.Branches[masterIndex] != "master" {
			apiCommit.Branch = p.Branches[0]
		} else {
			apiCommit.Branch = "master"
		}
	}

	for _, str := range p.Jobs {
		var s JobSummary
		e := json.Unmarshal([]byte(str), &s)
		if e != nil {
			utils.ErrorLog.Printf("Failed to unmarshal str %s into a job description", str)
		} else {
			apiCommit.Jobs = append(apiCommit.Jobs, &s)
		}
	}

	return &apiCommit
}

func (commit *Commit) Merge(c *CommitDB) {
	if c != nil {
		commit.Jobs = c.Jobs
		commit.JobsInit = c.JobsInit
		commit.JobsWait = c.JobsWait
		commit.JobsRun = c.JobsRun
		commit.JobsFail = c.JobsFail
		commit.JobsSuccess = c.JobsSuccess
	}
}

func (commit *Commit) ToCommitDB() *CommitDB {
	db := &CommitDB{}
	db.Date = commit.Date * 1e6
	db.AxTime = commit.Date * 1e6
	db.Revision = commit.Revision
	db.Repo = commit.Repo
	return db
}

func (commit *CommitDB) Update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, CommitTable, commit); axErr != nil {
		return axErr
	}
	UpdateETag()
	return nil
}

func GetAPICommitByRevision(revision, repo string) (*ApiCommit, *axerror.AXError) {
	commit, axErr := GetCommitByRevision(revision, repo)
	if axErr != nil {
		return nil, axErr
	}

	if commit == nil {
		return nil, nil
	}

	commitDB, axErr := GetCommitDBByRevision(revision, repo)
	if axErr != nil {
		return nil, axErr
	}

	commit.Merge(commitDB)

	return commit.NewApiCommit(), nil
}

func GetAPICommits(params map[string]interface{}) ([]ApiCommit, *axerror.AXError) {
	commits, axErr := GetCommits(params)
	if axErr != nil {
		return nil, axErr
	}

	if len(commits) > 0 {
		maxTime := commits[0].Date
		minTime := commits[len(commits)-1].Date

		dbParams := map[string]interface{}{
			axdb.AXDBQueryMaxTime: (maxTime + 1) * 1e6,
			axdb.AXDBQueryMinTime: (minTime - 1) * 1e6,
		}

		commitDBMaps, axErr := GetCommitDBsMap(dbParams)
		if axErr != nil {
			return nil, axErr
		}

		for i, _ := range commits {
			commits[i].Merge(commitDBMaps[commits[i].Revision])
		}
	}

	apiCommits := []ApiCommit{}
	for i, _ := range commits {
		apiCommits = append(apiCommits, *commits[i].NewApiCommit())
	}
	return apiCommits, nil
}

func GetCommitByRevision(revision, repo string) (*Commit, *axerror.AXError) {
	params := map[string]interface{}{
		"revision": revision,
	}

	if repo != "" {
		params[CommitRepo] = repo
	}

	commits, dbErr := GetCommits(params)

	if dbErr != nil {
		return nil, dbErr
	}

	if len(commits) == 0 {
		return nil, nil
	}
	return &commits[0], nil
}

type CommitData struct {
	Data []Commit `json:"data"`
}

func GetCommits(params map[string]interface{}) ([]Commit, *axerror.AXError) {
	data := CommitData{
		Data: []Commit{},
	}

	if params == nil {
		params = map[string]interface{}{}
	}

	axErr := utils.DevopsCl.Get("scm/commits", params, &data)
	if axErr != nil {
		return nil, axErr
	}
	return data.Data, nil
}

//func GetCommitMaps(params map[string]interface{}) ([]CommitMap, *axerror.AXError) {
//	var commits []CommitMap
//	dbErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, CommitTable, params, &commits)
//	if dbErr != nil {
//		return nil, dbErr
//	}
//	return commits, nil
//}

func GetCommitDBByRevision(revision, repo string) (*CommitDB, *axerror.AXError) {
	params := map[string]interface{}{
		"revision": revision,
	}

	if repo != "" {
		params[CommitRepo] = repo
	}

	commits, dbErr := GetCommitDBs(params)

	if dbErr != nil {
		return nil, dbErr
	}

	if len(commits) == 0 {
		return nil, nil
	}
	return commits[0], nil
}

func GetCommitDBs(params map[string]interface{}) ([]*CommitDB, *axerror.AXError) {
	var commits []*CommitDB
	dbErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, CommitTable, params, &commits)
	if dbErr != nil {
		return nil, dbErr
	}
	return commits, nil
}

func GetCommitDBsMap(params map[string]interface{}) (map[string]*CommitDB, *axerror.AXError) {
	commitsMap := map[string]*CommitDB{}
	commits, axErr := GetCommitDBs(params)

	if axErr != nil {
		return nil, axErr
	}

	for _, commit := range commits {
		commitsMap[commit.Revision] = commit
	}

	return commitsMap, nil
}
