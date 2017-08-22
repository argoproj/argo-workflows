#!/bin/bash

RES=`echo ruok | nc localhost 2181 | grep imok | wc -l`
if [ "$RES" != "1" ]; then
  exit 255
fi

RES=`echo stat | nc localhost 2181 | grep "This ZooKeeper instance is not currently serving requests" | wc -l`
if [ "$RES" == "1" ]; then
  exit 255
fi

exit 0