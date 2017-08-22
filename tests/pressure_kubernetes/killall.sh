#!/bin/bash

for p in `ps ax | grep run.sh | grep -v grep | awk '{print $1}'`; do
    kill ${p};
done

# Kill all dstat processes
for p in `ps ax | grep "dstat -am" | grep -v grep | awk '{print $1}'`; do
    kill ${p};
done

# Kill all kubectl
for p in `ps ax | grep "kubectl" | grep -v grep | awk '{print $1}'`; do
    kill ${p};
done
