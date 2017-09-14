# Dockerfile for volume_tools

FROM %%BUILDER_IMAGE_ID%%
ENV PATH $PATH:/ax/bin
ENV PYTHONPATH /ax/src

COPY docker_build/src /ax/src
COPY volume_tools.py /ax/bin/volume_tools
