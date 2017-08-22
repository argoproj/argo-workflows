apt-get install -y lxc wget bsdtar curl
apt-get install -y linux-image-extra-$(uname -r)
modprobe aufs

modprobe overlay

curl -sSl https://get.docker.com |sh
service docker stop
echo 'DOCKER_OPTS="--storage-driver=overlay"' >> /etc/default/docker
service docker start
