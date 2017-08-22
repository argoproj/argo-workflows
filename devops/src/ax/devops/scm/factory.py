import logging
import os
import re

from ax.devops.axdb.axops_client import AxopsClient
from .scm import CodeCommitClient, GitClient, HgClient

logger = logging.getLogger('ax.devops.scm')

# NOTE: axscm container is considered a customer like container and runs in axuser namespace/domain. Therefore, the
# hostname must be qualified with axsys domain. If and when this scm library is used for system level services (e.g. 
# for commitdata/commitmonitor) in the axsys namespace, it should use the unqualified hostname (i.e. 'axops').
AXOPS_HOST = 'axops-internal.axsys'
AXOPS_OLD = 'axops.axsys'
axops_client = AxopsClient(host=AXOPS_HOST)
axops_client_old = AxopsClient(host=AXOPS_OLD)


def create_scm_client(path, repo=None, **kwargs):
    """Return a ScmClient instance given a repo or path"""
    if repo:
        return _create_from_url(path=path, repo=repo, **kwargs)
    else:
        return _create_from_path(path=path, **kwargs)


def _find_axops_tool_config(repo):
    """Queries axops and returns the corresponding tool information for the repo"""
    match = re.match(r'^(\w+://)(.*)', repo)
    if not match:
        return None
    if match.group(1) == 'file://':
        return None
    try:
        tools = axops_client.get_tools(category='scm')
    except Exception as e:
        logger.warning("Cannot get tools from axops-internal. Your cluster needs upgrade. Error: %s", e)
        logger.info("Getting tools from axops instead of axops-internal")
        tools = axops_client_old.get_tools(category='scm')

    for tool in tools:
        if repo in tool.get('repos', []):
            return tool
    logging.warning("Credentials for %s not found in database. Assuming passwordless", repo)
    return None


def _create_from_url(path, repo, username=None, password=None, **kwargs):
    """Factory function to create a VcsClient instance given a repo url"""
    tool = _find_axops_tool_config(repo)
    username = username or (tool.get('username', None) if tool else None)
    password = password or (tool.get('password', None) if tool else None)

    if tool and tool['type'] == 'codecommit':
        return CodeCommitClient(path, repo, username=username, password=password, **kwargs)
    elif re.match(r"file://", repo):
        origin_path = os.path.normpath(repo[7:])
        if not os.path.isdir(origin_path):
            raise ValueError("{} does not exist".format(origin_path))
        if os.path.exists(os.path.join(origin_path, '.git')):
            return GitClient(path, repo, **kwargs)
        elif os.path.exists(os.path.join(origin_path, '.hg')):
            return HgClient(path, repo, **kwargs)
        else:
            raise ValueError("{} is not a git/hg repo".format(path))
    elif re.match(r".*\.git/?$", repo):
        return GitClient(path, repo, username=username, password=password, **kwargs)
    else:
        # TODO: add hg support when needed
        # return HgClient(path, repo, username=username, password=password, **kwargs)
        return GitClient(path, repo, username=username, password=password, **kwargs)


def _create_from_path(path, **kwargs):
    """Factory function to create a VcsClient instance given an existing repo path"""
    path = os.path.normpath(path)
    if not os.path.isdir(path):
        raise ValueError("{} does not exist".format(path))
    if os.path.exists(os.path.join(path, '.git')):
        return GitClient(path, **kwargs)
    elif os.path.exists(os.path.join(path, '.hg')):
        raise NotImplementedError("Mercurial unsupported")
    else:
        raise NotImplementedError("{} repo type unsupported".format(path))
