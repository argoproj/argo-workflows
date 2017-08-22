// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axdb

// AXDB version
const AXDBVersion = "v1"
const AXDBTableVersion = "0"
const AXDBServiceName = "axdb"

// AXDB REST API status codes
const (
	RestStatusOK            = 200 // for all successes
	RestStatusInvalid       = 400 // for invalid parameters (GET) or JSON body
	RestStatusDenied        = 401 // for unauthorized access
	RestStatusForbidden     = 403 // request is not forbidden. When POSTing to KeyValue store but the key exists already
	RestStatusNotFound      = 404 // for invalid url
	RestStatusInternalError = 500 // internal error
)

const (
	AXDBArrayMax = 100000
)

const (
	OneSecond = 1
	OneMinute = 60 * OneSecond
	OneHour   = 60 * OneMinute
	OneDay    = 24 * OneHour
	OneWeek   = 7 * OneDay
	OneYear   = 52 * OneWeek
)

const (
	AXDBNullUUID = "00000000-0000-0000-0000-000000000000"
)

const (
	AXDBRollUpInterval = OneHour
)

// AXDB update operation code
const (
	StatsTableUpdateAllowed    = 1 // allow updating statTable implies to allow updating original table as well
	OriginalTableUpdateAllowed = 2
	NoOperationAllowed         = 3
)

const Luceneffix = "$$ax$$"
