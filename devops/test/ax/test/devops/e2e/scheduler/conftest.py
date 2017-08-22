import logging
import pytest
import sys
from .mock import MockAxopsClient

logger = logging.getLogger(__name__)
logging.basicConfig(format="%(asctime)s.%(msecs)03d %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S",
                    level=logging.DEBUG,
                    stream=sys.stdout)
#
@pytest.fixture
def jobscheduler():
    from ax.devops.scheduler.jobscheduler import JobScheduler
    jobscheduler = JobScheduler()
    jobscheduler.axops_client = MockAxopsClient()
    yield jobscheduler
    jobscheduler.stop_scheduler()
