package main

import (
	"applatix.io/axdb/core"
	"applatix.io/monitor"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	var numNodes int64
	if len(os.Args) == 1 {
		numNodes = 1
	} else if len(os.Args) == 2 {
		n, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err != nil || n <= 0 {
			fmt.Println("usage: axdb_server [num_of_nodes]")
			os.Exit(1)
		} else if n == 2 {
			fmt.Println("Two-node cluster isn't recommended, please select a number >= 3")
			os.Exit(1)
		}
		numNodes = n
	} else {
		fmt.Println("usage: axdb_server [cassandra_mode]")
		os.Exit(1)
	}
	core.InitLoggers()

	go monitor.MonitorSystem(60*time.Second, core.GetDebugLogger())

	core.InitDB(numNodes)
	core.InitMessageClients()
	core.MonitorDB()
	core.ReloadDBTable()
	core.RollUpStats()
	core.StartRouter(false)
}
