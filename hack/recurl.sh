#!/bin/sh
set -eux

file=$1
url=$2

# loop forever
while ! curl -c -L -o "$file" -LO -- "$url" ;do
  echo "sleeping before trying again"
  sleep 10s
done

chmod +x "$file"