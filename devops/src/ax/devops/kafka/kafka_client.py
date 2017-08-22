#Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import logging
import json
import socket
import sys
import time
import uuid
from retrying import retry

from kafka import KafkaProducer, KafkaConsumer
from ax.devops.settings import AxSettings

logger = logging.getLogger(__name__)


class ProducerClient(object):
    """AX Kafka Producer client."""

    def __init__(self, host=None, port=None, acks=None, client_id=None, key_serializer=None, value_serializer=None,
                 retries=None, retry_interval=None, request_timeout_ms=None, retry_backoff_ms=None):
        """Initialize connection to Kafka producer.

        :param host:
        :param port:
        :param acks:
        :param client_id:
        :param key_serializer:
        :param value_serializer:
        :param retries:
        :param retry_interval:
        :param request_timeout_ms:
        :param retry_backoff_ms:
        :returns:
        """
        self.host = host or AxSettings.KAFKA_HOSTNAME
        self.port = port or AxSettings.KAFKA_PORT
        self.acks = acks or 1
        self.client_id = client_id or socket.gethostname()
        self.key_serializer = key_serializer or AxSettings.kafka_serialize_key
        self.value_serializer = value_serializer or AxSettings.kafka_serialize_value
        self.retries = retries or 300
        self.retry_interval = retry_interval or 5
        self.request_timeout_ms = request_timeout_ms or 30000
        self.retry_backoff_ms = retry_backoff_ms or 1000

        self.bootstrap_servers = '{}:{}'.format(self.host, self.port)
        self.producer = None

    def connect(self):

        @retry(stop_max_attempt_number=self.retries, wait_fixed=self.retry_interval)
        def create_producer():
            logger.info('Connecting kafka server (%s) as a producer ...', self.bootstrap_servers)
            self.producer = KafkaProducer(
                bootstrap_servers=self.bootstrap_servers,
                client_id=self.client_id,
                key_serializer=self.key_serializer,
                value_serializer=self.value_serializer,
                request_timeout_ms=self.request_timeout_ms,
                retries=self.retries,
                retry_backoff_ms=self.retry_backoff_ms,
                acks=self.acks,
                api_version_auto_timeout_ms=30000,
                max_request_size=10 * 1024 * 1024
            )

        create_producer()

    def send(self, topic, value, partition=None, key=None, timeout=300):
        """Send message to kafka.

        :param topic:
        :param value:
        :param partition:
        :param key:
        :param timeout:
        :returns:
        """
        if self.producer is None:
            self.connect()
        if key is None:
            key = str(uuid.uuid1())
        logger.info('Sending message (size: %s) to kafka server ...', sys.getsizeof(json.dumps(value)))
        logger.debug('Payload:\n%s', value)
        result = self.producer.send(topic, partition=partition, key=key, value=value)
        if timeout:
            return result.get(timeout=timeout)
        else:
            return result

    def close(self, timeout=300):
        """Close connection.

        :param timeout:
        :returns:
        """
        if self.producer is not None:
            return self.producer.close(timeout)


class ConsumerClient(object):
    """AX Kafka consumer client."""

    def __init__(self, topic, host=None, port=None, client_id=None, group_id=None, key_deserializer=None,
                 value_deserializer=None, retries=None, retry_interval=None, request_timeout_ms=None, retry_backoff_ms=None):
        """Initialize consumer.

        :param topic:
        :param host:
        :param port:
        :param client_id:
        :param group_id:
        :param key_deserializer:
        :param value_deserializer:
        :param retries:
        :param retry_interval:
        :param request_timeout_ms:
        :param retry_backoff_ms:
        :returns:
        """
        self.topic = topic
        self.host = host or AxSettings.KAFKA_HOSTNAME
        self.port = port or AxSettings.KAFKA_PORT
        self.bootstrap_servers = '{}:{}'.format(self.host, self.port)
        hostname = socket.gethostname().split('-')[0]
        self.client_id = client_id or 'kafka-client-{}'.format(hostname)
        self.group_id = group_id or 'kafka-group-{}'.format(hostname)
        self.key_deserializer = key_deserializer or AxSettings.kafka_deserialize_key
        self.value_deserializer = value_deserializer or AxSettings.kafka_deserialize_value
        self.retries = retries or 300
        self.retry_interval = retry_interval or 5
        self.request_timeout_ms = request_timeout_ms or 30000
        self.retry_backoff_ms = retry_backoff_ms or 1000
        self.consumer = None

    def connect(self):

        @retry(stop_max_attempt_number=self.retries, wait_fixed=self.retry_interval)
        def create_consumer():
            self.consumer = KafkaConsumer(
                self.topic,
                bootstrap_servers=self.bootstrap_servers,
                client_id=self.client_id, group_id=self.group_id,
                auto_offset_reset='earliest',
                key_deserializer=self.key_deserializer,
                value_deserializer=self.value_deserializer,
                request_timeout_ms=self.request_timeout_ms,
                retry_backoff_ms=self.retry_backoff_ms,
                api_version_auto_timeout_ms=30000)

        create_consumer()

    def poll(self, timeout=None):
        """Poll message out of kafka.

        :param timeout:
        :returns:
        """
        if not self.consumer:
            self.connect()
        return self.consumer.poll(timeout_ms=timeout * 1000 if timeout else 0)

    def close(self):
        """Close connection.

        :returns:
        """
        if self.consumer is not None:
            return self.consumer.close()


class ExecutorProducerClient(ProducerClient):
    """This Kafka producer is tailed for executor status report."""
    WAITING_STATE = 'WAITING'
    RUNNING_STATE = 'RUNNING'
    COMPLETE_STATE = 'COMPLETE'
    SUCCESS_RESULT = 'SUCCESS'
    FAILURE_RESULT = 'FAILURE'
    CANCELLED_RESULT = 'CANCELLED'
    SKIPPED_RESULT = 'SKIPPED'

    @staticmethod
    def send_executor_status(key, payload):
        """Report to Kafka when start and finish the task."""
        def exponential_sleep(cnt, wait_exponential_multiplier_ms=1000, wait_exponential_max_ms=20000):
            import random
            sleep_time_ms = (2 ** cnt) * wait_exponential_multiplier_ms + random.randint(0, 1000)
            sleep_time_ms = min(sleep_time_ms, wait_exponential_max_ms)
            time.sleep(sleep_time_ms / 1000.0)

        logger.info('[WFE] Report to Kafka: data: %s', json.dumps(payload))
        value = {'Op': 'status', 'Payload': payload}
        count = 0
        max_retry = 180
        while True:
            count += 1
            try:
                kafka_producer = ExecutorProducerClient()
                kafka_producer.send('devops_task', key=key, value=value)
                kafka_producer.producer.flush()
                break
            except Exception as e:
                logger.exception("report to Kafka, %s", str(e))
                if count < max_retry:
                    exponential_sleep(count - 1)
                else:
                    raise


class EventNotificationClient(ProducerClient):
    """Client for event notification."""

    topic = 'axnc'

    def __init__(self, facility):
        super(EventNotificationClient, self).__init__()
        self.facility = facility

    def send_message_to_notification_center(self, event_code, trace_id=None, recipients=None, detail=None):
        event_id = str(uuid.uuid1())
        trace_id = str(trace_id or event_id)

        if detail:
            for k in detail.keys():
                detail[k] = str(detail[k])

        event = {
            'event_id': event_id,
            'trace_id': trace_id,
            'code': event_code,
            'facility': self.facility,
            'recipients': recipients or [],
            'detail': detail or {},
            'timestamp': int(time.time() * 10 ** 6)
        }

        @retry(wait_fixed=5, stop_max_attempt_number=5)
        def _send():
            self.send(self.topic, key=event['trace_id'], value=event)
            self.producer.flush()

        return _send()
