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


def test_short_1():
    logger.info('Start: Testing test case one...')
    time.sleep(0.1)
    logger.info('End: Success: test case one')


def test_short_2():
    logger.info('Start: Testing test case two...')
    time.sleep(0.2)
    logger.info('End: Success: test case two')


def test_short_3():
    logger.info('Start: Testing test case three...')
    time.sleep(0.3)
    logger.info('End: Success: test case three')


def test_short_4():
    logger.info('Start: Testing test case four...')
    time.sleep(0.4)
    logger.info('End: Success: test case four')


def test_short_5():
    logger.info('Start: Testing test case five...')
    time.sleep(0.5)
    logger.info('End: Success: test case five')


def test_short_6():
    logger.info('Start: Testing test case six...')
    time.sleep(0.6)
    logger.info('End: Success: test case six')


def test_short_7():
    logger.info('Start: Testing test case seven...')
    time.sleep(0.7)
    logger.info('End: Success: test case seven')


def test_short_8():
    logger.info('Start: Testing test case eight...')
    time.sleep(0.8)
    logger.info('End: Success: test case eight')


def test_short_9():
    logger.info('Start: Testing test case nine...')
    time.sleep(0.0)
    logger.info('End: Success: test case nine')


def test_short_10():
    logger.info('Start: Testing test case ten...')
    time.sleep(1.0)
    logger.info('End: Success: test case ten')