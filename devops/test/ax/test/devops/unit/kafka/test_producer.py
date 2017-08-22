import logging

import pytest

logger = logging.getLogger(__name__)


@pytest.mark.parametrize('count', [1, 10, 100, 1000])
def test_count_of_message(kafka_producer, kafka_consumer, topic, count):
    ret = [kafka_producer.send(topic, value='msg %d' % i) for i in range(count)]
    assert len(ret) == count

    # msgs = set()
    # try:
    #     msgs.add(next(kafka_consumer).value)
    # except StopIteration:
    #     pass
    # assert msgs == set(['msg %d' % i for i in range(count)])


@pytest.mark.parametrize('size', [1, 1000, 1000 * 1000, 10 * 1000 * 1000])
def test_message_size(kafka_producer, kafka_consumer, topic, size):
    msg = size * '1'
    assert kafka_producer.send(topic, value=msg)
    # kafka_consumer.subscribe([topic])
    # recv_msg = kafka_consumer.value.decode('utf-8')
    # assert msg == recv_msg


@pytest.mark.parametrize('good_type_msg', [12, u'?', ['a', 'list'], ('a', 'tuple'), {'a': 'dict'}])
def test_message_good_type(kafka_producer, good_type_msg, topic):
    # This should not raise an exception
    kafka_producer.send(topic, value=good_type_msg)


@pytest.mark.parametrize('bad_type_msg', [b'*'])
def test_message_bad_type(kafka_producer, bad_type_msg, topic):
    with pytest.raises(Exception):
        kafka_producer.send(topic, value=bad_type_msg)
