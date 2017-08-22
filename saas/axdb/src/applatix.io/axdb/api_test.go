// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axdb_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axdb/core"
	"applatix.io/axerror"
	"gopkg.in/check.v1"
)

func deleteTable(c *check.C, tableName string) {
	status, _ := processDelete(c, appName, tableName, nil)
	if status != successCode && status != axerror.ERR_AXDB_TABLE_NOT_FOUND.Code {
		c.Logf("delete table got status %s", status)
		// fail(t) don't fail the test for now. There is a cassandra timeout that happens every now and then
		time.Sleep(5)
	}
}

func getTblDefinition(c *check.C, appname string, tblname string) axdb.Table {
	param := make(map[string]interface{})
	tblFullName := fmt.Sprintf("%s_%s", appname, tblname)
	param["ax_key"] = tblFullName
	tblDefArr := processQuery(c, "axint", "table_definition", param)
	c.Assert(len(tblDefArr), check.Equals, 1)

	tblDef := tblDefArr[0]["ax_value"].(string)
	c.Assert(tblDef, check.Not(check.Equals), "")

	buffer := bytes.NewBufferString(tblDef)
	var table axdb.Table
	decoder := json.NewDecoder(buffer)
	decoder.UseNumber()
	err := decoder.Decode(&table)
	c.Check(err, check.IsNil)
	return table
}

// returns whether it's successful
func processDelete(c *check.C, app string, table string, payload interface{}) (string, map[string]interface{}) {
	if verbose {
		payloadJson, _ := json.Marshal(payload)
		c.Logf("===> DELETE %s/%s %s", app, table, string(payloadJson))
	}

	resMap, err := axdbClient.Delete(app, table, payload)
	effectiveStatus := successCode
	if err != nil {
		effectiveStatus = err.Code
		c.Logf("%s: %s", err.Code, err.Message)
	}

	if verbose {
		resJson, _ := json.Marshal(resMap)
		c.Logf("<=== %s DELETE %s/%s %s", effectiveStatus, app, table, string(resJson))
	}

	return effectiveStatus, resMap
}

// returns whether it's successful
func processPut(c *check.C, app string, table string, payload interface{}) (bool, map[string]interface{}) {
	if verbose {
		payloadJson, _ := json.Marshal(payload)
		c.Logf("===> PUT %s/%s %s", app, table, string(payloadJson))
	}

	resMap, err := axdbClient.Put(app, table, payload)
	effectiveStatus := successCode
	if err != nil {
		effectiveStatus = err.Code
		c.Logf("%s: %s", err.Code, err.Message)
	}

	if verbose {
		resJson, _ := json.Marshal(resMap)
		c.Logf("<=== %s PUT %s/%s %s", effectiveStatus, app, table, string(resJson))
	}

	success := effectiveStatus == successCode
	c.Assert(success, check.Equals, true)

	return success, resMap
}

func processQuery(c *check.C, app string, table string, params map[string]interface{}) []map[string]interface{} {
	if verbose {
		paramsJson, _ := json.Marshal(params)
		c.Logf("===> GET %s/%s %s", app, table, string(paramsJson))
	}

	var resMapArray []map[string]interface{}
	err := axdbClient.Get(app, table, params, &resMapArray)
	effectiveStatus := successCode
	if err != nil {
		effectiveStatus = err.Code
		c.Logf("%s: %s", err.Code, err.Message)
	}

	if verbose {
		resJson, _ := json.Marshal(resMapArray)
		c.Logf("<=== %s GET %s/%s %s", effectiveStatus, app, table, string(resJson))
	}
	c.Assert(effectiveStatus, check.Equals, successCode)

	return resMapArray
}

// returns whether it's successful
func processPost(c *check.C, app string, table string, payload interface{}, expectedErr string) (bool, map[string]interface{}) {
	if verbose {
		payloadJson, _ := json.Marshal(payload)
		c.Logf("===> POST %s/%s %s", app, table, string(payloadJson))
	}

	resMap, err := axdbClient.Post(app, table, payload)
	effectiveStatus := successCode
	if err != nil {
		effectiveStatus = err.Code
		c.Logf("%s: %s", err.Code, err.Message)
	}

	if verbose {
		resJson, _ := json.Marshal(resMap)
		c.Logf("<=== %s POST %s/%s %s", effectiveStatus, app, table, string(resJson))
	}

	success := effectiveStatus == expectedErr
	c.Assert(success, check.Equals, true)
	return success, resMap
}

func processUpdate(c *check.C, app string, table string, payload interface{}) *axerror.AXError {
	if verbose {
		payloadJson, _ := json.Marshal(payload)
		c.Logf("===> PUT %s/%s %s", app, table, string(payloadJson))
	}

	_, err := axdbClient.Put(app, table, payload)
	return err
}

func deleteSingle(c *check.C, appName string, tableName string, params map[string]interface{}) {
	array := make([]map[string]interface{}, 1)
	array[0] = params
	status, _ := processDelete(c, appName, tableName, array)
	c.Assert(status, check.Equals, successCode)
}

func containsCol(c *check.C, tbl axdb.Table, colName string, col axdb.Column) bool {
	colExist, exist := tbl.Columns[colName]
	if !exist {
		return false
	}
	return colExist.Type == col.Type && colExist.Index == col.Index
}

func validateCols(c *check.C, results []map[string]interface{}, hasCols, hasNoCols *[]string) {
	if len(results) == 0 {
		return
	}

	result := results[0]

	if hasCols != nil {
		for _, hasCol := range *hasCols {
			if _, ok := result[hasCol]; !ok {
				c.Errorf("Result should have column %v", hasCol)
				fail(c)
				return
			}
		}
	}

	if hasNoCols != nil {
		for _, hasNoCol := range *hasNoCols {
			if _, ok := result[hasNoCol]; ok {
				c.Errorf("Result should not have column %v", hasNoCol)
				fail(c)
				return
			}
		}
	}
}

// verify data
func querySingleVerify(c *check.C, appName string, tableName string, params map[string]interface{}, expected map[string]interface{}) {
	resMapArray := processQuery(c, appName, tableName, params)
	if len(resMapArray) == 0 {
		c.Assert(expected, check.IsNil)
		return
	}

	c.Assert(len(resMapArray), check.Equals, 1)
	resMap := resMapArray[0]
	b := (expected == nil && len(resMapArray) != 0)
	c.Assert(b, check.Equals, false)
	c.Assert(verifyMatch(c, expected, resMap), check.Equals, true)
}

// Returns true if the two matches
func verifyMatch(c *check.C, original map[string]interface{}, fromJson map[string]interface{}) bool {
	// When parsing json text into the maps, json module may not use the same type we expect
	equals := true
	for k, v := range original {
		equals = true
		switch reflect.TypeOf(v).Kind() {
		case reflect.Map:
			equals = ((axdb.SerializeOrderedMap(v.(map[string]interface{}))) == (axdb.SerializeOrderedMap(fromJson[k].(map[string]interface{}))))
			break
		case reflect.Slice:
			break
		case reflect.Int:
			equals = (float64(v.(int)) == fromJson[k])
			break
		case reflect.Int32:
			equals = (float64(v.(int32)) == fromJson[k])
			break
		case reflect.Int64:
			equals = (float64(v.(int64)) == fromJson[k])
			break
		default:
			equals = (v == fromJson[k])
		}

		if !equals {
			c.Log("Expecting key", k, "to have type", reflect.TypeOf(v), "value", v, " but got type", reflect.TypeOf(fromJson[k]), "value", fromJson[k])
			return false
		}
	}
	return true
}

func deleteNonPartitionKey(c *check.C, appName string, tableName string, params map[string]interface{}) {
	array := make([]map[string]interface{}, 1)
	array[0] = params
	// to retrieve the number of rows corresponding to the non-partition keys
	resultArray := processQuery(c, appName, tableName, params)
	var expectedDeletedRows = len(resultArray)
	if expectedDeletedRows == 0 {
		return
	}
	// query the total number of rows in the table before deletion
	resultAll := processQuery(c, appName, tableName, nil)
	status, _ := processDelete(c, appName, tableName, array)
	c.Assert(status, check.Equals, successCode)

	//query the total number of rows in the table after deletion
	resultAfter := processQuery(c, appName, tableName, nil)
	c.Assert(expectedDeletedRows, check.Equals, len(resultAll)-len(resultAfter))
}

// this is to test the failure case, the query must return error; otherwise the test fails.
func processQueryWithError(c *check.C, app string, table string, params map[string]interface{}) *axerror.AXError {
	if verbose {
		paramsJson, _ := json.Marshal(params)
		c.Logf("===> GET %s/%s %s", app, table, string(paramsJson))
	}

	var resMapArray []map[string]interface{}
	err := axdbClient.Get(app, table, params, &resMapArray)

	effectiveStatus := successCode
	if err != nil {
		effectiveStatus = err.Code
		c.Logf("Failure tests, it's expected error %s: %s", err.Code, err.Message)
	}

	if verbose {
		resJson, _ := json.Marshal(resMapArray)
		c.Logf("<=== %s GET %s/%s %s", effectiveStatus, app, table, string(resJson))
	}
	return err
}

func compareTables(c *check.C, oldTbl axdb.Table, newTbl axdb.Table) bool {
	if oldTbl.Name != newTbl.Name || oldTbl.AppName != newTbl.AppName || oldTbl.Type != newTbl.Type {
		return false
	}
	oldCols := oldTbl.Columns
	newCols := newTbl.Columns

	if len(oldCols) != len(newCols) {
		return false
	}

	for colName, oldCol := range oldCols {
		newCol, exist := newCols[colName]
		if !exist || newCol.Index != oldCol.Index || newCol.Type != oldCol.Type {
			return false
		}
	}

	oldIndexOrder := oldTbl.IndexOrder
	newIndexOrder := newTbl.IndexOrder
	if len(oldIndexOrder) != len(newIndexOrder) {
		return false
	}

	for i, _ := range oldIndexOrder {
		if oldIndexOrder[i] != newIndexOrder[i] {
			return false
		}
	}

	oldStats := oldTbl.Stats
	newStats := newTbl.Stats
	if len(oldStats) != len(newStats) {
		return false
	}

	for colName, oldStatType := range oldStats {
		newStatType, exist := newStats[colName]
		if !exist || newStatType != oldStatType {
			return false
		}
	}

	return true
}

// failure tests; must return error
func doFailureTest(c *check.C, table axdb.Table) {
	// use unknown table name
	err := processQueryWithError(c, appName, "Unknown", nil)
	c.Assert(err, check.Not(check.Equals), nil)

	// select non-existing column
	err = processQueryWithError(c, appName, table.Name, map[string]interface{}{axdb.AXDBSelectColumns: "nonexistingcol"})
	c.Assert(err, check.Not(check.Equals), nil)

	// mismatched data type for where clause
	for name, col := range table.Columns {
		if col.Type == axdb.ColumnTypeInteger {
			err = processQueryWithError(c, appName, table.Name, map[string]interface{}{name: "hello"})
			c.Assert(err, check.Not(check.Equals), nil)
		}
	}
}

// the test function for the AXDB upgrade functionalities
func doUpdateTableTest(c *check.C, table axdb.Table) {
	// Test: The table doesn't exist; it should be created automatically
	tblName := table.Name
	var newTblName bytes.Buffer
	newTblName.WriteString(fmt.Sprintf("Tbl_%s", tblName))
	table.Name = newTblName.String()
	err := processUpdate(c, "axdb", "update_table", table)
	c.Assert(err, check.IsNil)

	table.Name = tblName
	// Test : The appName doesn't match: the table is still created successfully
	appName := table.AppName
	var newAppName bytes.Buffer
	newAppName.WriteString(fmt.Sprintf("App_%s", appName))
	table.AppName = newAppName.String()
	err = processUpdate(c, "axdb", "update_table", table)
	c.Assert(err, check.IsNil)

	table.AppName = appName
	// Test: the table type doesn't match
	// change the table type to the next of the table type constant list
	table.Type = (table.Type + 1) % axdb.NumberOfTableTypes
	err = processUpdate(c, "axdb", "update_table", table)
	c.Assert(err, check.ErrorMatches, ".*Table Type doesn't match.*")

	// restore to the original table type
	table.Type = (axdb.NumberOfTableTypes + table.Type - 1) % axdb.NumberOfTableTypes

	// Test: the data type of the column doesn't match
	// update operation should be denied.
	for colName, col := range table.Columns {
		oldDT := col.Type
		newDT := (oldDT + 1) % axdb.NumberOfDataTypes
		newCol := axdb.Column{Type: newDT, Index: col.Index}
		table.Columns[colName] = newCol
		err = processUpdate(c, "axdb", "update_table", table)

		if err == nil || !(strings.Contains(err.Message, "DataType of column") && strings.Contains(err.Message, "is different, data type change isn't supported")) {
			fail(c)
		}
		table.Columns[colName] = col
	}

	// Test: negative test cases, to cover the invalid index type changes
	// a map to record the index types in the table that have been tested, this is to ensure each type will be tested for only once
	testedTypes := make(map[int]bool)
	var allIndexTypes map[int]bool
	for colName, col := range table.Columns {
		oldIndex := col.Index
		if _, tested := testedTypes[oldIndex]; tested {
			continue
		}
		allIndexTypes = map[int]bool{axdb.ColumnIndexNone: true, axdb.ColumnIndexWeak: true, axdb.ColumnIndexStrong: true,
			axdb.ColumnIndexClustering: true, axdb.ColumnIndexPartition: true, axdb.ColumnIndexClusteringStrong: true}
		// delete from the allIndexTypes array the type that is the same as oldIndex
		delete(allIndexTypes, oldIndex)
		switch col.Index {
		// clustering key => other types: only ColumnIndexClusteringStrong is allowed, so delete it.
		case axdb.ColumnIndexClustering:
			delete(allIndexTypes, axdb.ColumnIndexClusteringStrong)
		// None => other types: only weak, secondary are allowed, so delete them
		case axdb.ColumnIndexNone:
			delete(allIndexTypes, axdb.ColumnIndexWeak)
			delete(allIndexTypes, axdb.ColumnIndexStrong)
		// Weak => other types: only none, secondary are allowed, so delete them
		case axdb.ColumnIndexWeak:
			delete(allIndexTypes, axdb.ColumnIndexNone)
			delete(allIndexTypes, axdb.ColumnIndexStrong)
		// secondary => other types: only none, weak are allowed, so delete them
		case axdb.ColumnIndexStrong:
			delete(allIndexTypes, axdb.ColumnIndexWeak)
			delete(allIndexTypes, axdb.ColumnIndexNone)
		//clusteringstrong => other types: only clustering is allowed, so delete it
		case axdb.ColumnIndexClusteringStrong:
			delete(allIndexTypes, axdb.ColumnIndexClustering)
		}

		for newType, _ := range allIndexTypes {
			newCol := axdb.Column{Type: col.Type, Index: newType}
			table.Columns[colName] = newCol
			//table.Columns[colName].Index = newType
			err = processUpdate(c, "axdb", "update_table", table)
			c.Assert(err, check.ErrorMatches, ".*Index type isn't compatible to upgrade:.*")
			table.Columns[colName] = col
		}
		testedTypes[oldIndex] = true
	}

	var iter int = 0
	for {
		if iter == 7 {
			break
		}

		var colAdded axdb.Column
		if iter == 0 {
			// Test: add a new column without an index
			// first check the definition_table for old table definition
			if table.Type == axdb.TableTypeKeyValue {
				colAdded = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
			} else if table.Type == axdb.TableTypeCounter {
				colAdded = axdb.Column{Type: axdb.ColumnTypeCounter, Index: axdb.ColumnIndexNone}
			} else {
				colAdded = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone}
			}
		} else if iter == 1 {
			if table.Type == axdb.TableTypeKeyValue || table.Type == axdb.TableTypeTimedKeyValue ||
				table.Type == axdb.TableTypeTimeSeries {
				//Test: Add a collection type of column
				colAdded = axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexNone}
			} else {
				// if it's counter table, collection type column isn't allowed to added
				break
			}
		} else if iter == 2 {
			// Test: add a new column with a secondary index
			// first check the definition_table for old table definition
			if table.Type == axdb.TableTypeKeyValue {
				colAdded = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexStrong}
			} else if table.Type == axdb.TableTypeCounter {
				colAdded = axdb.Column{Type: axdb.ColumnTypeCounter, Index: axdb.ColumnIndexStrong}
			} else {
				colAdded = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong}
			}
		} else if iter == 3 {
			if table.Type == axdb.TableTypeKeyValue || table.Type == axdb.TableTypeTimedKeyValue ||
				table.Type == axdb.TableTypeTimeSeries {
				//Test: Add a collection type of column, and build secondary index on it
				colAdded = axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong}
			} else {
				// if it's counter table, collection type column isn't allowed to added
				break
			}
		} else if iter == 4 {
			// add a map type of column for different type of index creation
			if table.Type == axdb.TableTypeKeyValue || table.Type == axdb.TableTypeTimedKeyValue ||
				table.Type == axdb.TableTypeTimeSeries {
				//Test: Add a collection type of column, and build secondary index on it; by default, it's a index built on value of the map
				colAdded = axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexStrong}
			} else {
				// if it's counter table, collection type column isn't allowed to added
				break
			}
		} else if iter == 5 {
			// add a map type of column for different type of index creation
			if table.Type == axdb.TableTypeKeyValue || table.Type == axdb.TableTypeTimedKeyValue ||
				table.Type == axdb.TableTypeTimeSeries {
				//Test: Add a collection type of column, and build secondary index on it; by default, it's a index built on value of the map
				colAdded = axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexStrong, IndexFlagForMapColumn: axdb.ColumnIndexMapKeys}
			} else {
				// if it's counter table, collection type column isn't allowed to added
				break
			}
		} else {
			// add a map type of column for different type of index creation
			if table.Type == axdb.TableTypeKeyValue || table.Type == axdb.TableTypeTimedKeyValue ||
				table.Type == axdb.TableTypeTimeSeries {
				//Test: Add a collection type of column, and build secondary index on it; by default, it's a index built on value of the map
				colAdded = axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexStrong, IndexFlagForMapColumn: axdb.ColumnIndexMapKeysAndValues}
			} else {
				// if it's counter table, collection type column isn't allowed to added
				break
			}
		}

		var colName string
		oldTable := getTblDefinition(c, appName, table.Name)
		if table.Type != axdb.TableTypeCounter {
			colName = fmt.Sprintf("col_%d", iter)
		} else {
			// counter table doesn't allow to insert a column that was previously dropped
			// so, we rename it here.
			colName = fmt.Sprintf("col_%d", time.Now().Unix())

		}
		table.Columns[colName] = colAdded
		err = processUpdate(c, "axdb", "update_table", table)
		c.Assert(err, check.IsNil)

		newTable := getTblDefinition(c, appName, table.Name)

		c.Assert(containsCol(c, newTable, colName, colAdded), check.Equals, true)
		delete(newTable.Columns, colName)

		//compare the two definition
		c.Assert(compareTables(c, oldTable, newTable), check.Equals, true)

		// Test: delete an existing column without an index
		// first check the old table definition
		oldTable = getTblDefinition(c, appName, table.Name)
		delete(table.Columns, colName)
		err = processUpdate(c, "axdb", "update_table", table)
		c.Assert(err, check.IsNil)

		newTable = getTblDefinition(c, appName, table.Name)
		newTable.Columns[colName] = colAdded
		c.Assert(compareTables(c, oldTable, newTable), check.Equals, true)
		iter++
	}

	//if it's a timeseries table, we will also test the update to statTable
	if table.Type == axdb.TableTypeTimeSeries {
		// first add a new column on which the stat will be collected
		newCol := axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}
		table.Columns["stat_col"] = newCol
		err = processUpdate(c, "axdb", "update_table", table)
		c.Assert(err, check.IsNil)

		statTypes := []int{axdb.ColumnStatSum, axdb.ColumnStatPercent}

		for _, statType := range statTypes {
			oldTable := getTblDefinition(c, appName, table.Name)
			// add current type of stats on "stat_col" column
			if table.Stats == nil {
				table.Stats = make(map[string]int)
			}
			table.Stats["stat_col"] = statType
			err = processUpdate(c, "axdb", "update_table", table)
			c.Assert(err, check.IsNil)

			newTable := getTblDefinition(c, appName, table.Name)
			if oldTable.Stats == nil {
				oldTable.Stats = make(map[string]int)
			}
			oldTable.Stats["stat_col"] = statType
			c.Assert(compareTables(c, oldTable, newTable), check.Equals, true)

			var params map[string]interface{}
			if statType == axdb.ColumnStatSum {
				params = map[string]interface{}{
					axdb.AXDBSelectColumns: []string{"stat_col" + axdb.AXDBSumColumnSuffix, "stat_col" + axdb.AXDBCountColumnSuffix, "ax_time"},
				}
			} else if statType == axdb.ColumnStatPercent {
				params = map[string]interface{}{
					axdb.AXDBSelectColumns: []string{"stat_col" + axdb.AXDBCountColumnSuffix, "stat_col" + axdb.AXDB10ColumnSuffix, "stat_col" + axdb.AXDB20ColumnSuffix,
						"stat_col" + axdb.AXDB30ColumnSuffix, "stat_col" + axdb.AXDB40ColumnSuffix, "stat_col" + axdb.AXDB50ColumnSuffix, "stat_col" + axdb.AXDB60ColumnSuffix,
						"stat_col" + axdb.AXDB70ColumnSuffix, "stat_col" + axdb.AXDB80ColumnSuffix, "stat_col" + axdb.AXDB90ColumnSuffix, "ax_time"},
				}
			}
			params["ax_interval"] = 100

			// there shouldn't be error; otherwise, processQuery will fail
			processQuery(c, appName, table.Name, params)

			// drop the columns added just now
			delete(table.Stats, "stat_col")
			err = processUpdate(c, "axdb", "update_table", table)
			c.Assert(err, check.IsNil)
			oldTable = newTable
			newTable = getTblDefinition(c, appName, table.Name)
			newTable.Stats["stat_col"] = statType
			c.Assert(compareTables(c, oldTable, newTable), check.Equals, true)
		}
	}
}

func (s *S) TestVersion(c *check.C) {
	var bodyMapArray []map[string]interface{}
	err := axdbClient.Get("axdb", "version", nil, &bodyMapArray)
	checkError(c, err)

	bodyMap := bodyMapArray[0]
	if bodyMap == nil || bodyMap["version"] == nil {
		bodyJson, _ := json.Marshal(bodyMap)
		c.Logf("version call failed, return body: %s", string(bodyJson))
		fail(c)
	}

	// only v1 is allowed for axdb version, the invalid version will cause an error
	invalidAxdbClient := axdbcl.NewAXDBClientWithTimeout(invalidAxdburl, time.Second*60)
	err = invalidAxdbClient.Get("axdb", "version", nil, &bodyMapArray)
	c.Assert(err, check.Not(check.Equals), nil)
}

func (s *S) TestCounter(c *check.C) {
	tableName := "CounterTable"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["part1"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["part2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexPartition}
	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeCounter, Columns: columns}
	success, _ := processPut(c, "axdb", "update_table", table)
	c.Assert(success, check.Equals, true)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	dataArray := []map[string]interface{}{
		map[string]interface{}{"part1": "p1.1", "part2": 2.1},
		map[string]interface{}{"part1": "p1.2", "part2": 2.1},
		map[string]interface{}{"part1": "p1.2", "part2": 2.2},
	}

	for i := 0; i < 10; i++ {
		for j := range dataArray {
			success, _ = processPost(c, appName, tableName, dataArray[j], successCode)
			c.Assert(success, check.Equals, true)
		}
	}

	processPut(c, appName, tableName, dataArray[1])
	processPut(c, appName, tableName, dataArray[1])
	processPut(c, appName, tableName, dataArray[2])

	dataArray[0][axdb.AXDBCounterColumnName] = 10
	dataArray[1][axdb.AXDBCounterColumnName] = 12
	dataArray[2][axdb.AXDBCounterColumnName] = 11
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part1": "p1.1", "part2": 2.1}, dataArray[0])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part1": "p1.2", "part2": 2.1}, dataArray[1])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part1": "p1.2", "part2": 2.2}, dataArray[2])

	doUpdateTableTest(c, table)
	//deleteTable

}

func (s *S) TestKeyValue(c *check.C) {
	tableName := "KeyValueTable"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["part"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["prim"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering}
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexClustering}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone}
	columns["val5"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong}

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeKeyValue, Columns: columns, IndexOrder: []string{"prim", "val1"}}

	//for i := 0; i < 4; i++ {
	//	// fire up multiple table creation in parallel to test this case.
	//	go processPost(t, "axdb", "create_table", table, successCode)
	//}

	success, _ := processPut(c, "axdb", "update_table", table)
	c.Assert(success, check.Equals, true)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	// data to insert
	dataArray := []map[string]interface{}{
		map[string]interface{}{"part": "p1", "prim": "m1", "val1": 1001, "val5": "'quoted string'"},
		map[string]interface{}{"part": "p2", "prim": "m1", "val1": 1, "val2": 2.1,
			"val3": map[string]interface{}{"v3k1": "v3v1", "v3k2": "v3v2"}, "val4": []string{"v4.1, v4.2, v4.3, v4.4"}, "val5": "'5.0'"},
		map[string]interface{}{"part": "p1", "prim": "m2", "val1": 2, "val5": "abc"},
		map[string]interface{}{"part": "p1", "prim": "m2", "val1": 3, "val5": "abc"},
		map[string]interface{}{"part": "p2", "prim": "m2", "val1": 3, "val2": 2.2, "val5": "5.1"},
	}

	for i := range dataArray {
		success, _ = processPut(c, appName, tableName, dataArray[i])
		c.Assert(success, check.Equals, true)
	}

	// verify the data, note that we delibrately overwritten some data, so can't verify everyline
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", "prim": "m1"}, dataArray[0])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"val5": "5.1"}, dataArray[4])

	//verify "contains key" for map column
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p2", "val3_contains_key": "v3k1"}, dataArray[1])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p2", "prim": "m1", "val3": "v3v1"}, dataArray[1])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p2", "val3_contains_key": "v3k2"}, dataArray[1])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p2", "prim": "m1", "val3": "v3v2"}, dataArray[1])

	resultArray = processQuery(c, appName, tableName, map[string]interface{}{"part": "p1", "prim": "m2"})
	c.Assert(len(resultArray), check.Equals, 2)

	if !verifyMatch(c, dataArray[2], resultArray[0]) || !verifyMatch(c, dataArray[3], resultArray[1]) {
		fail(c)
		return
	}

	success, _ = processPost(c, appName, tableName, map[string]interface{}{"part": "p1", "prim": "m2", "val1": 2},
		axerror.ERR_AXDB_INSERT_DUPLICATE.Code)
	c.Assert(success, check.Equals, true)

	resultArray = processQuery(c, appName, tableName, map[string]interface{}{"part": "p1", "prim": "m2"})
	c.Assert(len(resultArray), check.Equals, 2)
	c.Assert(verifyMatch(c, dataArray[2], resultArray[0]), check.Equals, true)

	for i := range dataArray {
		dataArray[i]["val2"] = rand.Float64()
		dataArray[i]["val4"] = []string{"v2.1, v2.2"}
		success, _ = processPut(c, appName, tableName, dataArray[i])
		c.Assert(success, check.Equals, true)
	}

	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", "prim": "m1"}, dataArray[0])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"val5": 5.1}, dataArray[4])
	resultArray = processQuery(c, appName, tableName, map[string]interface{}{"part": "p1", "prim": "m2"})
	c.Assert(len(resultArray), check.Equals, 2)
	c.Assert(verifyMatch(c, dataArray[2], resultArray[0]), check.Equals, true)
	c.Assert(verifyMatch(c, dataArray[3], resultArray[1]), check.Equals, true)

	deleteSingle(c, appName, tableName, dataArray[0])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", "prim": "m1"}, nil)

	//delete using non-partition key, only one row to be deleted
	deleteNonPartitionKey(c, appName, tableName, map[string]interface{}{"val5": "5.1"})
	querySingleVerify(c, appName, tableName, dataArray[4], nil)

	//delete using non-parition key, two rows to be deleted
	deleteNonPartitionKey(c, appName, tableName, map[string]interface{}{"prim": "m2"})
	querySingleVerify(c, appName, tableName, dataArray[1], nil)
	querySingleVerify(c, appName, tableName, dataArray[2], nil)

	// run update table tests
	doUpdateTableTest(c, table)
	// deleteTable(t, tableName)
}

func (s *S) TestTimeSeriesWithPartition(c *check.C) {
	tableName := "TimeSeriesTableWithPartition"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["part"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["part1"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeOrderedMap, Index: axdb.ColumnIndexNone}
	//the following two columns are added to cover the case for materialized view creation.
	//originally we didn't consider adding clustering keys of the base table to the primary key of the materialized view;
	//which caused materialized view creation failure.
	columns["val5"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering}
	columns["val6"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong}

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeTimeSeries, Columns: columns, Stats: map[string]int{"val2": axdb.ColumnStatPercent, "val1": axdb.ColumnStatSum}}
	success, _ := processPut(c, "axdb", "update_table", table)
	c.Assert(success, check.Equals, true)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	// data to insert
	dataArray := []map[string]interface{}{
		map[string]interface{}{"part": "p1", "part1": "m1", "val5": "v5", "val6": "v6"},
		map[string]interface{}{"part": "p2", "part1": "m1", "val1": 1, "val2": 2.1, "val3": "v3.0", "val4": map[string]interface{}{"App": "A", "Proj": "a"}, "val5": "v5", "val6": "v6"},
		map[string]interface{}{"part": "p1", "part1": "m2", "val1": 2, "val5": "v5", "val6": "v6"},
		map[string]interface{}{"part": "p1", "part1": "m2", "val1": 3, "val5": "v5", "val6": "v6"},
		map[string]interface{}{"part": "p2", "part1": "m2", "val1": 3, "val2": 2.2, "val3": "v3.1", "val4": map[string]interface{}{"App": "B", "Proj": "b"}, "val5": "v5", "val6": "v6"},
	}

	idArray := make([]string, len(dataArray))
	timeArray := make([]int64, len(dataArray))

	for i := range dataArray {
		_, resMap := processPost(c, appName, tableName, dataArray[i], successCode)
		idArray[i] = resMap[axdb.AXDBUUIDColumnName].(string)
		timeArray[i] = int64(resMap[axdb.AXDBTimeColumnName].(float64))
	}

	for i := range dataArray {
		querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[i]["part"], "part1": dataArray[i]["part1"], axdb.AXDBUUIDColumnName: idArray[i]},
			dataArray[i])
	}

	querySingleVerify(c, appName, tableName, map[string]interface{}{"val1": 1}, dataArray[1])

	// query a time range
	resultArray = processQuery(c, appName, tableName, map[string]interface{}{axdb.AXDBQueryMaxTime: timeArray[3], "part": dataArray[3]["part"], "part1": dataArray[3]["part1"]})
	c.Assert(len(resultArray), check.Equals, 1)
	if !verifyMatch(c, dataArray[2], resultArray[0]) {
		fail(c)
	}

	// delete an entry
	deleteSingle(c, appName, tableName, map[string]interface{}{"part": dataArray[0]["part"], "part1": dataArray[0]["part1"], axdb.AXDBUUIDColumnName: idArray[0], "val5": dataArray[0]["val5"], "val6": dataArray[0]["val6"]})
	// the deleted one is gone
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[0]["part"], "part1": dataArray[0]["part1"], "val5": dataArray[0]["val5"], "val6": dataArray[0]["val6"], axdb.AXDBUUIDColumnName: idArray[0]}, nil)
	// but the next one is still there
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[1]["part"], "part1": dataArray[1]["part1"], "val5": dataArray[1]["val5"], "val6": dataArray[1]["val6"], axdb.AXDBUUIDColumnName: idArray[1]},
		dataArray[1])

	// delete using non-partition key; only one row to delete
	deleteNonPartitionKey(c, appName, tableName, map[string]interface{}{axdb.AXDBUUIDColumnName: idArray[1]})
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[1]["part"], "part1": dataArray[1]["part1"], "val5": dataArray[1]["val5"], "val6": dataArray[1]["val6"], axdb.AXDBUUIDColumnName: idArray[1]}, nil)
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[2]["part"], "part1": dataArray[2]["part1"], "val5": dataArray[2]["val5"], "val6": dataArray[2]["val6"], axdb.AXDBUUIDColumnName: idArray[2]},
		dataArray[2])

	// test percent stat
	currentTime := time.Now().UnixNano() / 1e3
	data := make([]map[string]interface{}, 500)
	for i := 0; i < 500; i++ {
		data[i] = make(map[string]interface{})
		if i%2 == 1 {
			data[i]["part"] = "p3"
			data[i]["part1"] = "m3"
		} else {
			data[i]["part"] = "p4"
			data[i]["part1"] = "m4"
		}
		data[i]["val1"] = 1
		data[i]["val2"] = rand.Float64()
		data[i]["val3"] = strconv.Itoa(rand.Int())
		data[i]["ax_time"] = currentTime - int64((i+1)*1e6) // one second each
		data[i]["val5"] = "v5"
		data[i]["val6"] = "v6"

		processPost(c, appName, tableName, data[i], successCode)
	}

	// get stats at 100 second interval
	statParams := map[string]interface{}{"ax_interval": 100}
	statArray := processQuery(c, appName, tableName, statParams)
	c.Assert(len(statArray) < 15, check.Equals, false)

	for _, stat := range statArray {
		if stat["val2_count"].(float64) < 70 {
			// only checks the data with enough data point to be statistically valid
			continue
		}

		for i := 10; i <= 90; i += 10 {
			v := stat[fmt.Sprintf("val2_%d", i)].(float64)
			if math.Abs(v-float64(i)/100.0) > 0.2 {
				// not expecting a diff this big
				c.Logf("index %d value %v unexpected", i, v)
				fail(c)
			}
		}
	}

	// test percent stat and sum stat on the same column with type float64
	deleteTable(c, tableName)
	stats := map[string]int{"val2": axdb.ColumnStatAll}
	table.Stats = stats
	processUpdate(c, "axdb", "update_table", table)

	//re-insert the raw data
	for _, row := range data {
		processPost(c, appName, tableName, row, successCode)
	}
	statParams = map[string]interface{}{"ax_interval": 100}
	statArray = processQuery(c, appName, tableName, statParams)
	if len(statArray) < 15 {
		c.Logf("expecting 15 stat results, got %d", len(statArray))
		fail(c)
	} else {
		// The following two variables track the number of records that the percentage stats are calculated on the partition keys
		cntP3 := 0
		cntP4 := 0
		for _, stat := range statArray {
			if p := stat["part"]; p != nil {
				if p.(string) == "p3" {
					if p1 := stat["part1"]; p1 == nil || p1.(string) != "m3" {
						c.Logf("part1 column should has value m3")
						fail(c)
					}
					cntP3++
				} else if p.(string) == "p4" {
					if p1 := stat["part1"]; p1 == nil || p1.(string) != "m4" {
						c.Logf("part1 column should has value m4")
						fail(c)
					}
					cntP4++
				} else if p.(string) == "" {
					if p1 := stat["part1"]; p1 == nil || p1.(string) != "" {
						c.Logf("part column and part1 column must both be empty")
						fail(c)
					}
				}
			} else {
				c.Logf("column 'part' is expected to be in the stat Table")
				fail(c)
			}

			if cnt := stat["val2_count"]; cnt == nil || cnt.(float64) == 0 {
				c.Logf("column 'val2_cnt' is expected to be in the stat Table, and its value shouldn't be 0")
				fail(c)
			}

			if sum := stat["val2_sum"]; sum == nil || sum.(float64) == 0 {
				c.Logf("column 'val2_sum' is expected to be in the stat Table, and its value shouldn't be 0")
				fail(c)
			}

			if stat["val2_count"].(float64) < 70 {
				// only checks the data with enough data point to be statistically valid
				continue
			}

			for i := 10; i <= 90; i += 10 {
				v := stat[fmt.Sprintf("val2_%d", i)].(float64)
				if math.Abs(v-float64(i)/100.0) > 0.2 {
					// not expecting a diff this big
					c.Logf("index %d value %v unexpected", i, v)
					fail(c)
				}
			}
		}
		if cntP3 < 5 || cntP4 < 5 {
			c.Logf("expecting 5 stat results for cntP3 and cntP4, got cntP3 = %d, cntP4 = %d", cntP3, cntP4)
			fail(c)
		}
	}

	// test percent stat and sum stat on the same column with type int
	deleteTable(c, tableName)
	stats = map[string]int{"val1": axdb.ColumnStatAll}
	table.Stats = stats
	processUpdate(c, "axdb", "update_table", table)

	//re-insert the raw data
	for _, row := range data {
		processPost(c, appName, tableName, row, successCode)
	}
	statParams = map[string]interface{}{"ax_interval": 100}
	statArray = processQuery(c, appName, tableName, statParams)
	if len(statArray) < 15 {
		c.Logf("expecting 15 stat results, got %d", len(statArray))
		fail(c)
	} else {
		// The following two variables track the number of records that the percentage stats are calculated on the partition keys
		cntP3 := 0
		cntP4 := 0
		for _, stat := range statArray {
			if p := stat["part"]; p != nil {
				if p.(string) == "p3" {
					if p1 := stat["part1"]; p1 == nil || p1.(string) != "m3" {
						c.Logf("part1 column should has value m3")
						fail(c)
					}
					cntP3++
				} else if p.(string) == "p4" {
					if p1 := stat["part1"]; p1 == nil || p1.(string) != "m4" {
						c.Logf("part1 column should has value m4")
						fail(c)
					}
					cntP4++
				} else if p.(string) == "" {
					if p1 := stat["part1"]; p1 == nil || p1.(string) != "" {
						c.Logf("part column and part1 column must both be empty")
						fail(c)
					}
				}
			} else {
				c.Logf("column 'part' is expected to be in the stat Table")
				fail(c)
			}

			if cnt := stat["val1_count"]; cnt == nil || cnt.(float64) == 0 {
				c.Logf("column 'val1_cnt' is expected to be in the stat Table, and its value shouldn't be 0")
				fail(c)
			}

			if sum := stat["val1_sum"]; sum == nil || sum.(float64) == 0 {
				c.Logf("column 'val1_sum' is expected to be in the stat Table, and its value shouldn't be 0")
				fail(c)
			}

			if stat["val1_count"].(float64) < 70 {
				// only checks the data with enough data point to be statistically valid
				continue
			}
		}
		if cntP3 < 5 || cntP4 < 5 {
			c.Logf("expecting 5 stat results for cntP3 and cntP4, got cntP3 = %d, cntP4 = %d", cntP3, cntP4)
			fail(c)
		}
	}

	doUpdateTableTest(c, table)
	// deleteTable(t, tableName)
}

func (s *S) TestTimeSeriesWithoutPartition(c *check.C) {
	tableName := "TimeSeriesWithoutPartition"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong}

	stats := make(map[string]int)
	stats["val1"] = axdb.ColumnStatSum

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeTimeSeries, Columns: columns, Stats: stats}
	processPut(c, "axdb", "update_table", table)
	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	expectMap := make(map[string]map[string]interface{})

	currentTime := time.Now().UnixNano() / 1e3
	data := make(map[string]interface{})
	for i := 0; i < 500; i++ {
		val1 := rand.Int() % 300
		data["val1"] = 1
		data["val2"] = rand.Float64()
		data["val3"] = strconv.Itoa(rand.Int())
		data["ax_time"] = currentTime - int64((i+1)*1e6) // one second each
		_, resMap := processPost(c, appName, tableName, data, successCode)

		if val1%100 == 0 {
			id := resMap[axdb.AXDBUUIDColumnName].(string)
			expectMap[id] = make(map[string]interface{})
			for k, v := range data {
				expectMap[id][k] = v
			}
			expectMap[id][axdb.AXDBUUIDColumnName] = id
			expectMap[id][axdb.AXDBTimeColumnName] = int64(resMap[axdb.AXDBTimeColumnName].(float64))
		}
	}

	for k, v := range expectMap {
		querySingleVerify(c, appName, tableName, map[string]interface{}{axdb.AXDBUUIDColumnName: k}, v)
	}

	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 500)

	time0 := int64(resultArray[0][axdb.AXDBTimeColumnName].(float64))
	time1 := int64(resultArray[1][axdb.AXDBTimeColumnName].(float64))
	c.Assert(time1 >= time0, check.Equals, false)
	resultArray = processQuery(c, appName, tableName, map[string]interface{}{axdb.AXDBQueryMaxTime: time1})
	c.Assert(len(resultArray), check.Equals, 498)
	c.Assert(int64(resultArray[0][axdb.AXDBTimeColumnName].(float64)) >= time1, check.Equals, false)

	// get stats at 10 second interval
	statParams := map[string]interface{}{"ax_interval": 10, "ax_max_time": time0, "ax_min_time": int64(resultArray[99][axdb.AXDBTimeColumnName].(float64))}
	// we only return data at even 10 second intervals, so this may ignore the last 10 second interval
	statArray := processQuery(c, appName, tableName, statParams)
	if len(statArray) < 9 {
		c.Logf("expecting 9 or 10 stat results, got %d", len(statArray))
		fail(c)
	}
	for _, stat := range statArray {
		c.Logf("count %v sum %v", stat["val1_count"], stat["val1_sum"])
	}
	// the first entry may land on a partial timeslot
	for i := 1; i < len(statArray); i++ {
		stat := statArray[i]
		c.Assert(stat["val1_count"].(float64), check.Equals, float64(10))
		c.Assert(stat["val1_sum"].(float64), check.Equals, float64(10))
	}

	// delete some
	for k, _ := range expectMap {
		deleteSingle(c, appName, tableName, map[string]interface{}{axdb.AXDBUUIDColumnName: k})
	}
	// verify they are gone
	for k, _ := range expectMap {
		querySingleVerify(c, appName, tableName, map[string]interface{}{axdb.AXDBUUIDColumnName: k}, nil)
	}

	//deleteTable(t, tableName)
}

func (s *S) TestTimedKeyValue(c *check.C) {
	tableName := "TimedKeyValueTable"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["part"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["prim"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering}
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone}
	columns["val5"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong}

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeTimedKeyValue, Columns: columns}
	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	// data to insert
	currentTime := time.Now().UnixNano() / 1e3
	dataArray := []map[string]interface{}{
		map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 10*1e3, "prim": "m1", "val1": 1001},
		map[string]interface{}{"part": "p2", axdb.AXDBTimeColumnName: currentTime - 9*1e3, "prim": "m1", "val1": 1, "val2": 2.1,
			"val3": map[string]interface{}{"v3k1": "v3v1", "v3k2": "v3v2"}, "val4": []string{"v4.1, v4.2, v4.3, v4.4"}, "val5": "5.0"},
		map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 2*core.WeekInMicroSeconds, "prim": "m2", "val1": 2},
		map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 2*core.WeekInMicroSeconds, "prim": "m2", "val1": 3},
		map[string]interface{}{"part": "p2", axdb.AXDBTimeColumnName: currentTime - 100*1e3, "prim": "m2", "val1": 3, "val2": 2.2, "val5": "5.1"},
	}

	for i := range dataArray {
		processPut(c, appName, tableName, dataArray[i])
	}

	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 4)

	// verify result sorted
	var previous float64 = 0
	for _, res := range resultArray {
		if previous != 0 && previous < res[axdb.AXDBTimeColumnName].(float64) {
			fail(c)
			return
		}
		previous = res[axdb.AXDBTimeColumnName].(float64)
	}

	// verify the data, note that we delibrately overwritten some data, so can't verify everyline
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", "prim": "m1"}, dataArray[0])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 2*core.WeekInMicroSeconds, "prim": "m2"}, dataArray[3])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"val5": 5.1}, dataArray[4])

	//verify "contains key" for map column
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p2", "val3_contains_key": "v3k1"}, dataArray[1])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p2", "prim": "m1", "val3": "v3v1"}, dataArray[1])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p2", "val3_contains_key": "v3k2"}, dataArray[1])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p2", "prim": "m1", "val3": "v3v2"}, dataArray[1])

	processPost(c, appName, tableName, map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 2*core.WeekInMicroSeconds, "prim": "m2", "val1": 200},
		axerror.ERR_AXDB_INSERT_DUPLICATE.Code)

	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 2*core.WeekInMicroSeconds, "prim": "m2"}, dataArray[3])

	for i := range dataArray {
		dataArray[i]["val2"] = rand.Float64()
		processPut(c, appName, tableName, dataArray[i])
	}

	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 10*1e3, "prim": "m1"}, dataArray[0])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 2*core.WeekInMicroSeconds, "prim": "m2"}, dataArray[3])
	querySingleVerify(c, appName, tableName, map[string]interface{}{"val5": 5.1}, dataArray[4])

	deleteSingle(c, appName, tableName, map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 10*1e3, "prim": "m1"})
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", axdb.AXDBTimeColumnName: currentTime - 10*1e3, "prim": "m1"}, nil)

	doUpdateTableTest(c, table)
	// deleteTable(t, tableName)
}

func (s *S) TestKeyValueOrderedMap(c *check.C) {
	tableName := "TestKeyValueOrderedMap"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["part"] = axdb.Column{Type: axdb.ColumnTypeOrderedMap, Index: axdb.ColumnIndexPartition}
	columns["prim"] = axdb.Column{Type: axdb.ColumnTypeOrderedMap, Index: axdb.ColumnIndexClustering}
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeOrderedMap, Index: axdb.ColumnIndexStrong}

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeKeyValue, Columns: columns}

	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	// data to insert
	dataArray := []map[string]interface{}{
		map[string]interface{}{
			"part": map[string]interface{}{
				"A": "a",
				"B": "b",
				"C": "c",
			},
			"prim": map[string]interface{}{
				"A": "a",
				"B": "b",
				"C": "c",
			},
			"val1": map[string]interface{}{
				"A": "a",
				"B": "b",
				"C": "c",
			},
		},
		map[string]interface{}{
			"part": map[string]interface{}{
				"A": "e",
				"B": "f",
				"C": "g",
			},
			"prim": map[string]interface{}{
				"A": "h",
				"B": "i",
				"C": "j",
			},
			"val1": map[string]interface{}{
				"A": "k",
				"B": "l",
				"C": "m",
			},
		},
		map[string]interface{}{
			"part": map[string]interface{}{
				"A": "aa",
				"B": "bb",
				"C": "cc",
			},
			"prim": map[string]interface{}{
				"A": "aa",
				"B": "bb",
				"C": "cc",
			},
			"val1": map[string]interface{}{
				"A": "aa",
				"B": "bb",
				"C": "cc",
			},
		},
	}

	for i := range dataArray {
		processPut(c, appName, tableName, dataArray[i])
	}

	// verify the data, note that we delibrately overwritten some data, so can't verify everyline
	querySingleVerify(c, appName, tableName, map[string]interface{}{
		"part": map[string]interface{}{
			"A": "a",
			"B": "b",
			"C": "c",
		},
		"prim": map[string]interface{}{
			"A": "a",
			"B": "b",
			"C": "c",
		},
	}, dataArray[0])

	querySingleVerify(c, appName, tableName, map[string]interface{}{
		"val1": map[string]interface{}{
			"A": "aa",
			"B": "bb",
			"C": "cc",
		}}, dataArray[2])

	deleteSingle(c, appName, tableName, dataArray[0])
	querySingleVerify(c, appName, tableName, map[string]interface{}{
		"part": map[string]interface{}{
			"A": "a",
			"B": "b",
			"C": "c",
		},
		"prim": map[string]interface{}{
			"A": "a",
			"B": "b",
			"C": "c",
		},
	}, nil)
}

func (s *S) TestTimeSeriesOrderBy(c *check.C) {
	tableName := "TestTimeSeriesOrderBy"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeTimeSeries, Columns: columns}
	success, _ := processPut(c, "axdb", "update_table", table)
	c.Assert(success, check.Equals, true)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	data := make(map[string]interface{})
	for i := 0; i < 10; i++ {
		data["val1"] = i + 1
		success, _ := processPost(c, appName, tableName, data, successCode)
		c.Assert(success, check.Equals, true)
	}

	params := map[string]interface{}{
		axdb.AXDBQueryOrderByASC: []interface{}{axdb.AXDBUUIDColumnName},
	}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 10)

	// verify result sorted ASC
	var previous float64 = 0
	for _, res := range resultArray {
		c.Assert(previous != 0 && previous > res["val1"].(float64), check.Equals, false)
		previous = res["val1"].(float64)
	}

	params = map[string]interface{}{
		axdb.AXDBQueryOrderByDESC: []interface{}{axdb.AXDBUUIDColumnName},
	}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 10)

	// verify result sorted DESC
	previous = 0
	for _, res := range resultArray {
		c.Assert(previous != 0 && previous < res["val1"].(float64), check.Equals, false)
		previous = res["val1"].(float64)
	}

	// for JIRA-1000: populate some new data that span two weeks
	minTime := time.Now().UnixNano() / 1e3
	for i := 0; i < 10; i++ {
		// the last 5 elements will have timestamp one week after current time
		if i >= 5 {
			data["ax_time"] = minTime + int64((axdb.OneWeek+i-5)*1e6)
		}
		data["val1"] = i + 11
		success, _ := processPost(c, appName, tableName, data, successCode)
		c.Assert(success, check.Equals, true)
	}
	maxTime := time.Now().UnixNano() / 1e3
	params = map[string]interface{}{
		axdb.AXDBQueryOrderByASC: []interface{}{axdb.AXDBUUIDColumnName},
		axdb.AXDBQueryMinTime:    minTime,
		axdb.AXDBQueryMaxTime:    maxTime + int64((axdb.OneWeek+30)*1e6),
	}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 10)
	//verify result sorted ASC
	previous = 10
	for _, res := range resultArray {
		c.Assert(previous != 10 && previous > res["val1"].(float64), check.Equals, false)
		previous = res["val1"].(float64)
	}
}

func (s *S) TestUpdateTableTTL(c *check.C) {
	tableName := "TestUpdateTableTTL"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexPartition}

	table := axdb.Table{
		AppName: "test",
		Name:    tableName,
		Type:    axdb.TableTypeKeyValue,
		Columns: columns,
		Stats:   nil,
		Configs: map[string]interface{}{
			"default_time_to_live": 0,
			"comment":              "no commet at all",
		},
	}
	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	table.Configs["default_time_to_live"] = 2
	processUpdate(c, "axdb", "update_table", table)

	data := make(map[string]interface{})
	for i := 0; i < 2; i++ {
		data["val1"] = i + 1
		processPost(c, appName, tableName, data, successCode)
	}

	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 2)

	time.Sleep(time.Second * 2)
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)
}

func (s *S) TestKeyValueTableTTL(c *check.C) {
	tableName := "TestKeyValueTableTTL"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexPartition}

	table := axdb.Table{
		AppName: "test",
		Name:    tableName,
		Type:    axdb.TableTypeKeyValue,
		Columns: columns,
		Configs: map[string]interface{}{
			"comment":              "Today is good",
			"default_time_to_live": 5,
		},
	}
	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	data := make(map[string]interface{})
	for i := 0; i < 2; i++ {
		data["val1"] = i + 1
		processPost(c, appName, tableName, data, successCode)
	}

	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 2)
	time.Sleep(5 * time.Second)

	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)
}

func (s *S) TestTimeSeriesTableTTL(c *check.C) {
	tableName := "TestTimeSeriesTableTTL"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}

	stats := make(map[string]int)
	stats["val1"] = axdb.ColumnStatSum

	table := axdb.Table{
		AppName: "test",
		Name:    tableName,
		Type:    axdb.TableTypeTimeSeries,
		Columns: columns,
		Stats:   stats,
		Configs: map[string]interface{}{
			"default_time_to_live": 5,
			"comment":              "Today is good",
			"min_index_interval":   128,
		},
	}
	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	data := make(map[string]interface{})
	for i := 0; i < 2; i++ {
		data["val1"] = i + 1
		processPost(c, appName, tableName, data, successCode)

	}

	params := map[string]interface{}{
		axdb.AXDBIntervalColumnName: 1,
	}

	time.Sleep(8 * time.Second)

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 0)

	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)
}

func (s *S) TestTimeSeriesTableSelectCols(c *check.C) {
	tableName := "TestTimeSeriesTableSelectCols"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}

	stats := make(map[string]int)
	stats["val1"] = axdb.ColumnStatSum

	table := axdb.Table{
		AppName: "test",
		Name:    tableName,
		Type:    axdb.TableTypeTimeSeries,
		Columns: columns,
		Stats:   stats,
	}

	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	data := make(map[string]interface{})
	for i := 0; i < 5; i++ {
		data["val1"] = i + 1
		data["val2"] = i + 1
		data["val3"] = i + 1
		data["val4"] = i + 1
		processPost(c, appName, tableName, data, successCode)
	}

	params := map[string]interface{}{
		axdb.AXDBSelectColumns: []string{"val1", "val3"},
	}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 5)

	hasCols := &[]string{"val1", "val3"}
	hasNoCols := &[]string{"val2", "val4"}

	validateCols(c, resultArray, hasCols, hasNoCols)

	params = map[string]interface{}{
		axdb.AXDBSelectColumns: []string{"val1"},
	}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 5)

	hasCols = &[]string{"val1"}
	hasNoCols = &[]string{"val2", "val4", "val3"}

	validateCols(c, resultArray, hasCols, hasNoCols)
}

func (s *S) TestKeyValueTableSelectCols(c *check.C) {
	tableName := "TestKeyValueTableSelectCols"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexPartition}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}

	stats := make(map[string]int)
	stats["val1"] = axdb.ColumnStatSum

	table := axdb.Table{
		AppName: "test",
		Name:    tableName,
		Type:    axdb.TableTypeKeyValue,
		Columns: columns,
		Stats:   stats,
	}

	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	data := make(map[string]interface{})
	for i := 0; i < 5; i++ {
		data["val1"] = i + 1
		data["val2"] = i + 1
		data["val3"] = i + 1
		data["val4"] = i + 1
		processPost(c, appName, tableName, data, successCode)
	}

	params := map[string]interface{}{
		axdb.AXDBSelectColumns: []string{"val1", "val3"},
	}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 5)

	hasCols := &[]string{"val1", "val3"}
	hasNoCols := &[]string{"val2", "val4"}

	validateCols(c, resultArray, hasCols, hasNoCols)

	params = map[string]interface{}{
		axdb.AXDBSelectColumns: []string{"val1"},
	}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 5)

	hasCols = &[]string{"val1"}
	hasNoCols = &[]string{"val2", "val4", "val3"}

	validateCols(c, resultArray, hasCols, hasNoCols)

}

func (s *S) TestKeyValueTableSetType(c *check.C) {
	tableName := "TestKeyValueTableSetType"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexPartition}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone}

	table := axdb.Table{
		AppName: "test",
		Name:    tableName,
		Type:    axdb.TableTypeKeyValue,
		Columns: columns,
	}

	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	doFailureTest(c, table)

	data := make(map[string]interface{})
	data["val1"] = 1
	data["val2"] = []string{"a", "b", "c"}
	data["val3"] = []string{"a", "b", "c", "a", "b", "c"}
	data["val4"] = 1

	processPost(c, appName, tableName, data, successCode)
	params := map[string]interface{}{}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 1)

	result := resultArray[0]
	c.Logf("result:%v", result)
	c.Assert(result["val1"].(float64), check.Equals, 1.0)
	c.Assert(result["val4"].(float64), check.Equals, 1.0)
	c.Assert(len(result["val2"].([]interface{})), check.Equals, 3)
	c.Assert(len(result["val3"].([]interface{})), check.Equals, 3)

	data["val2"] = []string{"a", "b", "c", "a", "b"}
	data["val3"] = []string{"a", "b", "c", "d", "d", "d"}

	processPut(c, appName, tableName, data)

	params = map[string]interface{}{}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 1)

	result = resultArray[0]
	c.Logf("result:%v", result)
	c.Assert(result["val1"].(float64), check.Equals, 1.0)
	c.Assert(result["val4"].(float64), check.Equals, 1.0)
	c.Assert(len(result["val2"].([]interface{})), check.Equals, 3)
	c.Assert(len(result["val3"].([]interface{})), check.Equals, 4)

	data["val1"] = 2
	data["val2"] = []string{"g", "h", "j"}
	data["val3"] = []string{"g", "h", "j", "g", "h", "j", "a"}
	data["val4"] = 2

	processPost(c, appName, tableName, data, successCode)
	params = map[string]interface{}{
		"val3": "g",
	}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 1)

	result = resultArray[0]
	c.Logf("result:%v", result)
	c.Assert(result["val1"].(float64), check.Equals, 2.0)
	c.Assert(result["val4"].(float64), check.Equals, 2.0)

	params = map[string]interface{}{
		"val3": "a",
	}

	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 2)
}

func (s *S) TestLuceneIndexCreation(c *check.C) {
	c.Log("TestLuceneIndexCreation")

	tableName := "TestKeyValueTableLuceneCreate"

	columns := make(map[string]axdb.Column)
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	//columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexClustering}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexStrong}
	columns["val5"] = axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexStrong}
	columns["val6"] = axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexStrong}
	//columns["val7"] = axdb.Column{Type: axdb.ColumnTypeTimestamp, Index: axdb.ColumnIndexStrong}
	columns["val8"] = axdb.Column{Type: axdb.ColumnTypeUUID, Index: axdb.ColumnIndexStrong}
	columns["val9"] = axdb.Column{Type: axdb.ColumnTypeOrderedMap, Index: axdb.ColumnIndexStrong}
	columns["val10"] = axdb.Column{Type: axdb.ColumnTypeTimeUUID, Index: axdb.ColumnIndexStrong}
	columns["val11"] = axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong}

	table := axdb.Table{
		AppName:   "test",
		Name:      tableName,
		Type:      axdb.TableTypeKeyValue,
		Columns:   columns,
		UseSearch: true,
	}

	processPut(c, "axdb", "update_table", table)

	tableName = "TestTimedKeyValueTableLuceneCreate"
	table = axdb.Table{
		AppName:   "test",
		Name:      tableName,
		Type:      axdb.TableTypeTimedKeyValue,
		Columns:   columns,
		UseSearch: true,
	}

	processPut(c, "axdb", "update_table", table)

	tableName = "TestTimeSeriesTableLuceneCreate"
	table = axdb.Table{
		AppName:   "test",
		Name:      tableName,
		Type:      axdb.TableTypeTimeSeries,
		Columns:   columns,
		UseSearch: true,
	}

	processPut(c, "axdb", "update_table", table)

	c.Log("TestLuceneIndexCreation done")
}

func (s *S) TestLuceneIndexUpdate(c *check.C) {
	c.Log("TestLuceneIndexUpdate")

	appName := "test"

	tableNameBase := "TestLuceneCreate"

	// Test TableTypeTimeSeries/TableTypeKeyValue/TableTypeTimedKeyValue, except for TableTypeCounter
	for i := 0; i < axdb.TableTypeCounter; i++ {
		tableName := fmt.Sprintf("%s%v", tableNameBase, i)
		columns := make(map[string]axdb.Column)
		columns["val1"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
		//columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexClustering}
		columns["val3"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong}
		columns["val4"] = axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexStrong}
		columns["val5"] = axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexStrong}
		columns["val6"] = axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexStrong}
		//columns["val7"] = axdb.Column{Type: axdb.ColumnTypeTimestamp, Index: axdb.ColumnIndexStrong}
		columns["val8"] = axdb.Column{Type: axdb.ColumnTypeUUID, Index: axdb.ColumnIndexStrong}
		columns["val9"] = axdb.Column{Type: axdb.ColumnTypeOrderedMap, Index: axdb.ColumnIndexStrong}
		columns["val10"] = axdb.Column{Type: axdb.ColumnTypeTimeUUID, Index: axdb.ColumnIndexStrong}
		columns["val11"] = axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong}
		columns["val12"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong}

		// Create without lucene index
		table := axdb.Table{
			AppName:   appName,
			Name:      tableName,
			Type:      i,
			Columns:   columns,
			UseSearch: false,
		}
		processPut(c, "axdb", "update_table", table)

		data1 := make(map[string]interface{})
		data1["val1"] = "AAA"
		data1["val3"] = 1
		data1["val12"] = "aaa"

		data2 := make(map[string]interface{})
		data2["val1"] = "BBB"
		data2["val3"] = 2
		data2["val12"] = "bbb"

		data3 := make(map[string]interface{})
		data3["val1"] = "CCC"
		data3["val3"] = 3
		data3["val12"] = "ccc"

		data4 := make(map[string]interface{})
		data4["val1"] = "DDD'"
		data4["val3"] = 4
		data4["val12"] = "ddd'"

		processPost(c, appName, tableName, data1, successCode)
		processPost(c, appName, tableName, data2, successCode)
		processPost(c, appName, tableName, data3, successCode)
		processPost(c, appName, tableName, data4, successCode)

		// Update to have lucene index
		table.UseSearch = true
		processPut(c, "axdb", "update_table", table)

		time.Sleep(10 * time.Second)

		// Test must exact match
		luceneSearch := axdb.NewLuceneSearch()
		luceneSearch.AddQueryMust(axdb.NewLuceneWildCardFilterBase("val1", "aaa"))
		luceneSearch.AddQueryMust(axdb.NewLuceneWildCardFilterBase("val12", "aaa"))
		params := map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}

		resultArray := processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 1)
		c.Assert(resultArray[0]["val1"].(string), check.Equals, "AAA")

		// Test should filter
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddQueryShould(axdb.NewLuceneWildCardFilterBase("val1", "*a*"))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}

		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 1)
		c.Assert(resultArray[0]["val1"].(string), check.Equals, "AAA")

		// Test two should filters
		luceneSearch.AddQueryShould(axdb.NewLuceneWildCardFilterBase("val1", "*b*"))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 2)

		// Test must filters
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddQueryMust(axdb.NewLuceneWildCardFilterBase("val1", "*a*"))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 1)
		c.Assert(resultArray[0]["val1"].(string), check.Equals, "AAA")

		// Test two must filters
		luceneSearch.AddQueryMust(axdb.NewLuceneWildCardFilterBase("val1", "*aa*"))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 1)
		c.Assert(resultArray[0]["val1"].(string), check.Equals, "AAA")

		// Test two must + not filters
		luceneSearch.AddQueryNot(axdb.NewLuceneWildCardFilterBase("val1", "aaa"))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 0)

		// Test contains filter
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddQueryMust(axdb.NewLuceneContainsFilterBase("val3", []int64{1, 2, 3}))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 3)

		// Test sort by desc
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddSorter(axdb.NewLuceneSorterBase("val1", true))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 4)
		c.Assert(resultArray[0]["val1"].(string), check.Equals, "DDD'")
		c.Assert(resultArray[3]["val1"].(string), check.Equals, "AAA")

		// Test sort by asc
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddSorter(axdb.NewLuceneSorterBase("val1", false))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 4)
		c.Assert(resultArray[0]["val1"].(string), check.Equals, "AAA")
		c.Assert(resultArray[3]["val1"].(string), check.Equals, "DDD'")

		// Test two sorters
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddSorter(axdb.NewLuceneSorterBase("val12", false))
		luceneSearch.AddSorter(axdb.NewLuceneSorterBase("val1", true))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 4)
		c.Assert(resultArray[0]["val1"].(string), check.Equals, "AAA")
		c.Assert(resultArray[3]["val1"].(string), check.Equals, "DDD'")

		// Test search with single quotes
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddQueryShould(axdb.NewLuceneWildCardFilterBase("val1", "*'*"))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}

		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 1)
		c.Assert(resultArray[0]["val1"].(string), check.Equals, "DDD'")

		// Test sort by asc with limit 3
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddSorter(axdb.NewLuceneSorterBase("val3", false))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch:     luceneSearch,
			axdb.AXDBQueryMaxEntries: 3,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 3)
		c.Assert(int64(resultArray[0]["val3"].(float64)), check.Equals, int64(1))
		c.Assert(int64(resultArray[2]["val3"].(float64)), check.Equals, int64(3))

		// Test sort by asc with limit 3 offset 1
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddSorter(axdb.NewLuceneSorterBase("val3", false))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch:        luceneSearch,
			axdb.AXDBQueryMaxEntries:    3,
			axdb.AXDBQueryOffsetEntries: 1,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 3)
		c.Assert(int64(resultArray[0]["val3"].(float64)), check.Equals, int64(2))
		c.Assert(int64(resultArray[2]["val3"].(float64)), check.Equals, int64(4))

		// Test sort by asc with limit 2 offset 2
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddSorter(axdb.NewLuceneSorterBase("val3", false))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch:        luceneSearch,
			axdb.AXDBQueryMaxEntries:    2,
			axdb.AXDBQueryOffsetEntries: 2,
		}
		resultArray = processQuery(c, appName, tableName, params)
		c.Assert(len(resultArray), check.Equals, 2)
		c.Assert(int64(resultArray[0]["val3"].(float64)), check.Equals, int64(3))
		c.Assert(int64(resultArray[1]["val3"].(float64)), check.Equals, int64(4))

		// Drop a column to reindex
		delete(columns, "val11")
		table = axdb.Table{
			AppName:   appName,
			Name:      tableName,
			Type:      i,
			Columns:   columns,
			UseSearch: true,
		}
		processPut(c, "axdb", "update_table", table)

		// Add a column to reindex
		columns["val12"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong}
		table = axdb.Table{
			AppName:   appName,
			Name:      tableName,
			Type:      i,
			Columns:   columns,
			UseSearch: true,
		}
		processPut(c, "axdb", "update_table", table)

		// Drop the index
		table = axdb.Table{
			AppName:   appName,
			Name:      tableName,
			Type:      i,
			Columns:   columns,
			UseSearch: false,
		}
		processPut(c, "axdb", "update_table", table)

		// recreate lucene index with excluded column
		table.UseSearch = true
		table.ExcludedIndexColumns = map[string]bool{
			"val12": true,
		}
		processPut(c, "axdb", "update_table", table)

		time.Sleep(10 * time.Second)

		//make lucene index search on column val12 should be rejected
		luceneSearch = axdb.NewLuceneSearch()
		luceneSearch.AddQueryMust(axdb.NewLuceneWildCardFilterBase("val12", "aaa"))
		params = map[string]interface{}{
			axdb.AXDBQuerySearch: luceneSearch,
		}

		err := processQueryWithError(c, appName, tableName, params)
		c.Assert(err, check.Not(check.Equals), nil)
	}

	c.Log("TestLuceneIndexUpdate done")
}

func (s *S) TestRollUpStats(c *check.C) {
	tableName := "TimeSeriesTableRollUp"
	deleteTable(c, tableName)
	columns := make(map[string]axdb.Column)
	columns["part"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["part1"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeOrderedMap, Index: axdb.ColumnIndexNone}

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeTimeSeries, Columns: columns, Stats: map[string]int{"val2": axdb.ColumnStatPercent, "val1": axdb.ColumnStatSum}}
	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	// insert 500 rows of raw data
	currentTime := time.Now().UnixNano() / 1e3
	// align it to 500 seconds boundary
	currentTime = currentTime / (500 * 1e6) * 500 * 1e6
	data := make([]map[string]interface{}, 500)
	for i := 0; i < 500; i++ {
		data[i] = make(map[string]interface{})
		if i%2 == 1 {
			data[i]["part"] = "p3"
			data[i]["part1"] = "m3"
		} else {
			data[i]["part"] = "p4"
			data[i]["part1"] = "m4"
		}
		data[i]["val1"] = rand.Int()
		data[i]["val2"] = rand.Float64()
		data[i]["val3"] = strconv.Itoa(rand.Int())
		data[i]["ax_time"] = currentTime - int64((i+1)*1e6) // one second each
		processPost(c, appName, tableName, data[i], successCode)
	}

	// get stats at 100 second interval
	statParams := map[string]interface{}{"ax_interval": 100}
	rawstatArray := processQuery(c, appName, tableName, statParams)
	if len(rawstatArray) < 15 {
		// should be 15 rows since the stats are grouped by the partition keys, regardless the type of stats.
		c.Logf("expecting 15 stat results, got %d", len(rawstatArray))
		fail(c)
	}
	originalRes := make(map[string]map[string]int64)
	for _, data := range rawstatArray {
		key := data["part"].(string) + "," + data["part1"].(string)
		if originalRes[key] == nil {
			originalRes[key] = make(map[string]int64)
		}
		originalRes[key]["val1_count"] += int64(data["val1_count"].(float64))
		originalRes[key]["val1_sum"] += int64(data["val1_sum"].(float64))
		originalRes[key]["val2_count"] += int64(data["val2_count"].(float64))
	}

	// rollup stats for interval 500 seconds using interval = 100
	//statParams = map[string]interface{}{"ax_interval": 500, "ax_dst_interval": 500, "ax_src_interval": 100, "ax_rollup": 1}
	statParams = map[string]interface{}{"ax_interval": 500, "ax_src_interval": 100}
	statArray := processQuery(c, appName, tableName, statParams)
	if len(statArray) < 3 {
		c.Logf("expecting 3 rolled up stat results, got %d", len(statArray))
	}

	//check the rollup result is correct for Sum type of stats
	// and the count value of percent type of stats
	statParams = map[string]interface{}{"ax_interval": 500}
	statArray = processQuery(c, appName, tableName, statParams)
	for _, newData := range statArray {
		key := newData["part"].(string) + "," + newData["part1"].(string)
		// generated invalid partition key entry
		c.Assert(originalRes[key], check.NotNil)
		originalRes[key]["val1_count"] -= int64(newData["val1_count"].(float64))
		originalRes[key]["val1_sum"] -= int64(newData["val1_sum"].(float64))
		originalRes[key]["val2_count"] -= int64(newData["val2_count"].(float64))
	}

	for _, v := range originalRes {
		if v["val1_count"] != 0 || v["val1_sum"] != 0 || v["val2_count"] != 0 {
			fail(c)
			return
		}
	}
}

func (s *S) TestConditionUpdate(c *check.C) {
	tableName := "ConditionUpdateTable"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["part"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["part1"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering}
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeTimeUUID, Index: axdb.ColumnIndexNone}

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeTimeSeries, Columns: columns}
	var err *axerror.AXError
	var resMap map[string]interface{}
	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	// data to insert
	dataArray := []map[string]interface{}{
		map[string]interface{}{"part": "p1", "part1": "m1"},
		map[string]interface{}{"part": "p2", "part1": "m1", "val1": 1, "val2": 2.1, "val3": "v3.0"},
		map[string]interface{}{"part": "p1", "part1": "m2", "val1": 2},
		map[string]interface{}{"part": "p2", "part1": "m2", "val1": 3, "val2": 2.2, "val3": "v3.1"},
	}
	idArray := make([]string, len(dataArray))
	for i := range dataArray {
		_, resMap = processPost(c, appName, tableName, dataArray[i], successCode)
		idArray[i] = resMap[axdb.AXDBUUIDColumnName].(string)
	}

	//populate column "val4" with time
	for i := range dataArray {
		dataArray[i]["val4"] = idArray[i]
		dataArray[i][axdb.AXDBUUIDColumnName] = idArray[i]
		_, resMap = processPut(c, appName, tableName, dataArray[i])
	}

	// "if exists" and "if condition" co-exist; we must run into error.
	resMap, err = axdbClient.Put(appName, tableName, map[string]interface{}{"part": "p1", "part1": "m1", "val1": 4, axdb.AXDBUUIDColumnName: idArray[0], axdb.AXDBConditionalUpdateExist: "", "val1" + axdb.AXDBConditionalUpdateSuffix: nil})
	c.Assert(err, check.ErrorMatches, ".*error parsing conditional update clause for table.*")

	resMap, err = axdbClient.Put(appName, tableName, map[string]interface{}{"part": "p1", "part1": "m1", "val1": 4, axdb.AXDBUUIDColumnName: idArray[0], "val1" + axdb.AXDBConditionalUpdateSuffix: nil, axdb.AXDBConditionalUpdateExist: ""})
	c.Assert(err, check.ErrorMatches, ".*error parsing conditional update clause for table.*")

	// partition key is used in If condition
	resMap, err = axdbClient.Put(appName, tableName, map[string]interface{}{"part": "p1", "part1": "m1", "val1": 4, axdb.AXDBUUIDColumnName: idArray[0], "part" + axdb.AXDBConditionalUpdateSuffix: ""})
	c.Assert(err, check.ErrorMatches, ".*error parsing conditional update clause for table.*")

	// clustering key is used in If condition
	resMap, err = axdbClient.Put(appName, tableName, map[string]interface{}{"part": "p1", "part1": "m1", "val1": 4, axdb.AXDBUUIDColumnName: idArray[0], "part1" + axdb.AXDBConditionalUpdateSuffix: ""})
	c.Assert(err, check.ErrorMatches, ".*error parsing conditional update clause for table.*")

	// test if exists; if false, the row isn't inserted
	resMap, err = axdbClient.Put(appName, tableName, map[string]interface{}{"part": "p0", "part1": "m0", "val2": 3.0, axdb.AXDBUUIDColumnName: idArray[0], axdb.AXDBConditionalUpdateExist: ""})

	c.Assert(err, check.ErrorMatches, ".*Conditional Update failed, the row doesn't exist..*")

	// after processPut there are still 4 rows
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, len(dataArray))

	for i := range dataArray {
		querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[i]["part"], "part1": dataArray[i]["part1"], axdb.AXDBUUIDColumnName: idArray[i]}, dataArray[i])
	}

	// test if exists; if true, the row is updated
	_, resMap = processPut(c, appName, tableName, map[string]interface{}{"part": "p1", "part1": "m1", "val1": 4, axdb.AXDBUUIDColumnName: idArray[0], axdb.AXDBConditionalUpdateExist: ""})

	dataArray[0]["val1"] = 4
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[0]["part"], "part1": dataArray[0]["part1"], axdb.AXDBUUIDColumnName: idArray[0]}, dataArray[0])

	//test update ... if condition
	// update the secondary index column
	_, resMap = processPut(c, appName, tableName, map[string]interface{}{"part": "p2", "part1": "m1", "val1": 0, axdb.AXDBUUIDColumnName: idArray[1], "val3" + axdb.AXDBConditionalUpdateSuffix: "v3.0"})
	dataArray[1]["val1"] = 0
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[1]["part"], "part1": dataArray[1]["part1"], axdb.AXDBUUIDColumnName: idArray[1]}, dataArray[1])

	//test update if not condition on string column
	// the condition is false, no change is made
	resMap, err = axdbClient.Put(appName, tableName, map[string]interface{}{"part": "p2", "part1": "m2", "val1": 7, axdb.AXDBUUIDColumnName: idArray[3], "val3" + axdb.AXDBConditionalUpdateNotSuffix: "v3.1"})
	c.Assert(err, check.ErrorMatches, ".*Conditional Update failed, no changed was made.*")

	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[3]["part"], "part1": dataArray[3]["part1"], axdb.AXDBUUIDColumnName: idArray[3]}, dataArray[3])

	// the condition is true, change should be made
	_, resMap = processPut(c, appName, tableName, map[string]interface{}{"part": "p2", "part1": "m2", "val1": 7, axdb.AXDBUUIDColumnName: idArray[3], "val3" + axdb.AXDBConditionalUpdateNotSuffix: "v3.0"})
	dataArray[3]["val1"] = 7
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[3]["part"], "part1": dataArray[3]["part1"], axdb.AXDBUUIDColumnName: idArray[3]}, dataArray[3])

	// test both if and if not conditions in a single update
	// the condition is false, no change is made
	resMap, err = axdbClient.Put(appName, tableName, map[string]interface{}{"part": "p2", "part1": "m2", "val1": 7, axdb.AXDBUUIDColumnName: idArray[3], "val3" + axdb.AXDBConditionalUpdateNotSuffix: "v3.1", "val2" + axdb.AXDBConditionalUpdateSuffix: 2.2})
	c.Assert(err, check.ErrorMatches, ".*Conditional Update failed, no changed was made.*")

	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[3]["part"], "part1": dataArray[3]["part1"], axdb.AXDBUUIDColumnName: idArray[3]}, dataArray[3])

	// the condition is true, change should be made
	_, resMap = processPut(c, appName, tableName, map[string]interface{}{"part": "p2", "part1": "m2", "val1": 3, axdb.AXDBUUIDColumnName: idArray[3], "val3" + axdb.AXDBConditionalUpdateNotSuffix: "v3.0", "val2" + axdb.AXDBConditionalUpdateSuffix: 2.2})
	dataArray[3]["val1"] = 3
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[3]["part"], "part1": dataArray[3]["part1"], axdb.AXDBUUIDColumnName: idArray[3]}, dataArray[3])
	// test update if not condition on integer column
	_, resMap = processPut(c, appName, tableName, map[string]interface{}{"part": "p2", "part1": "m2", "val2": 2.3, axdb.AXDBUUIDColumnName: idArray[3], "val1" + axdb.AXDBConditionalUpdateNotSuffix: 2})
	dataArray[3]["val2"] = 2.3
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[3]["part"], "part1": dataArray[3]["part1"], axdb.AXDBUUIDColumnName: idArray[3]}, dataArray[3])

	// test update if not condition on float column
	_, resMap = processPut(c, appName, tableName, map[string]interface{}{"part": "p2", "part1": "m2", "val3": "v3.3", axdb.AXDBUUIDColumnName: idArray[3], "val2" + axdb.AXDBConditionalUpdateNotSuffix: 2.0})
	dataArray[3]["val3"] = "v3.3"
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[3]["part"], "part1": dataArray[3]["part1"], axdb.AXDBUUIDColumnName: idArray[3]}, dataArray[3])
}

func (s *S) TestConditionDelete(c *check.C) {
	tableName := "ConditionDeleteTable"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["part"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["part1"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering}
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeTimeUUID, Index: axdb.ColumnIndexNone}

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeTimeSeries, Columns: columns}
	var err *axerror.AXError
	var resMap map[string]interface{}
	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	// data to insert
	dataArray := []map[string]interface{}{
		map[string]interface{}{"part": "p1", "part1": "m1"},
		map[string]interface{}{"part": "p2", "part1": "m1", "val1": 1, "val2": 2.1, "val3": "v3.0"},
		map[string]interface{}{"part": "p1", "part1": "m2", "val1": 1},
		map[string]interface{}{"part": "p2", "part1": "m2", "val1": 3, "val2": 2.2, "val3": "v3.1"},
		map[string]interface{}{"part": "p3", "part1": "m3", "val1": 4, "val2": 3.1, "val3": "v3.3"},
		map[string]interface{}{"part": "p3", "part1": "m4", "val1": 5, "val2": 3.2, "val3": "v4.3"},
		map[string]interface{}{"part": "p3", "part1": "m5", "val1": 5, "val2": 3.3, "val3": "v4.3"},
		map[string]interface{}{"part": "p4", "part1": "m4", "val1": 6, "val2": 4.1, "val3": "v4.4"},
	}
	idArray := make([]string, len(dataArray))
	for i := range dataArray {
		_, resMap = processPost(c, appName, tableName, dataArray[i], successCode)
		idArray[i] = resMap[axdb.AXDBUUIDColumnName].(string)
	}

	//populate column "val4" with time
	for i := range dataArray {
		dataArray[i]["val4"] = idArray[i]
		dataArray[i][axdb.AXDBUUIDColumnName] = idArray[i]
		_, resMap = processPut(c, appName, tableName, dataArray[i])
	}

	// "if exists" and "if condition" co-exist; we must run into error.
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p1", "part1": "m1", axdb.AXDBUUIDColumnName: idArray[0], axdb.AXDBConditionalUpdateExist: "", "val1" + axdb.AXDBConditionalUpdateSuffix: nil}})
	c.Assert(err, check.ErrorMatches, ".*error parsing conditional update clause for table.*")

	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p1", "part1": "m1", "val1": 4, axdb.AXDBUUIDColumnName: idArray[0], "val1" + axdb.AXDBConditionalUpdateSuffix: nil, axdb.AXDBConditionalUpdateExist: ""}})
	c.Assert(err, check.ErrorMatches, ".*error parsing conditional update clause for table.*")

	// partition key is used in If condition
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p1", "part1": "m1", axdb.AXDBUUIDColumnName: idArray[0], "part" + axdb.AXDBConditionalUpdateSuffix: ""}})
	c.Assert(err, check.ErrorMatches, ".*error parsing conditional update clause for table.*")

	// clustering key is used in If condition
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p1", "part1": "m1", axdb.AXDBUUIDColumnName: idArray[0], "part1" + axdb.AXDBConditionalUpdateSuffix: ""}})
	c.Assert(err, check.ErrorMatches, ".*error parsing conditional update clause for table.*")

	// test if exists; if false, the row isn't deleted
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p0", "part1": "m0", axdb.AXDBUUIDColumnName: idArray[0], axdb.AXDBConditionalUpdateExist: ""}})
	c.Assert(err, check.ErrorMatches, ".*Conditional Update failed, the row doesn't exist..*")

	// after processPut there are still 8 rows
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, len(dataArray))

	for i := range dataArray {
		querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[i]["part"], "part1": dataArray[i]["part1"], axdb.AXDBUUIDColumnName: idArray[i]}, dataArray[i])
	}

	// test if exists; if true, the row is deleted
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p1", "part1": "m1", axdb.AXDBUUIDColumnName: idArray[0], axdb.AXDBConditionalUpdateExist: ""}})
	c.Assert(err, check.IsNil)
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[0]["part"], "part1": dataArray[0]["part1"], axdb.AXDBUUIDColumnName: idArray[0]}, nil)
	// after processPut there are 6 rows
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, len(dataArray)-1)

	//test update ... if condition
	// update the secondary index column
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"val1": 1, "val1" + axdb.AXDBConditionalUpdateSuffix: 1}})
	c.Assert(err, check.IsNil)
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p2", "part1": "m1", "val1": 1}, nil)
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": "p1", "part1": "m2", "val1": 1}, nil)
	// after processPut there are 4 rows
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, len(dataArray)-3)

	//test update if not condition on string column
	// the condition is false, no change is made
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p2", "part1": "m2", axdb.AXDBUUIDColumnName: idArray[3], "val3" + axdb.AXDBConditionalUpdateNotSuffix: "v3.1"}})
	c.Assert(err, check.ErrorMatches, ".*Conditional Update failed, no changed was made.*")
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[3]["part"], "part1": dataArray[3]["part1"], axdb.AXDBUUIDColumnName: idArray[3]}, dataArray[3])

	// the condition is true, change should be made
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p2", "part1": "m2", axdb.AXDBUUIDColumnName: idArray[3], "val3" + axdb.AXDBConditionalUpdateNotSuffix: "v3.0"}})
	c.Assert(err, check.IsNil)
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[3]["part"], "part1": dataArray[3]["part1"], axdb.AXDBUUIDColumnName: idArray[3]}, nil)
	// after processPut there are 4 rows
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, len(dataArray)-4)

	// test both if and if not conditions in a single update
	// the condition is false, no change is made
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p3", "part1": "m3", axdb.AXDBUUIDColumnName: idArray[4], "val3" + axdb.AXDBConditionalUpdateNotSuffix: "v3.1", "val2" + axdb.AXDBConditionalUpdateSuffix: 2.2}})
	c.Assert(err, check.ErrorMatches, ".*Conditional Update failed, no changed was made.*")
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[4]["part"], "part1": dataArray[4]["part1"], axdb.AXDBUUIDColumnName: idArray[4]}, dataArray[4])
	// after processPut there are still 4 rows
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, len(dataArray)-4)

	// the condition is true, change should be made
	resMap, err = axdbClient.Delete(appName, tableName, []map[string]interface{}{{"part": "p3", "part1": "m3", axdb.AXDBUUIDColumnName: idArray[4], "val3" + axdb.AXDBConditionalUpdateNotSuffix: "v3.0", "val2" + axdb.AXDBConditionalUpdateSuffix: 3.1}})
	c.Assert(err, check.IsNil)
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[4]["part"], "part1": dataArray[4]["part1"], axdb.AXDBUUIDColumnName: idArray[4]}, nil)
	// after processPut there are 3 rows
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, len(dataArray)-5)

	// test conditionally delete multiple rows
	data := []map[string]interface{}{
		{"part": "p3", "part1": "m4", axdb.AXDBUUIDColumnName: idArray[5], "val1" + axdb.AXDBConditionalUpdateSuffix: 5, "val2" + axdb.AXDBConditionalUpdateSuffix: 3.2, "val3" + axdb.AXDBConditionalUpdateSuffix: "v4.3"},
		{"part": "p3", "part1": "m5", axdb.AXDBUUIDColumnName: idArray[6], "val1" + axdb.AXDBConditionalUpdateSuffix: 5, "val2" + axdb.AXDBConditionalUpdateSuffix: 3.3, "val3" + axdb.AXDBConditionalUpdateSuffix: "v4.3"},
	}
	resMap, err = axdbClient.Delete(appName, tableName, data)
	c.Assert(err, check.IsNil)
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[5]["part"], "part1": dataArray[5]["part1"], axdb.AXDBUUIDColumnName: idArray[5]}, nil)
	querySingleVerify(c, appName, tableName, map[string]interface{}{"part": dataArray[6]["part"], "part1": dataArray[6]["part1"], axdb.AXDBUUIDColumnName: idArray[6]}, nil)
	// after processPut there are 1 rows
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, len(dataArray)-7)
}

func (s *S) TestIndexOnMapKey(c *check.C) {
	tableName := "IndexMapKeyTable"
	deleteTable(c, tableName)

	columns := make(map[string]axdb.Column)
	columns["part"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["part1"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering}
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	columns["val3"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone}
	columns["val4"] = axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexStrong, IndexFlagForMapColumn: axdb.ColumnIndexMapKeysAndValues}

	table := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeKeyValue, Columns: columns}
	processPut(c, "axdb", "update_table", table)

	resultArray := processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 0)

	// data to insert
	dataArray := []map[string]interface{}{
		map[string]interface{}{"part": "p1", "part1": "m1"},
		map[string]interface{}{"part": "p2", "part1": "m1", "val1": 1, "val2": 2.1, "val3": "v3.0", "val4": map[string]interface{}{"key1": "val1"}},
		map[string]interface{}{"part": "p1", "part1": "m2", "val1": 1, "val4": map[string]interface{}{"key1": "val1", "key2": "val2"}},
		map[string]interface{}{"part": "p2", "part1": "m2", "val1": 3, "val2": 2.2, "val3": "v3.1", "val4": map[string]interface{}{"key1": "val1", "key2": "val2", "key3": "val3"}},
		map[string]interface{}{"part": "p3", "part1": "m3", "val1": 4, "val2": 3.1, "val3": "v3.3", "val4": map[string]interface{}{"key1": "val1", "key2": "val2", "key3": "val3", "key4": "val4"}},
		map[string]interface{}{"part": "p3", "part1": "m4", "val1": 5, "val2": 3.2, "val3": "v4.3", "val4": map[string]interface{}{"key2": "val2", "key3": "val3"}},
	}
	for i := range dataArray {
		processPost(c, appName, tableName, dataArray[i], successCode)
	}

	// we insert 6 rows
	resultArray = processQuery(c, appName, tableName, nil)
	c.Assert(len(resultArray), check.Equals, 6)

	params := map[string]interface{}{
		"val4" + axdb.AXDBMapColumnKeySuffix: "key1",
	}
	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 4)

	params = map[string]interface{}{
		"val4" + axdb.AXDBMapColumnKeySuffix: "key2",
	}
	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 4)

	params = map[string]interface{}{
		"val4" + axdb.AXDBMapColumnKeySuffix: "key3",
	}
	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 3)

	params = map[string]interface{}{
		"val4" + axdb.AXDBMapColumnKeySuffix: "key4",
	}
	resultArray = processQuery(c, appName, tableName, params)
	c.Assert(len(resultArray), check.Equals, 1)

}
