// Copyright 2015-2016 Applatix, Inc. All rights reserved.

package core

import (
	"applatix.io/axdb"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gocql/gocql"
	"reflect"
	"sort"
	"strings"
	"time"
)

type TimeSeriesTable struct {
	Table
	timeView       *TimeSeriesTable // pointer to time ordered materialized view.
	statTable      *TimeSeriesTable // pointer to stat table, if any
	hasSumStat     bool
	hasPercentStat bool
	isStatTable    bool
}

func (table *TimeSeriesTable) createStatTable(initBackend bool) *axdb.AXDBError {
	// create stats table if needed
	if len(table.Stats) > 0 {
		t := axdb.Table{
			AppName: table.AppName,
			Name:    table.Name + axdb.AXDBStatSuffix,
			Type:    axdb.TableTypeTimeSeries,
			Columns: map[string]axdb.Column{
				axdb.AXDBIntervalColumnName: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexPartition},
			},
			// Stats table would have the same Configs
			Configs: table.Configs,
		}

		for colname, col := range table.Columns {
			if col.Index == axdb.ColumnIndexPartition || col.Index == axdb.ColumnIndexClustering || col.Index == axdb.ColumnIndexClusteringStrong {
				if colname != axdb.AXDBUUIDColumnName {
					t.Columns[colname] = col
				}
				continue
			}

			if table.Stats[colname] != 0 && (col.Type == axdb.ColumnTypeInteger || col.Type == axdb.ColumnTypeDouble) {
				t.Columns[colname+axdb.AXDBCountColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}

				if table.Stats[colname]&axdb.ColumnStatPercent != 0 {
					t.Columns[colname+axdb.AXDB10ColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					t.Columns[colname+axdb.AXDB20ColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					t.Columns[colname+axdb.AXDB30ColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					t.Columns[colname+axdb.AXDB40ColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					t.Columns[colname+axdb.AXDB50ColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					t.Columns[colname+axdb.AXDB60ColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					t.Columns[colname+axdb.AXDB70ColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					t.Columns[colname+axdb.AXDB80ColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					t.Columns[colname+axdb.AXDB90ColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					table.hasPercentStat = true
				}
				if table.Stats[colname]&axdb.ColumnStatSum != 0 {
					t.Columns[colname+axdb.AXDBSumColumnSuffix] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
					table.hasSumStat = true
				}
			}
		}

		table.statTable = &TimeSeriesTable{Table{t, nil, "", nil, nil, 0, 0}, nil, nil, false, false, true}
		table.statTable.real = table.statTable

		table.statTable.init()
		if initBackend {
			infoLog.Printf("Adding table %s.%s to DB, initBackend for stat table", table.AppName, table.Name)
			if theDB.replFactor > 1 {
				if err := theDB.WaitSchemaAgreement(); err != nil {
					return err
				}
			}
			axErr := table.statTable.initBackend()
			if axErr != nil {
				errorLog.Printf("Adding table %s.%s to DB, cannot initBackend for stat table", table.AppName, table.Name)
				return axErr
			}
			infoLog.Printf("Adding table %s.%s to DB, finished initBackend for stat table", table.AppName, table.Name)
		} else {
			infoLog.Printf("Adding table %s.%s to DB, skip initBackend for stat table", table.AppName, table.Name)
		}
	}
	return nil
}

func (table *TimeSeriesTable) init() *axdb.AXDBError {
	table.Table.init()

	if _, exist := table.Columns[axdb.AXDBIntervalColumnName]; !exist && len(table.partitionKeys) > 0 {
		// we need another view that's purely time based. We create the backend view when doing initBackend, but init the view
		// as a table here.
		t := axdb.Table{AppName: table.AppName, Name: table.Name + axdb.AXDBTimeViewSuffix, Type: axdb.TableTypeTimeSeries, Columns: map[string]axdb.Column{
			axdb.AXDBWeekColumnName: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexPartition},
			axdb.AXDBUUIDColumnName: axdb.Column{Type: axdb.ColumnTypeTimeUUID, Index: axdb.ColumnIndexClustering},
			axdb.AXDBTimeColumnName: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		}}

		for colname, col := range table.Columns {
			if col.Index == axdb.ColumnIndexPartition {
				col.Index = axdb.ColumnIndexClustering
			}
			t.Columns[colname] = col
		}
		t.Stats = table.Stats

		table.timeView = &TimeSeriesTable{Table{t, nil, "", nil, nil, 0, 0}, nil, nil, false, false, false}
		table.timeView.real = table.timeView

		table.timeView.init()
	}

	// add our ax_ columns
	table.Columns[axdb.AXDBWeekColumnName] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexPartition}
	if !table.isStatTable {
		table.Columns[axdb.AXDBUUIDColumnName] = axdb.Column{Type: axdb.ColumnTypeTimeUUID, Index: axdb.ColumnIndexClustering}
		table.Columns[axdb.AXDBTimeColumnName] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}
	} else {
		table.Columns[axdb.AXDBTimeColumnName] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexClustering}
	}

	return nil
}

func (table *TimeSeriesTable) createMatView() *axdb.AXDBError {
	if table.timeView != nil {
		var buf bytes.Buffer
		var keysBuf bytes.Buffer
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
				if colName != axdb.AXDBWeekColumnName && colName != axdb.AXDBUUIDColumnName {
					keysBuf.WriteString(", " + colName)
				}
			}
		}
		buf.WriteString("PRIMARY KEY (ax_week, ax_uuid")
		buf.WriteString(keysBuf.String())
		buf.WriteString(") WITH CLUSTERING ORDER BY (ax_uuid DESC)")
		infoLog.Println(buf.String())
		axErr := execQuery(buf.String(), false, true)
		if axErr != nil {
			return axErr
		}
	}
	return nil
}

func (table *TimeSeriesTable) initBackend() *axdb.AXDBError {
	partitionKeys := table.getIndexString(axdb.ColumnIndexPartition, true, false)
	primaryKeys := table.getIndexString(axdb.ColumnIndexClustering, false, true)

	var clusterColName string
	if table.Columns[axdb.AXDBUUIDColumnName].Index == axdb.ColumnIndexClustering || table.Columns[axdb.AXDBUUIDColumnName].Index == axdb.ColumnIndexClusteringStrong {
		clusterColName = axdb.AXDBUUIDColumnName
	} else {
		clusterColName = axdb.AXDBTimeColumnName
	}
	axErr := table.Table.initBackendWithPrimaryClause(fmt.Sprintf("PRIMARY KEY(%s, %s)) WITH CLUSTERING ORDER BY (%s DESC)",
		partitionKeys, primaryKeys, clusterColName))
	if axErr != nil {
		return axErr
	}

	axErr = table.createMatView()
	if axErr != nil {
		return axErr
	}

	axErr = table.createStatTable(true)
	if axErr != nil {
		return axErr
	}

	if table.timeView != nil {
		axErr = table.timeView.createStatTable(true)
		if axErr != nil {
			return axErr
		}
	}
	return nil
}

func (table *TimeSeriesTable) updateBackend(changedData UpdateData) *axdb.AXDBError {
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
		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}
		infoLog.Printf("Drop materialized view %s in order to drop a column.", table.timeView.fullName)
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

	// 1. the backend statTable exists; alter the statTable according to changedStats
	// 2. the statTable doesn't exist; it will be created at the first time a request with interval is made;
	//    we won't consider case 2 here because it will be dealt with by createStatsTable.

	// the stat table can be defined on the original table or timeseries materialized view
	if len(changedData.changedStatCols) != 0 {
		var tableNames []string
		tableName := fmt.Sprintf("%s%s", table.fullName, axdb.AXDBStatSuffix)
		tableNames = append(tableNames, tableName)
		tableName = fmt.Sprintf("%s%s%s", table.fullName, axdb.AXDBTimeViewSuffix, axdb.AXDBStatSuffix)
		tableNames = append(tableNames, tableName)

		for _, tableName := range tableNames {
			str := fmt.Sprintf("SELECT * FROM %s", tableName)
			err = execQuery(str)
			//the table exists
			if err == nil {
				table.updateBackendStatTable(changedData.changedStatCols, tableName)
			}
		}
	}

	return nil
}

func (table *TimeSeriesTable) updateBackendStatTable(changedStatCols map[string]bool, statTblName string) *axdb.AXDBError {
	var buffer bytes.Buffer
	var colNames []string
	var axErr *axdb.AXDBError
	for k, newColFlag := range changedStatCols {
		// the stat column to be added to the statTable
		if newColFlag {
			if table.Stats[k] != 0 && (table.Columns[k].Type == axdb.ColumnTypeInteger || table.Columns[k].Type == axdb.ColumnTypeDouble) {
				// it's a sum type of stat
				if table.Stats[k]&axdb.ColumnStatSum != 0 {
					colNames = []string{k + axdb.AXDBCountColumnSuffix, k + axdb.AXDBSumColumnSuffix}
				}
				if table.Stats[k]&axdb.ColumnStatPercent != 0 {
					// it's a percentage type of stat
					colNames = []string{k + axdb.AXDBCountColumnSuffix, k + axdb.AXDB10ColumnSuffix, k + axdb.AXDB20ColumnSuffix, k + axdb.AXDB30ColumnSuffix,
						k + axdb.AXDB40ColumnSuffix, k + axdb.AXDB50ColumnSuffix, k + axdb.AXDB60ColumnSuffix,
						k + axdb.AXDB70ColumnSuffix, k + axdb.AXDB80ColumnSuffix, k + axdb.AXDB90ColumnSuffix}
				}

				for _, colName := range colNames {
					buffer.Reset()
					buffer.WriteString(fmt.Sprintf("ALTER TABLE %s ADD %s double;", statTblName, colName))
					infoLog.Printf("*** ADD STATS COLUMNS: " + buffer.String())
					if theDB.replFactor > 1 {
						if err := theDB.WaitSchemaAgreement(); err != nil {
							return err
						}
					}
					axErr = execQuery(buffer.String())
				}
			}
		} else {
			// the column to be dropped from the statTable
			// We don't distinguish the type of stat (sum vs percentage) here, because we will ignore the error returned from execQuery.
			colNames = []string{k + axdb.AXDBCountColumnSuffix, k + axdb.AXDBSumColumnSuffix, k + axdb.AXDB10ColumnSuffix, k + axdb.AXDB20ColumnSuffix,
				k + axdb.AXDB30ColumnSuffix, k + axdb.AXDB40ColumnSuffix, k + axdb.AXDB50ColumnSuffix, k + axdb.AXDB60ColumnSuffix,
				k + axdb.AXDB70ColumnSuffix, k + axdb.AXDB80ColumnSuffix, k + axdb.AXDB90ColumnSuffix}
			for _, colName := range colNames {
				buffer.Reset()
				buffer.WriteString(fmt.Sprintf("ALTER TABLE %s DROP %s;", statTblName, colName))
				infoLog.Printf("*** DROP STATS COLUMNS: " + buffer.String())
				if theDB.replFactor > 1 {
					if err := theDB.WaitSchemaAgreement(); err != nil {
						return err
					}
				}
				axErr = execQuery(buffer.String())
			}
		}
	}
	return axErr
}

func (table *TimeSeriesTable) deleteBackend() *axdb.AXDBError {
	// Attempt tear down. We will ignore errors and try to proceed through. If there is any failures, on restart we should
	// try to recover.
	if table.statTable != nil {
		table.statTable.deleteBackend()
	}

	if table.timeView != nil {
		if table.timeView.statTable != nil {
			table.timeView.statTable.deleteBackend()
		}

		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}
		execQuery(fmt.Sprintf("DROP MATERIALIZED VIEW %s", table.timeView.fullName))
	}

	return table.Table.deleteBackend()
}

func (table *TimeSeriesTable) queryUseAscendTimeOrder(params map[string]interface{}) bool {
	cols, exist := params[axdb.AXDBQueryOrderByASC]
	if exist {
		for _, col := range cols.([]interface{}) {
			colName := col.(string)
			if colName == axdb.AXDBUUIDColumnName {
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

func (table *TimeSeriesTable) getMissedEndWeek(params map[string]interface{}) int64 {
	var week int64 = -1
	usePartitionIndex := table.queryUsesPartitionIndex(params)
	if params[axdb.AXDBUUIDColumnName] == nil {
		// if the user is not querying for a specific id, we need to use proper constraints to get the
		// right range back.
		if usePartitionIndex || !table.queryUsesSecondaryIndex(params) {
			if params[axdb.AXDBQueryMaxTime] != nil {
				maxTime := params[axdb.AXDBQueryMaxTime].(int64)
				week = maxTime / WeekInMicroSeconds
			} else {
				week = getWeekFromTime(time.Now())
			}
		}
	} else {
		time := uuidToTime(params[axdb.AXDBUUIDColumnName].(string))
		week = getWeekFromTime(time)
	}
	return week
}

// params are typed
func (table *TimeSeriesTable) getQueryStringForRequest(params map[string]interface{}) string {
	usePartitionIndex := table.queryUsesPartitionIndex(params)

	// if not use pagination or use lucene index, don't generate ax_week
	// find the proper week if it's not passed in
	if _, hasWeek := params[axdb.AXDBWeekColumnName]; !hasWeek {
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

	var tableName string
	if !table.queryUsesLuceneIndex(params) && !usePartitionIndex && (!table.queryUsesSecondaryIndex(params) || table.queryUsesTime(params)) {
		tableName = table.timeView.fullName
	} else {
		tableName = table.fullName
		if table.queryUsesLuceneIndex(params) {
			delete(params, axdb.AXDBWeekColumnName)
		}
	}

	return fmt.Sprintf("SELECT %s FROM %s %s %s LIMIT %d ALLOW FILTERING", table.getSelectColsClause(params), tableName, table.getWhereClause(params), table.getOrderByClause(params), getMaxEntries(params))
}

/* disable for now and use the parent's version. Need to figure out strategy when we do caching. One big question
 * we need to answer is whether there is any real use case to track life span per partition.
func (table *TimeSeriesTable) getBeginWeekFor(data map[string]interface{}) (int64, *axdb.AXDBError) {
	params := table.copyIndexData(data, axdb.ColumnIndexPartition)
	params[axdb.AXDBWeekColumnName] = 0
	params[axdb.AXDBUUIDColumnName] = NullUUID
	resArray, axErr := table.get(params)
	if len(resArray) == 0 {
		return 0, axErr
	}
	return resArray[0][axdb.AXDBTimeColumnName].(int64), axErr
}

func (table *TimeSeriesTable) saveBeginWeekIfNeeded(data map[string]interface{}, week int64) *axdb.AXDBError {
	beginWeek, axErr := table.getBeginWeekFor(data)
	if axErr == nil && beginWeek == 0 || beginWeek > week {
		params := table.copyIndexData(data, axdb.ColumnIndexPartition)
		params[axdb.AXDBWeekColumnName] = 0
		params[axdb.AXDBUUIDColumnName] = NullUUID
		params[axdb.AXDBTimeColumnName] = week
		axErr = table.execUpdate(params)
	}

	return axErr
}
*/

func (table *TimeSeriesTable) queryUsesPagination(params map[string]interface{}) bool {
	_, paramsHasId := params[axdb.AXDBUUIDColumnName]
	// if we use uuid, no pagination
	// if we use lucene, no pagination
	// if we use all partition keys, use pagination
	// if we don't use partition keys, and don't use secondary indexes, we will use the time materialized view, and use pagination
	return !paramsHasId && !table.queryUsesLuceneIndex(params) && (table.queryUsesPartitionIndex(params) || !table.queryUsesSecondaryIndex(params))
}

func (table *TimeSeriesTable) addGeneratedParams(params map[string]interface{}) *axdb.AXDBError {
	if _, exist := params[axdb.AXDBWeekColumnName]; !exist {
		if _, exist := params[axdb.AXDBUUIDColumnName]; exist {
			time := uuidToTime(params[axdb.AXDBUUIDColumnName].(string))
			if time.Equal(EpochTime) {
				return &axdb.AXDBError{RestStatus: axdb.RestStatusInvalid, Info: "trying to use epoch time in timeseries table"}
			}
			params[axdb.AXDBWeekColumnName] = getWeekFromTime(time)
		}
	}
	return nil
}

func (table *TimeSeriesTable) save(data map[string]interface{}, isNewInsert bool) (resMap map[string]interface{}, axErr *axdb.AXDBError) {
	isAccessible, axErr := table.tableAccessible()
	if axErr != nil || !isAccessible {
		return nil, axErr
	}

	if isNewInsert {
		var t time.Time
		if _, exist := data[axdb.AXDBTimeColumnName]; !exist {
			t = time.Now()
			data[axdb.AXDBTimeColumnName] = t.UnixNano() / 1e3
		} else {
			switch reflect.TypeOf(data[axdb.AXDBTimeColumnName]).Kind() {
			case reflect.Int64:
				t = time.Unix(data[axdb.AXDBTimeColumnName].(int64)/1e6, (data[axdb.AXDBTimeColumnName].(int64)%1e6)*1e3)
			default:
				timeus, err := data[axdb.AXDBTimeColumnName].(json.Number).Int64()
				if err != nil {
					errorLog.Println(err)
					return nil, &axdb.AXDBError{RestStatus: axdb.RestStatusInvalid, Info: "time field is not a valid number"}
				}
				t = time.Unix(timeus/1e6, (timeus%1e6)*1e3)
				data[axdb.AXDBTimeColumnName] = timeus
			}
		}
		uuid := gocql.UUIDFromTime(t)
		data[axdb.AXDBWeekColumnName] = data[axdb.AXDBTimeColumnName].(int64) / WeekInMicroSeconds
		if !table.isStatTable {
			data[axdb.AXDBUUIDColumnName] = uuid
			resMap = map[string]interface{}{axdb.AXDBUUIDColumnName: uuid, axdb.AXDBTimeColumnName: data[axdb.AXDBTimeColumnName]}
		} else {
			resMap = map[string]interface{}{axdb.AXDBTimeColumnName: data[axdb.AXDBTimeColumnName]}
		}

	} else {
		if data[axdb.AXDBUUIDColumnName] == nil {
			if !table.isStatTable {
				errorLog.Printf("Error: %s not found for PUT request", axdb.AXDBUUIDColumnName)
				return nil, &axdb.AXDBError{RestStatus: axdb.RestStatusInvalid, Info: "uuid is not specified for PUT request"}
			}
		}
		axErr = table.addGeneratedParams(data)
		if axErr != nil {
			return nil, axErr
		}
	}
	axErr = table.saveBeginWeekIfNeeded(data, data[axdb.AXDBWeekColumnName].(int64))
	if axErr != nil {
		return nil, axErr
	}

	return resMap, table.execUpdate(data)
}

func (table *TimeSeriesTable) get(params map[string]interface{}) (resultArray []map[string]interface{}, axErr *axdb.AXDBError) {
	if _, exist := params[axdb.AXDBIntervalColumnName]; exist {
		return table.getStats(params)
	}
	return table.Table.get(params)
}

// entry point for rolling up stats using lower level stats. i.e. rollup daily stats using hourly stats.
// input parameter:
//    data - the data point of lower level stats.
//    interval - the destination interval we want to roll up.

func (table *TimeSeriesTable) aggregateStats(minTime int64, maxTime int64, data []map[string]interface{}, interval int64, params map[string]interface{}) (resultArray []map[string]interface{}, axErr *axdb.AXDBError) {

	//get the partition keys of the base table, we will report the stats groupby the partition key values.
	var partitionParams map[string]interface{}
	if len(table.partitionKeys) != 0 {
		partitionParams = make(map[string]interface{})
		for _, key := range table.partitionKeys {
			partitionParams[key] = params[key]
		}
	}

	index := 0
	resultData := []map[string]interface{}{}

	//rollup multiple intervals, one iteration for each.
	for currentTime := maxTime; currentTime >= minTime; currentTime -= interval * 1e6 {
		intervalData := table.getStatCollector()
		intervalData.init(table, currentTime, interval, partitionParams, true)
		for {
			if index == len(data) {
				break
			}
			statTime := data[index][axdb.AXDBTimeColumnName].(int64)
			// current data is in the range of interval, add it.
			if statTime >= currentTime && statTime < currentTime+interval*1e6 {
				intervalData.addData(data[index])
				index++
			} else {
				//all data in the interval are collected, the aggregate data will be created and saved.
				//the aggregation logic for percent stats is in getAggregateHistogram()
				data := intervalData.getData()
				for _, rd := range data {
					rd[axdb.AXDBIntervalColumnName] = interval
					table.statTable.save(rd, true)
					resultData = append(resultData, rd)
				}
				break
			}
		}
		// run aggregation on the last one interval that is left over.
		data := intervalData.getData()
		if len(data) != 0 {
			for _, rd := range data {
				rd[axdb.AXDBIntervalColumnName] = interval
				table.statTable.save(rd, true)
				resultData = append(resultData, rd)
			}
		}
	}
	return resultData, nil
}

func (table *TimeSeriesTable) getStatCollector() StatCollector {
	tableStatFlag := 0
	for _, stat := range table.Stats {
		tableStatFlag |= stat
	}
	var intervalData StatCollector

	if tableStatFlag == axdb.ColumnStatAll {
		sumStatObj := SumStat{}
		percentStatObj := PercentStat{}
		intervalData = &AllStat{sumStat: sumStatObj, percentStat: percentStatObj}
	} else if tableStatFlag&axdb.ColumnStatPercent != 0 {
		intervalData = &PercentStat{}
	} else {
		intervalData = &SumStat{}
	}
	return intervalData
}

func (t *TimeSeriesTable) getStats(params map[string]interface{}) (resultArray []map[string]interface{}, axErr *axdb.AXDBError) {
	// real table and statTable we use
	var table *TimeSeriesTable
	var statTable *TimeSeriesTable
	if t.queryUsesPartitionIndex(params) {
		table = t
	} else {
		table = t.timeView
	}
	if table.statTable == nil {
		axErr = table.createStatTable(false)
		if axErr != nil {
			return nil, axErr
		}
	}
	statTable = table.statTable

	typedParams, axErr := statTable.getTypedParams(params)
	if axErr != nil {
		return nil, axErr
	}
	interval := typedParams[axdb.AXDBIntervalColumnName].(int64) * 1e6
	// time should be rounded to the nearest interval
	var minTime int64
	var maxTime int64
	nowUs := time.Now().UnixNano() / 1e3
	if _, exist := typedParams[axdb.AXDBQueryMaxTime]; exist {
		maxTime = typedParams[axdb.AXDBQueryMaxTime].(int64)
		if maxTime > nowUs {
			maxTime = nowUs
		}
	} else {
		maxTime = nowUs
	}
	if _, exist := typedParams[axdb.AXDBQueryMinTime]; exist {
		minTime = typedParams[axdb.AXDBQueryMinTime].(int64)
	} else {
		minTime = maxTime - interval*50
	}

	// stat datapoint timestamps rounded to interval.
	minTime = minTime / interval * interval
	maxTime = (maxTime - 1) / interval * interval
	if maxTime+interval >= nowUs {
		// this would result in one data point being partial, can't allow that.
		maxTime -= interval
	}
	currentTime := maxTime + interval

	//tell if we want to rollup a big interval using data from small interval
	isRollUp := false
	if typedParams[axdb.AXDBQuerySrcInterval] != nil {
		isRollUp = true
	}
	var statParams map[string]interface{}
	var dstInterval int64
	var srcInterval int64
	if isRollUp {
		dstInterval = typedParams[axdb.AXDBIntervalColumnName].(int64)
		val, _ := typedParams[axdb.AXDBQuerySrcInterval].(int)
		srcInterval = int64(val)
		//delete(typedParams, axdb.AXDBRollUpFlag)
		//delete(typedParams, axdb.AXDBQueryDstInterval)
		delete(typedParams, axdb.AXDBQuerySrcInterval)
		//srcInterval = int64(axdbRollupIntervalMap[int(dstInterval)])
		typedParams[axdb.AXDBIntervalColumnName] = srcInterval
		typedParams[axdb.AXDBQueryMinTime] = minTime
		typedParams[axdb.AXDBQueryMaxTime] = maxTime + interval

	} else {
		statParams = copyMap(typedParams)
		delete(statParams, axdb.AXDBIntervalColumnName)
		delete(statParams, axdb.AXDBSelectColumns)
		typedParams[axdb.AXDBQueryMinTime] = minTime
		typedParams[axdb.AXDBQueryMaxTime] = maxTime + interval
		infoLog.Printf("getStats min %d max %d current %d", minTime, maxTime, currentTime)
	}

	statArray, axErr := statTable.Table.get(typedParams)
	if axErr != nil {
		return nil, axErr
	}
	if isRollUp {
		infoLog.Printf(fmt.Sprintf("*** min %d, max %d, dstInterval %d", minTime, maxTime, dstInterval))
		return table.aggregateStats(minTime, maxTime, statArray, dstInterval, nil)
	}
	statArrayLen := len(statArray)
	infoLog.Printf("stat array len %d", statArrayLen)

	c := make(chan int, AXDBParallelLevel*10)
	totalRoutines := 0

	goBuildStats := func(beginTime int64, endTime int64) {
		timeRange := (endTime - beginTime) / AXDBParallelLevel / interval * interval
		if timeRange < interval {
			timeRange = interval
		}
		infoLog.Printf("goBuildStats begin %d end %d range %d", beginTime, endTime, timeRange)
		t := beginTime
		for ; t+timeRange < endTime; t += timeRange {
			go table.buildStats(t, t+timeRange-1, interval, statParams, c)
			totalRoutines++
		}
		go table.buildStats(t, endTime-1, interval, statParams, c)
		totalRoutines++
		/*
			go table.buildStats(beginTime, endTime -1, interval, statParams, c)
			totalRoutines++
		*/
	}

	// there may be multiple entries with the same stat time (when there are user specified clustering keys).
	for {
		totalRoutines = 0

		for _, stat := range statArray {
			statTime := stat[axdb.AXDBTimeColumnName].(int64)
			if statTime < currentTime {
				if statTime == currentTime-interval {
					currentTime = statTime
				} else {
					goBuildStats(statTime+interval, currentTime)
					currentTime = statTime
				}
			}
		}

		if currentTime > minTime {
			goBuildStats(minTime, currentTime)
			currentTime = maxTime + interval // reset for the next loop
		}

		if totalRoutines == 0 {
			break
		}

		for i := 0; i < totalRoutines; i++ {
			<-c
		}

		statArray, axErr = statTable.Table.get(typedParams)
		if axErr != nil {
			return nil, axErr
		}
		infoLog.Printf("again stat array len %d", len(statArray))
		if len(statArray) == statArrayLen {
			// if we can't build anymore it means we don't have raw data
			break
		}
		statArrayLen = len(statArray)
	}

	//var results []map[string]interface{}
	//for _,stat := range statArray {
	//	match, err := regexp.MatchString( "App=A", stat["tags"].(string));
	//	_ = "breakpoint"
	//	if err == nil && match == true {
	//		_ = "breakpoint"
	//		results = append(results, stat)
	//	}
	//}

	return statArray, nil
}

// Given the params, build the stats on disk and into outArray at position i, for total count. Returns the timestamp of the last entry (smallest timestamp)
func (table *TimeSeriesTable) buildStats(minTime int64, maxTime int64, interval int64, queryParams map[string]interface{}, doneChannel chan int) *axdb.AXDBError {
	params := copyMap(queryParams)
	params[axdb.AXDBQueryMaxTime] = maxTime
	params[axdb.AXDBQueryMinTime] = minTime
	infoLog.Printf("buildStats min %d max %d interval %d", minTime, maxTime, interval)

	// The partition keys are ax_interval and ax_week. The first clustering key is ax_time, with the other clustering keys defined by user.
	// For sum type of stats, we want to keep the accounting for the user defined clustering keys separate.
	// For histogram type of stats, we don't keep individual accounting for each user defined clustering key. The cost is
	// too high and we don't have a use case right now.

	var partitionParams map[string]interface{}
	if len(table.partitionKeys) != 0 {
		partitionParams = make(map[string]interface{})
		for _, key := range table.partitionKeys {
			partitionParams[key] = params[key]
		}
	}

	currentTime := maxTime / interval * interval // expected first data point
	// TODO change to get only the needed stats columns
	rawArray, axErr := table.get(params)
	if axErr != nil {
		doneChannel <- 1
		return axErr
	}
	intervalData := table.getStatCollector()
	/*
		tableStatFlag := 0
		for _, stat := range table.Stats {
			tableStatFlag |= stat
		}
		var intervalData StatCollector
		// Supporting only one type per table right now, implement AllStat later.
		if tableStatFlag == axdb.ColumnStatAll {
			sumStatObj := SumStat{}
			percentStatObj := PercentStat{}
			intervalData = &AllStat{sumStat: sumStatObj, percentStat: percentStatObj}
		} else if tableStatFlag&axdb.ColumnStatPercent != 0 {
			intervalData = &PercentStat{}
		} else {
			intervalData = &SumStat{}
		}
	*/
	intervalData.init(table, currentTime, interval, partitionParams, false)
	saveChannel := make(chan int, 2*(maxTime-minTime)/interval+1)
	saveCount := 0
	saveStatArray := func(dataArray []map[string]interface{}) {
		// TODO convert this to one operation to DB. Saving an array should be one operation.
		for _, d := range dataArray {
			table.statTable.save(d, true)
		}
		saveChannel <- 1
	}

	rawLen := len(rawArray)
	for i := 0; i < rawLen; i++ {
		data := rawArray[i]
		t := data[axdb.AXDBTimeColumnName].(int64)
		if t < currentTime {
			dataArray := intervalData.getData()
			if len(dataArray) > 0 {
				infoLog.Printf("saveStatToDB interval %d current %d", interval, currentTime)
				saveCount++
				go saveStatArray(dataArray)
			}

			currentTime = t / interval * interval
			intervalData.reset(currentTime)
		}
		intervalData.addData(data)
	}

	if rawLen < axdb.AXDBArrayMax {
		dataArray := intervalData.getData()
		if len(dataArray) > 0 {
			infoLog.Printf("saveStatToDB interval %d current %d", interval, currentTime)
			saveCount++
			go saveStatArray(dataArray)
		}
	}

	for i := 0; i < saveCount; i++ {
		<-saveChannel
	}
	infoLog.Printf("buildStats done, save count %d", saveCount)

	doneChannel <- 1
	return nil
}

type ColumnDescriptor struct {
	colName string
	colType int
}

type SumColumnDescriptor struct {
	ColumnDescriptor
	sumColName   string
	countColName string
}

type StatCollector interface {
	init(table *TimeSeriesTable, statTime int64, interval int64, params map[string]interface{}, isRollUp bool) // gathers table meta data information
	addData(data map[string]interface{})                                                                       // add one data point
	getData() []map[string]interface{}                                                                         // get the stat data
	reset(newTime int64)
}

type SumStat struct {
	columns  []SumColumnDescriptor
	keys     []string
	hasKeys  bool
	time     int64
	interval int64
	params   map[string]interface{}
	statData map[string](map[string]float64)
	isRollUp bool
}

func concatKeys(data map[string]interface{}, keys []string) string {
	var buf bytes.Buffer
	for _, key := range keys {
		if _, ok := data[key].(string); ok {
			buf.WriteString(data[key].(string) + ",")
		} else {
			valueMap := data[key].(map[string]interface{})
			valString := axdb.SerializeOrderedMap(valueMap)
			buf.WriteString(valString + ",")
		}
	}
	return buf.String()
}

func (s *SumStat) init(table *TimeSeriesTable, statTime int64, interval int64, params map[string]interface{}, isRollUp bool) {
	s.time = statTime
	s.interval = interval
	s.params = params
	s.isRollUp = isRollUp
	for statName, stat := range table.Stats {
		if stat&axdb.ColumnStatSum != 0 {
			desc := SumColumnDescriptor{
				ColumnDescriptor: ColumnDescriptor{colName: statName, colType: table.Columns[statName].Type},
				sumColName:       statName + axdb.AXDBSumColumnSuffix,
				countColName:     statName + axdb.AXDBCountColumnSuffix,
			}
			s.columns = append(s.columns, desc)
		}
	}
	for colname, col := range table.Columns {
		// TODO for now we only allow the keys to be string. Will need to have performance test in place before we play with flexibility
		if (col.Index == axdb.ColumnIndexClustering || col.Index == axdb.ColumnIndexClusteringStrong) &&
			axdbColumnTypeNames[col.Type] == "text" && colname != axdb.AXDBUUIDColumnName && colname != axdb.AXDBTimeColumnName {
			s.keys = append(s.keys, colname)
		}
	}
	if len(s.keys) != 0 {
		s.hasKeys = true
	}

	s.reset(statTime)
}

func (s *SumStat) reset(newTime int64) {
	s.time = newTime
	s.statData = make(map[string](map[string]float64))
}

func (s *SumStat) concatKeys(data map[string]interface{}) string {
	var buf bytes.Buffer
	for _, key := range s.keys {
		if _, ok := data[key].(string); ok {
			buf.WriteString(data[key].(string) + ",")
		} else {
			valueMap := data[key].(map[string]interface{})
			valString := axdb.SerializeOrderedMap(valueMap)
			buf.WriteString(valString + ",")
		}
	}
	return buf.String()
}

func (s *SumStat) addData(data map[string]interface{}) {
	for _, desc := range s.columns {
		var value float64
		var count float64
		if !s.isRollUp {
			raw, exist := data[desc.colName]
			if !exist {
				continue
			}
			if desc.colType == axdb.ColumnTypeInteger {
				value = float64(raw.(int64))
			} else if desc.colType == axdb.ColumnTypeDouble {
				value = raw.(float64)
			}
			count = 1
		} else {
			infoLog.Printf("** in SumStat.addData rollup logic.")
			raw, exist := data[desc.sumColName]
			if !exist {
				value = 0
			} else {
				value = raw.(float64)
			}
			raw, exist = data[desc.countColName]
			if !exist {
				count = 0
			} else {
				count = raw.(float64)
			}
		}

		if _, exist := s.statData[""]; !exist {
			s.statData[""] = make(map[string]float64)
		}
		s.statData[""][desc.sumColName] += value
		s.statData[""][desc.countColName] += count
		if s.hasKeys {
			str := concatKeys(data, s.keys)
			if _, exist := s.statData[str]; !exist {
				s.statData[str] = make(map[string]float64)
			}

			s.statData[str][desc.sumColName] += value
			s.statData[str][desc.countColName] += count
		}
	}
}

func (s *SumStat) getData() []map[string]interface{} {
	dataArray := make([]map[string]interface{}, len(s.statData))

	current := 0
	for k, v := range s.statData {
		saveData := copyMap(s.params)
		for k0, v0 := range v {
			saveData[k0] = v0
		}

		keyData := strings.Split(k, ",")
		for i, key := range s.keys {
			if i < len(keyData) {
				saveData[key] = keyData[i]
			} else {
				saveData[key] = ""
			}
		}
		saveData[axdb.AXDBIntervalColumnName] = s.interval / 1e6
		saveData[axdb.AXDBWeekColumnName] = s.time / WeekInMicroSeconds
		saveData[axdb.AXDBTimeColumnName] = s.time
		dataArray[current] = saveData
		current++
	}
	return dataArray
}

type PercentColumnDescriptor struct {
	ColumnDescriptor
}

type PercentStat struct {
	columns  []PercentColumnDescriptor
	keys     []string
	hasKeys  bool
	time     int64
	interval int64
	params   map[string]interface{}
	statData [](map[string]interface{})
	isRollUp bool
}

func (s *PercentStat) init(table *TimeSeriesTable, statTime int64, interval int64, params map[string]interface{}, isRollUp bool) {
	s.time = statTime
	s.interval = interval
	s.params = params
	s.isRollUp = isRollUp

	for statName, stat := range table.Stats {
		if stat&axdb.ColumnStatPercent != 0 {
			desc := PercentColumnDescriptor{ColumnDescriptor: ColumnDescriptor{colName: statName, colType: table.Columns[statName].Type}}
			s.columns = append(s.columns, desc)
		}
	}
	for colname, col := range table.Columns {
		// TODO for now we only allow the keys to be string. Will need to have performance test in place before we play with flexibility
		if (col.Index == axdb.ColumnIndexClustering || col.Index == axdb.ColumnIndexClusteringStrong) && axdbColumnTypeNames[col.Type] == "text" &&
			colname != axdb.AXDBUUIDColumnName && colname != axdb.AXDBTimeColumnName {
			s.keys = append(s.keys, colname)
		}
	}
	if len(s.keys) != 0 {
		s.hasKeys = true
	}

	s.reset(statTime)
}

func (s *PercentStat) reset(newTime int64) {
	s.time = newTime
	s.statData = nil
}

func (s *PercentStat) addData(data map[string]interface{}) {
	s.statData = append(s.statData, data)
}

func (s *PercentStat) concatKeys(data map[string]interface{}) string {
	var buf bytes.Buffer
	for _, key := range s.keys {
		if _, ok := data[key].(string); ok {
			buf.WriteString(data[key].(string) + ",")
		} else {
			valueMap := data[key].(map[string]interface{})
			valString := axdb.SerializeOrderedMap(valueMap)
			buf.WriteString(valString + ",")
		}
	}
	return buf.String()
}

type ColumnSorter struct {
	data       []map[string]interface{}
	colName    string
	keys       []string
	hasGroupBy bool // flag indicating the histogram group by partition keys
	rollUp     bool // flag indicating the aggregation of smaller interval data

}

// Len is part of sort.Interface.
func (s *ColumnSorter) Len() int {
	return len(s.data)
}

// Swap is part of sort.Interface.
func (s *ColumnSorter) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}

func (s *ColumnSorter) Less(i, j int) bool {
	panic("ColumnSorter can't be used directly")
}

func (s *ColumnSorter) getValueByKey(key string, index int) (string, bool) {
	value, exist := s.data[index][key]
	if !exist {
		return "", false
	} else {
		if _, ok := value.(string); ok {
			return value.(string), true
		} else {
			valueMap := value.(map[string]interface{})
			return axdb.SerializeOrderedMap(valueMap), true
		}
	}
}

// compare value indexed at i with value indexed at j
// return value 0: equal; -1: less than; 1: great than
// this function only used for groupby case
func (s *ColumnSorter) comparePartitionKeyValues(i, j int) int {
	if s.hasGroupBy {
		for _, key := range s.keys {
			gi, exist := s.getValueByKey(key, i)
			if !exist {
				return -1
			}

			gj, exist := s.getValueByKey(key, j)
			if !exist {
				return 1
			}

			cmpRes := strings.Compare(gi, gj)
			if cmpRes == 0 {
				//continue to compare the value of next partition key column
				continue
			} else {
				return cmpRes
			}
		}
	}
	return 0
}

type IntColumnSorter struct {
	ColumnSorter
}

type FloatColumnSorter struct {
	ColumnSorter
}

//The criteria to compare two rows:
// first to compare the values of partition keys (key1, key2). If the comparison result is equal, go to step 2.
// step 2: compare the real value of the column on which the stats are rolled up
func (s *IntColumnSorter) Less(i, j int) bool {
	cmpRes := s.ColumnSorter.comparePartitionKeyValues(i, j)
	if cmpRes == -1 {
		return true
	} else if cmpRes == 1 {
		return false
	}

	di, exist := s.data[i][s.colName]
	if !exist {
		return true
	}

	dj, exist := s.data[j][s.colName]
	if !exist {
		return false
	}

	return di.(int64) < dj.(int64)
}

func (s *FloatColumnSorter) Less(i, j int) bool {
	cmpRes := s.ColumnSorter.comparePartitionKeyValues(i, j)
	if cmpRes == -1 {
		return true
	} else if cmpRes == 1 {
		return false
	}

	di, exist := s.data[i][s.colName]
	if !exist {
		return true
	}

	dj, exist := s.data[j][s.colName]
	if !exist {
		return false
	}

	return di.(float64) < dj.(float64)
}

func (s *PercentStat) getAggregatedHistogramData() []map[string]interface{} {
	if len(s.statData) == 0 {
		return nil
	}

	var result []map[string]interface{}
	data := copyMap(s.params)
	data[axdb.AXDBIntervalColumnName] = s.interval
	data[axdb.AXDBWeekColumnName] = s.time / WeekInMicroSeconds
	data[axdb.AXDBTimeColumnName] = s.time

	// we only need one sort for the entire data , the sorter interface is always with type of float64
	// the colName field isn't important
	sorterImpl := ColumnSorter{data: s.statData, colName: "", hasGroupBy: true, keys: s.keys, rollUp: true}
	sorter := &FloatColumnSorter{sorterImpl}
	sort.Sort(sorter)

	//for each column we do a sort and calculate the aggregated stats
	for _, desc := range s.columns {
		keyMap := make(map[string]bool)
		index := 0
		var statPoints []StatPoint
		var accumulatedSum int64 = 0
		var preKeyStr string = ""
		for {
			var str string = ""
			if index < len(sorterImpl.data) {
				str = concatKeys(sorterImpl.data[index], s.keys)
			}

			if _, exist := keyMap[str]; (!exist || index == len(sorterImpl.data)) && len(statPoints) != 0 {
				//end of the previous record
				statPointSorter := &StatPointSorter{statPoints}
				sort.Sort(statPointSorter)

				var accumulatedCnt int64 = 0
				for i, _ := range statPointSorter.points {

					statPointSorter.points[i].cnt += accumulatedCnt
					accumulatedCnt = statPointSorter.points[i].cnt
				}

				//from the values in statPointSorter.points; we estimate the percentiles.
				histoData := copyMap(data)
				for i := 10; i <= 90; i += 10 {
					v := accumulatedCnt * int64(i) / 100
					// binary search to find a range
					begin := 0
					end := len(statPointSorter.points) - 1
					//the value of current percentile
					var pVal float64 = 0
					for begin < end {
						mid := begin + (end-begin)/2
						if statPointSorter.points[mid].cnt == v {
							pVal = statPointSorter.points[mid].data
							break
						} else if statPointSorter.points[mid].cnt < v {
							if mid < end && statPointSorter.points[mid+1].cnt >= v {
								d1 := statPointSorter.points[mid].data
								d2 := statPointSorter.points[mid+1].data
								v1 := statPointSorter.points[mid].cnt
								v2 := statPointSorter.points[mid+1].cnt
								pVal = d1 + (d2-d1)*float64(v-v1)/float64(v2-v1)
								break
							} else {
								begin = mid + 1
							}
						} else {
							if mid > begin && statPointSorter.points[mid-1].cnt <= v {
								d1 := statPointSorter.points[mid-1].data
								d2 := statPointSorter.points[mid].data
								v1 := statPointSorter.points[mid-1].cnt
								v2 := statPointSorter.points[mid].cnt
								pVal = d1 + (d2-d1)*float64(v-v1)/float64(v2-v1)
								break
							} else {
								end = mid - 1
							}
						}
					}
					histoData[fmt.Sprintf("%s_%d", desc.colName, i)] = pVal
				}
				histoData[desc.colName+axdb.AXDBCountColumnSuffix] = accumulatedSum
				if s.hasKeys {
					keyData := strings.Split(preKeyStr, ",")
					for i, key := range s.keys {
						if i < len(keyData) {
							histoData[key] = keyData[i]
						} else {
							histoData[key] = ""
						}
					}
				}

				result = append(result, histoData)
				//reset the accumulatedSum to zero
				accumulatedSum = 0
				preKeyStr = str
				statPoints = []StatPoint{}
				if index == len(sorterImpl.data) {
					break
				}
			}

			points := make([]StatPoint, 9)
			k := 0
			count := int64(sorterImpl.data[index][desc.colName+axdb.AXDBCountColumnSuffix].(float64))
			accumulatedSum += count

			for i := 10; i <= 90; i += 10 {
				colName := fmt.Sprintf("%s_%d", desc.colName, i)
				points[k] = StatPoint{data: sorterImpl.data[index][colName].(float64), cnt: count / 10}
				k++
			}
			statPoints = append(statPoints, points...)
			keyMap[str] = true
			index++
		}

	}
	return result
}

func (s *PercentStat) getHistogramData(GroupBy bool) []map[string]interface{} {
	if len(s.statData) == 0 {
		return nil
	}

	var result []map[string]interface{}
	histoData := make(map[string](map[string]interface{}))
	data := copyMap(s.params)
	data[axdb.AXDBIntervalColumnName] = s.interval / 1e6
	data[axdb.AXDBWeekColumnName] = s.time / WeekInMicroSeconds
	data[axdb.AXDBTimeColumnName] = s.time

	// calculate stats for each column; one column per iteration
	for _, desc := range s.columns {
		var sorter sort.Interface
		sorterImpl := ColumnSorter{data: s.statData, colName: desc.colName, hasGroupBy: GroupBy, keys: s.keys, rollUp: false}
		if desc.colType == axdb.ColumnTypeInteger {
			sorter = &IntColumnSorter{sorterImpl}
		} else if desc.colType == axdb.ColumnTypeDouble {
			sorter = &FloatColumnSorter{sorterImpl}
		}
		sort.Sort(sorter)
		if !GroupBy || !s.hasKeys {
			for _, key := range s.keys {
				data[key] = ""
			}

			first := 0
			for i, v := range sorterImpl.data {
				if v[sorterImpl.colName] != nil {
					first = i
					break
				}
			}
			count := len(sorterImpl.data) - first
			data[desc.colName+axdb.AXDBCountColumnSuffix] = count
			for i := 10; i <= 90; i += 10 {
				data[fmt.Sprintf("%s_%d", desc.colName, i)] = sorterImpl.data[count*i/100+first][desc.colName]
			}
		} else {
			first := 0
			for {
				if first >= len(sorterImpl.data) {
					break
				}
				//find the first element of the current partition key
				for {
					if sorterImpl.data[first][sorterImpl.colName] == nil {
						first++
					} else {
						break
					}
				}

				//concat the values of partition keys to construct the key
				// this is used for the final result assembly
				str := concatKeys(sorterImpl.data[first], s.keys)
				if _, exist := histoData[str]; !exist {
					histoData[str] = make(map[string]interface{})
				}

				// find the last element corresponding to the current partition key
				// for performance consideration, we use binary search
				// head and end are pointers to the end points of the range
				// last is the index to the last element we are looking for
				head := first
				end := len(sorterImpl.data) - 1
				last := first
				for head <= end {
					mid := head + (end-head)/2
					cmpRes := sorterImpl.comparePartitionKeyValues(first, mid)
					//if middle element has the same partition key values as first element
					if cmpRes == 0 {
						//we are at the end of the search range
						if mid == end {
							last = mid
							break
						} else {
							cmpRes1 := sorterImpl.comparePartitionKeyValues(first, mid+1)
							// if both elements at mid and mid+1 have the same partition key values
							// the final result is located at the right of mid
							if cmpRes1 == 0 {
								head = mid + 1
							} else if cmpRes1 == -1 {
								// if element at mid+1 has bigger partition key than mid,
								// then, mid is the last element we are looking for
								last = mid
								break
							} else {
								break
							}

						}
					} else if cmpRes == -1 {
						end = mid - 1
					} else {
						// this branch isn't supposed to be touched.
						break
					}
				}

				// calculate percentage for the partition key with range [first, last]
				count := last - first + 1
				histoData[str][desc.colName+axdb.AXDBCountColumnSuffix] = count
				for i := 10; i <= 90; i += 10 {
					histoData[str][fmt.Sprintf("%s_%d", desc.colName, i)] = sorterImpl.data[count*i/100+first][desc.colName]
				}
				first = last + 1
			}
		}
	}

	if !GroupBy || !s.hasKeys {
		result = append(result, data)
	} else {
		for k, v := range histoData {
			saveData := copyMap(data)
			for k0, v0 := range v {
				saveData[k0] = v0
			}
			keyData := strings.Split(k, ",")
			for i, key := range s.keys {
				if i < len(keyData) {
					saveData[key] = keyData[i]
				} else {
					saveData[key] = ""
				}
			}
			result = append(result, saveData)
		}
	}

	return result
}

// For percent stat, we also do separate reporting for each set of keys
func (s *PercentStat) getData() []map[string]interface{} {
	if s.isRollUp {
		return s.getAggregatedHistogramData()
	}
	var data []map[string]interface{}
	data = append(data, s.getHistogramData(false)...)
	data = append(data, s.getHistogramData(true)...)
	return data
}

type AllStat struct {
	sumStat     SumStat
	percentStat PercentStat
}

func (s *AllStat) init(table *TimeSeriesTable, statTime int64, interval int64, params map[string]interface{}, rollUp bool) {
	s.sumStat.init(table, statTime, interval, params, rollUp)
	s.percentStat.init(table, statTime, interval, params, rollUp)
}

func (s *AllStat) addData(data map[string]interface{}) {
	s.sumStat.addData(data)
	s.percentStat.addData(data)
}

func (s *AllStat) getData() []map[string]interface{} {
	var resultData []map[string]interface{}
	// TODO: consolidate the matched rows from sumStat and percentStat to one row, it's for performance purpose.
	resultData = append(resultData, s.sumStat.getData()...)
	resultData = append(resultData, s.percentStat.getData()...)
	return resultData
}

func (s *AllStat) reset(newTime int64) {
	s.sumStat.reset(newTime)
	s.percentStat.reset(newTime)
}

type StatPoint struct {
	data float64 //the data point in the x-axis
	cnt  int64   //number of point less than data
}

type StatPointSorter struct {
	points []StatPoint
}

func (p *StatPointSorter) Less(i, j int) bool {
	return p.points[i].data < p.points[j].data
}

func (p *StatPointSorter) Len() int {
	return len(p.points)
}

// Swap is part of sort.Interface.
func (p *StatPointSorter) Swap(i, j int) {
	p.points[i], p.points[j] = p.points[j], p.points[i]
}
