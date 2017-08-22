// Copyright 2015-2017 Applatix, Inc. All rights reserved.

package volops

import (
    "log"
    "encoding/json"
    "applatix.io/vol_plugin/constants"
    "os"
    "syscall"
    "applatix.io/vol_plugin/volmetadata"
)

// The RequestProcess is the main entry-point for processing mount/unmount requests. It will create and invoke
// appropriate objects based on the type of volume needed.
type RequestProcessor struct {
    op Op
    opLock *os.File
}

func (requestProcessor *RequestProcessor) AcquireLock() {
    file, err := os.Create(constants.AX_VOL_LOCK_PATH)
    if err != nil {
        panic(err)
    }

    // Intentionally a blocking call.
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
	if err != nil {
		file.Close()
		panic(err)
	}

    requestProcessor.opLock = file
    log.Printf("Acquired lock")
}

func (requestProcessor *RequestProcessor) ReleaseLock() {
    fd := requestProcessor.opLock.Fd()
    err := syscall.Flock(int(fd), syscall.LOCK_UN)
	if err != nil {
		panic(err)
	}
    log.Printf("Released lock")
}

func (requestProcessor *RequestProcessor) Run(operation string, mountDir string, mountDev string, jsonOpts string) {
    requestProcessor.AcquireLock()
    defer requestProcessor.ReleaseLock()

    var optMap map[string]interface{}
    if jsonOpts != "" {
        if err := json.Unmarshal([]byte(jsonOpts), &optMap); err != nil {
            panic(err)
        }
    } else {
        var dbManager volmetadata.DbManager
        optMap = make(map[string]interface{})
        optMap["ax_vol_type"] = dbManager.GetVolumeType(mountDir)
    }

    switch optMap["ax_vol_type"] {
        case "ax-anonymous":
            anonymousVolume := AnonymousVolume{mountDir: mountDir, mountDev: mountDev, optMap: optMap}
            if operation == "mount" {
                lvName := anonymousVolume.Mount()
                if lvName != "" {
                    log.Printf("Successfully created lv: %s", lvName)
                    requestProcessor.op.WriteSuccess()
                    return
                } else {
                    requestProcessor.op.WriteFailure("Failed to create logical volume")
                    return
                }
            } else if operation == "unmount" {
                anonymousVolume.Unmount()
                requestProcessor.op.WriteSuccess()
                return
            }
        case "ax-docker-graph-storage":
            dgsVolume := DGSVolume{mountDir: mountDir, mountDev: mountDev, optMap: optMap}
            if operation == "mount" {
                lvName := dgsVolume.Mount()
                if lvName != "" {
                    log.Printf("Successfully using lv: %s", lvName)
                    requestProcessor.op.WriteSuccess()
                    return
                } else {
                    requestProcessor.op.WriteFailure("Failed to find/create logical volume")
                    return
                }
            } else if operation == "unmount" {
                dgsVolume.Unmount()
                requestProcessor.op.WriteSuccess()
                return
            }
        default:
            log.Printf("Unknown volume type: %s", optMap["ax_vol_type"])
            requestProcessor.op.WriteFailure("Unknown volume type for requestProcessor")
            return
    }
}
