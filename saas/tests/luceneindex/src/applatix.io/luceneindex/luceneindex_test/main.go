package main

import (
	"applatix.io/axdb/axdbcl"
	"applatix.io/luceneindex"
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: luceneindex_test http://axdb_ip:axdb_port/axdb_version")
		os.Exit(1)
	}

	dbcl := axdbcl.NewAXDBClientWithTimeout(os.Args[1], 5*time.Minute)
	luceneindex.Dbcl = axdbcl.NewAXDBClientWithTimeout(os.Args[1], 5*time.Minute)
	// Wait at most 20 minutes for axdb to be ready
	count := 0
	for {
		count++
		var bodyArray []interface{}
		dbErr := dbcl.Get("axdb", "status", nil, &bodyArray)
		if dbErr == nil {
			break
		} else {
			fmt.Printf("waiting for axdb to be ready ...... count %v error %v\n", count, dbErr)
			if count > 1200 {
				// Give up, marathon would restart this container
				fmt.Printf("axdb is not available, exited\n")
				os.Exit(1)
			}
		}
		time.Sleep(1 * time.Second)
	}

	luceneindex.MainLoop()
}
