# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.234.0/containers/debian/.devcontainer/base.Dockerfile

# [Choice] Debian version (use bullseye on local arm64/Apple Silicon): bullseye, buster
ARG VARIANT="buster"
FROM --platform=linux/amd64 mcr.microsoft.com/vscode/devcontainers/base:0-${VARIANT}

ARG PROTOC_ZIP="protoc-3.14.0-linux-x86_64.zip"
RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/${PROTOC_ZIP} \
    && sudo unzip -o ${PROTOC_ZIP} -d /usr/local bin/protoc \
    && sudo unzip -o ${PROTOC_ZIP} -d /usr/local 'include/*' \
    && rm -f ${PROTOC_ZIP} \
    && chmod 755 /usr/local/bin/protoc \
    && sudo chmod -R 755 /usr/local/include/ \
    && wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
