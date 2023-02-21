FROM mcr.microsoft.com/vscode/devcontainers/universal

ARG PROTOC_ZIP="protoc-3.14.0-linux-x86_64.zip"

RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/${PROTOC_ZIP} \
    && unzip -o ${PROTOC_ZIP} -d /usr/local bin/protoc \
    && unzip -o ${PROTOC_ZIP} -d /usr/local 'include/*' \
    && rm -f ${PROTOC_ZIP} \
    && chmod 755 /usr/local/bin/protoc \
    && chmod -R 755 /usr/local/include/ 
