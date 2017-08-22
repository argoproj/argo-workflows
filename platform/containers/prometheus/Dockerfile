# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
# Dockerfile for prometheus

FROM %%ARGO_BASE_REGISTRY%%/argobase/prometheus:v1.6.2

ADD prometheus.yml /etc/prometheus/prometheus.yml
ADD run.sh /run.sh
RUN chmod +x /run.sh

ENTRYPOINT "/run.sh"
