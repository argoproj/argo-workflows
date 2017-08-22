# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
# Dockerfile for artifacts container

FROM %%BUILDER_IMAGE_ID%%

ENV PYTHONPATH=/ax/python

ADD run.sh /run.sh
RUN mkdir -p /ax-execu-host/art
RUN curl -sSL -o /ax-execu-host/art/ax_tar_ax https://s3-us-west-1.amazonaws.com/ax-public/static_tools/tar/tar-1-29 && chmod +x /ax-execu-host/art/ax_tar_ax
RUN curl -sSL -o /ax-execu-host/art/ax_gzip_ax https://s3-us-west-1.amazonaws.com/ax-public/static_tools/gzip/gzip-1-8 && chmod +x /ax-execu-host/art/ax_gzip_ax
RUN curl -sSL -o /ax-execu-host/art/ax_bash_ax https://s3-us-west-1.amazonaws.com/ax-public/static_tools/bash/bin/bash-static && chmod +x /ax-execu-host/art/ax_bash_ax
RUN curl -sSL -o /ax-execu-host/art/ax_exec_ax https://s3-us-west-1.amazonaws.com/ax-public/static_tools/ax_exec_ax/ax_exec_ax && chmod +x /ax-execu-host/art/ax_exec_ax
RUN curl -sSL -o /ax-execu-host/art/ax_cat_ax https://s3-us-west-1.amazonaws.com/ax-public/static_tools/coreutils/bin/cat && chmod +x /ax-execu-host/art/ax_cat_ax
RUN curl -sSL -o /ax-execu-host/art/ax_cp_ax https://s3-us-west-1.amazonaws.com/ax-public/static_tools/coreutils/bin/cp && chmod +x /ax-execu-host/art/ax_cp_ax
RUN curl -sSL -o /ax-execu-host/art/ax_echo_ax https://s3-us-west-1.amazonaws.com/ax-public/static_tools/coreutils/bin/echo && chmod +x /ax-execu-host/art/ax_echo_ax
RUN curl -sSL -o /ax-execu-host/art/ax_kill_ax https://s3-us-west-1.amazonaws.com/ax-public/static_tools/coreutils/bin/kill && chmod +x /ax-execu-host/art/ax_kill_ax
RUN curl -sSL -o /ax-execu-host/art/ax_mv_ax https://s3-us-west-1.amazonaws.com/ax-public/static_tools/coreutils/bin/mv && chmod +x /ax-execu-host/art/ax_mv_ax
RUN curl -sSL -o /ax-execu-host/art/busybox-i686 https://s3-us-west-1.amazonaws.com/ax-public/static_tools/busybox/busybox-i686 && chmod +x /ax-execu-host/art/busybox-i686
RUN curl -sSL -o /ax-execu-host/art/handshake https://s3-us-west-1.amazonaws.com/ax-public/static_tools/ax_handshake_ax/handshake && chmod +x /ax-execu-host/art/handshake

COPY docker_build/root /

# Install docker client. This has to be static binary.
RUN cd /tmp && \
    curl -o /tmp/docker.tgz -sSL https://download.docker.com/linux/static/stable/x86_64/docker-17.06.0-ce.tgz && \
    tar axvf /tmp/docker.tgz && \
    cp -pr /tmp/docker/docker /usr/local/bin/docker && \
    rm -rf /tmp/docker && \
    rm -f /tmp/docker.tgz

ENTRYPOINT ["/run.sh"]


