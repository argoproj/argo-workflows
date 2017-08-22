// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axdb

import (
	"applatix.io/axerror"
	"fmt"
	"strings"
)

type AXMap map[string]interface{}

type DBMetaData struct {
	Version string //the version of database; all tables share the same version
}

// defines one column
type Column struct {
	Type                  int // type of the column
	Index                 int // the type of index on the column
	IndexFlagForMapColumn int // flag to indicate if index on key, value or both for map column: 0 - value, 1 - key, 2 - both
}

// defines one table
type Table struct {
	Name                 string
	AppName              string                 // name of app
	Type                 int                    // time series tables will always return query results in reverse time order
	Columns              map[string]Column      // column name => Column
	Stats                map[string]int         // column name => bitmap of the ColumnStat we want to keep
	IndexOrder           []string               // the order of indexes, optional. Earlier ones are more important.
	Configs              map[string]interface{} // the table configuration like TTL
	UseSearch            bool                   // enable the full text search
	ExcludedIndexColumns map[string]bool        // columns that are not lucene indexed.
}

func (t Table) Copy() Table {
	copy := Table{
		Name:                 t.Name,
		AppName:              t.AppName,
		Type:                 t.Type,
		UseSearch:            t.UseSearch,
		ExcludedIndexColumns: map[string]bool{},
		Columns:              map[string]Column{},
		Stats:                map[string]int{},
		IndexOrder:           []string{},
		Configs:              map[string]interface{}{},
	}

	if t.Columns != nil {
		for k, v := range t.Columns {
			copy.Columns[k] = v
		}
	}

	if t.ExcludedIndexColumns != nil {
		for k, v := range t.ExcludedIndexColumns {
			copy.ExcludedIndexColumns[k] = v
		}
	}

	if t.Stats != nil {
		for k, v := range t.Stats {
			copy.Stats[k] = v
		}
	}

	if t.Configs != nil {
		for k, v := range t.Configs {
			copy.Configs[k] = v
		}
	}

	if t.IndexOrder != nil {
		for _, v := range t.IndexOrder {
			copy.IndexOrder = append(copy.IndexOrder, v)
		}
	}

	return copy
}

func NameIsGenerated(name string) bool {
	return (name == AXDBWeekColumnName || name == AXDBUUIDColumnName || name == AXDBTimeColumnName)
}

type AXDBError struct {
	RestStatus int    // what we decided to return to our client
	SysError   error  // the underlying error
	Info       string // our own error string
}

func NewAXDBError(status int, sysError error, info string) *AXDBError {
	return &AXDBError{
		RestStatus: status,
		SysError:   sysError,
		Info:       info,
	}
}

func (e *AXDBError) Error() string {
	var underlyingError string
	if e.SysError == nil {
		underlyingError = "none"
	} else {
		underlyingError = e.SysError.Error()
	}
	return fmt.Sprintf("AXDB RestErrorCode: %d, info %s, underlying DB error %s", e.RestStatus, e.Info, underlyingError)
}

func (e *AXDBError) ToAXError() *axerror.AXError {
	switch e.RestStatus {
	case RestStatusInvalid:
		// for artifact manager to distinguish different failure cases.
		if strings.Contains(e.Info, "Conditional Update failed, no changed was made:") {
			return axerror.ERR_AXDB_CONDITIONAL_UPDATE_FAILURE.NewWithMessagef(e.Info)
		} else if strings.Contains(e.Info, "Conditional Update failed, the row doesn't exist.") {
			return axerror.ERR_AXDB_CONDITIONAL_UPDATE_FAILURE_NOT_EXIST.NewWithMessagef(e.Info)
		} else if strings.Contains(e.Info, "Cannot achieve consistency level QUORUM") {
			return axerror.ERR_AXDB_INTERNAL.NewWithMessage(e.Info)
		} else if strings.Contains(e.Info, "Operation timed out") {
			return axerror.ERR_AXDB_INTERNAL.NewWithMessage(e.Info)
		} else {
			return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessage(e.Info)
		}
	case RestStatusDenied:
		return axerror.ERR_AXDB_AUTH_FAILED.NewWithMessage(e.Info)
	case RestStatusForbidden:
		return axerror.ERR_AXDB_INSERT_DUPLICATE.NewWithMessage(e.Info)
	case RestStatusNotFound:
		return axerror.ERR_AXDB_TABLE_NOT_FOUND.NewWithMessage(e.Info)
	default:
		return axerror.ERR_AXDB_INTERNAL.NewWithMessage(e.Info)
	}
}
