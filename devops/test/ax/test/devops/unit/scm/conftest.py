import logging
import os
import shutil
import sys

import responses
import pytest

from ax.devops.scm.factory import AXOPS_HOST
from .util import check_call
from .testdata import AXOPS_GET_TOOLS_MOCK_RESPONSE

logging.basicConfig(format="%(asctime)s.%(msecs)03d %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S",
                    level=logging.DEBUG,
                    stream=sys.stdout)

logger = logging.getLogger(__name__)


def pytest_addoption(parser):
    parser.addoption("--aws_secret_access_key", action="store", default=None,
                     help="AWS secret key to use for codecommit tests (or from environment variable)")
    parser.addoption("--aws_access_key_id", action="store", default=None,
                     help="AWS access key id to use for codecommit tests (or from environment variable)")


@pytest.fixture(autouse=True)
def fake_docker_env(monkeypatch):
    """Default fixture to simulate a container environment"""
    orig_isfile = os.path.isfile

    def _mock_isfile(path):
        if path == '/.dockerenv':
            return True
        else:
            return orig_isfile(path)

    monkeypatch.setattr(os.path, 'isfile', _mock_isfile)
    yield


@pytest.fixture(scope='session')
def gitrepo_setup(tmpdir_factory):
    """Creates a golden copy of a git repo with some conflicting and mergeable branches"""
    git_repo_path = os.path.join(str(tmpdir_factory.getbasetemp()), 'test_git_repo')
    check_call('git init {}'.format(git_repo_path))

    # remove our hooks if running form workspace
    try:
        os.remove(os.path.join(git_repo_path, '.git/hooks/commit-msg'))
    except OSError:
        pass

    os.chdir(git_repo_path)
    # Add file01
    check_call('echo "hello world" > ./file01', shell=True)
    check_call('git add ./file01')
    check_call('git commit -m "AA-01: initial commit"')
    # Add file02
    check_call('echo "foo bar" > ./file02', shell=True)
    check_call('git add ./file02')
    check_call('git commit -m "AA-02: add file02"')
    # Add file03 in 'mergable' branch
    check_call('git checkout -b mergeable')
    check_call('echo "asdf" > ./file03', shell=True)
    check_call('git add ./file03')
    check_call('git commit -m "AA-03: add file03 in branch \'mergeable\'"')
    # Add file04 in 'conflicting' branch
    check_call('git checkout master')
    check_call('git checkout -b conflicting')
    check_call('echo "abc" > ./file04', shell=True)
    check_call('git add ./file04')
    check_call('git commit -m "AA-04: add file04 in branch \'conflicting\'"')
    # Add file04 in 'master'
    check_call('git checkout master')
    check_call('echo "def" > ./file04', shell=True)
    check_call('git add ./file04')
    check_call('git commit -m "AA-05: add file04 in master"')
    # Add file05 in 'master'
    check_call('echo "xyz" > ./file05', shell=True)
    check_call('git add ./file05')
    check_call('git commit -m "AA-06: add file05 in master"')
    # Add file06 in 'mergeable'
    check_call('git checkout mergeable')
    check_call('echo "987" > ./file06', shell=True)
    check_call('git add ./file06')
    check_call('git commit -m "AA-07: add file06 in branch \'mergeable\'"')

    # Add file07 in ff-mergeable branch
    check_call('git checkout master')
    check_call('git checkout -b ff-mergeable')
    check_call('echo "567" > ./file07', shell=True)
    check_call('git add ./file07')
    check_call('git commit -m "AA-08: add file07 in branch \'ff-mergeable\'"')

    check_call('git status')
    check_call('git log --oneline --graph --all --decorate')

    logger.info("Git repo created at %s", git_repo_path)
    yield git_repo_path
    shutil.rmtree(git_repo_path)


@pytest.fixture
def gitrepo(gitrepo_setup, tmpdir):
    """Yields a copy of the golden copy of the git repo and converts to a bare repo"""
    pristine_copy = os.path.join(str(tmpdir), 'origin_repo')
    shutil.copytree(gitrepo_setup, pristine_copy)
    os.chdir(pristine_copy)
    check_call("git config --bool core.bare true")
    yield pristine_copy
    shutil.rmtree(pristine_copy)


@pytest.fixture
def axops(aws_credentials):
    """Mock axops get tools"""
    # For some reason, mock responses cannot accept the params
    # get_tools_mock_url = 'http://axops-internal.axsys:8085/v1/tools?category=scm'
    get_tools_mock_url = 'http://{}:8085/v1/tools'.format(AXOPS_HOST)
    if aws_credentials:
        for tool in AXOPS_GET_TOOLS_MOCK_RESPONSE['data']:
            if tool['type'] == 'codecommit':
                tool['username'] = aws_credentials['aws_access_key_id']
                tool['password'] = aws_credentials['aws_secret_access_key']
    with responses.mock:
        responses.add(responses.GET,
                      get_tools_mock_url,
                      json=AXOPS_GET_TOOLS_MOCK_RESPONSE,
                      status=200,
                      content_type='application/json')
        yield


@pytest.fixture(scope='session')
def aws_credentials(request):
    """Retrieves AWS credentials from args, or environment variable"""
    aws_secret_access_key = request.config.getoption("--aws_secret_access_key") or os.environ.get('AWS_SECRET_ACCESS_KEY')
    aws_access_key_id = request.config.getoption("--aws_access_key_id") or os.environ.get('AWS_ACCESS_KEY_ID')
    if not aws_access_key_id or not aws_secret_access_key:
        yield None
    else:
        yield {
            'aws_access_key_id': aws_access_key_id,
            'aws_secret_access_key': aws_secret_access_key
        }
