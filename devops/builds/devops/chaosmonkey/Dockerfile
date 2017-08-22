# Debug Dockerfile for chaosmonkey

FROM %%BUILDER_IMAGE_ID%%

ENV PATH $PATH:/ax/bin
ENV PYTHONPATH /ax/src
EXPOSE 9891

COPY chaosmonkey.py /ax/bin/chaosmonkey
COPY config.yaml /ax/etc/config.yaml
COPY docker_build/src /ax/src
CMD ["chaosmonkey"]
