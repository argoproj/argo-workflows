# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
# Dockerfile for axmon

FROM %%BUILDER_IMAGE_ID%%

RUN mkdir -p /root/.ssh/ \
    && touch /root/.ssh/known_hosts
COPY _retry.json /root/.aws/models/_retry.json

# add terraform
RUN apk update && apk --no-cache add ca-certificates && update-ca-certificates && \
    wget https://releases.hashicorp.com/terraform/0.9.5/terraform_0.9.5_linux_amd64.zip && \
    curl https://releases.hashicorp.com/terraform/0.9.5/terraform_0.9.5_SHA256SUMS | grep linux_amd64 > SHASUMS && \
    sha256sum -c SHASUMS && \
    unzip -d /usr/bin/ terraform_0.9.5_linux_amd64.zip && \
    rm terraform_0.9.5_linux_amd64.zip SHASUMS

COPY docker_build/root /

RUN curl -o /bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.6.7/bin/linux/amd64/kubectl && chmod +x /bin/kubectl
ADD restart_pod_for_eip.sh /ax/bin/restart_pod_for_eip.sh

ENV PYTHONPATH /ax/python
EXPOSE 8901
CMD /ax/bin/axmon
