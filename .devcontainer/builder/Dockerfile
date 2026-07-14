# TODO: Upgrade to Ubuntu 24.04 when https://bugs.launchpad.net/ubuntu/+source/curl/+bug/2073448 is addressed
FROM mcr.microsoft.com/vscode/devcontainers/base:ubuntu-22.04

# TARGETARCH is automatically set by BuildKit to the architecture (e.g. "amd64" or "arm64")
# Docs: https://docs.docker.com/reference/dockerfile/#automatic-platform-args-in-the-global-scope
ARG TARGETARCH

# Use bash to allow us to source hack/k8s-versions.sh
SHELL ["/bin/bash", "-c"]

# install protocol buffer tools
RUN apt-get update && apt-get install -y protobuf-compiler clang-format

# install kubectl
COPY hack/k8s-versions.sh /tmp/
RUN . /tmp/k8s-versions.sh && \
  wget -O /usr/local/bin/kubectl "https://dl.k8s.io/release/${K8S_VERSIONS[min]}/bin/linux/${TARGETARCH}/kubectl" && \
  chmod +x /usr/local/bin/kubectl
