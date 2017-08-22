#!/usr/bin/env bash

docker pull reinblau/php-apache2

cd /vagrant
result=results/`hostname`
test=tests/start_stop_apaches.py
date >> $result
python $test >> $result
python $test >> $result
python $test >> $result

