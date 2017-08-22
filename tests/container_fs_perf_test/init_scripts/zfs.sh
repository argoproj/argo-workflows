apt-get install software-properties-common
add-apt-repository ppa:zfs-native/stable
apt-get update
apt-get install -y ubuntu-zfs
modprobe zfs
apt-get install -y lxc wget bsdtar curl
apt-get install -y linux-image-extra-$(uname -r)
modprobe aufs
(echo n; echo p; echo 1; echo; echo; echo w; echo q) | /sbin/fdisk /dev/sdb
zpool create -f zpool-docker /dev/sdb1
curl -sSl https://get.docker.com |sh
service docker stop
rm -fr /var/lib/docker
mkdir -p /var/lib/docker
zfs create -o mountpoint=/var/lib/docker zpool-docker/docker
echo 'DOCKER_OPTS="--storage-driver=zfs"' >> /etc/default/docker
echo 'zfs mount -a' >> /etc/rc.local
service docker start
