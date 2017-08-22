// Copyright 2015-2016 Applatix, Inc. All rights reserved.

package main

import (
	"fmt"
	"os"
	"time"

	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axnc"
	"applatix.io/axops"
	"applatix.io/axops/artifact"
	"applatix.io/axops/schema_internal"
	"applatix.io/axops/tool"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/axops/volume"
	"applatix.io/common"
	"applatix.io/rediscl"
	"applatix.io/restcl"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: axops_initializer http://axdb_ip:axdb_port/axdb_version http://axmon_url")
		os.Exit(1)
	}

	utils.InitLoggers()

	// Since the upgrade might take long to finish, don't use timeout here
	axops.Dbcl = axdbcl.NewAXDBClientWithTimeout(os.Args[1], 5*time.Minute)
	utils.Dbcl = axdbcl.NewAXDBClientWithTimeout(os.Args[1], 5*time.Minute)
	utils.AxmonCl = restcl.NewRestClientWithTimeout(os.Args[2], 5*time.Minute)
	utils.RedisCacheCl = rediscl.NewRedisClient("redis.axsys:6379", "", utils.RedisCachingDatabase)
	dbcl := axops.Dbcl

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

	AX_NAMESPACE := common.GetAxNameSpace()
	AX_VERSION := common.GetAxVersion()

	if AX_NAMESPACE == "" {
		fmt.Printf("AX_NAMESAPCE is not available from environment variables. Abort.")
		os.Exit(1)
	}

	if AX_VERSION == "" {
		fmt.Printf("AX_VERSION is not available from environment variables. Abort.")
		os.Exit(1)
	}

	axErr := axops.CreateTables()
	if axErr != nil {
		fmt.Printf("axdb is not in good state, exited\n")
		os.Exit(1)
	}

	// Purge keys in redis
	utils.InfoLog.Println("Flushing redis DB")
	utils.RedisCacheCl.FlushDB()

	user.InitAdminInternalUser()

	tool.AddExampleRepository()

	axErr = artifact.PopulateDefaultArtifactMetaData()
	if axErr != nil {
		fmt.Printf("Failed to populate default artifact metadata, Err: %v", axErr)
		os.Exit(1)
	}

	axErr = artifact.PopulateDefaultRetentionPolicy()
	if axErr != nil {
		fmt.Printf("Failed to populate default retention policy, Err: %v", axErr)
		os.Exit(1)
	}

	axErr = volume.PopulateStorageProviderClasses()
	if axErr != nil {
		fmt.Printf("Failed to populate storage providers and classes, Err: %v", axErr)
		os.Exit(1)
	}

	axErr = axnc.PopulateDefaultRules()
	if axErr != nil {
		fmt.Printf("Failed to populate default rules, Err: %v", axErr)
		os.Exit(1)
	}

	// TODO: Break this down. Update when each schema groups are updated.
	for _, appName := range []string{axdb.AXDBAppAXSYS, axdb.AXDBAppAXDEVOPS, axdb.AXDBAppAXOPS, axdb.AXDBAppAMM} {
		namespace := schema_internal.AppKeyValue{
			AppName: appName,
			Key:     "AX_NAMESPACE",
			Value:   AX_NAMESPACE,
		}
		_, axErr = dbcl.Put(axdb.AXDBAppApp, schema_internal.AppTable, namespace)
		if axErr != nil {
			fmt.Printf("Failed to update the AX_NAMESAPCE for app %v: %v", appName, axErr)
			os.Exit(1)
		}

		version := schema_internal.AppKeyValue{
			AppName: appName,
			Key:     "AX_VERSION",
			Value:   AX_VERSION,
		}
		_, axErr = dbcl.Put(axdb.AXDBAppApp, schema_internal.AppTable, version)
		if axErr != nil {
			fmt.Printf("Failed to update the AX_NAMESAPCE for app %v: %v", appName, axErr)
			os.Exit(1)
		}
	}
}
