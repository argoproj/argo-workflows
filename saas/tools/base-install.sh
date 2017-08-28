#!bin/bash

apt-get update
apt-get install -y \
        curl \
        wget \
        vim  \
        bc \
        netcat \
        gnupg2

# oracle jdk
mkdir -p /opt/jre
cd /opt
wget --header "Cookie: oraclelicense=accept-securebackup-cookie" http://download.oracle.com/otn-pub/java/jdk/8u144-b01/090f390dda5b47b9b721c7dfaa008135/jre-8u144-linux-x64.tar.gz
tar -zxf jre-8u144-linux-x64.tar.gz -C /opt/jre
update-alternatives --install /usr/bin/java java /opt/jre/jre1.8.0_144/bin/java 10000
rm jre-8u144-linux-x64.tar.gz

# install confluent pacakge
wget -qO - http://packages.confluent.io/deb/3.1/archive.key | apt-key add -
echo "deb http://packages.confluent.io/deb/3.1 stable main" >> /etc/apt/sources.list
apt-get update && apt-get install -y confluent-platform-2.11

# Clean up unnecessary files.
apt-get purge -y gnupg2
apt-get autoremove -y
apt-get clean -y
rm -rf /var/lib/apt/lists
rm -rf /usr/share/locale /usr/share/doc /usr/share/man
