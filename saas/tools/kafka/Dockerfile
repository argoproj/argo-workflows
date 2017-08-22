# Builds a docker image running Confluent's distribution of Apache Kafka.
# Needs to be linked with a Zookeeper container with alias "zookeeper".
#
# Usage:
#   docker build -t confluent/kafka kafka
#   docker run -d --name kafka --link zookeeper:zookeeper confluent/kafka

FROM %%ARGO_BASE_REGISTRY%%/argobase/kafka-base:v1.4


ENV LOG_DIR "/ax/kafka/log"
ENV KAFKA_LOG_DIRS "/ax/kafka/log"
ENV KAFKA_DATA_DIRS "/ax/kafka/data"
#RUN ["mkdir", "-p", "/ax/kafka/log", "/ax/kafka/bin", "/ax/kafka/data/", "ax/kafka/security"]
#RUN ["mkdir", "-p", "/ax/kafka/log", "/ax/kafka/bin", "/ax/kafka/data/", "ax/kafka/security"]
COPY kafka-run.sh /ax/bin/kafka-run.sh
COPY server.properties /etc/kafka/server.properties
#RUN ["chown", "-R", "confluent:confluent", "/var/lib/kafka", "/var/log/kafka", "/etc/kafka/server.properties"]
RUN ["chmod", "+x", "/ax/bin/kafka-run.sh"]
#VOLUME ["/ax/kafka/log", "/ax/kafka/data", "/ax/kafka/security"]

#TODO Update the ports that are exposed.
#TODO Add support to expose JMX
EXPOSE 9092

#USER confluent
ENTRYPOINT ["/ax/bin/kafka-run.sh"]
