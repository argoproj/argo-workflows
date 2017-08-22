// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops

import (
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/restcl"
)

// system status
const (
	SystemOK           = 0
	SystemHalting      = 1
	SystemHalted       = 2
	SystemUnresponsive = 3
	SystemBooting      = 4
	SystemUpgrading    = 5
)

// interval we use
const (
	IntervalMinute = 60
	IntervalHour   = 3600
	IntervalDay    = (24 * IntervalHour)
)

const (
	RestData      = "data"
	RestBuildWait = "build_wait"
	RestTestWait  = "test_wait"
)

// query key words
const (
	QueryApp           = "app"
	QueryMinTime       = "min_time"
	QueryMaxTime       = "max_time"
	QueryLimit         = "limit"
	QueryOffset        = "offset"
	QueryCommit        = "commit"
	QueryRepo          = "repo"
	QueryBranch        = "branch"
	QueryName          = "name"
	QUeryBy            = "by"
	QueryFilterBy      = "filterBy"
	QueryFilterByValue = "filterByValue"
)

var nullMap = axdb.AXMap{}
var nullMapArray = []axdb.AXMap{}
var Dbcl *axdbcl.AXDBClient
var DevopsCl *restcl.RestClient
var WorkflowAdcCl *restcl.RestClient
var ArtifactCl *restcl.RestClient

//var AxmonCl *restcl.RestClient
var AxNotifierCl *restcl.RestClient

const (
	AxOpsApp     = "axops"
	axopsPerfApp = "axops_perf"
	usageHostId  = "host_id"

	COSTID  = "cost_id"
	APP     = "app"
	PROJ    = "project"
	SERVICE = "service"
)

// the variable is to specify if we return both total and subtotals on each partition key value
const StatsSummaryPlusGroupBy = "axops_allstats"

// HTTP headers used by NewSingleHostReverseProxyWithUserContext
const HTTPAxUserIDHeader = "X-AXUserID"
const HTTPAxUsernameHeader = "X-AXUsername"
