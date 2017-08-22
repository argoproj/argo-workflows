# Dockerfile for building & packaging python applications to single file executables.
# Is also used as a base image for debug builds
# Based off of: https://github.com/six8/pyinstaller-alpine

FROM %%ARGO_BASE_REGISTRY%%/argobase/debian:8.5

RUN apt-get update && apt-get install -y python3 curl && \
    rm -rf /var/lib/apt/lists && \
    find / -name *.pyc | xargs rm && \
    rm -rf /usr/share/locale /usr/share/doc /usr/share/man

RUN curl -L -o /tmp/get-pip.py http://bootstrap.pypa.io/get-pip.py && \
    python3 /tmp/get-pip.py && \
    pip3 install -U pip && \
    rm -f /tmp/get-pip.py

COPY docker_build/requirements /tmp/requirements
RUN apt-get update && apt-get install -y --no-install-recommends python3-dev git gcc g++ binutils && \
    pip3 install --no-cache-dir -r /tmp/requirements/requirements.txt && \
    (find / -type f | xargs strip -s || true ) && \
    find / -name *.pyc | xargs rm && \
    apt-get purge -y python3-dev git gcc g++ binutils && \
    apt-get autoremove -y && \
    apt-get clean -y && \
    rm -rf /var/lib/apt/lists && \
    rm -rf /usr/share/locale /usr/share/doc /usr/share/man
