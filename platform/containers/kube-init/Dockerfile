FROM alpine:3.5

RUN apk update && apk add curl && apk add coreutils && rm -rf /var/cache/apk/*
ADD wait_tcp /ax/bin/wait_tcp
ADD wait_http /ax/bin/wait_http
ADD wait_volume /ax/bin/wait_volume
RUN curl -o /ax/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.6.7/bin/linux/amd64/kubectl && chmod +x /ax/bin/kubectl

ENV PATH /ax/bin:${PATH}

