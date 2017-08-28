# Builds a docker image running Apache Zookeeper. Intended for use with Kafka.
#
# Usage:
#   docker build -t %%ARGO_BASE_REGISTRY%%/argobase/min/zookeeper:latest .
#   docker run -d --name zookeeper %%ARGO_BASE_REGISTRY%%/argobase/min/zookeeper:latest

FROM %%ARGO_BASE_REGISTRY%%/argobase/kafka-base:v1.5

ENV LOG_DIR "/ax/zookeeper/log"
ENV ZK_DATA_DIR "/ax/zookeeper/data"
ENV KAFKA_LOG4J_OPTS -Dlog4j.configuration=file:/etc/kafka/log4j.properties

RUN mkdir -p /ax/bin
#RUN mkdir -p /ax/zookeeper/log
#RUN mkdir -p /ax/zookeeper/data
#RUN mkdir -p /ax/zookeeper/conf

COPY zk-run.sh /ax/bin/zk-run.sh
COPY health.sh /ax/bin/health.sh
COPY log4j.properties /etc/kafka/log4j.properties
COPY zookeeper.properties /etc/kafka/zookeeper.properties

RUN ["chmod", "+x", "/ax/bin/zk-run.sh"]
RUN ["chmod", "+x", "/ax/bin/health.sh"]
#VOLUME ["/ax/zookeeper/data", "/ax/zookeeper/log"]
#TODO Expose JMX
# Expose client port (2188/tcp), peer connection port (2888/tcp), leader election port (3888/tcp)
EXPOSE 2181 2888 3888

#USER confluent
ENTRYPOINT ["/ax/bin/zk-run.sh"]
