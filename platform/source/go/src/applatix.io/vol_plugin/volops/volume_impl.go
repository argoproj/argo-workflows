// Copyright 2015-2017 Applatix, Inc. All rights reserved.

// Package volops contains the Op objects to be used by the volume plugin.
package volops

import (
    "errors"
    "os"
    "log"
    "strconv"
    "os/exec"
    "syscall"
    "strings"
    "github.com/satori/go.uuid"
    "applatix.io/vol_plugin/constants"
    "applatix.io/vol_plugin/volmetadata"
)

type VolumeBase struct {
    dbManager volmetadata.DbManager
}

func (volumeBase *VolumeBase) getFreeSize() int {
    command := "/sbin/vgs"
    commandArgs := strings.Split(constants.AX_VOL_VG_NAME + " --noheadings --unquoted -o vg_free --nosuffix --units m", " ")
    out, err := exec.Command(command, commandArgs...).Output()
    if err != nil {
        panic(err)
    }
    freeSizeFloat, err := strconv.ParseFloat(strings.TrimSpace(string(out[:])), 32)
    if err != nil {
        panic(err)
    }

    // Easier to deal in integers.
    freeSize := int(freeSizeFloat)

    log.Printf("Free size: %d", freeSize)
    return freeSize
}

func (volumeBase *VolumeBase) createLogicalVolume(lvName string, sizeMb int) {
    command := "/sbin/lvcreate"
    commandArgs := strings.Split("-L " + strconv.Itoa(sizeMb) + "M -n " + lvName + " " + constants.AX_VOL_VG_NAME, " ")
	output, err := exec.Command(command, commandArgs...).Output()
	if err != nil {
		panic(err)
	}
    log.Printf("Created logical volume %s. Size %dMB: %s", lvName, sizeMb, output)
}

func (volumeBase *VolumeBase) formatLogicalVolume(lvName string) {
    command := "mkfs.ext4"
    commandArgs := strings.Split("/dev/" + constants.AX_VOL_VG_NAME + "/" + lvName, " ")
	output, err := exec.Command(command, commandArgs...).Output()
	if err != nil {
		panic(err)
	}
    log.Printf("Created ext4 filesystem on %s. %s", lvName, output)
}

func (volumeBase *VolumeBase) mountLogicalVolume(lvName string, mountDir string) {
    // First, create the entire mountPath.
    err := os.MkdirAll(mountDir, os.ModePerm)
    if err != nil {
        panic(err)
    }

    // Mount the logical volume.
    command := "/bin/mount"
    commandArgs := strings.Split("/dev/" + constants.AX_VOL_VG_NAME + "/" + lvName + " " + mountDir, " ")
	output, err := exec.Command(command, commandArgs...).Output()
	if err != nil {
		panic(err)
	}
    log.Printf("Mounted %s on %s: %s", lvName, mountDir, output)
}

func (volumeBase *VolumeBase) getPodIdFromMountDir(mountDir string) string {
    // Mount path is of the format /mnt/ephemeral/kubernetes/kubelet/pods/<pod-id>/volumes/ax~vol_plugin/<vol-name>"
    var podId string
    mountParts := strings.Split(mountDir, "/")
    if len(mountParts) > 6 {
        podId = mountParts[6]
    }

    return podId
}

func (volumeBase *VolumeBase) createVolume(volumeType string, sizeMb int, mountDir string, optMap map[string]interface{}) string {
    lvName := "lv-" + uuid.NewV4().String()

    // It is possible that on a previous occassion, the mountDir was created but never got used because:
    //  i) The response never reached kubelet (e.g. if the plugin crashed after creating the
    //     volume but before replying to kubelet)
    // ii) Kubelet received the response but it crashed before using the mounted directory.
    // To get over the above situation, we simply unmount the mountDir and delete the lv.
    volumeBase.deleteVolume(mountDir)

    podId := volumeBase.getPodIdFromMountDir(mountDir)

    // Create an entry for the volume in the DB in the "CREATING" state.
    volumeBase.dbManager.InitVol(lvName, volumeType, sizeMb, mountDir, podId, optMap)

    // Create the Logical volume
    volumeBase.createLogicalVolume(lvName, sizeMb)

    // Format the newly created volume with ext4
    volumeBase.formatLogicalVolume(lvName)

    // Mount the volume on the user directory
    volumeBase.mountLogicalVolume(lvName, mountDir)

    // Mark volume as Ready in the DB.
    volumeBase.dbManager.CommitVol(lvName)

    return lvName
}

func (volumeBase *VolumeBase) unmountDir(mountDir string) {
    command := "/bin/umount"
    commandArgs := strings.Split("-f " + mountDir, " ")
	output, err := exec.Command(command, commandArgs...).Output()
	if err != nil {
        // /bin/mountpoint returns 0 if the given dir is a mountpoint.
        // Otherwise, the return code is 1.
        command := "/bin/mountpoint"
        commandArgs := strings.Split("-q " + mountDir, " ")
        _, errCode := exec.Command(command, commandArgs...).Output()
        if errCode != nil {
            log.Printf("%s is not a mountpoint. Ignoring ...", mountDir)
        } else {
            // Return code of 0 means that the dir is a mountpoint. This means that
            // there was a genuine failure to unmount. This shouldn't happen!
            log.Printf("Failed to unmount %s", mountDir)
            panic(err)
        }
	} else {
        log.Printf("Unmounted %s: %s", mountDir, output)
    }
}

func (volumeBase *VolumeBase) deleteLogicalVolume(lvName string) {
    command := "/sbin/lvremove"
    commandArgs := strings.Split("-f /dev/" + constants.AX_VOL_VG_NAME + "/" + lvName, " ")
	output, err := exec.Command(command, commandArgs...).Output()
	if err != nil {
		panic(err)
	}
    log.Printf("Deleted logical volume %s: %s", lvName, output)
}

func (volumeBase *VolumeBase) deleteVolume(mountDir string) {
    // Lookup the logical volume name from the DB.
    lvName := volumeBase.dbManager.GetVolumeName(mountDir)

    // Move on if there was no entry for this mountDir in the DB.
    if lvName == "" {
        return
    }

    // Unmount
    volumeBase.unmountDir(mountDir)

    // Delete the logical volume.
    volumeBase.deleteLogicalVolume(lvName)

    // Remove the entry from the DB.
    volumeBase.dbManager.DeleteVolume(lvName)

    log.Printf("Deleted volume %s(%s)", lvName, mountDir)
}

func getSizeOpt(optMap map[string]interface{}) int {
    sizeRequested := 0

    // Need to initialize err for reusing sizeRequested and keeping compiler happy.
    err := errors.New("Default error")
    if size_str, ok := optMap["size_mb"].(string); ok {
        sizeRequested, err = strconv.Atoi(size_str)
        if err != nil {
            panic(err)
        }
    } else {
        panic("Failed while parsing size")
    }

    return sizeRequested
}

func (volumeBase *VolumeBase) reuseVolume(mountDir string, sizeMb int) string {
    lvName := volumeBase.dbManager.GetFreeVolume(sizeMb)
    if lvName != "" {
        podId := volumeBase.getPodIdFromMountDir(mountDir)
        volumeBase.dbManager.MarkVolumeInUse(lvName, mountDir, podId)
        // Mount the volume on the user directory
        volumeBase.mountLogicalVolume(lvName, mountDir)
        log.Printf("Reusing LV: %s", lvName)
        return lvName
    }
    return ""
}

func (volumeBase *VolumeBase) markVolumeFree(mountDir string) {
    volumeBase.dbManager.MarkVolumeFree(mountDir)
}

type AnonymousVolume struct {
    base VolumeBase
    mountDir string
    mountDev string
    optMap map[string]interface{}
}

func (anonymousVolume *AnonymousVolume) Mount() string {
    sizeRequested := getSizeOpt(anonymousVolume.optMap)

    var lvName string
    if anonymousVolume.base.getFreeSize() > sizeRequested {
        lvName = anonymousVolume.base.createVolume("ax-anonymous", sizeRequested, anonymousVolume.mountDir, anonymousVolume.optMap)
    } else {
        log.Printf("Not enough disk space ...")
    }

    return lvName
}

func (anonymousVolume *AnonymousVolume) Unmount() {
    anonymousVolume.base.deleteVolume(anonymousVolume.mountDir)
}

type DGSVolume struct {
    base VolumeBase
    mountDir string
    mountDev string
    optMap map[string]interface{}
}

func (dgsVolume *DGSVolume) Mount() string {
    sizeRequested := getSizeOpt(dgsVolume.optMap)

    // 1. Find free volume matching size
    lvName := dgsVolume.base.reuseVolume(dgsVolume.mountDir, sizeRequested)
    if lvName == "" {
        // No reuse of volume. Try to create new volume
        if dgsVolume.base.getFreeSize() > sizeRequested {
            lvName = dgsVolume.base.createVolume("ax-docker-graph-storage", sizeRequested, dgsVolume.mountDir, dgsVolume.optMap)
        } else {
            // Couldn't create new volume. Reclaim all free volumes.
            // dgsVolume.base.ReclaimVolumes(sizeRequested)
            dbManager := dgsVolume.base.dbManager
            lvNames := dbManager.GetAllFreeVolumes()
            log.Printf("Volumes to reclaim: %s", lvNames)

            for _, lvName := range lvNames {
                // Delete the logical volume.
                dgsVolume.base.deleteLogicalVolume(lvName)
                // Remove the entry from the DB.
                dgsVolume.base.dbManager.DeleteVolume(lvName)
            }

            // Check if there is enough free size available.
            if dgsVolume.base.getFreeSize() > sizeRequested {
                lvName = dgsVolume.base.createVolume("ax-docker-graph-storage", sizeRequested, dgsVolume.mountDir, dgsVolume.optMap)
            } else {
                log.Printf("Not enough disk space ...")
            }
        }
    }
    return lvName
}

func (dgsVolume *DGSVolume) Unmount() {
    var freePercentage float64
    var stat syscall.Statfs_t
    err := syscall.Statfs(dgsVolume.mountDir, &stat)
	if err != nil {
		log.Printf("Failed to find volume usage. Ignoring error ...")
	} else {
        freePercentage = 100 * float64(stat.Bfree)/float64(stat.Blocks)
    }

    if freePercentage < 50.0 {
        log.Printf("Volume free % %.2f. Deleting ...", freePercentage)
        dgsVolume.base.deleteVolume(dgsVolume.mountDir)
    } else {
        log.Printf("Volume free % %.2f. Keeping volume for reuse", freePercentage)
        dgsVolume.base.unmountDir(dgsVolume.mountDir)
        dgsVolume.base.markVolumeFree(dgsVolume.mountDir)
    }
}