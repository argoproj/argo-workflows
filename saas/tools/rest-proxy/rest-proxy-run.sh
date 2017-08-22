#!/bin/bash

RP_CFG_FILE="/etc/kafka-rest/kafka-rest.properties"
sed -i 's/\"kafka\"//' /usr/bin/kafka-rest-run-class
#allow zookeeper and kafka to startup
sleep 10

exec /usr/bin/kafka-rest-start ${RP_CFG_FILE}
