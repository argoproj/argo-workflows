// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package main

import (
	"applatix.io/axdb/axdbcl"
	"applatix.io/axops"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/rediscl"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s \n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) > 0 {
		flag.Usage()
		return
	}

	utils.InitLoggers()

	axops.Dbcl = axdbcl.NewAXDBClientWithTimeout("http://axdb.axsys:8083/v1", 5*time.Minute)
	utils.Dbcl = axdbcl.NewAXDBClientWithTimeout("http://axdb.axsys:8083/v1", 5*time.Minute)
	utils.RedisCacheCl = rediscl.NewRedisClient("redis.axsys:6379", "", utils.RedisCachingDatabase)

	password, axErr := user.ResetAdminInternalPassword()

	if axErr != nil {
		os.Exit(1)
	}

	fmt.Println("Username:", user.INTERNAL_ADMIN_USER)
	fmt.Println("Password:", password)
}
