// Copyright 2015-2016 Applatix, Inc. All rights reserved.

package core

import (
	"applatix.io/axdb"
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type TimedKeyValueTable struct {
	Table
	timeView *TimedKeyValueTable // pointer to time ordered materialized view.
}

func (table *TimedKeyValueTable) init() *axdb.AXDBError {
	table.Table.init()
	// add our ax_ columns
	table.Columns[axdb.AXDBWeekColumnName] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexClustering}
	table.Columns[axdb.AXDBTimeColumnName] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexClustering}

	// we need another view that's purely time based. We create the backend view when doing initBackend, but init the view
	// as a table here. Note that the timeView table's partitionKeys array will be 0.
	if len(table.partitionKeys) > 0 {
		t := axdb.Table{AppName: table.AppName, Name: table.Name + axdb.AXDBTimeViewSuffix, Type: axdb.TableTypeTimedKeyValue, Columns: map[string]axdb.Column{
			axdb.AXDBWeekColumnName: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexPartition},
			axdb.AXDBTimeColumnName: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexClustering},
		}}

		for colname, col := range table.Columns {
			if col.Index == axdb.ColumnIndexPartition {
				col.Index = axdb.ColumnIndexClustering
			}
			t.Columns[colname] = col
		}
		t.Stats = table.Stats

		table.timeView = &TimedKeyValueTable{Table{t, nil, "", nil, nil, 0, 0}, nil}
		table.timeView.real = table.timeView
		table.timeView.init()
	}

	return nil
}

func (table *TimedKeyValueTable) createMatView() *axdb.AXDBError {
	if table.timeView != nil {
		var keysBuf bytes.Buffer
		var buf bytes.Buffer
		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}
		buf.WriteString(fmt.Sprintf("CREATE MATERIALIZED VIEW %s%s AS SELECT * FROM %s WHERE ",
			table.fullName, axdb.AXDBTimeViewSuffix, table.fullName))
		first := true
		for colName, column := range table.Columns {
			if column.Index == axdb.ColumnIndexPartition || column.Index == axdb.ColumnIndexClustering || column.Index == axdb.ColumnIndexClusteringStrong {
				if first {
					buf.WriteString(fmt.Sprintf("%s IS NOT NULL ", colName))
					first = false
				} else {
					buf.WriteString(fmt.Sprintf("AND %s IS NOT NULL ", colName))
				}
				if colName != axdb.AXDBWeekColumnName && colName != axdb.AXDBTimeColumnName {
					keysBuf.WriteString(", " + colName)
				}
			}
		}
		buf.WriteString("PRIMARY KEY (ax_week, ax_time")
		buf.WriteString(keysBuf.String())
		buf.WriteString(") WITH CLUSTERING ORDER BY (ax_time desc)")
		infoLog.Println(buf.String())
		axErr := execQuery(buf.String(), false, true)
		if axErr != nil {
			return axErr
		}
	}
	return nil
}

func (table *TimedKeyValueTable) initBackend() *axdb.AXDBError {
	partitionKeys := table.getIndexString(axdb.ColumnIndexPartition, true, false)
	primaryKeys := table.getIndexString(axdb.ColumnIndexClustering, false, false)

	primaryOrderString := strings.Replace(primaryKeys, ",", " DESC,", -1)
	axErr := table.Table.initBackendWithPrimaryClause(fmt.Sprintf("PRIMARY KEY(%s, %s)) WITH CLUSTERING ORDER BY (%s DESC)",
		partitionKeys, primaryKeys, primaryOrderString))
	if axErr != nil {
		return axErr
	}

	axErr = table.createMatView()
	if axErr != nil {
		return axErr
	}

	return nil
}

func (table *TimedKeyValueTable) updateBackend(changedData UpdateData) *axdb.AXDBError {
	changedFlags := changedData.changedFlags
	var needDropMV bool
	for _, v := range changedFlags {
		if v == -1 {
			needDropMV = true
			break
		}
	}
	// if a materialized view is defined, drop it first
	if needDropMV && table.timeView != nil {
		infoLog.Printf("Drop materialized view %s in order to drop a column.", table.timeView.fullName)
		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}
		execQuery(fmt.Sprintf("DROP MATERIALIZED VIEW %s", table.timeView.fullName))
	}

	err := table.Table.updateBackend(changedData)
	if err != nil {
		return err
	}

	//re-create materialzied view
	if needDropMV {
		err = table.createMatView()
		if err != nil {
			return err
		}
		infoLog.Printf("Re-created materialized view %s after dropping a column.", table.timeView.fullName)
	}
	return nil
}

func (table *TimedKeyValueTable) deleteBackend() *axdb.AXDBError {
	// Attempt tear down. We will ignore errors and try to proceed through. If there is any failures, on restart we should
	// try to recover.
	infoLog.Printf("deleting timedkeyvalue backend for table %s view %p", table.fullName, table.timeView)
	if table.timeView != nil {
		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}
		execQuery(fmt.Sprintf("DROP MATERIALIZED VIEW %s", table.timeView.fullName))
	}

	return table.Table.deleteBackend()
}

func (table *TimedKeyValueTable) addGeneratedParams(params map[string]interface{}) *axdb.AXDBError {
	if _, exist := params[axdb.AXDBTimeColumnName]; exist {
		switch reflect.TypeOf(params[axdb.AXDBTimeColumnName]).Kind() {
		case reflect.Int64:
		// do nothing
		default:
			t, err := params[axdb.AXDBTimeColumnName].(json.Number).Int64()
			if err != nil {
				errorLog.Println(err)
				return axdb.NewAXDBError(axdb.RestStatusInvalid, err, fmt.Sprintf("ax_time specified is not a number, value passed: %v", params[axdb.AXDBTimeColumnName]))
			}
			params[axdb.AXDBTimeColumnName] = t
		}

		params[axdb.AXDBWeekColumnName] = params[axdb.AXDBTimeColumnName].(int64) / WeekInMicroSeconds
	}

	return nil
}

func (table *TimedKeyValueTable) save(data map[string]interface{}, isNewInsert bool) (resMap map[string]interface{}, axErr *axdb.AXDBError) {
	if isNewInsert {
		if _, exist := data[axdb.AXDBTimeColumnName]; !exist {
			data[axdb.AXDBTimeColumnName] = time.Now().UnixNano() / 1e3
		}
		resMap = map[string]interface{}{axdb.AXDBTimeColumnName: data[axdb.AXDBTimeColumnName]}
	} else {
		if _, exist := data[axdb.AXDBTimeColumnName]; !exist {
			errStr := fmt.Sprintf("%s not found for PUT request", axdb.AXDBTimeColumnName)
			errorLog.Println(errStr)
			return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
		}
	}

	table.addGeneratedParams(data)

	axErr = table.saveBeginWeekIfNeeded(data, data[axdb.AXDBTimeColumnName].(int64)/WeekInMicroSeconds)
	if axErr != nil {
		return nil, axErr
	}

	if isNewInsert {
		return resMap, table.execInsert(data)
	} else {
		return nil, table.execUpdate(data)
	}
}

func (table *TimedKeyValueTable) queryUsesPagination(params map[string]interface{}) bool {
	_, paramsHasTime := params[axdb.AXDBTimeColumnName]

	// if we use time, no pagination
	// if we use lucene index, no pagination
	// if we use all partitions, no pagination - we are getting the exact entry.
	// if we use secondary keys, no pagination
	return !paramsHasTime && !table.queryUsesLuceneIndex(params) && !table.queryUsesPartitionIndex(params) && !table.queryUsesSecondaryIndex(params)
}

func (table *TimedKeyValueTable) queryUseAscendTimeOrder(params map[string]interface{}) bool {
	cols, exist := params[axdb.AXDBQueryOrderByASC]
	if exist {
		for _, col := range cols.([]interface{}) {
			colName := col.(string)
			if colName == axdb.AXDBTimeColumnName {
				column, ok := table.Columns[colName]
				if ok && (column.Index == axdb.ColumnIndexClustering || column.Index == axdb.ColumnIndexClusteringStrong) {
					return true
				} else {
					return false
				}
			}
		}
	}
	return false
}

func (table *TimedKeyValueTable) getMissedEndWeek(params map[string]interface{}) int64 {
	var week int64 = -1
	if params[axdb.AXDBTimeColumnName] == nil {
		// if the user is not querying for a specific time, we need to use proper constraints to get the
		// right range back.
		if params[axdb.AXDBQueryMaxTime] != nil {
			week = params[axdb.AXDBQueryMaxTime].(int64) / WeekInMicroSeconds
		} else {
			week = getWeekFromTime(time.Now())
		}
	} else {
		week = params[axdb.AXDBTimeColumnName].(int64) / WeekInMicroSeconds
	}
	return week

}

// params are typed
func (table *TimedKeyValueTable) getQueryStringForRequest(params map[string]interface{}) string {
	usesPartitionIndex := table.queryUsesPartitionIndex(params)
	var tableName string
	if table.queryUsesLuceneIndex(params) || usesPartitionIndex || table.queryUsesSecondaryIndex(params) && !table.queryUsesTime(params) {
		tableName = table.fullName
		if table.queryUsesLuceneIndex(params) {
			delete(params, axdb.AXDBWeekColumnName)
		}
	} else {
		tableName = table.timeView.fullName
		// find the proper week if it's not passed in
		if params[axdb.AXDBWeekColumnName] == nil {
			if table.queryUseAscendTimeOrder(params) {
				week, err := table.getBeginWeekFor(params, true)
				if err != nil {
					params[axdb.AXDBWeekColumnName] = 0
				} else {
					params[axdb.AXDBWeekColumnName] = week
				}
			} else {
				week := table.getMissedEndWeek(params)
				if week != -1 {
					params[axdb.AXDBWeekColumnName] = week
				}
			}
		}
	}

	return fmt.Sprintf("SELECT %s FROM %s %s %s LIMIT %d ALLOW FILTERING", table.getSelectColsClause(params), tableName, table.getWhereClause(params), table.getOrderByClause(params), getMaxEntries(params))
}

type KeyValueTable struct {
	Table
}

func (table *KeyValueTable) initBackend() *axdb.AXDBError {
	partitionKeys := table.getIndexString(axdb.ColumnIndexPartition, true, false)
	primaryKeys := table.getIndexString(axdb.ColumnIndexClustering, false, false)

	if partitionKeys == "" {
		errStr := fmt.Sprintf("no partition keys found for table %s", table.fullName)
		errorLog.Printf(errStr)
		return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("PRIMARY KEY(%s", partitionKeys))
	if primaryKeys != "" {
		buffer.WriteString(fmt.Sprintf(", %s", primaryKeys))
	}
	buffer.WriteString("))")

	return table.Table.initBackendWithPrimaryClause(buffer.String())
}

type CounterTable struct {
	Table
}

func (table *CounterTable) initBackend() *axdb.AXDBError {
	partitionKeys := table.getIndexString(axdb.ColumnIndexPartition, true, false)
	primaryKeys := table.getIndexString(axdb.ColumnIndexClustering, false, false)

	if partitionKeys == "" {
		errStr := fmt.Sprintf("no partition keys found for table %s", table.fullName)
		errorLog.Printf(errStr)
		return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}
	if primaryKeys != "" {
		errStr := fmt.Sprintf("now allowing primary key for counter type table %s", table.fullName)
		errorLog.Printf(errStr)
		return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}

	return table.Table.initBackendWithPrimaryClause(fmt.Sprintf("%s counter, PRIMARY KEY(%s))",
		axdb.AXDBCounterColumnName, partitionKeys))
}

func (table *CounterTable) save(data map[string]interface{}, isNewInsert bool) (map[string]interface{}, *axdb.AXDBError) {
	isAccessible, axErr := table.tableAccessible()
	if axErr != nil || !isAccessible {
		return nil, axErr
	}
	return nil, execQuery(fmt.Sprintf("UPDATE %s SET ax_counter = ax_counter +1 %s", table.fullName,
		table.getWhereClause(data)))
}
