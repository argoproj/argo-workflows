#!/bin/bash

# Copyright 2015 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Discover all the ephemeral disks

function format_disk_ext4() {
    local device=$1
    local opts=$2
    set +e
    counter=0

    echo "Formatting disk: " $device $opts
    while [[ $counter -le 20 ]]; do
        mkfs -t ext4 $opts $device && break
        counter=$((counter+1))
        echo "Retrying after 10 seconds..."
        sleep 10
    done

    if [ $counter -eq 21 ]; then
        echo "Failed to format disks. Shutting down now..."
        shutdown -h now
    fi
    set -e
}


# Mount or die if mount really fails
function idempotent-mount() {
    local mount_target=$1

    # Should not exit upon error in this function unless
    # do it explicitly
    set +e
    mount_output=$(mount ${mount_target} 2>&1)
    mount_ret=$?

    echo "AX_DEBUG::idempotent-mount::mount_output: [${mount_output}]"
    echo "AX_DEBUG::idempotent-mount::mount_ret: [${mount_ret}]"

    if [[ "${mount_ret}" == "0" ]]; then
        echo "AX_DEBUG::idempotent-mount: Mount is successful"
    else
        if [[ "${mount_output}" == *"is already mounted on ${mount_target}"* ]]; then
            echo "AX_DEBUG::idempotent-mount: Mount is successful because it is already mounted"
        else
            echo "AX_DEBUG::idempotent-mount: Mount really failed"
            exit 1
        fi
    fi

    # set it back as main bootstrap script
    set -e
}


function ensure-local-disks() {

# Skip if already mounted (a reboot)
if ( grep "/mnt/ephemeral" /proc/mounts ); then
  echo "Found /mnt/ephemeral in /proc/mounts; skipping local disk initialization"
  return
fi

block_devices=()

ephemeral_devices=$( (curl --silent http://169.254.169.254/2014-11-05/meta-data/block-device-mapping/ | grep ephemeral) || true )
for ephemeral_device in $ephemeral_devices; do
  echo "Checking ephemeral device: ${ephemeral_device}"
  aws_device=$(curl --silent http://169.254.169.254/2014-11-05/meta-data/block-device-mapping/${ephemeral_device})

  device_path=""
  if [ -b /dev/$aws_device ]; then
    device_path="/dev/$aws_device"
  else
    # Check for the xvd-style name
    xvd_style=$(echo $aws_device | sed "s/sd/xvd/")
    if [ -b /dev/$xvd_style ]; then
      device_path="/dev/$xvd_style"
    fi
  fi

  if [[ -z ${device_path} ]]; then
    echo "  Could not find disk: ${ephemeral_device}@${aws_device}"
  else
    echo "  Detected ephemeral disk: ${ephemeral_device}@${device_path}"
    block_devices+=(${device_path})
  fi
done

# These are set if we should move where docker/kubelet store data
# Note this gets set to the parent directory
move_docker=""
move_kubelet=""

docker_storage=${DOCKER_STORAGE:-aufs}

# Format the ephemeral disks
if [[ ${#block_devices[@]} == 0 ]]; then
  echo "No ephemeral block devices found; will use aufs on root"
  docker_storage="aufs"
else
  echo "Block devices: ${block_devices[@]}"

  # Remove any existing mounts
  for block_device in ${block_devices}; do
    echo "Unmounting ${block_device}"
    /bin/umount ${block_device} || echo "Ignoring failure umounting ${block_device}"
    sed -i -e "\|^${block_device}|d" /etc/fstab
  done

  # Remove any existing /mnt/ephemeral entry in /etc/fstab
  sed -i -e "\|/mnt/ephemeral|d" /etc/fstab

  # Mount the storage
  if [[ ${docker_storage} == "btrfs" ]]; then
    apt-get-install btrfs-tools

    if [[ ${#block_devices[@]} == 1 ]]; then
      echo "One ephemeral block device found; formatting with btrfs"
      mkfs.btrfs -f ${block_devices[0]}
    else
      echo "Found multiple ephemeral block devices, formatting with btrfs as RAID-0"
      mkfs.btrfs -f --data raid0 ${block_devices[@]}
    fi
    echo "${block_devices[0]}  /mnt/ephemeral  btrfs  noatime,nofail  0 0" >> /etc/fstab
    mkdir -p /mnt/ephemeral
    idempotent-mount /mnt/ephemeral

    mkdir -p /mnt/ephemeral/kubernetes

    move_docker="/mnt/ephemeral"
    move_kubelet="/mnt/ephemeral/kubernetes"
  elif [[ ${docker_storage} == "aufs-nolvm" ]]; then
    if [[ ${#block_devices[@]} != 1 ]]; then
      echo "aufs-nolvm selected, but multiple ephemeral devices were found; only the first will be available"
    fi

    format_disk_ext4 ${block_devices[0]} ""
    echo "${block_devices[0]}  /mnt/ephemeral  ext4     noatime,nofail  0 0" >> /etc/fstab
    mkdir -p /mnt/ephemeral
    idempotent-mount /mnt/ephemeral

    mkdir -p /mnt/ephemeral/kubernetes

    move_docker="/mnt/ephemeral"
    move_kubelet="/mnt/ephemeral/kubernetes"
  elif [[ ${docker_storage} == "devicemapper" || ${docker_storage} == "aufs" || ${docker_storage} == "overlay" || ${docker_storage} == "overlay2" ]]; then
    # We always use LVM, even with one device
    # In devicemapper mode, Docker can use LVM directly
    # Also, fewer code paths are good
    echo "Using LVM2 and ext4"
    apt-get-install lvm2

    # Don't output spurious "File descriptor X leaked on vgcreate invocation."
    # Known bug: e.g. Ubuntu #591823
    export LVM_SUPPRESS_FD_WARNINGS=1

    for block_device in ${block_devices}; do
      pvcreate ${block_device}
    done
    vgcreate vg-ephemeral ${block_devices[@]}

    if [[ ${docker_storage} == "devicemapper" ]]; then
      # devicemapper thin provisioning, managed by docker
      # This is the best option, but it is sadly broken on most distros
      # Bug: https://github.com/docker/docker/issues/4036

      # 80% goes to the docker thin-pool; we want to leave some space for host-volumes
      AX_DEVICE_MAPPER_DOCKER_DATA=${AX_DEVICE_MAPPER_DOCKER_DATA:-75}
      AX_DEVICE_MAPPER_DOCKER_META=${AX_DEVICE_MAPPER_DOCKER_META:-5}
      lvcreate -l ${AX_DEVICE_MAPPER_DOCKER_DATA}%VG -n docker-data vg-ephemeral
      lvcreate -l ${AX_DEVICE_MAPPER_DOCKER_META}%VG -n docker-meta vg-ephemeral

      DOCKER_OPTS="${DOCKER_OPTS:-} --storage-opt dm.datadev=/dev/mapper/vg--ephemeral-docker--data --storage-opt dm.metadatadev=/dev/mapper/vg--ephemeral-docker--meta"
      # Note that we don't move docker; docker goes direct to the thinpool

      # Remaining space (20%) is for kubernetes data
      # TODO: Should this be a thin pool?  e.g. would we ever want to snapshot this data?
      lvcreate -l 100%FREE -n kubernetes vg-ephemeral

      format_disk_ext4 /dev/vg-ephemeral/kubernetes ""

      mkdir -p /mnt/ephemeral/kubernetes
      echo "/dev/vg-ephemeral/kubernetes  /mnt/ephemeral/kubernetes  ext4  noatime,nofail  0 0" >> /etc/fstab
      idempotent-mount /mnt/ephemeral/kubernetes

      move_docker="/mnt/ephemeral"
      move_kubelet="/mnt/ephemeral/kubernetes"
     else
      # aufs, overlay or overlay2
      # We used to split docker & kubernetes, but we no longer do that, because
      # host volumes go into the kubernetes area, and it is otherwise very easy
      # to fill up small volumes.
      #
      # No need for thin pool since we are not over-provisioning or doing snapshots
      # (probably shouldn't be doing snapshots on ephemeral disk? Should be stateless-ish.)
      # Tried to do it, but it cause problems (#16188)

      lvcreate -l 100%VG -n ephemeral vg-ephemeral
      if [[ ${docker_storage} == "overlay" ]]; then
          format_disk_ext4 /dev/vg-ephemeral/ephemeral "-i 4096"
      else
          format_disk_ext4 /dev/vg-ephemeral/ephemeral ""
      fi
      mkdir -p /mnt/ephemeral
      echo "/dev/vg-ephemeral/ephemeral  /mnt/ephemeral  ext4  noatime,nofail 0 0" >> /etc/fstab
      idempotent-mount /mnt/ephemeral

      mkdir -p /mnt/ephemeral/kubernetes

      move_docker="/mnt/ephemeral"
      move_kubelet="/mnt/ephemeral/kubernetes"
     fi
 else
    echo "Ignoring unknown DOCKER_STORAGE: ${docker_storage}"
  fi
fi


if [[ ${docker_storage} == "btrfs" ]]; then
  DOCKER_OPTS="${DOCKER_OPTS:-} -s btrfs"
elif [[ ${docker_storage} == "aufs-nolvm" || ${docker_storage} == "aufs" ]]; then
  # Install aufs kernel module
  # Fix issue #14162 with extra-virtual
  if [[ `lsb_release -i -s` == 'Ubuntu' ]]; then
    apt-get-install linux-image-extra-$(uname -r) linux-image-extra-virtual
  fi

  # Install aufs tools
  apt-get-install aufs-tools

  DOCKER_OPTS="${DOCKER_OPTS:-} -s aufs"
elif [[ ${docker_storage} == "devicemapper" ]]; then
  DOCKER_OPTS="${DOCKER_OPTS:-} -s devicemapper"
elif [[ ${docker_storage} == "overlay" ]]; then
  modprobe overlay
  DOCKER_OPTS="${DOCKER_OPTS:-} -s overlay"
elif [[ ${docker_storage} == "overlay2" ]]; then
  modprobe overlay
  DOCKER_OPTS="${DOCKER_OPTS:-} -s overlay2"
else
  echo "Ignoring unknown DOCKER_STORAGE: ${docker_storage}"
fi

if [[ -n "${move_docker}" ]]; then
  # Stop docker if it is running, so we can move its files
  systemctl stop docker || true

  # Move docker to e.g. /mnt
  # but only if it is a directory, not a symlink left over from a previous run
  if [[ -d /var/lib/docker ]]; then
    # This racing condition happens more often when node is "stopped" and then "started",
    # rather than rebooted.
    # As /var/lib/docker is already created as a link from previous run, it is possible
    # that systemd haven't mounted according to /etc/fstab set by previous run by the time
    # ensure-local-disk() checks /proc/mounts. If this happens, docker can start before
    # this code path, and created a directory at its default root path (/var/lib/docker),
    # and get linked to /mnt/ephemeral/docker.
    # Under this scenario, [[ -d /var/lib/docker ]] will return true, and we ended up moving
    # 2 files that are the same, which is not desired.
    #
    # Same scenario for move kubelet
    if [[ -L /var/lib/docker ]]; then
      echo "AX_DEBUG::ensure-local-disks::mv: ${move_docker}/docker is already created, skip 'mv'."
    else
      mv /var/lib/docker ${move_docker}/ || true
    fi
  fi
  mkdir -p ${move_docker}/docker
  # If /var/lib/docker doesn't exist (it will exist if it is already a symlink),
  # then symlink it to the ephemeral docker area
  if [[ ! -e /var/lib/docker ]]; then
    ln -s ${move_docker}/docker /var/lib/docker
  fi
  DOCKER_ROOT="${move_docker}/docker"
  DOCKER_OPTS="${DOCKER_OPTS:-} -g ${DOCKER_ROOT}"
fi

if [[ -n "${move_kubelet}" ]]; then
  # Move /var/lib/kubelet to e.g. /mnt
  # (the backing for empty-dir volumes can use a lot of space!)
  # (As with /var/lib/docker, only if it is a directory; skip if symlink)
  if [[ -d /var/lib/kubelet ]]; then
    if [[ -L /var/lib/kubelet ]]; then
      echo "AX_DEBUG::ensure-local-disks::mv: ${move_kubelet}/kubelet is already created, skip 'mv'."
    else
      mv /var/lib/kubelet ${move_kubelet}/ || true
    fi
  fi
  mkdir -p ${move_kubelet}/kubelet
  # Create symlink for /var/lib/kubelet, unless it is already a symlink
  if [[ ! -e /var/lib/kubelet ]]; then
    ln -s ${move_kubelet}/kubelet /var/lib/kubelet
  fi
  KUBELET_ROOT="${move_kubelet}/kubelet"
fi

}
