package luceneindex

import (
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axerror"
	"fmt"
	"github.com/gocql/gocql"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"
)

var Dbcl *axdbcl.AXDBClient

// how many rows are there in table when the query is submitted
var minRows int = 0
var maxRows int = 500000

const (
	ArtifactsTable    = "artifacts"
	ArtifactsID       = "artifact_id"
	SourceArtifactsID = "source_artifact_id"
	ServiceInstanceID = "service_instance_id"
	Name              = "name"
	IsAlias           = "is_alias"
	Description       = "description"
	SrcPath           = "src_path"
	SrcName           = "src_name"
	Excludes          = "excludes"
	StorageMethod     = "storage_method"
	StoragePath       = "storage_path"
	InlineStorage     = "inline_storage"
	CompressionMode   = "compression_mode"
	SymlinkMode       = "symlink_mode"
	ArchiveMode       = "archive_mode"
	NumByte           = "num_byte"
	NumFile           = "num_file"
	NumDir            = "num_dir"
	NumSymlink        = "num_symlink"
	NumOther          = "num_other"
	NumSkipByte       = "num_skip_byte"
	NumSkip           = "num_skip"
	NumStoredByte     = "stored_byte"
	Meta              = "meta"
	Timestamp         = "timestamp"
	WorkflowID        = "workflow_id"
	Checksum          = "checksum"
	Tags              = "tags"
	RetentionTags     = "retention_tags"
	Deleted           = "deleted"
	DeletedDate       = "deleted_date"
	DeletedBy         = "deleted_by"
	ThirdPartyLinks   = "third_party"
	RelativePath      = "relative_path"
)

var ArtifactsSchema = axdb.Table{
	AppName: "perftest",
	Name:    ArtifactsTable,
	Type:    axdb.TableTypeTimeSeries,
	Columns: map[string]axdb.Column{
		ArtifactsID:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ServiceInstanceID: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		SourceArtifactsID: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Name:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		IsAlias:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		Description:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SrcPath:           axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SrcName:           axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Excludes:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		StorageMethod:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		StoragePath:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InlineStorage:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		CompressionMode:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SymlinkMode:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ArchiveMode:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		NumByte:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumFile:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumDir:            axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumSymlink:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumOther:          axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumSkipByte:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumSkip:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumStoredByte:     axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		Meta:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Timestamp:         axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		WorkflowID:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		Checksum:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Tags:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RetentionTags:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Deleted:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong},
		DeletedDate:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		DeletedBy:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ThirdPartyLinks:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RelativePath:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},

	UseSearch: true,
}

func generateRow() map[string]interface{} {
	row := make(map[string]interface{})
	tUUID := gocql.UUIDFromTime(time.Now())
	row[ArtifactsID] = fmt.Sprintf("%v", tUUID)
	sUUID := gocql.UUIDFromTime(time.Now())
	row[ServiceInstanceID] = fmt.Sprintf("%v", sUUID)
	row[SourceArtifactsID] = fmt.Sprintf("%v", sUUID)
	row[DeletedDate] = time.Now().UnixNano() / 1e3
	//row[Timestamp] = row[DeletedDate]
	row[DeletedBy] = fmt.Sprintf("deletor_%d", rand.Int())
	row[Tags] = fmt.Sprintf("random_%d", rand.Int())
	row[RelativePath] = fmt.Sprintf("randome_path_%f", rand.Float32())
	return row
}

func report_stats(op string, arr []float64) {
	sort.Float64s(arr)
	sum := float64(0)
	count := len(arr)
	for _, f := range arr {
		sum += f
	}

	fmt.Printf("%s latency report:  ", op)
	fmt.Printf("avg: %f, mean: %f, min: %f, max: %f\n", sum/float64(count), arr[count/2], arr[0], arr[count-1])
}

func GetMinRows() int {
	countStr, _ := os.LookupEnv("AX_MIN_ROWS")
	count, err := strconv.Atoi(countStr)
	if err == nil {
		return count
	} else {
		return 0
	}
}

func GetMaxRows() int {
	countStr, _ := os.LookupEnv("AX_MAX_ROWS")
	count, err := strconv.Atoi(countStr)
	if err == nil {
		return count
	} else {
		return 0
	}
}

func DropTable(table axdb.Table) *axerror.AXError {
	fmt.Printf("Starting dropping table: %v.\n", table)
	startTime := time.Now().UnixNano() / 1e9
	var axErr *axerror.AXError
	for {
		currentTime := time.Now().UnixNano() / 1e9
		if currentTime-startTime > 900 {
			axErr = axerror.ERR_AX_TIMEOUT.NewWithMessage("Drop table failed due to timeout (15 minutes)")
		} else {
			_, axErr = Dbcl.Delete(table.AppName, table.Name, nil)
			if axErr == nil {
				fmt.Printf("Successfully dropped table %v schema.\n", table.Name)
			} else {
				fmt.Printf("Drop table failure with err: %v", axErr)
				time.Sleep(1 * time.Second)
				continue
			}
		}
		break
	}
	return axErr
}

func CreateTable(table axdb.Table) *axerror.AXError {
	fmt.Printf("Starting updating table: %v.\n", table)
	startTime := time.Now().UnixNano() / 1e9
	var axErr *axerror.AXError
	for {
		currentTime := time.Now().UnixNano() / 1e9
		if currentTime-startTime > 900 {
			axErr = axerror.ERR_AX_TIMEOUT.NewWithMessage("Axops table creation failed due to timeout(15 minutes)")
		} else {
			_, axErr = Dbcl.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, table)
			if axErr == nil {
				fmt.Printf("Successfully updated table %v schema.\n", table.Name)
			} else {
				fmt.Printf("Create table failure with err: %v\n", axErr)
				time.Sleep(1 * time.Second)
				continue
			}
		}
		break
	}
	return axErr
}

func InsertRows(table axdb.Table, nums int) int {
	res := 0
	for i := 0; i < nums; i++ {

		_, axErr := Dbcl.Post(table.AppName, table.Name, generateRow())
		if axErr == nil {
			res++
		}
	}
	fmt.Printf("inserted %d rows\n", res)
	return res
}

func MainLoop() {
	r := GetMinRows()
	if r != 0 {
		minRows = r
	}

	r = GetMaxRows()
	if r != 0 && r >= minRows {
		maxRows = r
	}

	err := CreateTable(ArtifactsSchema)
	if err != nil {
		fmt.Printf("Failed to create table %s", ArtifactsSchema.Name)
		os.Exit(1)
	}

	incr := 50000
	preRows := minRows
	fmt.Printf("minRow=%d, maxRow=%d\n", minRows, maxRows)
	for i := minRows; i <= maxRows; i += incr {
		// insert (i - preRows) rows
		InsertRows(ArtifactsSchema, i-preRows)

		// run 200 tests
		// read and write latency array
		var reads []float64
		var writes []float64

		for t := 0; t < 200; t++ {
			row := make(map[string]interface{})
			tUUID := gocql.UUIDFromTime(time.Now())
			row[ArtifactsID] = fmt.Sprintf("%v", tUUID)
			sUUID := gocql.UUIDFromTime(time.Now())
			row[ServiceInstanceID] = fmt.Sprintf("%v", sUUID)
			row[SourceArtifactsID] = fmt.Sprintf("%v", sUUID)
			row[DeletedDate] = time.Now().UnixNano() / 1e3
			//row[Timestamp] = row[DeletedDate]
			row[DeletedBy] = fmt.Sprintf("deletor_%d", rand.Int())
			labels := fmt.Sprintf("fixed_%d_%d", i, rand.Int())
			row[Tags] = labels
			row[RelativePath] = fmt.Sprintf("randome_path_%f", rand.Float32())
			t1 := time.Now().UnixNano() / 1e3
			_, err := Dbcl.Post(ArtifactsSchema.AppName, ArtifactsSchema.Name, row)
			if err == nil {
				t2 := time.Now().UnixNano() / 1e3
				writes = append(writes, float64(t2-t1))
				// Test read latency
				luceneSearch := axdb.NewLuceneSearch()
				luceneSearch.AddQueryMust(axdb.NewLuceneWildCardFilterBase(Tags, labels))
				params := map[string]interface{}{
					axdb.AXDBQuerySearch: luceneSearch,
				}

				var resultArray []map[string]interface{}
				t1 := time.Now().UnixNano() / 1e3
				for {
					err := Dbcl.Get(ArtifactsSchema.AppName, ArtifactsSchema.Name, params, &resultArray)
					// db error, we just ignore this data point
					if err != nil {
						break
					}
					if len(resultArray) == 1 && resultArray[0][Tags].(string) == labels {
						t2 := time.Now().UnixNano() / 1e3
						reads = append(reads, float64(t2-t1))
						break
					}
				}
			} else {
				fmt.Printf("failure to insert a row: %v", err)
			}
		}
		report_stats("read", reads)
		report_stats("write", writes)
		preRows = i
	}

	fmt.Printf("drop table\n")
	err = DropTable(ArtifactsSchema)
	if err != nil {
		fmt.Printf("Failed to drop the existing table %s", ArtifactsSchema.Name)
		os.Exit(1)
	}
}
