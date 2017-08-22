#!/bin/bash
ZK_CFG_FILE="/etc/kafka/zookeeper.properties"
LOG_FILE="/etc/kafka/log4j.properties"

: ${ZK_MYID:=1}
: ${ZK_NUMBER_OF_NODES:=1}
: ${ZK_SERVICE_NAME:=zookeeper}

if [[ ${ZK_NUMBER_OF_NODES} > 1 ]]; then
  echo 'standaloneEnabled=false' >> ${ZK_CFG_FILE}
fi

if [ ! -z "$ZK_MYID" ] && [ ! -z "$ZK_NUMBER_OF_NODES" ]; then
  echo "Starting up in clustered mode"
  echo "" >> ${ZK_CFG_FILE}
  echo "#Server List" >> ${ZK_CFG_FILE}
  for i in $( eval echo {1..$ZK_NUMBER_OF_NODES}); do
    if [ "$i" = "$ZK_MYID" ]; then
        echo "server.$i=0.0.0.0:2888:3888" >> ${ZK_CFG_FILE}
    else
        echo "server.$i=${ZK_SERVICE_NAME}-$i:2888:3888" >> ${ZK_CFG_FILE}
    fi
  done
fi

# Persists the ID of the current instance of Zookeeper
if [ ! -d "/ax/zookeeper/data" ]; then
  mkdir -p /ax/zookeeper/data
fi
echo ${ZK_MYID} > /ax/zookeeper/data/myid

if [[ -n $ZK_MEM_SIZE ]] ; then
  [[ -n $MEM_MULT ]] || ( echo "Must set MEM_MULT" ; exit 1)
  mem_size=$(printf "%.0f" $(bc -l <<< "$ZK_MEM_SIZE * $MEM_MULT"))
  export KAFKA_HEAP_OPTS="-Xms${mem_size}M -Xmx${mem_size}M"
fi

sed -e "s/log4j.appender.zookeeper=org.apache.log4j.DailyRollingFileAppender/log4j.appender.zookeeper=org.apache.log4j.RollingFileAppender/" -i ${LOG_FILE}
echo "log4j.appender.zookeeper.MaxFileSize=102400KB"  >> ${LOG_FILE}
echo "log4j.appender.zookeeper.MaxBackupIndex=5" >> ${LOG_FILE}
exec /usr/bin/zookeeper-server-start ${ZK_CFG_FILE}
