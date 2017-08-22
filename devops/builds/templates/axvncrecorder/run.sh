#!/bin/bash

echo "secret" > /tmp/secret

/usr/local/bin/flvrec.py -P /tmp/secret -o ${output:-/tmp/output.flv} ${host:-localhost}:${port:-0}




