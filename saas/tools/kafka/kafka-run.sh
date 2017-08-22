#!/bin/bash

CONFIG_FILE="/etc/kafka/server.properties"
LOG_FILE="/etc/kafka/log4j.properties"
: ${ZOOKEEPER_CONNECT:=zk:2181}
: ${NUM_PARTITIONS:=1}
: ${BROKER_ID:=0}
: ${REPLICATION_FACTOR:=1}

#if [[ -n ${BROKER_ID} ]]; then
#  echo "broker.id=${BROKER_ID}" >> ${CONFIG_FILE}
#fi

if [[ -n ${ADVERTISED_HOSTNAME} ]]; then
  echo "advertised.host.name=${ADVERTISED_HOSTNAME}" >> ${CONFIG_FILE}
fi

if [[ -n ${REPLICATION_FACTOR} ]]; then
  echo "replication.factor=${REPLICATION_FACTOR}" >> ${CONFIG_FILE}
  echo "default.replication.factor=${REPLICATION_FACTOR}" >> ${CONFIG_FILE}
fi

sed -e "s/%BROKER_ID%/${BROKER_ID}/" -i ${CONFIG_FILE}
sed -e "s/%ZOOKEEPER_CONNECT%/${ZOOKEEPER_CONNECT}/" -i ${CONFIG_FILE}
sed -e "s/%NUM_PARTITIONS%/${NUM_PARTITIONS}/" -i ${CONFIG_FILE}
#sed -e "s/%REPLICATION_FACTOR%/${REPLICATION_FACTOR}/" -i ${CONFIG_FILE}
#sed -e "s/%DEFAULT_REPLICATION_FACTOR%/${DEFAULT_REPLICATION_FACTOR}/" -i ${CONFIG_FILE}

#sed -e "s/log4j.rootLogger=INFO/log4j.rootLogger=DEBUG/" -i ${LOG_FILE}
#sed -e "s/log4j.appender.kafkaAppender=org.apache.log4j.DailyRollingFileAppender/log4j.appender.kafkaAppender=org.apache.log4j.RollingFileAppender/" -i ${LOG_FILE}
sed -e "s/DailyRollingFileAppender/RollingFileAppender/" -i ${LOG_FILE}
#sed -e "s/#log4j.logger.kafka.producer.async.DefaultEventHandler/log4j.logger.kafka.producer.async.DefaultEventHandler/" -i ${LOG_FILE}
#sed -e "s/#log4j.logger.kafka.client.ClientUtils/log4j.logger.kafka.client.ClientUtils/" -i ${LOG_FILE}
#sed -e "s/log4j.logger.kafka=INFO/log4j.logger.kafka=DEBUG/" -i ${LOG_FILE}
echo "log4j.appender.kafkaAppender.MaxFileSize=102400KB"  >> ${LOG_FILE}
echo "log4j.appender.kafkaAppender.MaxBackupIndex=7" >> ${LOG_FILE}
echo "log4j.appender.stateChangeAppender.MaxFileSize=4096KB" >> ${LOG_FILE}
echo "log4j.appender.stateChangeAppender.MaxBackupIndex=7" >> ${LOG_FILE}
echo "log4j.appender.controllerAppender.MaxFileSize=4096KB"  >> ${LOG_FILE}
echo "log4j.appender.controllerAppender.MaxBackupIndex=7" >> ${LOG_FILE}
echo "log4j.appender.cleanerAppender.MaxFileSize=1024KB"  >> ${LOG_FILE}
echo "log4j.appender.cleanerAppender.MaxBackupIndex=7" >> ${LOG_FILE}

# Force the data directory and log paths to our volumes.
export KAFKA_LOG_DIRS='/ax/kafka/log'
export LOG_DIR='/ax/kafka/log'
export KAFKA_LOG4J_OPTS="-Dlog4j.configuration=file:/etc/kafka/log4j.properties"

if [[ -n $KAFKA_MEM_SIZE ]] ; then
  [[ -n $MEM_MULT ]] || ( echo "Must set MEM_MULT" ; exit 1)
  mem_size=$(printf "%.0f" $(bc -l <<< "$KAFKA_MEM_SIZE * $MEM_MULT"))
  export KAFKA_HEAP_OPTS="-Xms${mem_size}M -Xmx${mem_size}M"
fi

while [ 1 -eq 1 ]
do
  RES=`echo ruok | nc localhost 2181 | grep imok | wc -l`
  if [ "$RES" != "1" ]; then
    echo "not ok"
    sleep 1
    continue
  fi

  RES=`echo stat | nc localhost 2181 | grep "This ZooKeeper instance is not currently serving requests" | wc -l`
  if [ "$RES" == "1" ]; then
    echo "not ready to serve request"
    sleep 1
    continue
  else
    echo "ready, go!"
    break
  fi
done

exec /usr/bin/kafka-server-start ${CONFIG_FILE}
