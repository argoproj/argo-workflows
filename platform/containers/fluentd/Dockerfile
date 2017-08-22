# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
# Dockerfile for fluentd

FROM gcr.io/google-containers/fluentd-elasticsearch:1.23

ADD td-agent.conf /etc/td-agent/td-agent.conf
ADD ax-start.sh /ax-start.sh
RUN chmod +x /ax-start.sh
ENTRYPOINT /ax-start.sh
