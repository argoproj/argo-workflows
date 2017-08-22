// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package auth

import "applatix.io/axdb"

const (
	AuthRequestTableName = "auth_request"
	AuthRequestID        = "id"
	AuthRequestScheme    = "scheme"
	AuthRequestBody      = "request"
	AuthRequestCTIME     = "ctime"
	AuthRequestExpiry    = "expiry"
	AuthRequestData      = "data"
)

var AuthRequestSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    AuthRequestTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		AuthRequestID:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		AuthRequestScheme: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		AuthRequestBody:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		AuthRequestCTIME:  axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		AuthRequestExpiry: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		AuthRequestData:   axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
	},
	Configs: map[string]interface{}{
		"default_time_to_live": int64(1 * axdb.OneDay),
	},
}
