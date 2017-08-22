# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
# Dockerfile for axconsole debug build

FROM %%BUILDER_IMAGE_ID%%

ENV PYTHONPATH /ax/python
EXPOSE 81
COPY docker_build/root /
ENTRYPOINT ["/ax/bin/axconsole"]
