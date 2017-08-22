FROM debian:8.9

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

CMD /axamm/bin/axamm_server $AXDB $AMM $AXMON $ADC $FIXMANAGER $AXOPS $KAFKA

COPY ./bin/axam* /axamm/bin/
