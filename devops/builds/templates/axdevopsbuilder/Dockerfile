# Dockerfile for building & packaging python applications to single file executables.
# Is also used as a base image for debug builds
# Based off of: https://github.com/six8/pyinstaller-alpine

FROM alpine:3.5

# Needed to retrieve, build, install pyinstaller
# build dependencies for some pip packages (in addition to -dev packages above)
# to build gevent pre-release. remove when using official release
RUN apk --update --no-cache add \
    bash \
    python3 \
    wget \
    openssl \
    git \
    vim \
    curl

RUN pip3 install -U pip
RUN pip3 install --upgrade setuptools
COPY docker_build/requirements /tmp/requirements
RUN apk --update --no-cache add python3-dev musl-dev zlib-dev linux-headers gcc g++ make \
    && pip3 install --no-cache-dir -U pip \
    && pip3 install --no-cache-dir -r /tmp/requirements/requirements.txt \
    && rm -rf /tmp/requirements \
    && apk del python3-dev musl-dev zlib-dev linux-headers gcc g++ make

# Alpine bug for certifi
RUN pip3 install --upgrade certifi==2015.04.28
