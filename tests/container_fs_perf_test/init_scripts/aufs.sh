apt-get install -y lxc wget bsdtar curl
apt-get install -y linux-image-extra-$(uname -r)
modprobe aufs
curl -sSl https://get.docker.com |sh
service docker stop
rm -fr /var/lib/docker
mkdir -p /var/lib/docker
echo 'DOCKER_OPTS="--storage-driver=aufs"' >> /etc/default/docker
service docker start

