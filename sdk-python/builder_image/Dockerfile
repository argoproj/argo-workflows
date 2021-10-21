FROM ubuntu:18.04

RUN apt-get update && \
    apt install -y apt-transport-https ca-certificates curl software-properties-common && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu bionic stable" && \
    apt update

RUN apt-cache policy docker-ce && \
    apt install -y docker-ce build-essential rsync python3-pip jq moreutils

ADD requirements.txt /tmp/requirements.txt
ADD requirements-dev.txt /tmp/requirements-dev.txt
WORKDIR /tmp
RUN pip3 install -r requirements-dev.txt

ENTRYPOINT ["/bin/bash"]
