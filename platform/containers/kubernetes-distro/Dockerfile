FROM ubuntu:16.04

ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 1.12.6
ENV DOCKER_SHA256 cadc6025c841e034506703a06cf54204e51d0cadfae4bae62628ac648d82efdd
# Current version.
ENV KUBERNETES_VERSION=1.6.7
# All supported versions, including backward compatible versions.
ENV KUBERNETES_VERSIONS_LIST="1.5.7 1.6.7"
ENV GOOGLE_CLOUD_SDK_VERSION=163.0.0

# Install kubernetes binaries.
ADD install.sh /tmp/install.sh
ADD requirements.txt /tmp/requirements.txt
RUN /tmp/install.sh && rm -f /tmp/install.sh /tmp/requirements.txt
