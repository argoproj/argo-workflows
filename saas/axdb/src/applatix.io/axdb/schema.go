// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axdb

const (
	TableTypeTimeSeries    = 0 // time series, every insert will generate a new event
	TableTypeKeyValue      = 1 // key value pair, new insert will replace old values
	TableTypeTimedKeyValue = 2 // key value table, with ax_time column inserted by AXDB and return result will be ordered by time
	TableTypeCounter       = 3 // counter, new insert will increment a counter, which will be returned
)

// the number of different table types defined in AXDB. Now it's 4.
// Whenever we update the above Table Type constants, the number should be updated accordingly
const (
	NumberOfTableTypes = 4
)

// cassandra column data types we currently use
const (
	ColumnTypeString  = 0
	ColumnTypeDouble  = 1
	ColumnTypeInteger = 2
	ColumnTypeBoolean = 3
	ColumnTypeArray   = 4
	ColumnTypeMap     = 5
	//Removed
	//ColumnTypeTimestamp  = 6
	ColumnTypeUUID       = 7
	ColumnTypeOrderedMap = 8
	ColumnTypeTimeUUID   = 9
	ColumnTypeSet        = 10
	ColumnTypeCounter    = 11
)

// lucene index field data type
const (
	LuceneTypeString  = 1
	LuceneTypeDouble  = 2
	LuceneTypeLong    = 3
	LuceneTypeBoolean = 4
)

// the number of different data types defined in AXDB. Now it's 12.
// Whenever we update the above Data Type constants with new value, the number should be updated accordingly.
const (
	NumberOfDataTypes = 12
)

// column index types. For each index type, you can specify multiple columns. Partition indexes and primary
// indexes will be grouped and used together. For instance, if you specify col1, col2 as partition index and
// col3 and col4 as primary index, we will partition using (col1, col2) and support searching for
// ((col1, col2), col3, col4)
//
// 1) KeyValue table, you must set partition index. (partition, clustering) combined together must be unique. If
//    partition is unique by itself, clustering key is not needed.
// 2) ObjectStore table. Partition key can be specified but are not required. AXDB will generate ax_id
// 	  so that (partition, ax_id) are unique. If partition key is not specified ax_id
//    alone will be unique. You shouldn't specify a clustering key. It will be ignored.
// 3) TimeSeries table, partition and clustering indexes are advisory. We will generate a unique id for each
//    event inserted into the table.
// 4) Counter table, you can only specify partition indexes. Having other column types will result in error.
//
const (
	ColumnIndexNone             = 0 // not a key
	ColumnIndexStrong           = 1 // we will do query on this column, and we expect high cardinality
	ColumnIndexWeak             = 2 // we will do query on this column, and we expect low cardinality
	ColumnIndexClustering       = 3 // Use this for clustering index, can specify multiple columns
	ColumnIndexPartition        = 4 // Use this as partition key, can specify multiple columns
	ColumnIndexClusteringStrong = 5 // It's both a clustering key and a secondary key
)

// When a column is of type ColumnTypeMap, and we create ColumnIndexStrong index on it, we can further
// choose if we want to index the keys, the values, or both. Default will be to index values only.
const (
	ColumnIndexMapValues        = 0 // index the map values only
	ColumnIndexMapKeys          = 1 // index the map keys only
	ColumnIndexMapKeysAndValues = 2 // index both the map keys and values
)

const (
	ColumnStatSum     = 1                                 // just keep sum, count is always kept
	ColumnStatPercent = (1 << 1)                          // percentile stat
	ColumnStatAll     = ColumnStatSum | ColumnStatPercent // support both types of stats
)

// AXDB generated column names. All "ax_" column names are reserved.
const (
	AXDBTimeColumnName         = "ax_time"
	AXDBWeekColumnName         = "ax_week"
	AXDBCounterColumnName      = "ax_counter"
	AXDBKeyColumnName          = "ax_key"
	AXDBUUIDColumnName         = "ax_uuid"
	AXDBIntervalColumnName     = "ax_interval"
	AXDBValueColumnName        = "ax_value"
	AXDBStatusColumnName       = "ax_status"
	AXDBSelectColumns          = "ax_select_cols"
	AXDBMetaDataKeyName        = "ax_dbmeta"
	AXDBLeaderIPName           = "ax_leader_ip"
	AXDBQuerySearch            = "ax_search"
	AXDBQueryExactSearch       = "ax_exact_search"
	AXDBConditionalUpdateExist = "ax_update_if_exist"

	// Suffixes for stat columns
	AXDB10ColumnSuffix             = "_10"
	AXDB20ColumnSuffix             = "_20"
	AXDB30ColumnSuffix             = "_30"
	AXDB40ColumnSuffix             = "_40"
	AXDB50ColumnSuffix             = "_50"
	AXDB60ColumnSuffix             = "_60"
	AXDB70ColumnSuffix             = "_70"
	AXDB80ColumnSuffix             = "_80"
	AXDB90ColumnSuffix             = "_90"
	AXDBSumColumnSuffix            = "_sum"
	AXDBCountColumnSuffix          = "_count"
	AXDBStatSuffix                 = "_axst"
	AXDBTimeViewSuffix             = "_axtv"
	AXDBLuceneIndexSuffix          = "_lucene_idx"
	AXDBConditionalUpdateSuffix    = "_update_if"
	AXDBConditionalUpdateNotSuffix = "_update_ifnot"
	AXDBVectorColumnPlusSuffix     = "_col_plus"
	AXDBVectorColumnMinusSuffix    = "_col_minus"
	AXDBMapColumnKeySuffix         = "_contains_key"
)

const (
	AXDBTableDefinitionTable = "table_definition"
	AXDBObjectCounterTable   = "object_counter"
	AXDBSystemInfoTable      = "system_info"
	AXDBPerfTable            = "profile"
)

const (
	AXDBAppAXDB     = "axdb"     // AXDB app, supports version and create_table
	AXDBAppAXSYS    = "axsys"    // sys configuration
	AXDBAppAXDEVOPS = "axdevops" // devops / workflow related
	AXDBAppAXOPS    = "axops"    // devops related
	AXDBAppAXINT    = "axint"    // AXDB internal
	AXDBAppApp      = "axapp"
	AXDBAppAMM      = "axamm"
	AXDBAppAXNC     = "axnc"
	AXDBPerf        = "axdb_perf"
)

const (
	AXDBTableContainerUsage = "container_usage"
	AXDBTableHostUsage      = "host_usage"
	AXDBTableHost           = "host"
	AXDBTableContainer      = "container"
	AXDBTableCommit         = "commit"
	AXDBTableImage          = "image"
)

const (
	AXDBOpVersion     = "version"
	AXDBOpCreateTable = "create_table"
	AXDBOpUpdateTable = "update_table"
)

const (
// AXDBIdBucketSize = 1000 // id partition bucket size
)

// query related hard coded strings
const (
	AXDBQueryMaxTime       = "ax_max_time"
	AXDBQueryMinTime       = "ax_min_time"
	AXDBQueryMaxEntries    = "ax_max_entries"
	AXDBQueryOffsetEntries = "ax_offset_entries"
	AXDBQueryOrderByASC    = "ax_orderby_asc"
	AXDBQueryOrderByDESC   = "ax_orderby_desc"
	AXDBQuerySessionID     = "ax_session_id"
	AXDBQuerySrcInterval   = "ax_src_interval"
	//AXDBQueryDstInterval = "ax_dst_interval"
	//AXDBRollUpFlag       = "ax_rollup"
)
