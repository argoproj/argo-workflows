FROM debian:8.9

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY ./bin/argo /usr/bin/argo
ENTRYPOINT ["/usr/bin/argo"]
