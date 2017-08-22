# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
# Dockerfile for axstats debug build

FROM %%BUILDER_IMAGE_ID%%
ENV PYTHONPATH /ax/python

COPY start.sh /start.sh
COPY docker_build/root /
CMD /start.sh
