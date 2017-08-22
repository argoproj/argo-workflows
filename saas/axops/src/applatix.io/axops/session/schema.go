// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package session

import "applatix.io/axdb"

const (
	SessionTableName = "session"
	SessionID        = "id"
	SessionUserId    = "userid"
	SessionUserName  = "username"
	SessionUserState = "state"
	SessionScheme    = "scheme"
	SessionCTIME     = "ctime"
	SessionExpiry    = "expiry"
)

var SessionSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    SessionTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		SessionID:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		SessionUserId:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		SessionUserName:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		SessionUserState: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		SessionScheme:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SessionCTIME:     axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		SessionExpiry:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
	Configs: map[string]interface{}{
		"default_time_to_live": int64(4 * axdb.OneDay),
	},
}
