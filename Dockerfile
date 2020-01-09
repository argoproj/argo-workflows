FROM golang:1.11.5 as builder

RUN apt-get update && apt-get install -y \
    git \
    make \
    wget \
    gcc \
    zip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /tmp

ENV DOCKER_CHANNEL stable
ENV DOCKER_VERSION 18.09.1
RUN wget -O docker.tgz "https://download.docker.com/linux/static/${DOCKER_CHANNEL}/x86_64/docker-${DOCKER_VERSION}.tgz" && \
    tar --extract --file docker.tgz --strip-components 1 --directory /usr/local/bin/ && \
    rm docker.tgz

FROM debian:9.6-slim as argoexec
# NOTE: keep the version synced with https://storage.googleapis.com/kubernetes-release/release/stable.txt
ENV KUBECTL_VERSION=1.15.1
RUN apt-get update && \
    apt-get install -y curl jq procps git tar mime-support && \
    rm -rf /var/lib/apt/lists/* && \
    curl -L -o /usr/local/bin/kubectl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/local/bin/kubectl
COPY hack/ssh_known_hosts /etc/ssh/ssh_known_hosts
COPY --from=builder /usr/local/bin/docker /usr/local/bin/
COPY argoexec /usr/local/bin/
ENTRYPOINT [ "argoexec" ]

FROM scratch as workflow-controller
COPY workflow-controller /bin/
ENTRYPOINT [ "workflow-controller" ]

FROM scratch as argocli
COPY argo /bin/
ENTRYPOINT [ "argo" ]

FROM scratch as argo-server
COPY argo-server /bin/
ENTRYPOINT [ "argo-server" ]
