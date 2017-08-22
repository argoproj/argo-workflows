package core

import (
	"applatix.io/axdb"
	"github.com/gocql/gocql"
	"time"
)

const (
	CassandraReplStrategy          = "SimpleStrategy"
	CassandraReplRedundancy        = 1
	CassandraClusterReplRedundancy = 3
	CassandraAXKeyspace            = "axdb"
	CassandraSysKeyspace           = "system"
	CassandraProtoVersion          = 4
	CassandraStandloneMode         = "Standlone"
	CassandraClusterMode           = "Cluster"
	CassandraNodes                 = 1
)

const (
	UpdateDropOldColumn                  = -1
	UpdateDropSecondaryIndex             = 0
	UpdateAddNewColumn                   = 1
	UpdateAddSecondaryIndex              = 2
	UpdateAddNewColumnWithSecondaryIndex = 3
	UpdateReCreateSecondaryIndex         = 4
)

const (
	UpdateLuceneIndexNoChange = 0
	UpdateLuceneIndexDrop     = 1
	UpdateLuceneIndexReCreate = 2
)

// constant indicating the status of schema update
const (
	TableAccessAvailable = 0
	TableUpdateInProcess = 1
	TableUpdateFailed    = 2
)

// kafka service address
const (
	KafkaAddress = "kafka-zk.axsys:9092"
	KafkaTopic   = "axdb_schema"
)

var axdbColumnTypeNames = map[int]string{
	axdb.ColumnTypeString:  "text",
	axdb.ColumnTypeDouble:  "double",
	axdb.ColumnTypeInteger: "bigint",
	axdb.ColumnTypeBoolean: "boolean",
	axdb.ColumnTypeArray:   "list<text>",
	axdb.ColumnTypeMap:     "map<text, text>",
	//Removed
	//axdb.ColumnTypeTimestamp:  "timestamp",
	axdb.ColumnTypeUUID:       "uuid",
	axdb.ColumnTypeTimeUUID:   "timeuuid",
	axdb.ColumnTypeOrderedMap: "text",
	axdb.ColumnTypeSet:        "set<text>",
	axdb.ColumnTypeCounter:    "counter",
}

var axdbColumnIndexTypeNames = map[int]string{
	axdb.ColumnIndexStrong:           "secondary key",
	axdb.ColumnIndexNone:             "none",
	axdb.ColumnIndexWeak:             "none",
	axdb.ColumnIndexClustering:       "clustering key",
	axdb.ColumnIndexPartition:        "partition key",
	axdb.ColumnIndexClusteringStrong: "clutering key with secondary index",
}

var axdbLuceneIndexTypeNames = map[int]string{
	axdb.LuceneTypeString:  "string",
	axdb.LuceneTypeDouble:  "double",
	axdb.LuceneTypeLong:    "long",
	axdb.LuceneTypeBoolean: "boolean",
}

func adaptToLuceneIndexType(axdbCol int) int {
	switch axdbCol {
	case axdb.ColumnTypeString,
		axdb.ColumnTypeArray,
		axdb.ColumnTypeMap,
		axdb.ColumnTypeSet,
		axdb.ColumnTypeUUID,
		axdb.ColumnTypeTimeUUID,
		axdb.ColumnTypeOrderedMap:
		return axdb.LuceneTypeString
	case axdb.ColumnTypeDouble:
		return axdb.LuceneTypeDouble
	case axdb.ColumnTypeInteger:
		return axdb.LuceneTypeLong
	case axdb.ColumnTypeBoolean:
		return axdb.LuceneTypeBoolean
	case axdb.ColumnTypeCounter:
		return -1
	default:
		// Require 1 to 1 mapping from cassandra column type to lucene column type, panic to bring attention to the issue instead of hiding the problem
		panic("Missing the mapping from cassandra column type to lucene column type, please check the logic in axdb")
	}
}

const (
	WeekInSeconds      = (7 * 24 * 3600)
	WeekInMicroSeconds = (7 * 24 * 3600 * 1e6)
	AXDBParallelLevel  = 1 // number of threads we want to use when processing a task that can be broken down
)

var axdbRollupIntervalMap = map[int]int{
	axdb.OneDay: axdb.OneHour,
}

// Parameters. Later expose API as needed to control them.
const (
	TimeSeriesTableDataRefreshDelay = 30 // seconds. Delay between insertion of very old data and the time they show up on queries.
)

var NullUUID gocql.UUID = gocql.UUID{0, 0, 0, 0, 0, 0, 0x10, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var EpochTime time.Time = time.Unix(0, 0)

var axintTableDefinitionTable = axdb.Table{AppName: "axint", Name: "table_definition", Type: axdb.TableTypeKeyValue, Columns: map[string]axdb.Column{
	axdb.AXDBKeyColumnName:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
	axdb.AXDBValueColumnName:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	axdb.AXDBTimeColumnName:   axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	axdb.AXDBStatusColumnName: axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
}}

var axintObjectCounterTable = axdb.Table{AppName: "axint", Name: "object_counter", Type: axdb.TableTypeCounter, Columns: map[string]axdb.Column{
	axdb.AXDBKeyColumnName: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
}}

// internal table to save system information used by axdb
var axintSystemInfo = axdb.Table{AppName: "axint", Name: axdb.AXDBSystemInfoTable, Type: axdb.TableTypeKeyValue, Columns: map[string]axdb.Column{
	axdb.AXDBKeyColumnName:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
	axdb.AXDBValueColumnName: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
}}

const (
	AXDBPerfTableName  = "table_name"
	AXDBPerfOperation  = "cql_op"
	AXDBPerfParameters = "cql_parameters"
	AXDBPerfExecTime   = "cql_exec_time"
	// we only record the sucsseful cql query at this moment
	//RetCode    = "rest_return_code"
)

// profile table
var axintProfileTable = axdb.Table{AppName: axdb.AXDBPerf, Name: axdb.AXDBPerfTable, Type: axdb.TableTypeTimeSeries, Columns: map[string]axdb.Column{
	AXDBPerfTableName:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
	AXDBPerfOperation:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
	AXDBPerfParameters: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
	AXDBPerfExecTime:   axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
},
	Stats: map[string]int{
		AXDBPerfExecTime: axdb.ColumnStatPercent,
	},
	UseSearch: false,
	Configs: map[string]interface{}{
		"default_time_to_live": int64(30 * axdb.OneDay),
	},
}

var internalTables = []*axdb.Table{
	&axintTableDefinitionTable,
	&axintObjectCounterTable,
	&axintSystemInfo,
}

var userTables = []*axdb.Table{
	&axintProfileTable,
}
