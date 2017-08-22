# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
# Dockerfile for axnotification

FROM %%BUILDER_IMAGE_ID%%

ENV PYTHONPATH /ax/python
EXPOSE 9889
COPY docker_build/root /

CMD python3 /ax/bin/axnotification
