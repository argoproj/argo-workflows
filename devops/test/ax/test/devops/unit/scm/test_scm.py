import logging
import os
import time

import pytest

from ax.devops.scm.cli import main
from ax.devops.scm import create_scm_client
from ax.exceptions import AXTimeoutException
from .util import check_output
from ax.devops.exceptions import AXScmException

logger = logging.getLogger(__name__)


def test_scm_git_clone_remote_head(gitrepo, tmpdir):
    """Verify git clone will checkout remote head when commit/branch is omitted"""
    local_dir = os.path.join(str(tmpdir), 'local')
    main(args=['clone', 'file://' + gitrepo, local_dir])
    head_commit = check_output('git -C {} rev-parse HEAD'.format(gitrepo))
    actual_commit = check_output('git -C {} rev-parse HEAD'.format(local_dir))
    assert head_commit == actual_commit


def test_scm_git_clone_commit(gitrepo, tmpdir):
    """Verify we can clone using a specific commit"""
    local_dir = os.path.join(str(tmpdir), 'local')
    commit = check_output('git -C {} log --oneline --all | grep AA-03'.format(gitrepo), shell=True).split()[0]
    main(args=['clone', 'file://' + gitrepo, local_dir, '--commit', commit])
    actual_commit = check_output('git -C {} rev-parse HEAD'.format(local_dir))
    assert actual_commit.startswith(commit)


def test_scm_git_clone_branch(gitrepo, tmpdir):
    """Verify we can clone using a specific branch"""
    local_dir = os.path.join(str(tmpdir), 'local')
    branch_tip_commit = check_output('git -C {} show-ref mergeable'.format(gitrepo)).split()[0]
    main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', 'mergeable'])
    actual_commit = check_output('git -C {} rev-parse HEAD'.format(local_dir))
    assert branch_tip_commit == actual_commit


@pytest.mark.parametrize("branches", [('master', 'mergeable'), ('mergeable', 'master')])
def test_scm_git_merge_no_conflicts(gitrepo, tmpdir, branches):
    """Verify we are able to perform a merge with no conflicts (master->private and vice versa)"""
    local_dir = os.path.join(str(tmpdir), 'local')
    main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', branches[0], '--merge', branches[1]])
    for f in ['file01', 'file02', 'file03', 'file04', 'file05', 'file06']:
        assert os.path.exists(os.path.join(local_dir, f))


@pytest.mark.parametrize("branches", [('master', 'conflicting'), ('conflicting', 'master')])
def test_scm_git_merge_conflicts(gitrepo, tmpdir, branches):
    """Verify merge with conflicts raises exception (private->master and vice versa)"""
    local_dir = os.path.join(str(tmpdir), 'local')
    with pytest.raises(SystemExit):
        main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', branches[0], '--merge', branches[1]])


def test_scm_git_merge_ff_branch(gitrepo, tmpdir):
    """Verify when we merge+push a fast-forwardable branch, we still use a merge commit"""
    local_dir = os.path.join(str(tmpdir), 'local')
    before = check_output('git -C {} log --oneline --graph --all --decorate'.format(gitrepo))
    num_commits_before = before.count('*')
    main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', 'master', '--merge', 'ff-mergeable', '--push'])
    after = check_output('git -C {} log --oneline --graph --all --decorate'.format(gitrepo))
    num_commits_after = after.count('*')
    assert num_commits_after == num_commits_before + 1, "Unexpected number of commits"


@pytest.mark.parametrize("branches", [('master', 'mergeable'), ('mergeable', 'master')])
def test_scm_git_push(gitrepo, tmpdir, branches):
    """Verify we can push changes from a local repo"""
    before = check_output('git -C {} log --oneline --graph --all --decorate'.format(gitrepo))
    num_commits_before = before.count('*')
    local_dir = os.path.join(str(tmpdir), 'local')
    main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', branches[0], '--merge', branches[1]])
    assert "Your branch is ahead" in check_output('git -C {} status'.format(local_dir))
    main(args=['push', local_dir])
    assert "Your branch is up-to-date" in check_output('git -C {} status'.format(local_dir))
    after = check_output('git -C {} log --oneline --graph --all --decorate'.format(gitrepo))
    num_commits_after = after.count('*')
    assert num_commits_after == num_commits_before + 1, "Unexpected number of commits"


def test_scm_git_merge_with_author(gitrepo, tmpdir):
    """Verify e-mail address is tested as a valid e-mail"""
    local_dir = os.path.join(str(tmpdir), 'local')
    mickeymouse = 'Mickey Mouse <mickey@disney.com>'
    main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', 'master', '--merge', 'mergeable', '--author', mickeymouse])
    gitlog = check_output('git -C {} log -1'.format(local_dir))
    assert mickeymouse in gitlog


def test_scm_git_merge_invalid_author(gitrepo, tmpdir):
    """Verify e-mail address is tested as a valid e-mail"""
    local_dir = os.path.join(str(tmpdir), 'local')
    with pytest.raises(SystemExit):
        main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', 'master', '--merge', 'mergeable', '--author', 'invalid'])


def test_scm_git_get_remote_head(gitrepo, tmpdir):
    """See if we can get remote head of a branch"""
    local_dir = os.path.join(str(tmpdir), 'local')
    client = create_scm_client(repo='file://' + gitrepo, path=local_dir)
    client.init()
    client.fetch()
    actual_master_head_commit = check_output('git -C {} show-ref refs/heads/master'.format(gitrepo)).split()[0]
    master_head_commit = client.get_remote_head('master')
    assert master_head_commit == actual_master_head_commit


def test_scm_git_merge_existing_commit_same_branch(gitrepo, tmpdir):
    """Verify merge request for a commit that is already in current branch is a no-op"""
    local_dir = os.path.join(str(tmpdir), 'local')
    before = check_output('git -C {} log --oneline --graph --all --decorate'.format(gitrepo))
    num_commits_before = before.count('*')
    commit = check_output('git -C {} log --oneline --all | grep AA-02'.format(gitrepo), shell=True).split()[0]
    main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', 'master', '--merge', commit])
    after = check_output('git -C {} log --oneline --graph --all --decorate'.format(local_dir))
    num_commits_after = after.count('*')
    assert num_commits_after == num_commits_before, "Unexpected number of commits"


def test_scm_git_merge_branch_same_branch(gitrepo, tmpdir):
    """Verify  merge request from the current branch to the current branch is a no-op"""
    local_dir = os.path.join(str(tmpdir), 'local')
    before = check_output('git -C {} log --oneline --graph --all --decorate'.format(gitrepo))
    num_commits_before = before.count('*')
    main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', 'mergeable', '--merge', 'mergeable'])
    after = check_output('git -C {} log --oneline --graph --all --decorate'.format(local_dir))
    num_commits_after = after.count('*')
    assert num_commits_after == num_commits_before, "Unexpected number of commits"


def test_scm_git_merge_behind_into_ahead(gitrepo, tmpdir):
    """Verify a merge from a branch which is behind, into a branch which ahead is a no-op"""
    local_dir = os.path.join(str(tmpdir), 'local')
    before = check_output('git -C {} log --oneline --graph --all --decorate'.format(gitrepo))
    num_commits_before = before.count('*')
    main(args=['clone', 'file://' + gitrepo, local_dir, '--branch', 'ff-mergeable', '--merge', 'master'])
    expected = check_output('git -C {} log --oneline --graph --all --decorate'.format(gitrepo))
    num_commits_before = before.count('*')
    after = check_output('git -C {} log --oneline --graph --all --decorate'.format(local_dir))
    num_commits_after = after.count('*')
    assert num_commits_after == num_commits_before, "Unexpected number of commits"


def test_scm_client_run_cmd_timeout(gitrepo, tmpdir):
    """Tests timeout facility of ScmClient"""
    client = create_scm_client(path=str(tmpdir), repo='file://' + gitrepo)
    start_time = time.time()
    with pytest.raises(AXTimeoutException):
        client.run_cmd("sleep 10", timeout=1)
    elapsed = time.time() - start_time
    assert elapsed == pytest.approx(1, abs=0.2)


def test_scm_client_run_cmd_no_timeout(gitrepo, tmpdir):
    """Verify timeout facility is not invoked when command completes in time"""
    client = create_scm_client(path=str(tmpdir), repo='file://' + gitrepo)
    start_time = time.time()
    client.run_cmd("sleep 1", timeout=2)
    elapsed = time.time() - start_time
    assert elapsed == pytest.approx(1, abs=0.2)


def test_scm_client_run_cmd_nonzero(gitrepo, tmpdir):
    """Verifies exception is raised upon non-zero return code"""
    client = create_scm_client(path=str(tmpdir), repo='file://' + gitrepo)
    with pytest.raises(AXScmException):
        client.run_cmd("exit 1", shell=True, timeout=2)


def test_scm_client_run_cmd_constructor_timeout(gitrepo, tmpdir):
    """Verifies cmd timeout is honored from constructor"""
    client = create_scm_client(path=str(tmpdir), repo='file://' + gitrepo, cmd_timeout=1)
    with pytest.raises(AXTimeoutException):
        client.run_cmd("sleep 2")
