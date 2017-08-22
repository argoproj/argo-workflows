FROM debian:8.9

RUN apt-get update && \
    apt-get install -y xmlsec1 ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY ./public /public
COPY ./swagger /swagger
COPY ./bin/axops_server /axops/bin/
COPY ./bin/axops_initializer /axops/bin/
COPY ./bin/axpassword /axops/bin/
COPY ./bin/argocli /public/files/argocli

CMD /axops/bin/axops_server $AXDB $GATEWAY $AXWORKFLOWADC $AXMON $AXNOTIFIER $FIXMANAGER $SCHEDULER $AXARTIFACTMANAGER

EXPOSE 8081
