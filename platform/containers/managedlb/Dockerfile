# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
# Dockerfile for axmon debug build

FROM %%BUILDER_IMAGE_ID%%

RUN apk update && apk add ca-certificates && update-ca-certificates
RUN mkdir -p /root/.ssh/ \
    && touch /root/.ssh/known_hosts
COPY _retry.json /root/.aws/models/_retry.json

COPY docker_build/root /


ENV PYTHONPATH /ax/python
CMD /ax/bin/managed_elb_creator
