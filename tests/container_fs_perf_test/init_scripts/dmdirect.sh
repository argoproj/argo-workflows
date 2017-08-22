apt-get install -y lxc wget bsdtar curl
apt-get install -y linux-image-extra-$(uname -r)
modprobe aufs

(echo n; echo p; echo 1; echo; echo; echo w; echo q) | /sbin/fdisk /dev/sdb
apt-get install -y lvm2
pvcreate /dev/sdb1
vgcreate vg-docker /dev/sdb1
lvcreate -L 36G -n data vg-docker
lvcreate -L 3G -n metadata vg-docker

curl -sSl https://get.docker.com |sh
service docker stop
rm -fr /var/lib/docker
mkdir -p /var/lib/docker
echo 'DOCKER_OPTS="--storage-driver=devicemapper --storage-opt dm.datadev=/dev/vg-docker/data --storage-opt dm.metadatadev=/dev/vg-docker/metadata"' >> /etc/default/docker
service docker start
