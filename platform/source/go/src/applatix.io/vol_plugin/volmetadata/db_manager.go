// Copyright 2015-2017 Applatix, Inc. All rights reserved.

// Package volmetadata contains the objects needed to handle the volume metadata.
package volmetadata

import (
    "os"
    "log"
    "strconv"
    "encoding/json"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "applatix.io/vol_plugin/constants"
)

type DbManager struct {}

func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

func (dbManager *DbManager) commitOrRollback(tx *sql.Tx) {
    err := recover()

    if err != nil {
        log.Printf("Failed to insert version: %s. Rolling back...", err)
        tx.Rollback()
    } else {
        tx.Commit()
    }
}

func (dbManager *DbManager) initVersion(db *sql.DB) {
    // Insert the "version" string in the ax_vol_version table. Subequent "upgrades" of the schema will
    // be handled with a special "upgrade" command line option.
    tx, err := db.Begin()
    checkErr(err)

    // This method will be invoked when this method completes execution or errors out. Depending on the
    // the whether there was an error or not, commitOrRollback will do the right thing accordingly.
    defer dbManager.commitOrRollback(tx)

    rows, err := db.Query("SELECT COUNT(*) FROM ax_vol_version")
    defer rows.Close()
    if err != nil {
        log.Printf("Failed to insert version: %s. Rolling back!", err)
        tx.Rollback()
    }

    rowCount := 0
    for rows.Next() {
        err = rows.Scan(&rowCount)
    }
    err = rows.Err()
    checkErr(err)

    if rowCount == 0 {
        log.Printf("No rows in the DB. Creating version '1.0.0'")
        sqlStmt, err := db.Prepare("INSERT INTO ax_vol_version VALUES('1.0.0')")
        checkErr(err)
        _, err = tx.Stmt(sqlStmt).Exec()
        checkErr(err)
    }
}

func (dbManager *DbManager) CreateSchema() {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS ax_vol_version (version text)`)
    checkErr(err)

    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS ax_vols(
            lv_name text PRIMARY KEY,
            mount_path text,
            size_mb real,
            state text,
            in_use int,
            vol_type text,
            pod_id text,
            used_counter int,
            create_time datetime,
            last_used_time datetime,
            json_opts text
        )`)
    checkErr(err)

    dbManager.initVersion(db)
}

func (dbManager *DbManager) TableExists(tableName string) bool {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    sqlStmt := `SELECT name FROM sqlite_master WHERE type='table' AND name='` + tableName + `'`
    rows, err := db.Query(sqlStmt)
    checkErr(err)
    defer rows.Close()

    exists := false
    for rows.Next() {
        exists = true
    }

	err = rows.Err()
	if err != nil {
		panic(err)
	}

    return exists
}

func (dbManager *DbManager) DeleteSchema() {
    err := os.RemoveAll(constants.AX_VOL_DB_NAME)
    checkErr(err)
}

func (dbManager *DbManager) InitVol(lvName string, volumeType string, sizeMb int,
                                    mountDir string, podId string, optMap map[string]interface{}) {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    jsonString, _ := json.Marshal(optMap)
    sqlQuery := "INSERT INTO ax_vols(lv_name, mount_path, size_mb, state, in_use, vol_type, pod_id, " +
        "used_counter, create_time, last_used_time, json_opts) VALUES(?, ?, ?, " +
        "'CREATING', 1, ?, ?, 1, datetime('now', 'localtime'), " +
        "datetime('now', 'localtime'), ?)"
    _, err = db.Exec(sqlQuery, lvName, mountDir, strconv.Itoa(sizeMb), volumeType, podId, string(jsonString))
    checkErr(err)
}

func (dbManager *DbManager) CommitVol(lvName string) {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    sqlQuery := "UPDATE ax_vols SET state = 'READY' WHERE lv_name = ?"
    _, err = db.Exec(sqlQuery, lvName)
    checkErr(err)
}

func (dbManager *DbManager) getColHelper(sqlQuery string, queryArg string) string {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    rows, err := db.Query(sqlQuery, queryArg)
    checkErr(err)
    defer rows.Close()

    retVal := ""
    for rows.Next() {
        err = rows.Scan(&retVal)
    }
    err = rows.Err()
    if err != nil {
        panic(err)
    }

    return retVal
}

func (dbManager *DbManager) GetVolumeName(mountDir string) string {
    return dbManager.getColHelper("SELECT lv_name FROM ax_vols WHERE mount_path = ?", mountDir)
}

func (dbManager *DbManager) GetVolumeType(mountDir string) string {
    return dbManager.getColHelper("SELECT vol_type FROM ax_vols WHERE mount_path = ?", mountDir)
}

func (dbManager *DbManager) DeleteVolume(lvName string) {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    sqlQuery := "DELETE FROM ax_vols WHERE lv_name = ?"
    _, err = db.Exec(sqlQuery, lvName)
    checkErr(err)
    log.Printf("Delete volume from DB: %s", lvName)
}

func (dbManager *DbManager) MarkVolumeInUse(lvName string, mountDir string, podId string) {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    sqlQuery := "UPDATE ax_vols SET in_use = 1, used_counter = used_counter + 1, mount_path = ?, pod_id = ?, last_used_time = datetime('now', 'localtime') WHERE lv_name = ?"
    _, err = db.Exec(sqlQuery, mountDir, podId, lvName)
    checkErr(err)
}

func (dbManager *DbManager) MarkVolumeFree(mountDir string) {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    sqlQuery := "UPDATE ax_vols SET in_use = 0, mount_path = '', pod_id = '' WHERE mount_path = ?"
    _, err = db.Exec(sqlQuery, mountDir)
    checkErr(err)
}

func (dbManager *DbManager) GetFreeVolume(sizeMb int) string {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    sqlQuery := `SELECT lv_name FROM ax_vols WHERE vol_type =
        'ax-docker-graph-storage' AND in_use = 0 AND size_mb = ? LIMIT 1`
    rows, err := db.Query(sqlQuery, sizeMb)
    checkErr(err)
    defer rows.Close()

    var lv_name string
    for rows.Next() {
        err = rows.Scan(&lv_name)
    }
    err = rows.Err()
    if err != nil {
        panic(err)
    }

    return lv_name
}

func (dbManager *DbManager) GetAllFreeVolumes() []string {
    db, err := sql.Open("sqlite3", constants.AX_VOL_DB_NAME)
    checkErr(err)
    defer db.Close()

    sqlQuery := `SELECT lv_name FROM ax_vols WHERE vol_type = 'ax-docker-graph-storage' AND in_use = 0`
    rows, err := db.Query(sqlQuery)
    checkErr(err)
    defer rows.Close()

    var lvNames []string
    var lvName string
    for rows.Next() {
        err = rows.Scan(&lvName)
        lvNames = append(lvNames, lvName)
    }
    err = rows.Err()
    if err != nil {
        panic(err)
    }

    return lvNames
}