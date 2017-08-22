#!/bin/bash

set -ex

apt-get update
apt-get install -y gnupg2
export GNUPGHOME="$(mktemp -d)"
# pub   4096R/A15703C6 2016-01-11 [expires: 2018-01-10]
#       Key fingerprint = 0C49 F373 0359 A145 1858  5931 BC71 1F9B A157 03C6
# uid                  MongoDB 3.4 Release Signing Key <packaging@mongodb.com>
export GPG_KEYS=0C49F3730359A14518585931BC711F9BA15703C6
for key in $GPG_KEYS; do
        gpg --keyserver ha.pool.sks-keyservers.net --recv-keys "$key"
done
gpg --export $GPG_KEYS > /etc/apt/trusted.gpg.d/mongodb.gpg
rm -r "$GNUPGHOME"
apt-key list

echo "deb http://repo.mongodb.org/apt/debian jessie/mongodb-org/$MONGO_MAJOR main" > /etc/apt/sources.list.d/mongodb-org.list
apt-get update
apt-get install -y \
        ${MONGO_PACKAGE}=$MONGO_VERSION \
        ${MONGO_PACKAGE}-server=$MONGO_VERSION \
        ${MONGO_PACKAGE}-shell=$MONGO_VERSION \
        ${MONGO_PACKAGE}-mongos=$MONGO_VERSION \
        ${MONGO_PACKAGE}-tools=$MONGO_VERSION

pip3 install pymongo==3.4.0
find /usr/local/lib/python3.4 -name *.pyc | xargs rm -f

apt-get purge -y gnupg2
apt-get autoremove -y
rm -rf /var/lib/apt/lists/*
rm -rf /usr/share/locale /usr/share/doc /usr/share/man
rm -rf /var/lib/mongodb
mkdir -p /data/db /data/configdb
chown -R mongodb:mongodb /data/db /data/configdb
