#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import logging
import json
import time

from future.utils import with_metaclass

from ax.util.retry_exception import AXRetry, ax_retry
from ax.util.singleton import Singleton
from kafka import KafkaProducer

DEFAULT_EVENT_HOST = "kafka-zk.axsys"
DEFAULT_EVENT_PORT = 9092

logger = logging.getLogger(__name__)


class AXEventClient(with_metaclass(Singleton, object)):

    _producer = None

    def __init__(self, *args, **kwargs):
        pass

    @classmethod
    def producer(cls):
        if not cls._producer:
            count = 0
            max_retry = 300
            while True:
                count += 1
                try:
                    cls._producer = KafkaProducer(bootstrap_servers="{}:{}".format(DEFAULT_EVENT_HOST, DEFAULT_EVENT_PORT),
                                                  key_serializer=lambda v: v.encode('utf-8'),
                                                  value_serializer=lambda v: json.dumps(v).encode('utf-8'))
                    break
                except Exception:
                    logger.exception("Create Kafka Client Failure, retrying...")
                    if count < max_retry:
                        time.sleep(5)
                    else:
                        raise

        return cls._producer

    @staticmethod
    def post(topic, op, data, key=None):
        """
        Post an event to axevent server.

        :param path: pathname for event.
        :param data: event json data.
        :return POST response or None
        """
        logger.debug("Post {} {} {}".format(topic, key, json.dumps(data)))
        retry = AXRetry(retry_exception=(Exception,))
        # kafka consumer has its own timeout, by default is 30 seconds
        return ax_retry(AXEventClient.producer().send, retry, topic=topic, key=key, value=AXEventClient.pack_message_body(op, data))

    @staticmethod
    def post_from_axmon(topic, op, data, key=None):
        """
        This method should be called from axmon container only
        """
        logger.debug("Post {} {} {} ".format(topic, key, json.dumps(data)))

        # post without retry as this is typically called for rt container events
        AXEventClient.producer().send(topic=topic, key=key, value=AXEventClient.pack_message_body(op, data))

    @staticmethod
    def pack_message_body(op, payload):
        message_body = {'Op': op, 'Payload': payload}
        return message_body