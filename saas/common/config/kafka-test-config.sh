#!/bin/bash


# change the schema-registry listening port to 8091 to avoid conflict on 8081
echo "port=8091" >> /tmp/schema-registry.properties
echo "kafkastore.connection.url=localhost:2181" >> /tmp/schema-registry.properties
echo "kafkastore.topic=_schemas" >> /tmp/schema-registry.properties
echo "listeners=http://0.0.0.0:8091" >> /tmp/schema-registry.properties
cp /tmp/schema-registry.properties /etc/schema-registry/schema-registry.properties

# change rest-proxy listening port to 8092 to avoid conflict on 8082
echo "port=8092" >> /tmp/kafka-rest.properties
echo "schema.registry.url=http://localhost:8091" >> /tmp/kafka-rest.properties
echo "zookeeper.connect=localhost:2181" >> /tmp/kafka-rest.properties
echo "consumer.request.timeout.ms=5000" >> /tmp/kafka-rest.properties
cp /tmp/kafka-rest.properties /etc/kafka-rest/kafka-rest.properties

sed -i -e 's/KAFKA_HEAP_OPTS="-Xmx1G -Xms1G"/KAFKA_HEAP_OPTS="-Xmx200M -Xms200M"/' /usr/bin/kafka-server-start
sed -i -e 's/KAFKA_HEAP_OPTS="-Xmx512M -Xms512M"/KAFKA_HEAP_OPTS="-Xmx50M -Xms50M"/' /usr/bin/zookeeper-server-start
