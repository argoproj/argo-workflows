# Builds a docker image for the Confluent schema registry.
# Expects links to "kafka" and "zookeeper" containers.
#
# Usage:
#   docker build -t confluent/schema-registry schema-registry
#   docker run -d --name schema-registry --link zookeeper:zookeeper --link kafka:kafka \
#       --env SCHEMA_REGISTRY_AVRO_COMPATIBILITY_LEVEL=none confluent/schema-registry

FROM %%ARGO_BASE_REGISTRY%%/argobase/kafka-base:v1.1

RUN mkdir -p /ax/schema-registry/bin
COPY schema-registry-run.sh /ax/schema-registry/bin/schema-registry-run.sh
COPY schema-registry.properties /etc/schema-registry/schema-registry.properties

#TODO Schema Registry needs a log directory.
#RUN ["chown", "-R", "confluent:confluent", "/etc/schema-registry/schema-registry.properties"]
RUN ["chmod", "+x", "/ax/schema-registry/bin/schema-registry-run.sh"]

EXPOSE 8091

#USER confluent
ENTRYPOINT ["/ax/schema-registry/bin/schema-registry-run.sh"]
