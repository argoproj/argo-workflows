import logging
import json
import random
import string
import sys

import pytest
from kafka import KafkaConsumer

LOG_PROMPT_FMT = '%(asctime)s.%(msecs)s:%(name)s:%(thread)d:%(levelname)s:%(process)d:%(message)s'
LOG_DATE_FMT = '%Y-%m-%dT%H:%M:%S'

logger = logging.getLogger(__name__)

logging.basicConfig(format=LOG_PROMPT_FMT,
                    datefmt=LOG_DATE_FMT,
                    level=logging.DEBUG,
                    stream=sys.stdout
                    )


def pytest_addoption(parser):
    parser.addoption('--host', action='store', default=None,
                     help='IP or hostname of a Kafka broker')
    parser.addoption('--port', action="store", default=9092,
                     help='Kafka broker port')


def random_string(l):
    return "".join(random.choice(string.ascii_letters) for _ in range(l))


@pytest.fixture()
def topic():
    yield 'test_{}'.format(random_string(5))


@pytest.fixture()
def broker(request):
    host = request.config.getoption('--host')
    port = request.config.getoption('--port')
    if not host:
        host = 'kafka-zk.axsys'
    if not port:
        port = 9092
    yield '{}:{}'.format(host, port)


@pytest.fixture()
def kafka_producer(broker):
    from ax.devops.kafka.kafka_client import ProducerClient
    host, port = broker.split(':')
    k_producer = ProducerClient(host, port=port)
    yield k_producer
    k_producer.producer.close()


@pytest.fixture()
def kafka_consumer(broker, topic):
    k_consumer = KafkaConsumer(topic,
                               bootstrap_servers=broker,
                               auto_offset_reset='smallest',
                               value_deserializer=lambda v: json.loads(v.decode('utf-8'))
                               )
    yield k_consumer




