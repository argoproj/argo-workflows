// Copyright 2015-2017 Applatix, Inc. All rights reserved.

// Package volops contains the Op objects to be used by the volume plugin.
package volops

import (
    "fmt"
    "os/exec"
    "strings"
    "log"
    "encoding/json"
    "applatix.io/vol_plugin/constants"
    "applatix.io/vol_plugin/volmetadata"
)

type Op struct {}

func (o *Op) WriteNotSupported() {
    successResponse := map[string]string{"status": "Not supported"}
    res, _ := json.Marshal(successResponse)
    fmt.Print(string(res))
}

func (o *Op) WriteSuccess() {
    successResponse := map[string]string{"status": "Success"}
    res, _ := json.Marshal(successResponse)
    fmt.Print(string(res))
}

func (o *Op) WriteSuccessDetails(message string, device string, volumeName string, attached string) {
    successResponse := string(
        "{\"status\": \"Success\", " +
        "\"message\": \"" +  message + "\", "  +
        "\"device\": \"" + device  + "\", "  +
        "\"volumeName\":\"" + volumeName + "\", "  +
        "\"attached\":" + attached + "}")
    fmt.Print(successResponse)
}

func (o *Op) WriteFailure(message string) {
    failureResponse := map[string]string{"status": "Failure", "message": message}
    res, _ := json.Marshal(failureResponse)
    fmt.Println(string(res))
}

type InitOp struct {
    Op
    dbManager volmetadata.DbManager
}

func (initOp *InitOp) volumeSetup() {
    command := "/sbin/vgs"
    commandArgs := strings.Split(constants.AX_VOL_VG_NAME + " --noheadings --unquoted --nosuffix -o vg_name", " ")
    output, err := exec.Command(command, commandArgs...).CombinedOutput()
    if err != nil {
        // The above command failed. Check if it was because there is no vg-ax. If so, move on.
        if strings.Contains(string(output), "Volume group \"vg-ax\" not found") {
            log.Printf("%s", output)
        } else {
            panic(err)
        }
    } else {
        // There was no error in the above command. Check if the vg exists.
        if strings.HasSuffix(strings.TrimSpace(string(output[:])), constants.AX_VOL_VG_NAME) {
            log.Printf("Volume group %s already exists\n", constants.AX_VOL_VG_NAME)
            return
        } else {
            log.Printf("vgs output: %s", output)
        }
    }
    log.Printf("Creating Physical volume...")
    command = "/sbin/pvcreate"
	_, err = exec.Command(command, "/dev/xvdz").Output()
	if err != nil {
        log.Printf("pvcreate error: %s", err)
		panic(err)
	}
    log.Printf("Creating physical volume and volume-group")

    command = "/sbin/vgcreate"
    commandArgs = strings.Split(constants.AX_VOL_VG_NAME + " /dev/xvdz", " ")
	_, err = exec.Command(command, commandArgs...).Output()
	if err != nil {
		panic(err)
	}
    log.Printf("Created physical volume")

}

func (initOp *InitOp) Run() {
    // 1. Create the DB schema.
    initOp.dbManager.CreateSchema()

    // 2. Create the physical volume and volume-group.
    initOp.volumeSetup()

    successResponse := string(
        "{\"status\": \"Success\", " +
        "\"capabilities\": {\"attach\": false}}")
    fmt.Print(successResponse)
}
