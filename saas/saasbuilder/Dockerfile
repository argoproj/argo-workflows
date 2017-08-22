# Dockerfile for desktop dev environment.
FROM golang:1.7.6

# Basic utilities.
RUN apt-get update -y && \
    apt-get install -y \
		apt-utils \
		vim \
		less \
		psmisc \
		sudo \
		git  \
		vim \
        python3 \
        maven && \
    rm -rf /var/lib/apt/lists/* && \

    # pip
    curl -O https://bootstrap.pypa.io/get-pip.py && \
    python get-pip.py && \
    python3 get-pip.py && \
    rm -f get-pip.py && \

    # oracle jdk
    mkdir /opt/jdk && \
    wget --header "Cookie: oraclelicense=accept-securebackup-cookie" http://download.oracle.com/otn-pub/java/jdk/8u131-b11/d54c1d3a095b4ff2b6607d096fa80163/jdk-8u131-linux-x64.tar.gz && \
    tar -zxf jdk-8u131-linux-x64.tar.gz -C /opt/jdk && \
    update-alternatives --install /usr/bin/java java /opt/jdk/jdk1.8.0_131/bin/java 10000 && \
    update-alternatives --install /usr/bin/javac javac /opt/jdk/jdk1.8.0_131/bin/javac 10000 && \
    rm -f /go/jdk-8u131-linux-x64.tar.gz && \

    # install Cassandra
    echo "deb http://debian.datastax.com/datastax-ddc 3.7 main" | tee -a /etc/apt/sources.list.d/cassandra.sources.list && \
    curl -L https://debian.datastax.com/debian/repo_key | apt-key add - && \
    apt-get update -y && \
    apt-get install -y datastax-ddc && \
    rm -rf /var/lib/apt/lists/* && \
    service cassandra stop && \
    rm -rf /var/lib/cassandra/data/system/* && \
    # not allowed in container
    sed '/ulimit -l unlimited/d' /etc/init.d/cassandra > /tmp/cassandra && \
    cp /tmp/cassandra /etc/init.d/cassandra

ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 1.11.2
ENV DOCKER_SHA256 8c2e0c35e3cda11706f54b2d46c2521a6e9026a7b13c7d4b8ae1f3a706fc55e1

RUN cd /usr/bin && \
    curl -fSL "https://${DOCKER_BUCKET}/builds/Linux/x86_64/docker-$DOCKER_VERSION.tgz" -o docker.tgz && \
    echo "${DOCKER_SHA256} *docker.tgz" | sha256sum -c - && \
    tar -xzvf docker.tgz && \
    mv docker/* /usr/local/bin/ && \
    rmdir docker && \
    rm docker.tgz && \
    docker -v

# 7000: intra-node communication
# 7001: TLS intra-node communication
# 7199: JMX
# 9042: CQL
# 9160: thrift service
EXPOSE 7000 7001 7199 9042 9160

# install cassandra-lucene-index
RUN cd /root && \
    git clone http://github.com/Stratio/cassandra-lucene-index && \
    cd cassandra-lucene-index && \
    git checkout 3.7.0 && \
    mvn clean package && \
    cp /root/cassandra-lucene-index/plugin/target/cassandra-lucene-index-plugin-*.jar /usr/share/cassandra/lib && \
    rm -rf /root/cassandra-lucene-index

EXPOSE 2181 9092  8091 8092

# install confluent pacakge
# First install Confluent’s public key, which is used to sign the packages in the apt repository.
RUN wget -qO - http://packages.confluent.io/deb/3.0/archive.key | sudo apt-key add - && \

    #Add the repository
    #sudo add-apt-repository "deb http://packages.confluent.io/deb/3.0 stable main"
    echo "deb http://packages.confluent.io/deb/3.0 stable main" >> /etc/apt/sources.list && \

    # install confluent platform
    sudo apt-get update -y && \
    sudo apt-get install -y confluent-platform-2.11 && \

    # seed provider library dependency
    cd /root && \
    git clone https://github.com/kubernetes/kubernetes.git && \
    cd kubernetes/examples/storage/cassandra/java && \
    mvn clean package && \
    mv target/kubernetes-cassandra*.jar /usr/share/cassandra/lib/kubernetes-cassandra.jar && \
    rm -rf /root/kubernetes && \
    apt-get install -y xmlsec1 && \
    rm -rf /var/lib/apt/lists/*

# AWS CLI
RUN curl "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" -o "awscli-bundle.zip" && \
    unzip awscli-bundle.zip && \
    ./awscli-bundle/install -i /usr/local/aws -b /usr/local/bin/aws && \
    rm -rf awscli-bundle && \
    rm awscli-bundle.zip

# swagger to generate the documentation
RUN go get github.com/yvasiyarov/swagger && \
    go install github.com/yvasiyarov/swagger

COPY glide.yaml /go/glide.yaml
COPY glide.lock /go/glide.lock

RUN curl https://glide.sh/get | sh && \
    cd /go && \
    glide install --strip-vendor && \
    rm -rf src && \
    mv vendor src && \
    rm -rf /go/.glide && \
    chmod -R 777 /go

WORKDIR /
