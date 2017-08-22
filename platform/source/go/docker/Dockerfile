FROM golang:1.7

RUN apt-get update && apt-get install python2.7 && \
    apt-get install -y python-pip=1.5.6-5 && \
    pip install awscli==1.11.6 --upgrade --ignore-installed six

ENV GOPATH=/source/platform/source/go
RUN go get github.com/mattn/go-sqlite3 && go get github.com/satori/go.uuid

# applatix.io should be volume mounted from the host
WORKDIR /source/platform/source/go/src/applatix.io/vol_plugin
