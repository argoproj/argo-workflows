apt-get install -y lxc wget bsdtar curl
apt-get install -y linux-image-extra-$(uname -r)
modprobe aufs

(echo n; echo p; echo 1; echo; echo; echo w; echo q) | /sbin/fdisk /dev/sdb
modprobe btrfs
apt-get install -y btrfs-tools


mkfs.btrfs -f /dev/sdb1
mkdir -p /var/lib/docker

UUID=`blkid /dev/sdb1 |grep -o " UUID=\S\+"`; echo /dev/sdb1 /var/lib/docker btrfs defaults 0 0 $UUID /var/lib/docker btrfs defaults 0 0 >> /etc/fstab
mount -a

curl -sSl https://get.docker.com |sh
service docker stop
echo 'DOCKER_OPTS="--storage-driver=btrfs"' >> /etc/default/docker
service docker start
