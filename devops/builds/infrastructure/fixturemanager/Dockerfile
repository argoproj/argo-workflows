# Dockerfile for fixturemanager

FROM %%BUILDER_IMAGE_ID%%

ENV MONGO_MAJOR 3.4
ENV MONGO_VERSION 3.4.4
ENV MONGO_PACKAGE mongodb-org

COPY install.sh /tmp/install.sh
RUN /tmp/install.sh && rm -f /tmp/install.sh

ENV PATH $PATH:/ax/bin
ENV PYTHONPATH /ax/src
EXPOSE 8912
COPY fixturemanager.py /ax/bin/fixturemanager
COPY docker_build/src /ax/src
CMD ["fixturemanager"]
