# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.234.0/containers/debian/.devcontainer/base.Dockerfile

ARG VARIANT="bullseye"
FROM --platform=linux/amd64 mcr.microsoft.com/vscode/devcontainers/base:0-${VARIANT}

ARG NEW_USERNAME=$(whoami)
ARG HOME=/root
ARG OLD_USERNAME="vscode"
ARG VSCODE_SCRIPT_VERSION="v0.234.0"

ARG PROTOC_ZIP="protoc-3.14.0-linux-x86_64.zip"
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y apt-transport-https ca-certificates git sudo \
    # use new user instead of vscode user
    && usermod -l $NEW_USERNAME -d /home/$NEW_USERNAME -m $OLD_USERNAME \
    && groupmod -n $NEW_USERNAME $OLD_USERNAME \
    && echo $NEW_USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$NEW_USERNAME \
    && chmod 0440 /etc/sudoers.d/$NEW_USERNAME \
    # kubectl
    && curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list \
    && apt-get update \
    && apt-get install -y kubectl \
    # protobuf
    && curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/${PROTOC_ZIP} \
    && unzip -o ${PROTOC_ZIP} -d /usr/local bin/protoc \
    && unzip -o ${PROTOC_ZIP} -d /usr/local 'include/*' \
    && rm -f ${PROTOC_ZIP} \
    && chmod 755 /usr/local/bin/protoc \
    && chmod -R 755 /usr/local/include/ \
    # k3d
    && wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash \
    # go
    && bash -ec "$(curl -fsSL "https://raw.githubusercontent.com/microsoft/vscode-dev-containers/$VSCODE_SCRIPT_VERSION/script-library/go-debian.sh")" -- "1.18" "/usr/local/go" "$HOME/go" "automatic" "true" "false" \
    # dind
    && bash -ec "$(curl -fsSL "https://raw.githubusercontent.com/microsoft/vscode-dev-containers/$VSCODE_SCRIPT_VERSION/script-library/docker-in-docker-debian.sh")" -- "true" "automatic" "true" "20.10" "v1" \
    # node
    && bash -ec "$(curl -fsSL "https://raw.githubusercontent.com/microsoft/vscode-dev-containers/$VSCODE_SCRIPT_VERSION/script-library/node-debian.sh")" -- "/usr/local/share/nvm" "16" "automatic" "true" "true" \
    # python
    && bash -ec "$(curl -fsSL "https://raw.githubusercontent.com/microsoft/vscode-dev-containers/$VSCODE_SCRIPT_VERSION/script-library/python-debian.sh")" -- "3.9" \
    && apt-get clean -y && rm -rf /var/lib/apt/lists/*
