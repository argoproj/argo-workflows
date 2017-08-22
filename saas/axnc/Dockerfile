FROM debian:8.9

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

CMD /axnc/bin/server

COPY ./bin/* /axnc/bin/
COPY event-skeleton /axnc
