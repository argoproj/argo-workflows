import os

import pytest

from ax.devops.scm.cli import main 


@pytest.mark.skip(reason="Need real SCM repo")
def test_scm_axops_github(axops, tmpdir):
    """Verify we can clone from a github account"""
    local_dir = os.path.join(str(tmpdir), 'local')
    repo_url = 'https://github.com/demo/ansible-example.git'
    main(args=['clone', repo_url, local_dir])


@pytest.mark.skip(reason="Need real SCM repo")
def test_scm_axops_bitbucket(axops, tmpdir):
    """Verify we can clone from a bitbucket account"""
    local_dir = os.path.join(str(tmpdir), 'local')
    repo_url = 'https://bitbucket.org/atlassian_tutorial/helloworld.git'
    main(args=['clone', repo_url, local_dir])


@pytest.mark.skip(reason="Need real SCM repo")
def test_scm_axops_generic_git(axops, tmpdir):
    """Verify we can clone from a generic git server"""
    local_dir = os.path.join(str(tmpdir), 'local')
    repo_url = 'https://bitbucket.org/atlassian_tutorial/helloworld.git'
    main(args=['clone', repo_url, local_dir])


@pytest.mark.skip(reason="Need real SCM repo")
def test_scm_axops_generic_git_passwordless(axops, tmpdir):
    """Verify we can clone from a generic git server"""
    local_dir = os.path.join(str(tmpdir), 'local')
    repo_url = 'https://github.com/demo/goexample.git'
    main(args=['clone', repo_url, local_dir])


@pytest.mark.skip(reason="Need real SCM repo")
def test_scm_axops_codecommit(axops, tmpdir, aws_credentials):
    """Verify we can clone from a code commit repository"""
    if not aws_credentials:
        pytest.skip("AWS credentials must be supplied from command line or env variables")
    local_dir = os.path.join(str(tmpdir), 'local')
    repo_url = 'https://git-codecommit.us-east-1.amazonaws.com/v1/repos/goexample'
    main(args=['clone', repo_url, local_dir])
