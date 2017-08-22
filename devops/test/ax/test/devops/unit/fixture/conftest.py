import logging
import shlex
import subprocess
import sys

import pytest

from .mock import MockAxopsClient, MockAxdbClient, MockAxsysClient

logger = logging.getLogger(__name__)

logging.basicConfig(format="%(asctime)s.%(msecs)03d %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S",
                    level=logging.DEBUG,
                    stream=sys.stdout)

def pytest_addoption(parser):
    parser.addoption("--redis", action="store", default=None,
                     help="IP or hostname of a redis database instance")
    parser.addoption("--mongodb", action="store", default=None,
                     help="IP or hostname of a mongodb database instance")
    parser.addoption("--axops", action="store", default=None,
                     help="IP or hostname of an axops container instance")
    parser.addoption("--axdb", action="store", default=None,
                     help="IP or hostname of an axdb container instance")
    parser.addoption("--axmon", action="store", default=None,
                     help="IP or hostname of an axmon container instance")

def run_container(image):
    """Launch a container and return its container id and port"""
    logger.info("Starting %s container", image)
    cmd = "docker run --detach -P {}".format(image)
    container_id = subprocess.check_output(shlex.split(cmd), universal_newlines=True).strip()
    port = get_container_port(container_id)
    logger.info("Created %s container: %s on port %s", image, container_id, port)
    return container_id, port

def get_container_port(container_id):
    cmd = "docker port {}".format(container_id)
    port_output = subprocess.check_output(shlex.split(cmd), universal_newlines=True).strip()
    return int(port_output.split(':')[1])

def remove_container(container_id):
    logger.info("Shutting down container %s", container_id)
    cmd = "docker rm --volumes -f {}".format(container_id)
    subprocess.check_call(shlex.split(cmd))

@pytest.fixture(scope="session")
def redis(request):
    redis_host = request.config.getoption("--redis")
    container_id = None
    if not redis_host:
        container_id, port = run_container('redis')
        redis_host = 'localhost:{}'.format(port)
    yield redis_host
    if container_id:
        remove_container(container_id)

@pytest.fixture(scope="session")
def mongodb(request):
    mongodb_host = request.config.getoption("--mongodb")
    container_id = None
    if not mongodb_host:
        container_id, port = run_container('mongo:3.4.4')
        mongodb_host = 'localhost:{}'.format(port)
    yield mongodb_host
    if container_id:
        remove_container(container_id)

@pytest.fixture(scope="session")
def axops(request):
    return request.config.getoption("--axops")

@pytest.fixture(scope="session")
def axdb(request):
    return request.config.getoption("--axdb")

@pytest.fixture(scope="session")
def axmon(request):
    return request.config.getoption("--axmon")

@pytest.fixture
def fixmgr(redis, mongodb, axops, axdb, axmon):
    from ax.devops.fixture.manager import FixtureManager
    fixmgr = FixtureManager(mongodb_host=mongodb, redis_host=redis, axops_host=axops)
    if axops is None:
        fixmgr.axops_client = MockAxopsClient()
    if axdb is None:
        fixmgr.axdb_client = MockAxdbClient()
    else:
        for v in fixmgr.volumemgr.axdb_client.get_volumes():
            fixmgr.volumemgr.axdb_client.delete_volume(v['id'])
    if axmon is None:
        fixmgr.volumemgr.axsys_client = MockAxsysClient()
    fixmgr.initdb()
    fixmgr.reqproc.requestdb.initdb()
    import ax.devops.fixture.rest as rest
    rest.fixmgr = fixmgr
    yield fixmgr
    fixmgr.volumemgr.stop_workers()
    fixmgr.reqproc.stop_processor()
    if not isinstance(fixmgr.volumemgr.axsys_client, MockAxsysClient):
        # if we are not running with a mock axsys client, delete the real ebs volumes from platform manually for cleanup purposes
        for v in fixmgr.volumemgr.get_volumes():
            try:
                logger.info("Cleaning up %s", v)
                fixmgr.volumemgr.axsys_client.delete_volume(v.id)
            except Exception:
                logger.exception("Failed to clean up %s", v)
