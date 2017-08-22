// Copyright 2015-2016 Applatix, Inc. All rights reserved.

package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"applatix.io/axdb"
	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"github.com/gocql/gocql"
)

// for cassandra cluster, it's good practice to allow some oeprations running on only one node.
// These kinds of operations will run on axdb-0; so that both standlone and cluster mode work.
// We think axdb-0 as leader node.
func isLeaderNode() bool {
	if theDB.numNodes == 1 {
		return true
	}
	//check the hostname
	cmd := exec.Command("hostname")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errorLog.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		errorLog.Fatal(err)
	}
	defer cmd.Wait()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		hostname = strings.Trim(line, " ")
		infoLog.Printf(fmt.Sprintf("HostName = %s", hostname))
		// axdb-0 is the expected hostname
		leaderName := fmt.Sprintf("%s-%d", axdb.AXDBServiceName, 0)
		return strings.Contains(line, leaderName)
	}
	return false
}

func getConfigsDiff(oldConfigs map[string]interface{}, newConfigs map[string]interface{}) map[string]interface{} {
	//now we only consider TTL
	//TODO: add other configuration if needed
	var changedConfigs map[string]interface{}
	oldTTL := getTTL(oldConfigs)
	newTTL := getTTL(newConfigs)
	if oldTTL != newTTL {
		changedConfigs = make(map[string]interface{})
		changedConfigs["default_time_to_live"] = newTTL
	}
	return changedConfigs
}

func getStatsDefDiff(oldStatDef map[string]int, newStatDef map[string]int) map[string]bool {
	// we currently only consider adding/removing a column to/from the statTable
	// don't consider the change to the stat type of a column
	// changedStatDef[colName] = true : stat columns for colName will be added to the statTable
	// changedStatDef[colName] = false : stat columns for colName will be removed from the statTable
	changedStatDef := make(map[string]bool)
	for colName, _ := range oldStatDef {
		if _, exist := newStatDef[colName]; !exist {
			changedStatDef[colName] = false
		}
	}
	for colName, _ := range newStatDef {
		if _, exist := oldStatDef[colName]; !exist {
			changedStatDef[colName] = true
		}
	}
	return changedStatDef
}

func getTTL(configs map[string]interface{}) float64 {
	var ttl float64
	if configs == nil {
		ttl = 0
	} else {
		v, exist := configs["default_time_to_live"].(float64)
		if exist {
			ttl = v
		} else {
			ttl = 0
		}
	}
	return ttl
}

func execQuery(queryString string, flags ...bool) *axdb.AXDBError {
	query := theDB.session.Query(queryString).RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: 5})
	retMap := make(map[string]interface{})
	var err error
	var hasReturn bool
	var schemaOperation bool = false
	var applied bool
	if len(flags) == 0 {
		hasReturn = false
	} else if len(flags) == 1 {
		hasReturn = flags[0]
	} else {
		hasReturn = flags[0]
		schemaOperation = flags[1]
	}

	if schemaOperation {
		query = query.Consistency(gocql.Quorum)
	}

	if !hasReturn {
		err = query.Exec()
	} else {
		applied, err = query.MapScanCAS(retMap)
		if err == nil && !applied {
			// the record doesn't exist
			if len(retMap) == 0 {
				return axdb.NewAXDBError(axdb.RestStatusInvalid, err, fmt.Sprintf("Conditional Update failed, the row doesn't exist."))
			} else {
				return axdb.NewAXDBError(axdb.RestStatusInvalid, err, fmt.Sprintf("Conditional Update failed, no changed was made: %s", queryString))
			}
		}
	}

	if err != nil {
		retcode := axdb.GetAXDBErrCodeFromDBError(err)
		if retcode != axdb.RestStatusOK {
			errorLog.Printf("failed to run query: %s", queryString)
			errorLog.Printf("error from db: %v", err)
			return axdb.NewAXDBError(retcode, err, fmt.Sprintf("DB error on query: %s", queryString))
		} else { // some 0x2200 errors could be ignored, and we won't return error in those cases.
			infoLog.Printf("ignoring error %v for query %s", err, queryString)
		}
	}
	return nil
}

func copyMap(data map[string]interface{}) map[string]interface{} {
	newData := make(map[string]interface{})
	if data != nil {
		for k, v := range data {
			newData[k] = v
		}
	}
	return newData
}

// represent the DB for a specific axdb app
type App struct {
	mutex sync.Mutex
	name  string

	tables map[string]TableInterface
	//flag indicating the current state of each table
	flags map[string]int

	// later we will add app specific things such as query strategy and per-app query statistics managment here
}

// data structure to represent the changes made to table schama
type UpdateData struct {
	changedCols  map[string]axdb.Column
	changedFlags map[string]int
	//table configuration
	changedConfigs map[string]interface{}
	// stat table definition
	changedStatCols map[string]bool
	// lucene index change
	changedLuceneIndex int
}

func getDBMetaDataFromJsonByte(metaJson []byte) (axdb.DBMetaData, *axdb.AXDBError) {
	var metadata axdb.DBMetaData
	err := json.Unmarshal(metaJson, &metadata)
	if err != nil {
		errStr := fmt.Sprintf("failed to unmarshal the following json into a meta object: %s", metaJson)
		warningLog.Printf(errStr)
		return metadata, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}
	return metadata, nil
}

func addAppTableFromJsonByte(tableJson []byte, initBackend bool, opportunistic bool) *axdb.AXDBError {
	table, err := getAppTableDefFromJsonByte(tableJson)
	if err != nil {
		return err
	}
	return addAppTable(&table, initBackend, opportunistic)
}

func getAppTableDefFromJsonByte(tableJson []byte) (axdb.Table, *axdb.AXDBError) {
	var table axdb.Table
	err := json.Unmarshal(tableJson, &table)
	if err != nil {
		errStr := fmt.Sprintf("failed to unmarshal the following json into a table object: %s", tableJson)
		warningLog.Printf(errStr)
		return table, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}

	if table.AppName == "" {
		warningLog.Printf("missing appName: %s", tableJson)
		return table, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, "missing app name")
	}

	return table, nil
}

func updateAppTableFromJsonByte(tableJson []byte) *axdb.AXDBError {
	// first compare the current vesion against that in the metadata table
	table, err := getAppTableDefFromJsonByte(tableJson)
	if err != nil {
		return err
	}
	return updateAppTable(&table)
}

func addAppTables(tables []*axdb.Table, initBackend bool, opportunistic bool) *axdb.AXDBError {
	for _, table := range tables {
		err := addAppTable(table, initBackend, opportunistic)
		if err != nil {
			return err
		}
	}
	return nil
}

// A table may be added multiple times. Create table operation for the same table may hit
// different nodes at the same time. We use the atomic operation on the table_definition table
// to resolve conflicts. We can't use any in-memory data structure for synchronization.
func addAppTable(t *axdb.Table, initBackend bool, opportunistic bool) *axdb.AXDBError {
	table, axErr := initTableInterface(t)
	if axErr != nil {
		return axErr
	}

	tableJsonStr, axErr := jsonMarshal(table)
	if axErr != nil {
		return axErr
	}

	// The table structure has been verified already. Add to table definition table first.
	// If for some reason we crash before we create the table, we will retry once we restart.
	// Adding to backend table is idempotent.
	if initBackend && table.getAppName() != axdb.AXDBAppAXINT {
		v := make(map[string]interface{})
		v[axdb.AXDBKeyColumnName] = table.getFullName()
		v[axdb.AXDBValueColumnName] = tableJsonStr
		v[axdb.AXDBTimeColumnName] = time.Now().UnixNano() / 1e3
		_, axErr = tableDefinitionTable.save(v, true)
		if axErr != nil {
			if axErr.RestStatus != axdb.RestStatusForbidden {
				return axErr
			} else {
				// table exists in table definition table, load the existing definition
				loadDBTable(table.getFullName(), false)
				if opportunistic {
					theDB.getApp(t.AppName).getTable(t.Name).waitForReady(10)
					return nil
				}
			}
		}
	}

	app := theDB.getApp(table.getAppName())
	table = app.addTableToMem(table, false)

	if !initBackend {
		infoLog.Printf("Adding table %s.%s to DB, skip initBackend", app.name, t.Name)
		return nil
	}

	// need to flag the status column for table_definition table.
	v := make(map[string]interface{})
	v[axdb.AXDBKeyColumnName] = table.getFullName()

	axErr = table.initBackend()
	if axErr != nil {
		errorLog.Printf("Adding table %s.%s to DB, cannot initBackend", app.name, t.Name)
		v[axdb.AXDBStatusColumnName] = false
		if table.getAppName() != axdb.AXDBAppAXINT {
			_, axErr1 := tableDefinitionTable.save(v, false)
			if axErr1 != nil {
				errorLog.Printf("Update status column for table %s.%s failed.", app.name, t.Name)
			}
		}
		return axErr
	}

	v[axdb.AXDBStatusColumnName] = true
	if table.getAppName() != axdb.AXDBAppAXINT {
		_, axErr = tableDefinitionTable.save(v, false)
		if axErr != nil {
			errorLog.Printf("Update status column for table %s.%s failed.", app.name, t.Name)
			return axErr
		}
	}
	infoLog.Printf("Added table %s.%s to DB", app.name, t.Name)
	return nil
}

/*
 *  this func is used to check if the name of the table to be update matches the name of old table,
 *  and if the appName initialized the update matches the original appName to create the old table
 */
func getExistingTable(table TableInterface) (*axdb.Table, *axdb.AXDBError) {
	// load the schema of the table to be updated from definition_table
	params := make(map[string]interface{})
	tableName := table.getFullName()
	appName := table.getAppName()
	if len(tableName) == 0 {
		errStr := "Name of the table to be updated is empty"
		errorLog.Printf(errStr)
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}
	params[axdb.AXDBKeyColumnName] = tableName
	resultArray, axErr := tableDefinitionTable.get(params)
	if axErr != nil {
		errStr := fmt.Sprintf("Failed on query metastore for (%s/%s)", appName, tableName)
		errorLog.Printf(errStr)
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}

	if len(resultArray) == 0 {
		infoStr := fmt.Sprintf("The table %s/%s doesn't exist, will be initialized.", appName, tableName)
		infoLog.Printf(infoStr)
		return nil, nil
	} else if len(resultArray) > 1 {
		// this error shouldn't be expected to touch since the name of the table is primary key
		errStr := fmt.Sprintf("There should be at most one record in the definition_table for table %s/%s",
			appName, tableName)
		errorLog.Printf(errStr)
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	} else {
		if resultArray[0][axdb.AXDBStatusColumnName] == nil {
			infoStr := fmt.Sprintf("The table %s/%s wasn't initialized correctly, will be initialized again.", appName, tableName)
			infoLog.Printf(infoStr)
			return nil, nil
		}
		if resultArray[0][axdb.AXDBStatusColumnName].(bool) != true {
			// We found an existing table but it's status was false, which means previous attempt to upgrade failed
			// We allow upgrade to try again, but log a warning
			errorLog.Printf("Previous initialization/upgrade of table %s/%s failed. Will attempt upgrade again", appName, tableName)
		}
	}

	// now we have the schema of old definition
	oldTable, axErr := getAppTableDefFromJsonByte([]byte(resultArray[0][axdb.AXDBValueColumnName].(string)))
	if axErr != nil {
		return nil, axErr
	}

	// check if the appName matches; return error if not match
	if strings.Compare(appName, oldTable.AppName) != 0 {
		errStr := fmt.Sprintf("AppName doesn't match: oldAppName=%s, updateAppName=%s", oldTable.AppName, appName)
		errorLog.Printf(errStr)
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}

	// check if the table types are the same
	if table.getTableType() != oldTable.Type {
		errStr := fmt.Sprintf("Table Type doesn't match: oldType=%d, newType=%d", oldTable.Type, table.getTableType())
		errorLog.Printf(errStr)
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}

	return &oldTable, nil
}

func initTableInterface(t *axdb.Table) (TableInterface, *axdb.AXDBError) {
	var table TableInterface
	if t.Type == axdb.TableTypeTimeSeries {
		tb := &TimeSeriesTable{Table{*t, nil, "", nil, nil, 0, 0}, nil, nil, false, false, false}
		tb.real = tb
		table = tb
	} else if t.Type == axdb.TableTypeKeyValue {
		tb := &KeyValueTable{Table{*t, nil, "", nil, nil, 0, 0}}
		tb.real = tb
		table = tb
	} else if t.Type == axdb.TableTypeTimedKeyValue {
		tb := &TimedKeyValueTable{Table{*t, nil, "", nil, nil, 0, 0}, nil}
		tb.real = tb
		table = tb
	} else if t.Type == axdb.TableTypeCounter {
		tb := &CounterTable{Table{*t, nil, "", nil, nil, 0, 0}}
		tb.real = tb
		table = tb
	} else {
		errStr := fmt.Sprintf("Table type %d not found for table %s/%s", t.Type, t.AppName, t.Name)
		errorLog.Printf(errStr)
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}

	axErr := table.init()
	if axErr != nil {
		return nil, axErr
	}
	return table, nil
}

func getChangedLuceneIndex(old, new *axdb.Table) int {
	if old.UseSearch == false && new.UseSearch == false {
		return UpdateLuceneIndexNoChange
	}

	if old.UseSearch == false && new.UseSearch == true {
		return UpdateLuceneIndexReCreate
	}

	if old.UseSearch == true && new.UseSearch == false {
		return UpdateLuceneIndexDrop
	}

	if len(old.Columns) != len(new.Columns) {
		return UpdateLuceneIndexReCreate
	}
	//if the column definitions are different in new and old tables;  need to recreate
	for col := range old.Columns {
		if _, exist := new.Columns[col]; !exist {
			return UpdateLuceneIndexReCreate
		}
	}

	// compare the excluded columns for lucene index
	if old.ExcludedIndexColumns == nil && new.ExcludedIndexColumns != nil || old.ExcludedIndexColumns != nil && new.ExcludedIndexColumns == nil {
		return UpdateLuceneIndexReCreate
	}
	for col := range old.ExcludedIndexColumns {
		if _, exist := new.ExcludedIndexColumns[col]; !exist {
			return UpdateLuceneIndexReCreate
		}
	}

	for col := range new.ExcludedIndexColumns {
		if _, exist := old.ExcludedIndexColumns[col]; !exist {
			return UpdateLuceneIndexReCreate
		}
	}

	return UpdateLuceneIndexNoChange
}

func updateAppTable(t *axdb.Table) *axdb.AXDBError {
	//first, we initialize a table interface
	table, axErr := initTableInterface(t)
	if axErr != nil {
		return axErr
	}

	// check if the use input table name and appName are valid
	oldTable, axErr := getExistingTable(table)
	if axErr != nil {
		return axErr
	}

	// if the table doesn't exist, just create it.
	if oldTable == nil {
		return addAppTable(t, true, false)
	}

	// we only compare non-system generated columns, the system generated columns are excluded
	// step 1: to compare the columns between old and new definitions
	changedCols := make(map[string]axdb.Column)
	changedFlags := make(map[string]int)
	changedLuceneIndex := getChangedLuceneIndex(oldTable, t)
	for k, col := range t.Columns {
		_, ok := oldTable.Columns[k]
		// the column in the new table isn't in old table
		if !ok {
			// although it's a new column, we need to make sure the index type is correctly defined
			if col.Index == axdb.ColumnIndexPartition || col.Index == axdb.ColumnIndexClustering || col.Index == axdb.ColumnIndexClusteringStrong {
				errStr := fmt.Sprintf("Index type (%s) of the new column isn't allowed",
					axdbColumnIndexTypeNames[col.Index])
				errorLog.Printf(errStr)
				return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
			}
			// if the new column isn't a counter type, it's not allowed to insert into counter table
			if t.Type == axdb.TableTypeCounter {
				if col.Type != axdb.ColumnTypeCounter {
					errStr := fmt.Sprintf("Non counter column(%s) isn't allowed to add to a counter table", k)
					errorLog.Printf(errStr)
					return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
				if col.Index != axdb.ColumnIndexNone && col.Index != axdb.ColumnIndexWeak {
					errStr := fmt.Sprintf("Index type (%s) of the new column isn't allowed on counter type column",
						axdbColumnIndexTypeNames[col.Index])
					errorLog.Printf(errStr)
					return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
			} else {
				if col.Type == axdb.ColumnTypeCounter {
					errStr := fmt.Sprintf("Counter type column(%s) isn't allowed to add to a non-counter table", k)
					errorLog.Printf(errStr)
					return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
			}
			// a new column to be added
			changedCols[k] = col
			// a new column to add
			// here, axdb.ColumnIndexClusteringStrong isn't expected
			if col.Index == axdb.ColumnIndexStrong {
				changedFlags[k] = UpdateAddNewColumnWithSecondaryIndex
			} else {
				changedFlags[k] = UpdateAddNewColumn
			}
		} else {
			// if the types in both definitions are different
			if oldTable.Columns[k].Type != t.Columns[k].Type {
				errStr := fmt.Sprintf("DataType of column(%s) is different, data type change isn't supported.", k)
				errorLog.Printf(errStr)
				return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
			}

			// the column in both definitions
			oldIndex := oldTable.Columns[k].Index
			newIndex := t.Columns[k].Index

			// if the index types are the same; do nothing unless it's a map type of column
			if oldIndex == newIndex {
				if t.Columns[k].Type == axdb.ColumnTypeMap && (newIndex == axdb.ColumnIndexStrong || newIndex == axdb.ColumnIndexClusteringStrong) {
					if oldTable.Columns[k].IndexFlagForMapColumn != t.Columns[k].IndexFlagForMapColumn {
						changedCols[k] = col
						changedFlags[k] = UpdateReCreateSecondaryIndex
					}
				}
				continue

			}

			switch oldIndex {
			// If the old index type is partition index; update operation isn't allowed
			case axdb.ColumnIndexPartition:
				errStr := fmt.Sprintf("Index type isn't compatible to upgrade: old type (%s), new type (%s)",
					axdbColumnIndexTypeNames[oldIndex], axdbColumnIndexTypeNames[newIndex])
				errorLog.Printf(errStr)
				return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
			case axdb.ColumnIndexClustering:
				switch newIndex {
				case axdb.ColumnIndexClusteringStrong:
					changedCols[k] = col
					changedFlags[k] = UpdateAddSecondaryIndex
				default:
					errStr := fmt.Sprintf("Index type isn't compatible to upgrade: old type (%s), new type (%s)",
						axdbColumnIndexTypeNames[oldIndex], axdbColumnIndexTypeNames[newIndex])
					errorLog.Printf(errStr)
					return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
			case axdb.ColumnIndexStrong:
				switch newIndex {
				case axdb.ColumnIndexNone, axdb.ColumnIndexWeak:
					changedCols[k] = col
					// drop an existing secondary index
					changedFlags[k] = UpdateDropSecondaryIndex
				default:
					errStr := fmt.Sprintf("Index type isn't compatible to upgrade: old type (%s), new type (%s)",
						axdbColumnIndexTypeNames[oldIndex], axdbColumnIndexTypeNames[newIndex])
					errorLog.Printf(errStr)
					return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
			case axdb.ColumnIndexClusteringStrong:
				switch newIndex {
				case axdb.ColumnIndexClustering:
					changedCols[k] = col
					changedFlags[k] = UpdateDropSecondaryIndex
				default:
					errStr := fmt.Sprintf("Index type isn't compatible to upgrade: old type (%s), new type (%s)",
						axdbColumnIndexTypeNames[oldIndex], axdbColumnIndexTypeNames[newIndex])
					errorLog.Printf(errStr)
					return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
			case axdb.ColumnIndexNone:
				switch newIndex {
				case axdb.ColumnIndexStrong:
					changedCols[k] = col
					// add a secondary index on an existing column
					changedFlags[k] = UpdateAddSecondaryIndex
				case axdb.ColumnIndexPartition, axdb.ColumnIndexClustering, axdb.ColumnIndexClusteringStrong:
					errStr := fmt.Sprintf("Index type isn't compatible to upgrade: old type (%s), new type (%s)",
						axdbColumnIndexTypeNames[oldIndex], axdbColumnIndexTypeNames[newIndex])
					errorLog.Printf(errStr)
					return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				}
			}
		}
	}

	for k, col := range oldTable.Columns {
		// the column in the old table isn't in new table
		if _, ok := t.Columns[k]; !ok {
			//although it's an old column to remove, but we need to make sure its index type allows the removal
			if col.Index == axdb.ColumnIndexPartition || col.Index == axdb.ColumnIndexClustering || col.Index == axdb.ColumnIndexClusteringStrong {
				errStr := fmt.Sprintf("The column (%s) isn't allowed to drop due to the index type (%s)",
					k, axdbColumnIndexTypeNames[col.Index])
				errorLog.Printf(errStr)
				return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
			}
			changedCols[k] = col
			// an old column to drop
			changedFlags[k] = UpdateDropOldColumn
		}
	}

	// check if it's required to update configuration of the table
	// here we only consider TTL for simplicity;
	// TODO: deal with other configurations if required
	oldConfigs := oldTable.Configs
	newConfigs := t.Configs
	changedConfigs := getConfigsDiff(oldConfigs, newConfigs)

	// check if the stat table definition changed
	changedStatCols := getStatsDefDiff(oldTable.Stats, t.Stats)
	// if there is no change at all, return immediately
	if len(changedCols) == 0 && len(changedConfigs) == 0 && len(changedStatCols) == 0 && changedLuceneIndex == UpdateLuceneIndexNoChange {
		return nil
	}
	changedData := UpdateData{changedCols, changedFlags, changedConfigs, changedStatCols, changedLuceneIndex}

	//prepare DB schema update
	app := theDB.getApp(table.getAppName())
	err := app.updateMemTableStatus(t.Name, TableUpdateInProcess, false)
	if err != nil {
		return err
	}

	// Mark table status to be false before we attempt update. Will mark as true upon success
	if table.getAppName() != axdb.AXDBAppAXINT {
		v := make(map[string]interface{})
		v[axdb.AXDBKeyColumnName] = table.getFullName()
		v[axdb.AXDBStatusColumnName] = false
		_, axErr = tableDefinitionTable.save(v, false)
		if axErr != nil {
			errorLog.Printf("Failed to init table definition status of %s.%s to DB: %v", app.name, t.Name, axErr)
			app.updateMemTableStatus(t.Name, TableUpdateFailed, true)
			return axErr
		}
	}

	// Attempt the actual update of the definition table for the new table schema.
	retryCount := 0
	for {
		retryCount++
		axErr = table.updateBackend(changedData)
		// it's also OK if axErr return Cassandra 2200 code; This has been processed in execQuery()
		if axErr == nil {
			break
		} else {
			// retry for 2 minutes
			if retryCount > 120 {
				errorLog.Printf("axdb update failed, exited")
				app.updateMemTableStatus(t.Name, TableUpdateFailed, true)
				os.Exit(1)
			}
		}
		time.Sleep(1 * time.Second)
	}
	infoLog.Printf("Updated table %s.%s to DB", app.name, t.Name)

	// update in-memory structure
	table = app.addTableToMem(table, true)
	app.updateMemTableStatus(t.Name, TableAccessAvailable, false)

	// Table update was successful. Mark table status as true, and commit the updated table definition
	if table.getAppName() != axdb.AXDBAppAXINT {
		v := make(map[string]interface{})
		v[axdb.AXDBKeyColumnName] = table.getFullName()
		v[axdb.AXDBStatusColumnName] = true
		tableJSONStr, axErr := jsonMarshal(table)
		if axErr != nil {
			app.updateMemTableStatus(t.Name, TableUpdateFailed, true)
			return axErr
		}
		infoLog.Printf(fmt.Sprintf("*** update table definition statement =  %s", tableJSONStr))
		v[axdb.AXDBValueColumnName] = tableJSONStr
		_, axErr = tableDefinitionTable.save(v, false)
		if axErr != nil {
			errorLog.Printf("Update status column for table %s.%s failed: %v", app.name, t.Name, axErr)
			return axErr
		}
	}

	// We only perform schema updates on axdb-0. Now send schema change notification to other nodes via kafka (if applicable)
	if theDB.replFactor > 1 {
		produceMsg := &sarama.ProducerMessage{Topic: KafkaTopic, Key: sarama.StringEncoder(table.getFullName()),
			Value: sarama.StringEncoder("update")}
		retryCount = 0
		for {
			retryCount++
			if _, _, err := producer.SendMessage(produceMsg); err != nil {
				if retryCount > 120 {
					errorLog.Printf(fmt.Sprintf("Failed to send schema change event:%v\n", err))
					break
				} else {
					time.Sleep(1 * time.Second)
				}

			} else {
				break
			}
		}
	}
	return nil
}

// the parameter force means a system error happens, we need to force to change the status
func (app *App) updateMemTableStatus(tblName string, status int, force bool) *axdb.AXDBError {
	app.mutex.Lock()
	defer app.mutex.Unlock()
	flag, exist := app.flags[tblName]
	if !exist {
		errStr := fmt.Sprintf("The table (%s) doesn't exist, can't update in-memory table structure.", tblName)
		errorLog.Printf(errStr)
		return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}
	if flag == TableUpdateInProcess && !force {
		errStr := fmt.Sprintf("The table (%s) update is in-process, access is denied.", tblName)
		errorLog.Printf(errStr)
		return axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
	}
	app.flags[tblName] = status
	return nil
}

func (app *App) getMemTableStatus(tblName string) (int, *axdb.AXDBError) {
	//trim the view suffix to get the real table name
	for {
		if strings.HasSuffix(tblName, axdb.AXDBTimeViewSuffix) {
			tblName = strings.TrimSuffix(tblName, axdb.AXDBTimeViewSuffix)
		} else if strings.HasSuffix(tblName, axdb.AXDBStatSuffix) {
			tblName = strings.TrimSuffix(tblName, axdb.AXDBStatSuffix)
		} else {
			break
		}
	}
	// here I still use lock to prevent reader from reading the table status in the middle of status change
	app.mutex.Lock()
	defer app.mutex.Unlock()
	st, exist := app.flags[tblName]
	if !exist {
		errStr := fmt.Sprintf("The table (%s) doesn't exist, can't update in-memory table structure.", tblName)
		errorLog.Printf(errStr)
		// the table isn't found
		return -1, axdb.NewAXDBError(axdb.RestStatusNotFound, nil, errStr)
	}
	return st, nil
}

func (app *App) addTableToMem(table TableInterface, overwrite bool) TableInterface {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	existing := app.tables[table.getName()]
	if existing != nil && !overwrite {
		return existing
	}

	if table.getAppName() != app.name {
		panic("AppName not match")
	}

	app.tables[table.getName()] = table
	app.flags[table.getName()] = TableAccessAvailable
	return table
}

func (app *App) deleteTable(tableName string) (resMap map[string]interface{}, axErr *axdb.AXDBError) {
	app.mutex.Lock()

	if app.tables[tableName] == nil {
		app.mutex.Unlock()
		loadDBTable(GetTableFullName(app.name, tableName), false)
		app.mutex.Lock()
	}

	existing := app.tables[tableName]
	if existing == nil {
		errStr := fmt.Sprintf("deleting a table that doesn't exist: %s.%s", app.name, tableName)
		warningLog.Printf(errStr)
		axErr = axdb.NewAXDBError(axdb.RestStatusNotFound, nil, errStr)
	} else {
		delete(app.tables, tableName)
	}
	app.mutex.Unlock()

	if existing != nil {
		axErr = existing.deleteBackend()
	}
	return nil, axErr
}

func (app *App) getTable(tableName string) TableInterface {
	app.mutex.Lock()
	table := app.tables[tableName]
	app.mutex.Unlock()

	if table == nil {
		loadDBTable(GetTableFullName(app.name, tableName), false)
		app.mutex.Lock()
		table = app.tables[tableName]
		app.mutex.Unlock()
	}
	return table
}

func createApp(name string) *App {
	newapp := App{sync.Mutex{}, name, nil, nil}
	newapp.tables = make(map[string]TableInterface)
	newapp.flags = make(map[string]int)
	return &newapp
}

// represent the backend DB
type DB struct {
	// the default consistency level of cluster is quorum, we don't need to change.
	cluster    *gocql.ClusterConfig // cassandra cluster
	session    *gocql.Session
	apps       map[string]*App // maps app name to App struct
	mutex      sync.Mutex
	numNodes   int64 // number of nodes in the cluster
	replFactor int64 // the replication factor
}

func getClusterID() string {
	var id string = ""
	cmd := exec.Command("nodetool", "info")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errorLog.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		errorLog.Fatal(err)
	}
	defer cmd.Wait()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, ":")
		if strings.Contains(tokens[0], "ID") && len(tokens[1]) > 8 {
			id = strings.Trim(tokens[1], " ")
			break
		}
	}
	return id
}

func (db *DB) backendIsRunning() bool {
	// check if the DB has been started already
	if getClusterID() == "" {
		return false
	}

	return true
}

func (db *DB) clusterIsAllReady() bool {
	// check if the cassandra cluster is in ready status
	infoLog.Printf("CLUSTER STATUS: db.repl=%d", db.replFactor)
	cmd := exec.Command("nodetool", "status")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errorLog.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		errorLog.Fatal(err)
	}
	defer cmd.Wait()

	scanner := bufio.NewScanner(stdout)
	var upNodes int64 = 0
	for scanner.Scan() {
		line := scanner.Text()
		infoLog.Printf("NODETOOL: %s", line)
		if strings.HasPrefix(line, "UN") {
			upNodes++
			//tokens := strings.Split(line, " ")
		}
	}
	infoLog.Printf(fmt.Sprintf("total number of nodes (%d), number of Up nodes (%d)", db.numNodes, upNodes))
	if upNodes == db.numNodes {
		return true
	} else {
		return false
	}
}

func (db *DB) clusterIsReady() bool {
	// check if the cassandra cluster is in ready status
	infoLog.Printf("CLUSTER STATUS: db.repl=%d", db.replFactor)
	cmd := exec.Command("nodetool", "status")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errorLog.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		errorLog.Fatal(err)
	}
	defer cmd.Wait()

	scanner := bufio.NewScanner(stdout)
	var upNodes int64 = 0
	for scanner.Scan() {
		line := scanner.Text()
		infoLog.Printf("NODETOOL: %s", line)
		if strings.HasPrefix(line, "UN") {
			upNodes++
		}
	}

	//given replication factor N, the cluster is still available with ceil(N/2) - 1 nodes down
	var allowedDown int64
	if db.replFactor == CassandraReplRedundancy {
		allowedDown = 0
	} else {
		allowedDown = int64(math.Ceil(float64(db.replFactor)/2)) - 1
	}

	infoLog.Printf(fmt.Sprintf("total number of nodes (%d), number of Up nodes (%d), allowedDown nodes (%d)", db.numNodes, upNodes, allowedDown))
	if upNodes >= db.numNodes-allowedDown {
		return true
	} else {
		return false
	}
}

func (db *DB) startBackend() {
	// start the backend DB
	infoLog.Println("starting cassandra")
	cmd := exec.Command("/usr/sbin/cassandra", "-R")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		errorLog.Panicln(err)
		panic("Can't start cassandra")
	}

	// connect to cassandra system keyspace, to check if we can access.
	db.cluster.Keyspace = CassandraSysKeyspace
	for true {
		session, err := db.cluster.CreateSession()
		if err == nil {
			session.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (db *DB) WaitSchemaAgreement() *axdb.AXDBError {
	var agreed bool = false
	startTime := time.Now().UnixNano() / 1e9
	for {
		endTime := time.Now().UnixNano() / 1e9
		if endTime-startTime > 300 {
			return axdb.NewAXDBError(axdb.RestStatusInternalError, nil, "Can't reach schema agreement in 5 minutes")
		}
		infoLog.Printf("[WSA]: waiting schema agreement....")
		// query system.peers table
		iter := db.session.Query("select schema_version from system.peers").Consistency(gocql.One).Iter()
		schemaVersions, err := iter.SliceMap()
		if err != nil || int64(len(schemaVersions)) != db.replFactor-1 {
			infoLog.Printf("[WSA]: invalid query to peers table, retry.")
			time.Sleep(1 * time.Second)
			continue
		}
		expectedVersion := ""
		var idx int = 0
		for i, schemaversion := range schemaVersions {
			infoLog.Printf("[WSA]: peer %d schema version: %v", i, schemaversion)
			currentVersion := fmt.Sprintf("%v", schemaversion["schema_version"])
			if i == 0 {
				expectedVersion = currentVersion
			} else {
				if currentVersion != expectedVersion {
					infoLog.Printf("[WSA]: schema mismatch found in peers, retry.")
					break
				}
			}
			idx = i
		}
		if idx+1 == len(schemaVersions) {
			infoLog.Printf("[WSA]: peer schemas matched, will expore local schema next.")
			agreed = true
		}
		if !agreed {
			time.Sleep(1 * time.Second)
			continue
		} else {
			// match schema_version in system.local table
			var localSchema map[string]interface{}
			for {
				iter := db.session.Query("select schema_version from system.local").Consistency(gocql.One).Iter()
				ret, err := iter.SliceMap()
				if err != nil || len(ret) != 1 {
					infoLog.Printf("[WSA]: invalid query to local table, retry.")
					continue
				} else {
					infoLog.Printf("[WSA]: local schema version: %v", ret[0])
					localSchema = ret[0]
					break
				}
			}
			localVersion := fmt.Sprintf("%v", localSchema["schema_version"])
			if localVersion == expectedVersion {
				infoLog.Printf("[WSA]: schema agreement reached, done!")
				return nil
			}
			infoLog.Printf("[WSA]: local doesn't match peers, retry.")
		}
	}
}

// init backend DB
func (db *DB) initDB(numNodes int64) {

	db.apps = make(map[string]*App)
	db.cluster = gocql.NewCluster("127.0.0.1")
	db.cluster.Timeout = 60 * time.Second
	db.cluster.ProtoVersion = CassandraProtoVersion
	// query will be retried 30 times in case of failures
	//db.cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 30}
	db.numNodes = numNodes

	var mode string
	var rc int64
	if numNodes >= 3 {
		mode = CassandraClusterMode
		rc = CassandraClusterReplRedundancy
	} else {
		mode = CassandraStandloneMode
		rc = CassandraReplRedundancy
	}
	infoLog.Printf(fmt.Sprintf("AXDB Mode: %s", mode))
	db.replFactor = rc
	infoLog.Printf(fmt.Sprintf("REPLICATION=%d, rc = %d", db.replFactor, rc))
	if !db.backendIsRunning() {
		db.startBackend()
	}
	/*
		for {
			if db.clusterIsAllReady() {
				break
			} else {
				time.Sleep(10 * time.Second)
			}
		}
	*/

	db.cluster.Keyspace = CassandraAXKeyspace
	session, err := db.cluster.CreateSession()
	if err == nil {
		db.session = session
		return
	}

	db.cluster.Keyspace = CassandraSysKeyspace
	session, err = db.cluster.CreateSession()
	if err != nil {
		warningLog.Printf("create system session failed")
		return
	}
	db.session = session

	execQuery(fmt.Sprintf(`CREATE KEYSPACE IF NOT EXISTS %s
		WITH replication = {
			'class' : '%s',
			'replication_factor' : %d
		}`, CassandraAXKeyspace, CassandraReplStrategy, rc))

	session.Close()
	db.session = nil
	db.cluster.Keyspace = CassandraAXKeyspace
	session, err = db.cluster.CreateSession()
	if err != nil {
		warningLog.Printf("create axdb session failed")
		return
	}
	db.session = session
}

func (db *DB) getApp(appName string) *App {
	db.mutex.Lock()

	app := db.apps[appName]
	if app == nil {
		infoLog.Printf("DB adding %s app", appName)
		db.apps[appName] = createApp(appName)
		app = db.apps[appName]
	}

	db.mutex.Unlock()

	return app
}

func (db *DB) SwitchProfileStatus(status string, isCoordinator bool) {
	var b bool
	if status == "on" {
		b = true
	} else {
		b = false
	}

	// only when status != profileSwitchStatus, we do a switch
	if b != profileSwitchStatus {
		switchMutex.Lock()
		defer switchMutex.Unlock()
		profileSwitchStatus = b
	}

	// if it is coordinator, use kafka to populate the switch value to other nodes;
	if isCoordinator && theDB.replFactor > 1 {
		produceMsg := &sarama.ProducerMessage{Topic: KafkaTopic, Key: sarama.StringEncoder("profile_switch"),
			Value: sarama.StringEncoder(status)}
		retryCount := 0
		for {
			retryCount++
			if _, _, err := producer.SendMessage(produceMsg); err != nil {
				if retryCount > 120 {
					errorLog.Printf(fmt.Sprintf("Failed to send profile_switch message:%v\n", err))
					break
				} else {
					time.Sleep(1 * time.Second)
				}

			} else {
				break
			}
		}
	}
}

var theDB DB

// internal tables. Used frequently and static, no point to go through maps to get to them. Cache here.
var tableDefinitionTable TableInterface
var objectCounterTable TableInterface
var systemInfoTable TableInterface
var profileTable TableInterface
var producer sarama.SyncProducer
var consumer cluster.Consumer
var hostname string

// nodeID in cassandra cluster
var nodeID string

// the id address of axdb-0
var leaderIp string

// the profile switch; by default it's off
var profileSwitchStatus bool = false
var switchMutex sync.Mutex

func isInternalTablesExists(tables []*axdb.Table) bool {
	for _, table := range tables {
		if exist := isTableExists(table); !exist {
			return false
		}
	}
	return true
}

func isTableExists(table *axdb.Table) bool {
	if table.AppName == axdb.AXDBAppAXINT {
		axErr := execQuery(fmt.Sprintf("SELECT * FROM %s_%s LIMIT 1", table.AppName, table.Name))
		if axErr != nil {
			return false
		} else {
			return true
		}
	} else {
		tblName := fmt.Sprintf("%s_%s", table.AppName, table.Name)
		params := map[string]interface{}{
			axdb.AXDBKeyColumnName: tblName,
		}
		resultArray, err := tableDefinitionTable.get(params)
		if err != nil || len(resultArray) == 0 {
			return false
		} else {
			return true
		}
	}
}

//this is applicable only in a cluster, we use kafka to sync the states of axdb instances
func InitMessageClients() {
	if theDB.replFactor == 1 {
		return
	}
	var err error
	producer, err = sarama.NewSyncProducer([]string{KafkaAddress}, nil)
	if err != nil {
		//here, we fail axdb
		errorLog.Printf(fmt.Sprintf("failed to create a kafka producer for axdb, Err: %v", err))
		os.Exit(1)
	}

	// start the consumer long-running loop
	go func() {
		config := cluster.NewConfig()
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
		client, err := cluster.NewClient([]string{KafkaAddress}, config)
		if err != nil {
			errorLog.Printf(fmt.Sprintf("failed to create a kafka consumer client, Err: %v", err))
			os.Exit(1)
		}
		consumer, err := cluster.NewConsumerFromClient(client, hostname, []string{KafkaTopic})
		if err != nil {
			errorLog.Printf(fmt.Sprintf("failed to create a kafka consumer, Err: %v", err))
			os.Exit(1)
		}
		infoLog.Printf("consumer group is created successfully!")

		defer func() {
			if err := consumer.Close(); err != nil {
				errorLog.Printf(fmt.Sprintf("failed to close the consumer connection, Err: %v", err))
			}
		}()

		for {
			select {
			case msg := <-consumer.Messages():
				infoLog.Printf(fmt.Sprintf("AXDB Consumer %s Received message with topic = %s, key = %s, partition = %d, offset = %d", hostname, msg.Topic, msg.Key, msg.Partition, msg.Offset))
				val := string(msg.Value[:])
				key := string(msg.Key[:])
				// we only deal with schema update event
				if val == "update" {
					params := map[string]interface{}{
						axdb.AXDBKeyColumnName: key,
					}
					resultArray, err := tableDefinitionTable.get(params)
					if err != nil || len(resultArray) != 1 {
						errorLog.Printf("The table isn't well defined")
					}

					table, err := getAppTableDefFromJsonByte([]byte(resultArray[0][axdb.AXDBValueColumnName].(string)))
					if err != nil {
						errorLog.Printf("The table isn't well defined as a Json format")
					}
					t, err := initTableInterface(&table)
					if err != nil {
						errorLog.Printf("Failed to init a table interface")
					}
					infoLog.Printf("before reloading, the definition in memory: %v", theDB.getApp(t.getAppName()).getTable(table.Name))
					theDB.getApp(t.getAppName()).addTableToMem(t, true)
					infoLog.Printf("after reloading, the definition in memory: %v", theDB.getApp(t.getAppName()).getTable(table.Name))
					// need to commit the offset in order to avoid consume the same message multiple time.
				} else if key == "profile_switch" {
					theDB.SwitchProfileStatus(val, false)
				}
				infoLog.Printf(fmt.Sprintf("AXDB Consumer %s prepare to commit the offset = %d for topic %s, partition = %d", hostname, msg.Offset, msg.Topic, msg.Partition))
				consumer.MarkOffset(msg, "axdbmetadata")
				consumer.CommitOffsets()
				infoLog.Printf(fmt.Sprintf("AXDB Consumer %s finished committing the offset for topic %s, partition = %d", hostname, msg.Topic, msg.Partition))
			}
		}
	}()
}

func initDBTables(tables []*axdb.Table) {
	if theDB.replFactor > 1 {
		if isLeaderNode() {
			if !isInternalTablesExists(tables) {
				addAppTables(tables, true, false)
			} else {
				addAppTables(tables, false, false)
			}
		} else {
			for {
				if !isInternalTablesExists(tables) {
					infoLog.Printf("Internal tables don't exist, waiting for axdb-0 to create them....")
					time.Sleep(1 * time.Second)
				} else {
					break
				}
			}
			addAppTables(tables, false, false)
		}
	} else {
		addAppTables(tables, true, false)
	}
}

func InitDB(numNodes int64) {
	theDB.initDB(numNodes)
	initDBTables(internalTables)
	tableDefinitionTable = theDB.getApp(axdb.AXDBAppAXINT).tables[axdb.AXDBTableDefinitionTable]
	objectCounterTable = theDB.getApp(axdb.AXDBAppAXINT).tables[axdb.AXDBObjectCounterTable]
	systemInfoTable = theDB.getApp(axdb.AXDBAppAXINT).tables[axdb.AXDBSystemInfoTable]

	if theDB.replFactor > 1 && isLeaderNode() {
		saveLeaderNodeIp()
	}

	// compare the version in table_definition with current version
	// only axdb-0 is allowed to do this operation
	if isLeaderNode() {
		metaRow, op := VersionCheck()
		if op == axdb.StatsTableUpdateAllowed {
			upgradeDBTables(metaRow)
		} else if op == axdb.NoOperationAllowed {
			panic("It's not allowed to run an older DB or on older tables")
		}
	}

	initDBTables(userTables)
	profileTable = theDB.getApp(axdb.AXDBPerf).tables[axdb.AXDBPerfTable]

	go ProfilerWorker()
}

func saveLeaderNodeIp() {
	nodeID = getClusterID()
	infoLog.Printf("my ID: %s", nodeID)
	cmd := exec.Command("nodetool", "status")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errorLog.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		errorLog.Fatal(err)
	}
	defer cmd.Wait()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "UN") && strings.Contains(line, nodeID) {
			tokens := strings.Fields(line)
			if len(tokens) > 1 {
				leaderIp = tokens[1]
				infoLog.Printf("leader IP= %s", leaderIp)
			}
			break
		}
	}
	if leaderIp != "" {
		retryCount := 0
		for {
			data := map[string]interface{}{
				axdb.AXDBKeyColumnName:   axdb.AXDBLeaderIPName,
				axdb.AXDBValueColumnName: leaderIp,
			}
			retryCount++
			_, err := systemInfoTable.save(data, false)
			if err != nil {
				if retryCount >= 30 {
					panic("Failed to insert axdb-0 IP address to table.")
				}
				time.Sleep(1 * time.Second)
			} else {
				infoLog.Printf("leader IP saved successfully!")
				break
			}
		}
	}
}

func runNodetoolCmd(cmd string) {
	nodetoolCmd := exec.Command("nodetool", cmd)
	err := nodetoolCmd.Run()
	if err != nil {
		errorLog.Printf("Got an error when running nodetool %s command, Err: %v", cmd, err)
	}
}

func loadLeaderNodeIp() {
	params := make(map[string]interface{})
	params[axdb.AXDBKeyColumnName] = axdb.AXDBLeaderIPName

	retryCount := 0
	var resultArray []map[string]interface{}
	var err *axdb.AXDBError
	for {

		resultArray, err = systemInfoTable.get(params)
		if err != nil {
			if retryCount < 60 {
				time.Sleep(250 * time.Millisecond)
				retryCount++
				continue
			} else {
				panic("Failed to retrieve IP of leader node.")
			}
		} else if len(resultArray) == 0 {
			//this means axdb-0 hasn't finished write its ip to the table, keep retry.
			infoLog.Printf("No leader IP got, waiting...")
			time.Sleep(250 * time.Millisecond)
			continue
		} else {
			break
		}
	}

	for _, result := range resultArray {
		if result[axdb.AXDBValueColumnName] != nil && result[axdb.AXDBValueColumnName].(string) != "" {
			leaderIp = result[axdb.AXDBValueColumnName].(string)
			infoLog.Printf("Got leaderIp: %s", leaderIp)
		}
	}
}

func VersionCheck() (map[string]interface{}, int) {
	metaRow, err := getMetaVersion()
	if err != nil {
		return nil, axdb.OriginalTableUpdateAllowed
	}
	if metaRow == nil {
		return nil, axdb.StatsTableUpdateAllowed
	} else {
		metadata, err := getDBMetaDataFromJsonByte([]byte(metaRow[axdb.AXDBValueColumnName].(string)))
		if err != nil {
			panic("not a valid meta data")
		}
		// if version in meta table is empty, we think it's the oldest version
		if metadata.Version == "" {
			return metaRow, axdb.StatsTableUpdateAllowed
		} else {
			strArr := strings.Split(metadata.Version, "_")
			curDBVersion := axdb.AXDBVersion
			curTableVersion := axdb.AXDBTableVersion
			// TODO: need to deal with the complicated version comparison. i.e. v1.10 vs v1.2.5
			// Here we assume the version will be alphabetical increasing
			if len(strArr) != 2 {
				panic("not a valid version")
			}
			if strings.Compare(curDBVersion, strArr[0]) == 1 ||
				strings.Compare(curDBVersion, strArr[0]) == 0 && strings.Compare(curTableVersion, strArr[1]) == 1 {
				return metaRow, axdb.StatsTableUpdateAllowed
			} else if strings.Compare(curDBVersion, strArr[0]) == -1 ||
				strings.Compare(curDBVersion, strArr[0]) == 0 && strings.Compare(curTableVersion, strArr[1]) == -1 {
				return metaRow, axdb.NoOperationAllowed
			} else {
				return metaRow, axdb.OriginalTableUpdateAllowed
			}
		}
	}
}

func getMetaVersion() (map[string]interface{}, *axdb.AXDBError) {
	params := make(map[string]interface{})
	params[axdb.AXDBKeyColumnName] = axdb.AXDBMetaDataKeyName
	var resultArray []map[string]interface{}
	var err *axdb.AXDBError
	//it's possible that when running the following query, axdb-1 or axdb-2 hasn't been ready yet.
	//as a result, we could get an error; we will try for 5 minutes.
	startTime := time.Now().UnixNano() / 1e9
	for {
		resultArray, err = tableDefinitionTable.get(params)
		if err == nil {
			break
		} else {
			endTime := time.Now().UnixNano() / 1e9
			if endTime-startTime > 300 {
				return nil, axdb.NewAXDBError(axdb.RestStatusInternalError, nil, "failed to query Metadata version.")
			}
			time.Sleep(1 * time.Second)
		}
	}

	if len(resultArray) == 0 {
		return nil, nil
	} else if len(resultArray) != 1 {
		panic("not a valid meta table")
	} else {
		return resultArray[0], nil
	}
}

//used to remove the statTable associated with a timeseries table,
//and upgrade the table definition in the metadata table
func upgradeDBTables(data map[string]interface{}) *axdb.AXDBError {
	params := make(map[string]interface{})
	resultArray, err := tableDefinitionTable.get(params)
	if err == nil {
		for _, tblDef := range resultArray {
			// if the current row is the data meta information; just skip it
			if tblDef[axdb.AXDBKeyColumnName] == axdb.AXDBMetaDataKeyName {
				continue
			}
			//if the table definition is null, don't deal with it.
			if tblDef[axdb.AXDBValueColumnName] == nil || len(tblDef[axdb.AXDBValueColumnName].(string)) == 0 {
				continue
			}
			v := []byte(tblDef[axdb.AXDBValueColumnName].(string))
			table, err1 := getAppTableDefFromJsonByte(v)
			if err1 != nil {
				return err1
			}
			//only consider the timeseries table which has Stats defined.
			if table.Type == axdb.TableTypeTimeSeries && len(table.Stats) > 0 {
				infoLog.Printf("Drop StatTable associated with table %s for DB upgrade.", table.Name)
				if theDB.replFactor > 1 {
					if err := theDB.WaitSchemaAgreement(); err != nil {
						return err
					}
				}
				//first to drop the statTable created on original table
				execQuery(fmt.Sprintf("DROP TABLE %s_%s%s", table.AppName, table.Name, axdb.AXDBStatSuffix))
				if theDB.replFactor > 1 {
					if err := theDB.WaitSchemaAgreement(); err != nil {
						return err
					}
				}
				//it's also possible that the statTable was created on materialized view
				execQuery(fmt.Sprintf("DROP TABLE %s_%s%s%s", table.AppName, table.Name, axdb.AXDBTimeViewSuffix, axdb.AXDBStatSuffix))

				//we don't assign the table.Stats to nil, the change on the statTable schema will be dealt with in update_table call
				//table.Stats = nil

				tableJsonStr, axErr := jsonMarshal(table)
				if axErr != nil {
					return axErr
				}
				tblDef[axdb.AXDBValueColumnName] = tableJsonStr
				// don't need to change the timestamp of the table since it's used to indicate the time when the oldest record was generated.
				//tblDef[axdb.AXDBTimeColumnName] = time.Now().UnixNano() / 1e3
				_, axErr = tableDefinitionTable.save(tblDef, false)
				if axErr != nil {
					infoLog.Printf(fmt.Sprintf("update metatable exception: %v", axErr))
					return axErr

				}
			}
		}
		//finally we need to change the table version in the metadata table
		var metadata axdb.DBMetaData
		var newInsert bool = false
		if data == nil {
			data = make(map[string]interface{})
			metadata = axdb.DBMetaData{}
			data[axdb.AXDBKeyColumnName] = axdb.AXDBMetaDataKeyName
			newInsert = true
		} else {
			metadata, _ = getDBMetaDataFromJsonByte([]byte(data[axdb.AXDBValueColumnName].(string)))
		}

		metadata.Version = fmt.Sprintf("%s_%s", axdb.AXDBVersion, axdb.AXDBTableVersion)
		dataJsonStr, _ := jsonMarshal(metadata)
		data[axdb.AXDBValueColumnName] = dataJsonStr
		data[axdb.AXDBTimeColumnName] = time.Now().UnixNano() / 1e3
		_, axErr := tableDefinitionTable.save(data, newInsert)
		if axErr != nil {
			return axErr
		}

		return nil
	} else {
		panic("not a valid meta table")
	}
}

func MonitorDB() {
	go func() {
		for {
			running := theDB.backendIsRunning()
			if !running {
				// Give up, marathon would restart this container
				fmt.Printf("Cassandra is not responding, exited")
				os.Exit(1)
			}
			time.Sleep(20 * time.Second)
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for {
			select {
			case <-ticker.C:
				infoLog.Printf("Doing nodetool flush by backgroup thread")
				runNodetoolCmd("flush")
			}
		}

	}()
}

func RollUpStats() {
	// isAggregate flag indicates if we rollup from row data or from low granularity stat data
	goRollUpStats := func(intervalInSecond int, isAggregate bool) {
		for {
			interval := int64(intervalInSecond) * 1e6
			// ignore any errors in the process
			// transform the current time to seconds
			beginTicket := time.Now().UnixNano() / 1e3
			// right at the boundary of an interval, sleep a few seconds to pass it
			if beginTicket%interval == 0 {
				time.Sleep(time.Second * 20)
				continue
			}

			var maxTime int64
			maxTime = beginTicket
			//round maxTime to interval boundary
			maxTime = (maxTime - 1) / interval * interval
			infoLog.Printf(fmt.Sprintf("*** rollup is triggered! current time = %d, last interval boundary = %d", beginTicket, maxTime))
			RollUpStatsExecutor(intervalInSecond, maxTime, isAggregate)

			// get time(in seconds) once again to determine the sleep time in order to pass the next interval boundary
			endTicket := time.Now().UnixNano() / 1e3
			//after a rollup, the endTicket passes an interval boundary
			//the next rollup can be triggered right away
			if 1+beginTicket/interval == (endTicket-1)/interval {
				infoLog.Printf(fmt.Sprintf("*** rollup is done!, the next can be triggered right away"))
				continue
			} else {
				// sleep sufficient time to pass the interval boundary
				nextBoundary := (endTicket-1)/interval*interval + interval
				diffNanoSec := (nextBoundary-endTicket)*1e3 + axdb.OneSecond*1e9*20
				infoLog.Printf(fmt.Sprintf("*** rollup is done!, has to sleep for %d(nanoSec) to run next rollup", diffNanoSec))
				time.Sleep(time.Duration(diffNanoSec))
			}
		}
	}

	if isLeaderNode() {
		go goRollUpStats(axdb.AXDBRollUpInterval, false)
		go goRollUpStats(axdb.OneDay, true)
	}
}

// interval is passed in seconds, maxTime is passed in micro-seconds
func RollUpStatsExecutor(interval int, maxTime int64, isAggregate bool) {
	// first, to get the table definition from the in-memory data structure.
	for _, app := range theDB.apps {
		for _, table := range app.tables {
			stats := table.getStatList()
			infoLog.Printf(fmt.Sprintf("*** table (%s) has stats definition %v", table.getFullName(), stats))
			// if the table doesn't have stats defined
			if len(stats) == 0 {
				continue
			}
			//construct the param for stat creation
			params := make(map[string]interface{})
			minTime := maxTime - int64(interval)*1e6
			params[axdb.AXDBIntervalColumnName] = int64(interval)
			params[axdb.AXDBQueryMinTime] = int64(minTime)
			if isAggregate {
				params[axdb.AXDBQuerySrcInterval] = axdb.OneHour
			}
			table.get(params)
		}
	}
}

func loadOneDBTable(tblStr []byte, initBackend bool, c chan int) {
	addAppTableFromJsonByte(tblStr, initBackend, !initBackend)
	c <- 1
}

func loadDBTable(tableName string, initBackend bool) {
	params := make(map[string]interface{})
	if len(tableName) != 0 {
		params[axdb.AXDBKeyColumnName] = tableName
	}
	//we only re-initBacked for tables with status = false; this means the table is partially created before, we need to restore it.
	allResultArray, err := tableDefinitionTable.get(params)
	if err == nil {
		var oldVersion bool = true
		for _, entry := range allResultArray {
			if entry[axdb.AXDBStatusColumnName] != nil {
				oldVersion = false
				break
			}
		}

		// if it's for an existing cluster, just skip the initBackend operations
		var skippedArray []map[string]interface{}
		if oldVersion || !isLeaderNode() {
			skippedArray = allResultArray
		} else {
			var resultArray []map[string]interface{}
			for _, entry := range allResultArray {
				if entry[axdb.AXDBStatusColumnName] == nil || entry[axdb.AXDBStatusColumnName].(bool) != true {
					resultArray = append(resultArray, entry)
				} else {
					skippedArray = append(skippedArray, entry)
				}
			}
			// need to batch
			n := len(resultArray)
			batch := 8
			index := 0
			for index < n {
				actualBatch := 0
				if index+batch >= n {
					actualBatch = n - index
				} else {
					actualBatch = batch
				}

				c := make(chan int, 10*actualBatch)

				for i := 0; i < actualBatch; i++ {
					v := resultArray[i+index]
					b := []byte(v[axdb.AXDBValueColumnName].(string))
					go loadOneDBTable(b, initBackend, c)
				}

				for i := 0; i < actualBatch; i++ {
					<-c
				}
				index += batch
			}

		}
		for _, v := range skippedArray {
			b := []byte(v[axdb.AXDBValueColumnName].(string))
			addAppTableFromJsonByte(b, false, true)
		}

	} else {
		panic("not a valid meta table")
	}
}

func ReloadDBTable() {
	loadDBTable("", true)
}

func GetTableFullName(appName string, tableName string) string {
	return strings.Join([]string{appName, tableName}, "_")
}
