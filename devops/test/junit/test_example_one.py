import logging
import sys
import time

LOG_PROMPT_FMT = '%(asctime)s.%(msecs)s:%(name)s:%(thread)d:%(levelname)s:%(process)d:%(message)s'
LOG_DATE_FMT = '%Y-%m-%dT%H:%M:%S'

logger = logging.getLogger(__name__)

logging.basicConfig(format=LOG_PROMPT_FMT,
                    datefmt=LOG_DATE_FMT,
                    level=logging.DEBUG,
                    stream=sys.stdout
                    )


def test_long_1():
    logger.info('Start: Testing test case one...')
    time.sleep(1)
    logger.info('End: Success: test case one')


def test_long_2():
    logger.info('Start: Testing test case two...')
    time.sleep(2)
    logger.info('End: Success: test case two')


def test_long_3():
    logger.info('Start: Testing test case three...')
    time.sleep(3)
    logger.info('End: Success: test case three')


def test_long_4():
    logger.info('Start: Testing test case four...')
    time.sleep(4)
    logger.info('End: Success: test case four')


def test_long_5():
    logger.info('Start: Testing test case five...')
    time.sleep(5)
    logger.info('End: Success: test case five')


def test_long_6():
    logger.info('Start: Testing test case six...')
    time.sleep(1)
    logger.info('End: Success: test case six')


def test_long_7():
    logger.info('Start: Testing test case seven...')
    time.sleep(2)
    logger.info('End: Success: test case seven')


def test_long_8():
    logger.info('Start: Testing test case eight...')
    time.sleep(3)
    logger.info('End: Success: test case eight')


def test_long_9():
    logger.info('Start: Testing test case nine...')
    time.sleep(4)
    logger.info('End: Success: test case nine')


def test_long_10():
    logger.info('Start: Testing test case ten...')
    time.sleep(5)
    logger.info('End: Success: test case ten')
