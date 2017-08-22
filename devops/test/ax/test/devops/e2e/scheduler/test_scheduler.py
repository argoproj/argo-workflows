import time
import logging

logger = logging.getLogger(__name__)


def mock_create_service(policy):
    logger.info(policy)
    raise False


def test_refresh_scheduler(jobscheduler, monkeypatch):
    logger.info(jobscheduler.axops_client.ping())
    jobscheduler.refresh_scheduler()
    counter = 0
    while True:
        logger.info(jobscheduler.get_schedules())
        time.sleep(5)
        counter += 1
        if counter == 10:
            jobscheduler.refresh_scheduler()
            counter = 0
