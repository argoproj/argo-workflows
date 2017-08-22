// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package user

import (
	"applatix.io/axdb"
)

const (
	UserTableName       = "user"
	UserID              = "id"
	UserName            = "username"
	UserFirstName       = "first_name"
	UserLastName        = "last_name"
	UserPassword        = "password"
	UserState           = "state"
	UserAuthSchemes     = "auth_schemes"
	UserGroups          = "groups"
	UserSettings        = "settings"
	UserViewPreferences = "view_preferences"
	UserLabels          = "labels"
	UserCtime           = "ctime"
	UserMtime           = "mtime"
)

var UserSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    UserTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		UserID:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		UserName:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		UserFirstName:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		UserLastName:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		UserPassword:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		UserState:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		UserAuthSchemes:     axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexNone},
		UserGroups:          axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong},
		UserSettings:        axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
		UserViewPreferences: axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
		UserLabels:          axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong},
		UserCtime:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		UserMtime:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}

const (
	SysReqTableName = "system_request"
	SysReqID        = "id"
	SysReqUserId    = "user_id"
	SysReqUserName  = "user_name"
	SysReqTarget    = "target"
	SysReqExpiry    = "expiry"
	SysReqType      = "type"
	SysReqData      = "data"
)

var SystemRequestSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    SysReqTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		SysReqID:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		SysReqUserId:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SysReqUserName: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SysReqTarget:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SysReqExpiry:   axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		SysReqType:     axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		SysReqData:     axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
	},
	Configs: map[string]interface{}{
		"default_time_to_live": int64(7 * axdb.OneDay),
	},
}

const (
	GroupTableName = "group"
	GroupID        = "id"
	GroupName      = "name"
	GroupUserNames = "usernames"
)

var GroupSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    GroupTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		GroupID:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		GroupName:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		GroupUserNames: axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexWeak},
	},
}
