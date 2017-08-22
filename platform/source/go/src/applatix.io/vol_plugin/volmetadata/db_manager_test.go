// Copyright 2015-2017 Applatix, Inc. All rights reserved.

package volmetadata

import (
    "fmt"
    "database/sql"
    "testing"
    "applatix.io/vol_plugin/constants"
    _ "github.com/mattn/go-sqlite3"
)

func TestDBManagerBasic(t *testing.T) {
    var dbManager DbManager
    // Delete any DBs from prior runs.
    dbManager.DeleteSchema()

    // Create the schema.
    dbManager.CreateSchema()

    // Verify that the version is correctly populated.
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    if err != nil {
        panic("Failed to get open DB")
    }

    rows, err := db.Query("SELECT * FROM ax_vol_version")
    defer rows.Close()
    if err != nil {
        panic("Failed to get version")
    }

    version := ""
    for rows.Next() {
        err = rows.Scan(&version)
    }
    err = rows.Err()
    if err != nil {
        panic("Failed to scan rows for getting version")
    }

    if version != "1.0.0" {
        fmt.Println("Version mismatch: Expected: 1.0.0, Found:", version)
        t.Fail()
    }
}

func TestInitVol(t *testing.T) {
    var dbManager DbManager
    dbManager.DeleteSchema()
    dbManager.CreateSchema()

    optMap := make(map[string]interface{})
    optMap["size_mb"] = "100"
    optMap["ax_vol_type"] = "ax-anonymous"
    dbManager.InitVol("some_lv_name", "ax-anonyous", 100, "/mnt/testDir", optMap)

    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    if err != nil {
        panic("Failed to get open DB")
    }

    rows, err := db.Query("SELECT lv_name FROM ax_vols")
    defer rows.Close()
    if err != nil {
        panic("Failed to get ax_vols")
    }

    lvName := ""
    for rows.Next() {
        err = rows.Scan(&lvName)
    }
    err = rows.Err()
    if err != nil {
        panic("Failed to scan rows for getting ax_vols")
    }

    if lvName != "some_lv_name" {
        fmt.Println("Volume name mismatch: Expected: some_lv_name, Found:", lvName)
        t.Fail()
    }
}

func TestCommitVol(t *testing.T) {
    TestInitVol(t)

    var dbManager DbManager
    dbManager.CommitVol("some_lv_name")

    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    if err != nil {
        panic("Failed to get open DB")
    }

    rows, err := db.Query("SELECT state FROM ax_vols WHERE lv_name = 'some_lv_name'")
    defer rows.Close()
    if err != nil {
        panic("Failed to get ax_vols")
    }

    state := ""
    for rows.Next() {
        err = rows.Scan(&state)
    }
    err = rows.Err()
    if err != nil {
        panic("Failed to scan rows for getting ax_vols")
    }

    if state != "READY" {
        fmt.Println("Volume state mismatch: Expected: READY, Found:", state)
        t.Fail()
    }
}