#!/bin/sh

/usr/bin/uwsgi --ini /ax/gateway.ini &
/usr/sbin/nginx &
/usr/sbin/crond &
sleep 9223372036854775807
