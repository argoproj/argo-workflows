// Copyright 2015-2017 Applatix, Inc. All rights reserved.

package main

import (
    "fmt"
    "flag"
    "log"
    "log/syslog"
    "applatix.io/vol_plugin/volops"
)

var Version = "No version provided"

func process(operation string) {
    // Configure logger to write to the syslog.
    logwriter, err := syslog.New(syslog.LOG_INFO, "ax_volume_plugin")
    if err == nil {
        log.SetOutput(logwriter)
    }
    log.Printf("Processing: %s", operation)

    switch operation {
        case "init":
            op := volops.InitOp{}
            op.Run()
        case "unmount": // Mount and unmount are processed by the same object. Order is important!
            mount_dir := flag.Arg(1)
            op := volops.RequestProcessor{}
            op.Run(operation, mount_dir, "", "")
        case "mount":
            mount_dir := flag.Arg(1)
            json_opts := flag.Arg(2)
            op := volops.RequestProcessor{}
            op.Run(operation, mount_dir, "", json_opts)
        case "version":
            fmt.Println("AX volume plugin version: ", Version)
        case "getvolumename":
            fallthrough;
        case "waitforattach":
            fallthrough;
        case "attach":
            fallthrough;
        case "detach":
            fallthrough;
        case "isattached":
            fallthrough;
        case "mountdevice":
            fallthrough;
        case "unmountdevice":
            op := volops.Op{}
            op.WriteNotSupported()
        default:
            op := volops.Op{}
            op.WriteFailure("Invalid option...")
    }

   log.Printf("Done Processing: %s", operation)
}

func main() {
    flag.Parse()

    if len(flag.Args()) < 1 {
        log.Panicf("Not enough command line arguments ...")
        return
    }

    operation := flag.Arg(0)
    process(operation)
}
