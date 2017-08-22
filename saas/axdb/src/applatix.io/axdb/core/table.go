// Copyright 2015-2016 Applatix, Inc. All rights reserved.

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"applatix.io/axdb"
	"github.com/gocql/gocql"
)

// defines one column
type Column struct {
	axdb.Column
}

/*
 * Real table implementations need to implement the following interfaces. The common pattern:
 * Embed struct Table in TableImplementation.
 * Implement the following required functions
 * initBackend()
 * save()
 *
 * Override other functions as needed
 */
type TableInterface interface {
	// init in-memory structure
	init() *axdb.AXDBError
	// init DB table
	initBackend() *axdb.AXDBError
	// update DB table
	updateBackend(changedData UpdateData) *axdb.AXDBError

	// wait for the table to be ready
	waitForReady(sec int) bool
	// delete DB table
	deleteBackend() *axdb.AXDBError

	// save data
	save(data map[string]interface{}, isNewInsert bool) (map[string]interface{}, *axdb.AXDBError)
	// get data
	get(params map[string]interface{}) (resultArray []map[string]interface{}, axErr *axdb.AXDBError)
	// delete data
	delete(paramsArray []map[string]interface{}) (map[string]interface{}, *axdb.AXDBError)

	getAppName() string
	getName() string
	getFullName() string
	getTableType() int
	getStatList() map[string]int

	// given user supplied params, get the query string
	getQueryStringForRequest(typedParams map[string]interface{}) string
	// given user supplied params, return whether we will trigger pagination
	queryUsesPagination(params map[string]interface{}) bool
	// add AXDB generated columns based on user supplied params
	addGeneratedParams(params map[string]interface{}) *axdb.AXDBError
	// get the proper ending week if it's missed
	getMissedEndWeek(params map[string]interface{}) int64
	// whether we will query by ascending time order
	queryUseAscendTimeOrder(params map[string]interface{}) bool
}

func getMaxEntries(typedParams map[string]interface{}) (maxEntries int64) {
	if typedParams[axdb.AXDBQueryMaxEntries] != nil {
		maxEntries = typedParams[axdb.AXDBQueryMaxEntries].(int64)
	}
	if maxEntries == 0 {
		return axdb.AXDBArrayMax
	} else {
		return maxEntries + getOffsetEntries(typedParams)
	}
}

func getOffsetEntries(typedParams map[string]interface{}) (offsetEntries int64) {
	if typedParams[axdb.AXDBQueryOffsetEntries] != nil {
		offsetEntries = typedParams[axdb.AXDBQueryOffsetEntries].(int64)
	}
	return offsetEntries
}

// Base table implementation. This implements the common table functions, but can't be TableInterface
type Table struct {
	axdb.Table
	real           TableInterface // the real table implementation
	fullName       string         // full DB name
	partitionKeys  []string       // array of partition key names
	clusteringKeys []string       // array of user defined clustering key names, in the order of importance
	beginWeek      int64          // week of the earliest data entry
	refreshTime    int64          // last time we refreshed the beginWeek value
}

func (table *Table) getAppName() string {
	return table.AppName
}

func (table *Table) getName() string {
	return table.Name
}

func (table *Table) getFullName() string {
	return table.fullName
}

func (table *Table) getTableType() int {
	return table.Type
}

func (table *Table) getStatList() map[string]int {
	return table.Stats
}

func (table *Table) getQueryStringForRequest(typedParams map[string]interface{}) string {
	return fmt.Sprintf("SELECT %s FROM %s %s %s LIMIT %d ALLOW FILTERING", table.getSelectColsClause(typedParams), table.fullName, table.getWhereClause(typedParams), table.getOrderByClause(typedParams), getMaxEntries(typedParams))
}

// Look for the specific index entries and return "" if nothing exists, "col1" if one exist, "(col1, col2...)"
// if multiple are found
func (table *Table) getIndexString(indexType int, needBracket bool, generatedFirst bool) string {
	var buffer bytes.Buffer
	count := 0

	printColumn := func(name string) {
		if count == 0 {
			buffer.WriteString(fmt.Sprintf("%s", name))
		} else {
			buffer.WriteString(fmt.Sprintf(",%s", name))
		}
		count++
	}
	printGenerated := func() {
		if col, exist := table.Columns[axdb.AXDBUUIDColumnName]; exist &&
			(col.Index == indexType || col.Index == axdb.ColumnIndexClusteringStrong && indexType == axdb.ColumnIndexClustering) {
			printColumn(axdb.AXDBUUIDColumnName)
		}
		if col, exist := table.Columns[axdb.AXDBTimeColumnName]; exist &&
			(col.Index == indexType || col.Index == axdb.ColumnIndexClusteringStrong && indexType == axdb.ColumnIndexClustering) {
			printColumn(axdb.AXDBTimeColumnName)
		}
		if col, exist := table.Columns[axdb.AXDBWeekColumnName]; exist &&
			(col.Index == indexType || col.Index == axdb.ColumnIndexClusteringStrong && indexType == axdb.ColumnIndexClustering) {
			printColumn(axdb.AXDBWeekColumnName)
		}
	}

	if generatedFirst {
		printGenerated()
	}
	var keyArray []string
	if indexType == axdb.ColumnIndexPartition {
		keyArray = table.partitionKeys
	} else if indexType == axdb.ColumnIndexClustering {
		keyArray = table.clusteringKeys
	} else {
		panic(fmt.Sprintf("getIndexString for index type %d not supported", indexType))
	}

	for _, colname := range keyArray {
		printColumn(colname)
	}
	if !generatedFirst {
		printGenerated()
	}
	if count == 0 {
		return ""
	} else if count == 1 || !needBracket {
		return buffer.String()
	} else {
		return fmt.Sprintf("(%s)", buffer.String())
	}
}

func jsonMarshal(data interface{}) (jsonStr string, axErr *axdb.AXDBError) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		errorLog.Println(err)
		errStr := fmt.Sprintf("failed to marshal %v into json\n", data)
		errorLog.Printf(errStr)
		axErr = axdb.NewAXDBError(axdb.RestStatusInvalid, err, errStr)
	} else {
		jsonStr = string(jsonData[:])
	}
	return jsonStr, axErr
}

func (table *Table) createSecondaryIndex(colName string) *axdb.AXDBError {
	col, ok := table.Columns[colName]
	if !ok {
		errStr := fmt.Sprintf("The column (%s) doesn't exist, can not create index on it", colName)
		errorLog.Println(errStr)
		return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)

	}

	if col.Index == axdb.ColumnIndexStrong || col.Index == axdb.ColumnIndexClusteringStrong {
		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}
		var queryString string
		if col.IndexFlagForMapColumn == axdb.ColumnIndexMapKeys {
			queryString = fmt.Sprintf("create index on %s (keys(%s))", table.fullName, colName)

		} else {
			queryString = fmt.Sprintf("create index on %s (%s)", table.fullName, colName)
		}

		err := createSecondaryIndexExecutor(queryString)
		if err != nil {
			errorLog.Println(err)
			return err
		}

		if col.IndexFlagForMapColumn == axdb.ColumnIndexMapKeysAndValues {
			queryString = fmt.Sprintf("create index on %s (keys(%s))", table.fullName, colName)
			err := createSecondaryIndexExecutor(queryString)
			if err != nil {
				errorLog.Println(err)
				return err
			}
		}
	}
	return nil
}

func createSecondaryIndexExecutor(queryString string) *axdb.AXDBError {
	infoLog.Println(queryString)
	err := execQuery(queryString, false, true)
	if err != nil {
		errorLog.Println(err)
		return err
	}
	return nil
}

// create secondary indexes
func (table *Table) createSecondaryIndexes() *axdb.AXDBError {
	for colName, _ := range table.Columns {
		err := table.createSecondaryIndex(colName)
		if err != nil {
			errorLog.Println(err)
			return err
		}
	}
	return nil
}

// Create lucene index
func (table *Table) createLuceneIndex() *axdb.AXDBError {
	if table.UseSearch {
		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}
		schema := NewLuceneIndexSchema()
		schema.addFields(table.Columns, table.ExcludedIndexColumns)
		if !schema.isEmpty() {
			schemaBytes, axErr := jsonMarshal(schema)
			if axErr != nil {
				return axErr
			}
			schemaStr := string(schemaBytes)
			schemaStr = strings.Replace(schemaStr, "\"$$ax$$", "", -1)
			schemaStr = strings.Replace(schemaStr, "$$ax$$\"", "", -1)

			schemaStr = fmt.Sprintf(LuceneIndexTemplate, table.fullName, axdb.AXDBLuceneIndexSuffix, table.fullName, schemaStr)

			infoLog.Println(schemaStr)
			err := execQuery(schemaStr, false, true)
			if err != nil {
				errorLog.Printf(fmt.Sprintf("Failed to create the lucene index for table %v:%v", table.fullName, err))
				return err
			}
		} else {
			debugLog.Printf("Skip creating lucene index for table %v, no column ", table.fullName)
		}
	} else {
		debugLog.Printf("Skip creating lucene index for table %v, UseSearch %v", table.fullName, table.UseSearch)
	}

	return nil
}

// Exec insert query.
func (table *Table) execInsert(data map[string]interface{}) *axdb.AXDBError {

	var err error
	for colname, v := range data {
		col, ok := table.Columns[colname]
		if !ok {
			errStr := fmt.Sprintf("Data field %s not found in DB", colname)
			errorLog.Printf(errStr)
			return axdb.NewAXDBError(axdb.RestStatusInvalid, err, errStr)
		}

		// Serialize to string for orderedMap
		if col.Type == axdb.ColumnTypeOrderedMap {
			valueMap := v.(map[string]interface{})
			valString := axdb.SerializeOrderedMap(valueMap)
			data[colname] = valString
		}
	}

	jsonData, axErr := jsonMarshal(data)
	if axErr != nil {
		return axErr
	}

	queryString := fmt.Sprintf("INSERT INTO %s JSON '%s' IF NOT EXISTS", table.fullName, jsonData)
	applied := false

	infoLog.Println(queryString)
	existing := make(map[string]interface{})
	qry := theDB.session.Query(queryString)
	applied, err = qry.MapScanCAS(existing)
	if err != nil {
		errStr := fmt.Sprintf("DB error for request %s", queryString)
		errorLog.Println(errStr)
		errorLog.Println(err)
		axErr = axdb.NewAXDBError(axdb.GetAXDBErrCodeFromDBError(err), err, errStr)
	}

	if axErr == nil && !applied {
		infoLog.Printf("insert failed, entry existed already")
		axErr = axdb.NewAXDBError(axdb.RestStatusForbidden, nil, "Can't POST the same entry twice")
	}

	return axErr
}

// Exec update query.
func (table *Table) execUpdate(data map[string]interface{}) *axdb.AXDBError {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("UPDATE %s SET ", table.fullName))

	var err error
	first := true
	for colname, v := range data {
		// it's a parameter for conditional update; just skip it now.
		if colname == axdb.AXDBConditionalUpdateExist || strings.HasSuffix(colname, axdb.AXDBConditionalUpdateSuffix) || strings.HasSuffix(colname, axdb.AXDBConditionalUpdateNotSuffix) {
			continue
		}
		if strings.HasSuffix(colname, axdb.AXDBVectorColumnPlusSuffix) || strings.HasSuffix(colname, axdb.AXDBVectorColumnMinusSuffix) {
			continue
		}
		col, ok := table.Columns[colname]
		if !ok {
			errorLog.Printf("Error updating Table %s with data:\n%v", table.fullName, data)
			errStr := fmt.Sprintf("Data field %s not found in DB", colname)
			errorLog.Printf(errStr)
			return axdb.NewAXDBError(axdb.RestStatusInvalid, err, errStr)
		}

		if col.Index == axdb.ColumnIndexPartition || col.Index == axdb.ColumnIndexClustering || col.Index == axdb.ColumnIndexClusteringStrong {
			continue
		}

		if !first {
			buffer.WriteString(", ")
		} else {
			first = false
		}

		if v == nil {
			buffer.WriteString(fmt.Sprintf("%s = null", colname))
			delete(data, colname)
			continue
		}

		switch col.Type {
		case axdb.ColumnTypeString:
			buffer.WriteString(fmt.Sprintf("%s = '%v'", colname, v))
		case axdb.ColumnTypeArray:
			firstElement := true
			buffer.WriteString(fmt.Sprintf("%s = [", colname))
			valueArray := v.([]interface{})
			for i := range valueArray {
				if !firstElement {
					buffer.WriteString(", ")
				} else {
					firstElement = false
				}
				buffer.WriteString(fmt.Sprintf("'%v'", valueArray[i]))
			}
			buffer.WriteString("]")
		case axdb.ColumnTypeSet:
			firstElement := true
			plus := fmt.Sprintf("%s%s", colname, axdb.AXDBVectorColumnPlusSuffix)
			minus := fmt.Sprintf("%s%s", colname, axdb.AXDBVectorColumnMinusSuffix)
			if data[plus] != nil {
				delete(data, plus)
				buffer.WriteString(fmt.Sprintf("%s = %s + {", colname, colname))
			} else if data[minus] != nil {
				delete(data, minus)
				buffer.WriteString(fmt.Sprintf("%s = %s - {", colname, colname))
			} else {
				buffer.WriteString(fmt.Sprintf("%s = {", colname))
			}

			valueArray := v.([]interface{})
			for i := range valueArray {
				if !firstElement {
					buffer.WriteString(", ")
				} else {
					firstElement = false
				}
				buffer.WriteString(fmt.Sprintf("'%v'", valueArray[i]))
			}
			buffer.WriteString("}")
		case axdb.ColumnTypeMap:
			firstElement := true
			buffer.WriteString(fmt.Sprintf("%s = {", colname))
			valueMap := v.(map[string]interface{})
			for valueK, valueV := range valueMap {
				if !firstElement {
					buffer.WriteString(", ")
				} else {
					firstElement = false
				}
				buffer.WriteString(fmt.Sprintf("'%s': '%v'", valueK, valueV))
			}
			buffer.WriteString("}")
		case axdb.ColumnTypeOrderedMap:
			if _, ok := v.(string); ok {
				buffer.WriteString(fmt.Sprintf("%s = '%v'", colname, v))
			} else {
				valueMap := v.(map[string]interface{})
				valString := axdb.SerializeOrderedMap(valueMap)
				buffer.WriteString(fmt.Sprintf("%s = '%v'", colname, valString))
			}
		default:
			buffer.WriteString(fmt.Sprintf("%s = %v", colname, v))
		}

		// remove the key from where clause
		delete(data, colname)
	}

	conditionUpdStr, axErr := table.getConditionalUpdateClause(data)
	if axErr != nil {
		return axErr
	}
	queryString := fmt.Sprintf("%s %s %s", buffer.String(), table.getWhereClause(data), conditionUpdStr)
	infoLog.Println(queryString)

	if len(conditionUpdStr) != 0 {
		return execQuery(queryString, true)
	} else {
		return execQuery(queryString, false)
	}

}

// Do the necessary in memory data structure init work.
func (table *Table) init() *axdb.AXDBError {
	addedClustering := make(map[string]int)
	if len(table.IndexOrder) != 0 {
		for _, colname := range table.IndexOrder {
			table.clusteringKeys = append(table.clusteringKeys, colname)
			addedClustering[colname] = 1
		}
	}
	for colname, col := range table.Columns {
		if !axdb.NameIsGenerated(colname) {
			if col.Index == axdb.ColumnIndexPartition {
				table.partitionKeys = append(table.partitionKeys, colname)
			} else if (col.Index == axdb.ColumnIndexClusteringStrong || col.Index == axdb.ColumnIndexClustering) && addedClustering[colname] != 1 {
				table.clusteringKeys = append(table.clusteringKeys, colname)
				addedClustering[colname] = 1
			}
		}
	}
	table.fullName = GetTableFullName(table.AppName, table.Name)
	return nil
}

// alter table by adding new columns and removing old columns
//                updating the index
//                updating table configuration (i.e. TTL)
func (table *Table) updateBackend(changedData UpdateData) *axdb.AXDBError {
	changedCols := changedData.changedCols
	changedFlags := changedData.changedFlags
	changedLuceneIndex := changedData.changedLuceneIndex

	infoLog.Printf(fmt.Sprintf("**** changed cols: %v, changed flags: %v, changed lucene index: %v", changedCols, changedFlags, changedLuceneIndex))

	if changedLuceneIndex == UpdateLuceneIndexDrop || changedLuceneIndex == UpdateLuceneIndexReCreate {
		dropLuceneIndex := fmt.Sprintf("DROP INDEX IF EXISTS %s%s", table.fullName, axdb.AXDBLuceneIndexSuffix)
		infoLog.Printf(dropLuceneIndex)
		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}
		err := execQuery(dropLuceneIndex)
		if err != nil {
			errorLog.Println(err)
			return err
		}
	}

	for k, v := range changedFlags {
		var buffer bytes.Buffer
		if v == UpdateAddNewColumn || v == UpdateAddNewColumnWithSecondaryIndex {
			buffer.WriteString(fmt.Sprintf("ALTER TABLE %s ADD %s %s;", table.fullName, k, axdbColumnTypeNames[table.Columns[k].Type]))
		} else if v == UpdateDropOldColumn {
			buffer.WriteString(fmt.Sprintf("ALTER TABLE %s DROP %s;", table.fullName, k))
		} else if v == UpdateDropSecondaryIndex {
			buffer.WriteString(fmt.Sprintf("DROP INDEX %s_%s_idx", table.fullName, k))
		}

		infoLog.Printf(fmt.Sprintf("Update the schema %s", buffer.String()))
		infoLog.Printf(fmt.Sprintf("isDrop = %d, index type = %d", v, table.Columns[k].Index))

		// the column to be dropped has a secondary index defined on it
		if v == UpdateDropOldColumn && changedCols[k].Index == axdb.ColumnIndexStrong {
			var dropIndexBuf bytes.Buffer
			dropIndexBuf.WriteString(fmt.Sprintf("DROP INDEX %s_%s_idx", table.fullName, k))
			infoLog.Printf(fmt.Sprintf("*** drop index: %s ", dropIndexBuf.String()))
			if theDB.replFactor > 1 {
				if err := theDB.WaitSchemaAgreement(); err != nil {
					return err
				}
			}
			err := execQuery(dropIndexBuf.String())
			if err != nil {
				//skip the error if the index to be dropped doesn't exist
				//this has been completed in execQuery()
				errorLog.Println(err)
				return err
			}
			if changedCols[k].IndexFlagForMapColumn == axdb.ColumnIndexMapKeysAndValues {
				dropIndexBuf.Reset()
				dropIndexBuf.WriteString(fmt.Sprintf("DROP INDEX %s_%s_idx_1", table.fullName, k))
				err := execQuery(dropIndexBuf.String())
				if err != nil {
					errorLog.Println(err)
					return err
				}
			}
		}
		if v != UpdateAddSecondaryIndex {
			if theDB.replFactor > 1 {
				if err := theDB.WaitSchemaAgreement(); err != nil {
					return err
				}
			}

			err := execQuery(buffer.String())
			if err != nil {
				errorLog.Println(err)
				return err
			}
			if v == UpdateReCreateSecondaryIndex {
				buffer.Reset()
				buffer.WriteString(fmt.Sprintf("DROP INDEX %s_%s_idx_1", table.fullName, k))
				err := execQuery(buffer.String())
				if err != nil {
					errorLog.Println(err)
					return err
				}
			}
		}

		// add secondary index
		if v == UpdateAddSecondaryIndex || v == UpdateAddNewColumnWithSecondaryIndex || v == UpdateReCreateSecondaryIndex {
			err := table.createSecondaryIndex(k)
			if err != nil {
				errorLog.Println(err)
				return err
			}
		}
	}

	// if TTL changed
	changedConfigs := changedData.changedConfigs
	for configName, _ := range changedConfigs {
		var buffer bytes.Buffer
		if configName == "default_time_to_live" {
			infoLog.Printf(fmt.Sprintf("*** TTL value to be update: %d", int64(changedConfigs[configName].(float64))))
			buffer.WriteString(fmt.Sprintf("ALTER TABLE %s WITH %s = %d", table.fullName, configName, int64(changedConfigs[configName].(float64))))
		} else {
			continue
		}
		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}

		infoLog.Printf(fmt.Sprintf("Update the schema %s", buffer.String()))
		err := execQuery(buffer.String())
		if err != nil {
			errorLog.Println(err)
			return err
		}
	}

	if changedLuceneIndex == UpdateLuceneIndexReCreate {
		return table.createLuceneIndex()
	}

	return nil
}

// pass in the cassandra clause starting "primary key (" all the way to the end.
func (table *Table) initBackendWithPrimaryClause(primaryKeyClause string) *axdb.AXDBError {
	var buffer bytes.Buffer
	if theDB.replFactor > 1 {
		if err := theDB.WaitSchemaAgreement(); err != nil {
			return err
		}
	}

	buffer.WriteString(fmt.Sprintf("CREATE TABLE %s (", table.fullName))
	for colname, col := range table.Columns {
		buffer.WriteString(fmt.Sprintf("%s %s, ", colname, axdbColumnTypeNames[col.Type]))
	}

	buffer.WriteString(primaryKeyClause)

	if len(table.Configs) > 0 {
		hasWITH := strings.Contains(primaryKeyClause, "WITH")
		for k, v := range table.Configs {
			debugLog.Printf("config %v type %T", k, v)
			if hasWITH {
				switch v.(type) {
				case string:
					buffer.WriteString(fmt.Sprintf(" AND %s = '%s'", k, v))
				case float64:
					buffer.WriteString(fmt.Sprintf(" AND %s = %d", k, int64(v.(float64))))
				default:
					buffer.WriteString(fmt.Sprintf(" AND %s = %v", k, v))
				}
			} else {
				switch v.(type) {
				case string:
					buffer.WriteString(fmt.Sprintf(" WITH %s = '%s'", k, v))
				case float64:
					buffer.WriteString(fmt.Sprintf(" WITH %s = %d", k, int64(v.(float64))))
				default:
					buffer.WriteString(fmt.Sprintf(" WITH %s = %v", k, v))
				}
				hasWITH = true
			}
		}
	}

	infoLog.Println(buffer.String())
	err := execQuery(buffer.String(), false, true)

	if err != nil {
		// retry. A table creation immediately following the same table deletion will hit a timeout.
		// loop for 20 seconds.
		tableIsReady := table.waitForReady(20)
		if !tableIsReady {
			err = execQuery(buffer.String(), false, true)
			if err != nil {
				return err
			}
		}
	}

	if theDB.replFactor > 1 {
		if err := theDB.WaitSchemaAgreement(); err != nil {
			return err
		}
	}
	if err = table.createSecondaryIndexes(); err != nil {
		errorLog.Println(err)
		return err
	}

	return table.createLuceneIndex()
}

func (table *Table) deleteBackend() *axdb.AXDBError {
	table.real = nil
	_, err := tableDefinitionTable.delete([]map[string]interface{}{map[string]interface{}{axdb.AXDBKeyColumnName: table.fullName}})
	if err == nil && table.exists() {
		if theDB.replFactor > 1 {
			if err := theDB.WaitSchemaAgreement(); err != nil {
				return err
			}
		}
		err = execQuery(fmt.Sprintf("DROP TABLE %s", table.fullName))
		if err != nil {
			// sometimes the operation succeeds, but we still get an error (such as timeout) back. Check and make sure.
			// TODO we need to figure out this spurious timeout error
			for i := 0; i < 20; i++ {
				time.Sleep(1 * time.Second)
				if !table.exists() {
					return nil
				}
			}
		}
	}
	return err
}

// wait for the max number of seconds for the table to be ready. Returns whether the table is ready
func (table *Table) waitForReady(sec int) bool {
	for i := 0; i < sec; i++ {
		if table.exists() {
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

// check whether the table exists
func (table *Table) exists() bool {
	axErr := execQuery(fmt.Sprintf("SELECT * FROM %s LIMIT 1", table.fullName))
	return axErr == nil
}

func getWeekFromTime(t time.Time) int64 {
	return t.Unix() / WeekInSeconds
}

func (table *Table) getOrderByClause(params map[string]interface{}) string {
	first := true
	var buffer bytes.Buffer
	cols, ok := params[axdb.AXDBQueryOrderByASC]
	if ok {
		for _, col := range cols.([]interface{}) {
			column, exist := table.Columns[col.(string)]
			if !exist || column.Index != axdb.ColumnIndexClustering && column.Index != axdb.ColumnIndexClusteringStrong {
				warningLog.Printf("Order by condition for column %s is ignored", col)
				continue
			}

			if first {
				buffer.WriteString(fmt.Sprintf("ORDER BY %s ASC", col))
				first = false
			} else {
				buffer.WriteString(fmt.Sprintf(", %s ASC", col))
			}

		}
	}

	cols, ok = params[axdb.AXDBQueryOrderByDESC]
	if ok {
		for _, col := range cols.([]interface{}) {
			column, exist := table.Columns[col.(string)]
			if !exist || column.Index != axdb.ColumnIndexClustering && column.Index != axdb.ColumnIndexClusteringStrong {
				warningLog.Printf("Order by condition for invalid column %s is ignored", col)
				continue
			}

			if first {
				buffer.WriteString(fmt.Sprintf("ORDER BY %s DESC", col))
				first = false
			} else {
				buffer.WriteString(fmt.Sprintf(", %s DESC", col))
			}

		}
	}
	return buffer.String()
}

func (table *Table) getSelectColsClause(params map[string]interface{}) string {
	if cols, ok := params[axdb.AXDBSelectColumns]; ok {
		return strings.Join(cols.([]string), ",")
	} else {
		return "*"
	}
}

func (table *Table) getConditionalUpdateClause(params map[string]interface{}) (string, *axdb.AXDBError) {
	first := true
	hasExistClause := false
	/* the behavior of condition update:
	   1. if exists and if condition cannot co-exist. Our implementation will do the following:
	        if params contain both exists and conditions, we will ignore exists and only use condition.
	   2. if exists can be applied only once. Our implementation will ignore the duplication.
	   3. primary key columns are not allowed in the conditions. Our implementation will report error if it's detected.
	   4. collection type of column isn't allowed in condition
	   5. UUID type of column is supported in condition
	   6. The column can be updated and used in condition even a secondary index is defined on it.
	*/
	var buffer bytes.Buffer
	for k, v := range params {
		if k == axdb.AXDBConditionalUpdateExist {
			if hasExistClause {
				errStr := fmt.Sprintf("error parsing conditional update clause for table %s: Multiple EXISTS are found", table.fullName)
				errorLog.Printf(errStr)
				return "", axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
			} else {
				if !first {
					errStr := fmt.Sprintf("error parsing conditional update clause for table %s: both EXISTS and CONDITIONS are found", table.fullName)
					errorLog.Printf(errStr)
					return "", axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				} else {
					buffer.WriteString(" IF EXISTS")
					hasExistClause = true
					first = false
				}
			}
		} else if strings.HasSuffix(k, axdb.AXDBConditionalUpdateSuffix) || strings.HasSuffix(k, axdb.AXDBConditionalUpdateNotSuffix) {
			if hasExistClause {
				errStr := fmt.Sprintf("error parsing conditional update clause for table %s: both EXISTS and CONDITIONS are found", table.fullName)
				errorLog.Printf(errStr)
				return "", axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
			}
			equalCondition := strings.HasSuffix(k, axdb.AXDBConditionalUpdateSuffix)
			var colName string
			if equalCondition {
				colName = strings.TrimSuffix(k, axdb.AXDBConditionalUpdateSuffix)
			} else {
				colName = strings.TrimSuffix(k, axdb.AXDBConditionalUpdateNotSuffix)
			}

			// if k == "ax_min_time_upd_if" || "ax_max_time_upd_if"
			if colName == axdb.AXDBQueryMaxTime || colName == axdb.AXDBQueryMinTime {
				// invalid JSON input in the payload
				if !equalCondition {
					errStr := fmt.Sprintf("error parsing conditional update clause for table %s: inequality condtion isn't allowed on (min|max)_time", table.fullName)
					errorLog.Printf(errStr)
					return "", axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
				if timeColumn, exist := table.Columns[axdb.AXDBTimeColumnName]; exist &&
					(timeColumn.Index == axdb.ColumnIndexClustering || timeColumn.Index == axdb.ColumnIndexClusteringStrong || timeColumn.Index == axdb.ColumnIndexPartition) {
					errStr := fmt.Sprintf("error parsing conditional update clause for table %s: primary key column cannot be used in condition update", table.fullName)
					errorLog.Printf(errStr)
					return "", axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
				if colName == axdb.AXDBQueryMaxTime {
					if first {
						buffer.WriteString(fmt.Sprintf(" IF %s < %v ", axdb.AXDBTimeColumnName, params[k].(int64)))
						first = false
					} else {
						buffer.WriteString(fmt.Sprintf(" AND %s < %v ", axdb.AXDBTimeColumnName, params[k].(int64)))
					}
				} else {
					if first {
						buffer.WriteString(fmt.Sprintf(" IF %s >= %v ", axdb.AXDBTimeColumnName, params[k].(int64)))
						first = false
					} else {
						buffer.WriteString(fmt.Sprintf(" AND %s >= %v ", axdb.AXDBTimeColumnName, params[k].(int64)))
					}
				}
			} else {
				column, exist := table.Columns[colName]
				if !exist {
					continue
				}
				if column.Index == axdb.ColumnIndexPartition || column.Index == axdb.ColumnIndexClustering || column.Index == axdb.ColumnIndexClusteringStrong {
					errStr := fmt.Sprintf("error parsing conditional update clause for table %s: primary key column cannot be used in condition update", table.fullName)
					errorLog.Printf(errStr)
					return "", axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
				if column.Type == axdb.ColumnTypeCounter || column.Type == axdb.ColumnTypeSet || column.Type == axdb.ColumnTypeArray ||
					column.Type == axdb.ColumnTypeMap || column.Type == axdb.ColumnTypeOrderedMap {
					errStr := fmt.Sprintf("error parsing conditional update clause for table %s: type of column %s cannot be used in condition update", table.fullName, colName)
					errorLog.Printf(errStr)
					return "", axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
				equalSign := "="
				if !equalCondition {
					equalSign = "!="
				}
				if first {
					buffer.WriteString(fmt.Sprintf("IF %s %s ", colName, equalSign))
					first = false
				} else {
					buffer.WriteString(fmt.Sprintf(" AND %s %s ", colName, equalSign))
				}
				if column.Type == axdb.ColumnTypeString {
					buffer.WriteString(fmt.Sprintf("'%s'", v))
				} else {
					buffer.WriteString(fmt.Sprintf("%v", v))
				}
			}
		} else {
			continue
		}
	}

	return buffer.String(), nil
}

func (table *Table) getWhereClause(params map[string]interface{}) string {
	first := true
	var buffer bytes.Buffer
	for k, v := range params {
		if k == axdb.AXDBQueryOrderByASC || k == axdb.AXDBQueryOrderByDESC || k == axdb.AXDBSelectColumns || k == axdb.AXDBQueryMaxEntries || k == axdb.AXDBQueryOffsetEntries || k == axdb.AXDBConditionalUpdateExist {
			continue
		}

		// if params contains params that are used in update ... if syntax
		if strings.HasSuffix(k, axdb.AXDBConditionalUpdateSuffix) || strings.HasSuffix(k, axdb.AXDBConditionalUpdateNotSuffix) {
			continue
		}

		if k == axdb.AXDBQuerySearch {
			searchString := params[axdb.AXDBQuerySearch].(string)
			debugLog.Printf(searchString)
			// Escaping the single quotes in the CQL
			//searchString = strings.Replace(searchString, "'", "''", -1)
			searchString = strings.Replace(searchString, "\"$$ax$$", "", -1)
			searchString = strings.Replace(searchString, "$$ax$$\"", "", -1)

			if first {
				buffer.WriteString(fmt.Sprintf(" WHERE expr(%s%s,'%s') ", table.fullName, axdb.AXDBLuceneIndexSuffix, searchString))
				first = false
			} else {
				buffer.WriteString(fmt.Sprintf(" AND expr(%s%s,'%s') ", table.fullName, axdb.AXDBLuceneIndexSuffix, searchString))
			}

			continue
		}

		if k == axdb.AXDBQueryMaxTime {
			maxTimeStamp := params[axdb.AXDBQueryMaxTime].(int64)

			if timeColumn, exist := table.Columns[axdb.AXDBTimeColumnName]; exist && (timeColumn.Index == axdb.ColumnIndexClustering || timeColumn.Index == axdb.ColumnIndexClusteringStrong) {
				if first {
					buffer.WriteString(fmt.Sprintf(" WHERE %s < %v ", axdb.AXDBTimeColumnName, maxTimeStamp))
					first = false
				} else {
					buffer.WriteString(fmt.Sprintf(" AND %s < %v ", axdb.AXDBTimeColumnName, maxTimeStamp))
				}
			}

			// TODO this is not exactly right. Converting time to uuid is not repeatable. We should
			// 1) allow passing in a max_uuid and use that for pagination
			// 2) if using a time, use Cassandra's minTimeuuid function. What we have here is close enough for now.
			if uuidColumn, exist := table.Columns[axdb.AXDBUUIDColumnName]; exist && (uuidColumn.Index == axdb.ColumnIndexClustering || uuidColumn.Index == axdb.ColumnIndexClusteringStrong) {
				maxTime := time.Unix(maxTimeStamp/1e6, (maxTimeStamp%1e6)*1e3-1)
				maxTimeUUID := gocql.UUIDFromTime(maxTime)
				if first {
					buffer.WriteString(fmt.Sprintf(" WHERE %s < %v ", axdb.AXDBUUIDColumnName, maxTimeUUID))
					first = false
				} else {
					buffer.WriteString(fmt.Sprintf(" AND %s < %v ", axdb.AXDBUUIDColumnName, maxTimeUUID))
				}
			}
			continue
		}
		if k == axdb.AXDBQueryMinTime {
			minTimeStamp := params[axdb.AXDBQueryMinTime].(int64)

			if timeColumn, exist := table.Columns[axdb.AXDBTimeColumnName]; exist && (timeColumn.Index == axdb.ColumnIndexClustering || timeColumn.Index == axdb.ColumnIndexClusteringStrong) {
				if first {
					buffer.WriteString(fmt.Sprintf(" WHERE %s >= %v ", axdb.AXDBTimeColumnName, minTimeStamp))
					first = false
				} else {
					buffer.WriteString(fmt.Sprintf(" AND %s >= %v ", axdb.AXDBTimeColumnName, minTimeStamp))
				}
			}

			if uuidColumn, exist := table.Columns[axdb.AXDBUUIDColumnName]; exist && (uuidColumn.Index == axdb.ColumnIndexClustering || uuidColumn.Index == axdb.ColumnIndexClusteringStrong) {
				minTime := time.Unix(minTimeStamp/1e6, (minTimeStamp%1e6)*1e3-1)
				minTimeUUID := gocql.UUIDFromTime(minTime)
				if first {
					buffer.WriteString(fmt.Sprintf(" WHERE %s >= %v ", axdb.AXDBUUIDColumnName, minTimeUUID))
					first = false
				} else {
					buffer.WriteString(fmt.Sprintf(" AND %s >= %v ", axdb.AXDBUUIDColumnName, minTimeUUID))
				}
			}
			continue
		}

		// identify if it's a "contains key" query on map column
		var mapContainsKey bool = false
		if strings.HasSuffix(k, axdb.AXDBMapColumnKeySuffix) {
			mapContainsKey = true
			k = strings.TrimSuffix(k, axdb.AXDBMapColumnKeySuffix)
		}

		column := table.Columns[k]

		equalSign := "="
		if column.Type == axdb.ColumnTypeArray || column.Type == axdb.ColumnTypeSet {
			equalSign = "contains"
		}

		if column.Type == axdb.ColumnTypeMap {
			if mapContainsKey {
				equalSign = "contains key"
			} else {
				equalSign = "contains"
			}
		}

		if first {
			buffer.WriteString(fmt.Sprintf("WHERE %s %s ", k, equalSign))
			first = false
		} else {
			buffer.WriteString(fmt.Sprintf(" AND %s %s ", k, equalSign))
		}
		if column.Type == axdb.ColumnTypeString || column.Type == axdb.ColumnTypeMap || column.Type == axdb.ColumnTypeArray || column.Type == axdb.ColumnTypeSet {
			buffer.WriteString(fmt.Sprintf("'%s'", v.(string)))
		} else if column.Type == axdb.ColumnTypeOrderedMap {
			if _, ok := v.(string); ok {
				buffer.WriteString(fmt.Sprintf("'%s'", v.(string)))
			} else {
				valueMap := v.(map[string]interface{})
				valString := axdb.SerializeOrderedMap(valueMap)
				buffer.WriteString(fmt.Sprintf("'%s'", valString))
			}
		} else {
			buffer.WriteString(fmt.Sprintf("%v", v))
		}
	}
	return buffer.String()
}

// Does this query uses secondary index
func (table *Table) queryUsesSecondaryIndex(params map[string]interface{}) bool {
	for k, _ := range params {
		if strings.HasSuffix(k, axdb.AXDBMapColumnKeySuffix) {
			continue
		}
		if table.Columns[k].Index == axdb.ColumnIndexStrong || table.Columns[k].Index == axdb.ColumnIndexClusteringStrong {
			return true
		}
	}
	return false
}

func (table *Table) queryIgnoreWeek(params map[string]interface{}) bool {
	return (!table.queryUsesPagination(params) || table.queryUsesLuceneIndex(params)) && !table.queryUsesPartitionIndex(params)
}

func (table *Table) queryUsesLuceneIndex(params map[string]interface{}) bool {
	_, ok := params[axdb.AXDBQuerySearch]
	return ok
}

func (table *Table) queryUsesTime(params map[string]interface{}) bool {
	if params[axdb.AXDBQueryMinTime] != nil || params[axdb.AXDBQueryMaxTime] != nil {
		return true
	} else {
		return false
	}
}

// Does this query uses proper partition key
func (table *Table) queryUsesPartitionIndex(params map[string]interface{}) bool {
	for i := range table.partitionKeys {
		_, exist := params[table.partitionKeys[i]]
		if !exist {
			return false
		}
	}
	return true
}

// transfer data based on index type
func (table *Table) copyIndexData(data map[string]interface{}, indexType int) map[string]interface{} {
	params := make(map[string]interface{})
	for k, v := range data {
		if strings.HasSuffix(k, axdb.AXDBMapColumnKeySuffix) {
			continue
		}
		if table.Columns[k].Index == indexType {
			params[k] = v
		}
	}
	return params
}

// don't return the cached table.beginWeek. Another node might have updated it to a smaller number.
func (table *Table) getBeginWeekFor(data map[string]interface{}, force bool) (int64, *axdb.AXDBError) {
	currentTime := time.Now().Unix()
	if !force && table.beginWeek != 0 && table.refreshTime > currentTime-TimeSeriesTableDataRefreshDelay {
		return table.beginWeek, nil
	}

	name := table.fullName
	for {
		l := len(name)
		if l > len(axdb.AXDBStatSuffix) {
			suffix := name[l-len(axdb.AXDBStatSuffix) : l]
			if suffix == axdb.AXDBStatSuffix || suffix == axdb.AXDBTimeViewSuffix {
				name = name[0 : l-len(axdb.AXDBStatSuffix)]
			} else {
				break
			}
		} else {
			break
		}
	}
	params := map[string]interface{}{axdb.AXDBKeyColumnName: name}
	resArray, axErr := tableDefinitionTable.get(params)
	if len(resArray) == 0 {
		return 0, axErr
	}
	table.beginWeek = resArray[0][axdb.AXDBTimeColumnName].(int64) / WeekInMicroSeconds
	table.refreshTime = currentTime
	return table.beginWeek, axErr
}

func (table *Table) saveBeginWeekIfNeeded(data map[string]interface{}, week int64) *axdb.AXDBError {
	if table.beginWeek <= week && table.beginWeek != 0 {
		return nil
	}

	beginWeek, axErr := table.getBeginWeekFor(data, true)
	if axErr == nil && beginWeek == 0 || beginWeek > week {
		params := map[string]interface{}{axdb.AXDBKeyColumnName: table.fullName}
		params[axdb.AXDBTimeColumnName] = week * WeekInMicroSeconds
		_, axErr = tableDefinitionTable.save(params, false)
		table.beginWeek = week
	}

	return axErr
}

func uuidToTime(uuidStr string) time.Time {
	uuid, err := gocql.ParseUUID(uuidStr)
	if err != nil {
		errorLog.Printf("Can't parse UUID string: %s", uuidStr)
		return EpochTime
	}
	if uuid.Version() != 1 {
		errorLog.Printf("UUID is not a version 1 uuid %s", uuidStr)
		return EpochTime
	}
	return uuid.Time()
}

// overridden by the tables that actually uses pagination
func (table *Table) queryUsesPagination(params map[string]interface{}) bool {
	return false
}

// overridden by the tables that actually query by ascending time order
func (table *Table) queryUseAscendTimeOrder(params map[string]interface{}) bool {
	return false
}

func (table *Table) getTypedParams(params map[string]interface{}) (map[string]interface{}, *axdb.AXDBError) {
	var err error
	typedParams := make(map[string]interface{})
	var colType int
	for k, v := range params {
		switch v.(type) {
		case string:
			str := v.(string)
			if k == axdb.AXDBQueryMinTime || k == axdb.AXDBQueryMaxTime || k == axdb.AXDBQueryMaxEntries || k == axdb.AXDBQueryOffsetEntries {
				typedParams[k], err = strconv.ParseInt(str, 10, 64)
			} else if k == axdb.AXDBQueryOrderByASC || k == axdb.AXDBQueryOrderByDESC {
				var listValue []interface{}
				err = json.Unmarshal([]byte(str), &listValue)
				typedParams[k] = listValue
			} else {
				var colname string
				if strings.HasSuffix(k, axdb.AXDBMapColumnKeySuffix) {
					colname = strings.TrimSuffix(k, axdb.AXDBMapColumnKeySuffix)
				} else {
					colname = k
				}
				column := table.Columns[colname]
				colType = column.Type
				switch colType {
				case axdb.ColumnTypeBoolean:
					typedParams[k], err = strconv.ParseBool(str)
				case axdb.ColumnTypeInteger:
					typedParams[k], err = strconv.ParseInt(str, 10, 64)
				case axdb.ColumnTypeDouble:
					typedParams[k], err = strconv.ParseFloat(str, 64)
				case axdb.ColumnTypeOrderedMap:
					mapValue := make(map[string]interface{})
					err = json.Unmarshal([]byte(str), &mapValue)
					typedParams[k] = mapValue
				default:
					typedParams[k] = v
				}
			}
			if err != nil {
				errStr := fmt.Sprintf("error parsing table %s column %s, expecting type %s, got value %s, error %v",
					table.fullName, k, axdbColumnTypeNames[colType], v, err)
				errorLog.Printf(errStr)
				return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, err, errStr)
			}
		case []string:
			typedParams[k] = v
		default:
			typedParams[k] = v
		}
	}
	return typedParams, nil
}

// by default don't add any new params
func (table *Table) addGeneratedParams(params map[string]interface{}) *axdb.AXDBError {
	return nil
}

func (table *Table) tableAccessible() (bool, *axdb.AXDBError) {
	if table.getAppName() == axdb.AXDBAppAXINT {
		return true, nil
	}
	tblStatus, err := theDB.getApp(table.getAppName()).getMemTableStatus(table.Name)
	if err != nil {
		return false, err
	}
	if tblStatus == TableUpdateInProcess {
		errStr := fmt.Sprintf("The table (%s) is under updating, access is denied", table.Name)
		// in this case we should allow the client to retry, so change the error code to InternalError
		return false, axdb.NewAXDBError(axdb.RestStatusInternalError, nil, errStr)
	}
	return true, nil
}

func (table *Table) getMissedEndWeek(params map[string]interface{}) int64 {
	return -1
}

func (table *Table) get(params map[string]interface{}) (resultArray []map[string]interface{}, axErr *axdb.AXDBError) {
	isAccessible, err := table.tableAccessible()
	if err != nil || !isAccessible {
		return nil, err
	}
	// http params are parsed as strings. We need to convert to the proper types
	typedParams, err := table.getTypedParams(params)
	if err != nil {
		return nil, err
	}

	var maxEntries int64
	if typedParams[axdb.AXDBQueryMaxEntries] != nil {
		maxEntries = typedParams[axdb.AXDBQueryMaxEntries].(int64)
	}
	if maxEntries == 0 {
		maxEntries = axdb.AXDBArrayMax
	}

	var offsetEntries int64
	if typedParams[axdb.AXDBQueryOffsetEntries] != nil {
		offsetEntries = typedParams[axdb.AXDBQueryOffsetEntries].(int64)
	}

	usePagination := table.real.queryUsesPagination(params)

	resultArray = make([]map[string]interface{}, maxEntries)

	var current int64 = 0
	var beginWeek int64 = 0
	var endWeek int64 = 0
	if usePagination {
		beginWeek, axErr = table.getBeginWeekFor(params, false)
		if beginWeek == 0 || axErr != nil {
			return resultArray[0:0], axErr
		}
	}

	queryByAscendingTimeOrder := table.real.queryUseAscendTimeOrder(typedParams)
	if queryByAscendingTimeOrder {
		endWeek = table.real.getMissedEndWeek(typedParams)
	}

	for true {
		queryString := table.real.getQueryStringForRequest(typedParams)
		infoLog.Print(queryString)
		qry := theDB.session.Query(queryString)
		iter := qry.Iter()
		returnedRows, err := iter.SliceMap()
		if err != nil {
			return resultArray[0:0], axdb.NewAXDBError(axdb.GetAXDBErrCodeFromDBError(err), err, err.Error())
		}

		for _, row := range returnedRows {
			if offsetEntries > 0 {
				offsetEntries--
			} else {
				for k, v := range row {
					// Recover the map representation
					if table.Columns[k].Type == axdb.ColumnTypeOrderedMap {
						orderedMap := axdb.DeserializeOrderedMap(v.(string))
						row[k] = orderedMap
					}
				}
				resultArray[current] = row
				current++
				if current >= maxEntries {
					break
				}
			}
		}
		if !usePagination || table.queryUsesLuceneIndex(params) || current >= maxEntries {
			break
		}

		if !queryByAscendingTimeOrder {
			if typedParams[axdb.AXDBWeekColumnName].(int64) <= beginWeek || beginWeek == 0 {
				break
			} else {
				typedParams[axdb.AXDBWeekColumnName] = typedParams[axdb.AXDBWeekColumnName].(int64) - 1
			}
		} else {
			if typedParams[axdb.AXDBWeekColumnName].(int64) >= endWeek || endWeek == 0 {
				break
			} else {
				typedParams[axdb.AXDBWeekColumnName] = typedParams[axdb.AXDBWeekColumnName].(int64) + 1
			}
		}
	}
	return resultArray[0:current], nil
}

func (table *Table) save(data map[string]interface{}, isNewInsert bool) (map[string]interface{}, *axdb.AXDBError) {
	isAccessible, axErr := table.tableAccessible()
	if axErr != nil || !isAccessible {
		return nil, axErr
	}
	axErr = table.real.addGeneratedParams(data)
	if axErr != nil {
		return nil, axErr
	}

	if isNewInsert {
		return nil, table.execInsert(data)
	} else {
		return nil, table.execUpdate(data)
	}
}

// Delete data from the table. Pass in an array of params. Each params map contains the key to use in where clause
func (table *Table) delete(paramsArray []map[string]interface{}) (map[string]interface{}, *axdb.AXDBError) {
	var err *axdb.AXDBError
	var isAccessible bool
	isAccessible, err = table.tableAccessible()
	if err != nil || !isAccessible {
		return nil, err
	}
	for _, param := range paramsArray {
		if needQueryBeforeDelete := !validDelete(param, table); needQueryBeforeDelete {
			newParamArray, err1 := table.get(param)
			if err1 != nil {
				errStr := fmt.Sprintf("Error when retrieving (%s)", param)
				warningLog.Printf(errStr)
				continue
			}
			// if param contains conditional delete columns, add them to newParamArray
			for k, v := range param {
				if strings.HasSuffix(k, axdb.AXDBConditionalUpdateExist) || strings.HasSuffix(k, axdb.AXDBConditionalUpdateSuffix) || strings.HasSuffix(k, axdb.AXDBConditionalUpdateNotSuffix) {
					for i := range newParamArray {
						newParamArray[i][k] = v
					}
				}
			}
			_, err = table.delete(newParamArray)
			if err != nil {
				return nil, err
			} else {
				continue
			}
		}

		// get the condition clause before they are discarded later
		conditionDelStr, axErr := table.getConditionalUpdateClause(param)
		if axErr != nil {
			return nil, axErr
		}

		// remove the entries that are not partition and clustering keys
		for colname, _ := range param {
			col, ok := table.Columns[colname]
			if colname == axdb.AXDBConditionalUpdateExist || strings.HasSuffix(colname, axdb.AXDBConditionalUpdateSuffix) || strings.HasSuffix(colname, axdb.AXDBConditionalUpdateNotSuffix) {
				continue
			}
			if !ok {
				errStr := fmt.Sprintf("Data field %s not found in DB", colname)
				errorLog.Printf(errStr)
				return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
			}

			if col.Index != axdb.ColumnIndexPartition && col.Index != axdb.ColumnIndexClustering &&
				col.Index != axdb.ColumnIndexClusteringStrong {
				delete(param, colname)
			}
		}

		// TODO error handling. What's the right schemantic to the client? Pass the failed data back?
		if err = table.real.addGeneratedParams(param); err == nil {
			queryString := fmt.Sprintf("DELETE FROM %s %s %s", table.fullName, table.getWhereClause(param), conditionDelStr)
			infoLog.Print(queryString)
			// for conditional deletion, we will return immediately if we run into error;
			// and there would be no-op for remaining rows in param array
			if len(conditionDelStr) != 0 {
				err = execQuery(queryString, true)
				if err != nil {
					return nil, err
				}
			} else {
				execQuery(queryString, false) // no error checking, we try to delete the next one in the param array
			}

		}
	}
	return nil, nil
}

// check if the data param contains all partition keys
func validDelete(data map[string]interface{}, table *Table) bool {
	for _, colname := range table.partitionKeys {
		_, existing := data[colname]
		if !existing {
			return false
		}
	}
	return true
}
