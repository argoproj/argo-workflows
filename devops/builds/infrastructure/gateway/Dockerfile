# Dockerfile for gateway

FROM %%BUILDER_IMAGE_ID%%

ENV PATH $PATH:/ax/bin
ENV PYTHONPATH /ax/src
EXPOSE 8889

COPY gateway.py /ax/bin/gateway
COPY docker_build/src /ax/src
CMD ["gateway"]
