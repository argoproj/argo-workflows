# Copyright 2015-2017 Applatix, Inc.  All rights reserved.

import logging
import os
import pytest
import sys

from ax.devops.client.artifact_client import AxArtifactManagerClient

logging.basicConfig(format="%(asctime)s.%(msecs)03d %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S",
                    level=logging.INFO,
                    stream=sys.stdout)


def pytest_addoption(parser):
    parser.addoption("--concurrency", action="store", dest='concurrency', type=int,
                     default=20, help="Number of concurrent requests to be sent")
    parser.addoption("--max-request", action="store", dest='max_request', type=int,
                     default=1000, help="Number of maximal requests to be sent")


@pytest.fixture
def artifact_manager():
    hostname = 'axartifactmanager.axsys' if os.environ.get('KUBERNETES_SERVICE_HOST') else 'localhost'
    return AxArtifactManagerClient(host=hostname, timeout=60 * 5)


@pytest.fixture
def concurrency(request):
    return request.config.getoption('--concurrency')


@pytest.fixture
def max_request(request):
    return request.config.getoption('--max-request')
