FROM node:6.9.5-alpine

RUN apk update && apk add ca-certificates wget && update-ca-certificates

WORKDIR /tmp

RUN wget https://storage.googleapis.com/kubernetes-release/release/v1.6.4/bin/linux/amd64/kubectl && chmod +x kubectl && mv ./kubectl /usr/bin/ && \
    wget https://download.docker.com/linux/static/stable/x86_64/docker-17.03.0-ce.tgz && tar -xzf docker-17.03.0-ce.tgz && mv ./docker/docker /usr/bin/ && \
    rm /tmp/docker-17.03.0-ce.tgz && rm -rf rm /tmp/docker

ADD ./dist /app/dist
ADD ./node_modules /app/node_modules

WORKDIR /app

EXPOSE 8080
