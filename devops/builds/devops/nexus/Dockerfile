# Debug Dockerfile for axnexus

FROM %%BUILDER_IMAGE_ID%%

ENV PATH $PATH:/ax/bin
ENV PYTHONPATH /ax/src

COPY axnexus.py /ax/bin/axnexus
COPY docker_build/src /ax/src
CMD ["axnexus"]
