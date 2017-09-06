#!/bin/bash

set -e
apt-get update
apt-get install -y \
        curl \
        git \
        maven \
        wget \
        vim \
        bc \
        procps \
        gnupg2

# oracle jdk
mkdir -p /opt/jdk
cd /opt
wget --header "Cookie: oraclelicense=accept-securebackup-cookie" http://download.oracle.com/otn-pub/java/jdk/8u144-b01/090f390dda5b47b9b721c7dfaa008135/jdk-8u144-linux-x64.tar.gz
tar -zxf jdk-8u144-linux-x64.tar.gz -C /opt/jdk
update-alternatives --install /usr/bin/java java /opt/jdk/jdk1.8.0_144/bin/java 10000
rm jdk-8u144-linux-x64.tar.gz

# install Cassandra
echo "deb http://debian.datastax.com/datastax-ddc 3.7 main" | tee -a /etc/apt/sources.list.d/cassandra.sources.list
curl -L https://debian.datastax.com/debian/repo_key | apt-key add -
apt-get update -y && apt-get install -y \
        datastax-ddc python-cassandra
service cassandra stop
rm -rf /var/lib/cassandra/data/system/*

# install cassandra-lucene-index
cd /root && \
    git clone http://github.com/Stratio/cassandra-lucene-index && \
    cd cassandra-lucene-index && \
    git checkout 3.7.1 && \
    mvn clean package
cp /root/cassandra-lucene-index/plugin/target/cassandra-lucene-index-plugin-*.jar /usr/share/cassandra/lib
rm -rf /root/cassandra-lucene-index

# Clean up unnecessary files.
rm -f /opt/jdk/jdk1.8.0_144/src.zip /opt/jdk/jdk1.8.0_144/javafx-src.zip
find /usr/lib/python2.7 -name "*.pyc" | xargs rm
rm -rf /root/.m2
apt-get purge -y maven git gnupg2
apt-get autoremove -y
apt-get clean -y
rm -rf /var/lib/apt/lists
rm -rf /usr/share/locale /usr/share/doc /usr/share/man
