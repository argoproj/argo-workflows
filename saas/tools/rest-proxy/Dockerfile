# Builds a docker image for the Kafka REST Proxy.
# Expects links to "schema-registry" and "zookeeper" containers.
#
# Usage:
#   docker build -t confluent/rest-proxy rest-proxy
#   docker run -d --name rest-proxy --link zookeeper:zookeeper --link schema-registry:schema-registry \
#       confluent/rest-proxy

FROM %%ARGO_BASE_REGISTRY%%/argobase/kafka-base:v1.1

RUN ["mkdir", "-p", "/ax/rest-proxy/bin"]
COPY rest-proxy-run.sh /ax/rest-proxy/bin/rest-proxy-run.sh
COPY kafka-rest.properties /etc/kafka-rest/kafka-rest.properties
#RUN ["chown", "-R", "confluent:confluent", "/etc/kafka-rest", "/usr/local/bin/rest-proxy-docker.sh"]
RUN ["chmod", "+x", "/ax/rest-proxy/bin/rest-proxy-run.sh"]

EXPOSE 8092

CMD [ "/ax/rest-proxy/bin/rest-proxy-run.sh" ]

