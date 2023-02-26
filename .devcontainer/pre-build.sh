#!/usr/bin/env sh
set -eux

# install protoc
curl -Lo protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/protoc-3.14.0-linux-x86_64.zip \
    && unzip -o protoc.zip -d /usr/local bin/protoc \
    && unzip -o protoc.zip -d /usr/local 'include/*' \
    && rm -f protoc.zip \
    && chmod 755 /usr/local/bin/protoc \
    && chmod -R 755 /usr/local/include/ 


# install kit
curl -q https://raw.githubusercontent.com/kitproj/kit/main/install.sh | sh

pwd
ls

# do time consuming tasks, e.g. download deps and initial build
CI=1 kit pre-up