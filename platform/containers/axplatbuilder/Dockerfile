# Dockerfile for building & packaging python applications to single file executables.
# Is also used as a base image for debug builds

FROM alpine:3.5

RUN apk --update --no-cache add \
    vim \
    bash \
    python \
    python3 \
    wget \
    curl \
    openssl \
    git \
    inotify-tools

RUN wget -P /tmp/ http://bootstrap.pypa.io/get-pip.py && \
    python2 /tmp/get-pip.py && \
    cp /usr/bin/pip /usr/bin/pip2 && \
    pip3 install -U pip && \
    rm -f /tmp/get-pip.py

RUN pip2 install --upgrade setuptools && pip3 install --upgrade setuptools
COPY docker_build/requirements /tmp/requirements

RUN apk --update --no-cache add \
    python-dev python3-dev musl-dev zlib-dev linux-headers gcc g++ make &&\
    pip2 install --no-cache-dir -r /tmp/requirements/requirements.txt && \
    pip3 install --no-cache-dir -r /tmp/requirements/requirements.txt && \
    apk del python-dev python3-dev musl-dev zlib-dev linux-headers gcc g++ make

# Alpine bug for certifi
RUN pip2 install --upgrade certifi==2015.04.28 && pip3 install --upgrade certifi==2015.04.28
