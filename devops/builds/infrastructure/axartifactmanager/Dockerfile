# Copyright 2015-2016 Applatix, Inc. All rights reserved.

# Debug Dockerfile for artifact manager

FROM %%BUILDER_IMAGE_ID%%

ENV PATH $PATH:/ax/bin
ENV PYTHONPATH /ax/src
EXPOSE 9892

COPY artifactmanager.py /ax/bin/artifactmanager
COPY run.sh /ax/bin/run.sh
RUN chmod +x /ax/bin/run.sh
COPY docker_build/src /ax/src

CMD ["artifactmanager"]
