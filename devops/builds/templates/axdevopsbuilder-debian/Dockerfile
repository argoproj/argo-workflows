# Debian version of axdevopsbuilder

FROM %%ARGO_BASE_REGISTRY%%/argobase/debian:8.5

# git needed to clone pre-release version of gevent
RUN apt-get update && apt-get install -y \
  python3 \
  wget \
  git \
  && rm -rf /var/lib/apt/lists/*

COPY docker_build/requirements /tmp/requirements

RUN apt-get update -y && apt-get install -y python3-dev gcc g++ make && \
    wget -P /tmp/ https://bootstrap.pypa.io/get-pip.py && \
    python3 /tmp/get-pip.py && \
    pip install --no-cache-dir -U setuptools && \
    pip install --no-cache-dir -r /tmp/requirements/requirements.txt && \
    rm -f /tmp/get-pip.py && \
    rm -rf /tmp/requirements && \
    apt-get purge -y python3-dev gcc g++ make && \
    apt-get autoremove -y && \
    apt-get clean -y && \
    rm -rf /usr/share/locale /usr/share/doc /usr/share/man
