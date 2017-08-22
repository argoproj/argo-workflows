# -*- coding: utf-8 -*-
#
# Copyright (C) 2015-2016 Applatix Inc.
#

import json


class RedisSettings(object):
    """Redis settings."""

    server = 'redis'
    port = 6379
    db = 0
    url = 'redis://{}:{}/{}'.format(server, port, db)


class AxSettings(object):
    """Store all constants like default hostname."""

    AXDB_HOSTNAME = 'axdb.axsys'
    AXDB_PORT = 8083
    AXDB_VERSION = 'v1'

    AXOPS_HOSTNAME = 'axops-internal.axsys'
    AXOPS_PORT = 8085
    AXOPS_PROTOCOL = 'http'
    AXOPS_VERSION = 'v1'
    AXOPS_USERNAME = 'admin@internal'

    AXNOTIFICATION_HOSTNAME = 'axnotification.axsys'
    AXNOTIFICATION_PORT = 9889

    AXMON_HOSTNAME = 'axmon.axsys'
    AXMON_PORT = 8901

    REDIS_HOSTNAME = 'redis.axsys'
    REDIS_PORT = 6379

    AXARTIFACTMANAGER_HOSTNAME = 'axartifactmanager.axsys'
    AXARTIFACTMANAGER_PORT = 9892
    AXARTIFACTMANAGER_VERSION = 'v1'

    AXAMM_HOSTNAME = 'axamm.axsys'
    AXAMM_PORT = 8966
    AXAMM_VERSION = 'v1'

    KAFKA_HOSTNAME = 'kafka-zk.axsys'
    KAFKA_PORT = 9092
    TOPIC_DEVOPS_CI_EVENT = 'devops_ci_event'
    TOPIC_GC_EVENT = 'repo_gc'

    PROMETHEUS_HOSTNAME = 'prometheus.axsys'
    PROMETHEUS_PORT = 9090


    @staticmethod
    def kafka_serialize_key(k):
        return k.encode(encoding='utf-8', errors='replace')


    @staticmethod
    def kafka_deserialize_key(k):
        return k.decode(encoding='utf-8', errors='replace')


    @staticmethod
    def kafka_serialize_value(v):
        return json.dumps(v).encode(encoding='utf-8', errors='replace')


    @staticmethod
    def kafka_deserialize_value(v):
        return json.loads(v.decode(encoding='utf-8', errors='replace'))

