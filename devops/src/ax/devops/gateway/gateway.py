import boto3
import json
import logging
import os
import re
import shutil
import socket

from concurrent.futures import ThreadPoolExecutor, as_completed
from retrying import retry
try:
    from urlparse import urlparse
except ImportError:
    from urllib.parse import urlparse
from urllib.parse import unquote

from .constants import AxEventTypes, ScmVendors, SUPPORTED_TYPES, DEFAULT_CONCURRENCY, DEFAULT_INTERVAL
from .scm_rest.bitbucket_client import BitBucketClient
from .scm_rest.github_client import GitHubClient
from .scm_rest.gitlab_client import GitLabClient
from .repo_manager import RepoManager
from .event_trigger import EventTrigger

from ax.exceptions import AXApiInvalidParam, AXApiAuthFailed, AXApiForbiddenReq, AXApiInternalError
from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.exceptions import AXScmException
from ax.devops.jira.jira_client import JiraClient
from ax.devops.kafka.kafka_client import EventNotificationClient
from ax.notification_center import FACILITY_GATEWAY
from ax.notification_center import CODE_JOB_CI_ELB_CREATION_FAILURE, CODE_JOB_CI_ELB_VERIFICATION_TIMEOUT, \
    CODE_JOB_CI_WEBHOOK_CREATION_FAILURE, CODE_JOB_CI_ELB_DELETION_FAILURE, CODE_JOB_CI_WEBHOOK_DELETION_FAILURE, \
    CODE_CONFIGURATION_SCM_CONNECTION_ERROR
from ax.devops.redis.redis_client import RedisClient, DB_REPORTING
from ax.devops.scm.scm import GitClient


logger = logging.getLogger(__name__)


class Gateway(object):
    """Repo Controller"""
    BASE_DIR = '/ax/data/repos'
    BRANCH_CACHE_TTL = 5 * 60  # 5 minutes TTL as we expect we won't finish upgrade within 5 minutes
    NAMESPACE = 'gateway'

    CLUSTER_NAME_ID = os.environ.get('AX_CLUSTER')
    CUSTOMER_ID = os.environ.get('AX_CUSTOMER_ID')
    S3_BUCKET_NAME = os.getenv("ARGO_DATA_BUCKET_NAME") or 'applatix-cluster-{account}-{seq}'.format(account=CUSTOMER_ID, seq=0)
    s3_bucket = boto3.resource('s3', aws_access_key_id=os.environ.get("ARGO_S3_ACCESS_KEY_ID", None),
                aws_secret_access_key=os.environ.get("ARGO_S3_ACCESS_KEY_SECRET", None),
                endpoint_url=os.environ.get("ARGO_S3_ENDPOINT", None)).Bucket(S3_BUCKET_NAME)

    def __init__(self):
        self.axdb_client = AxdbClient()
        self.axops_client = AxopsClient()
        self.axsys_client = AxsysClient()
        self.redis_client = RedisClient('redis', db=DB_REPORTING)
        self.event_notification_client = EventNotificationClient(FACILITY_GATEWAY)
        self.scm_clients = {
            ScmVendors.BITBUCKET: BitBucketClient(),
            ScmVendors.GITHUB: GitHubClient(),
            ScmVendors.GITLAB: GitLabClient()
        }
        self.repo_manager = RepoManager(DEFAULT_CONCURRENCY, DEFAULT_INTERVAL)
        self.event_trigger = EventTrigger()

    def get_repos(self, scm_type, url, username, password):
        """Get all repos owned by the user."""
        if scm_type in {ScmVendors.BITBUCKET, ScmVendors.GITHUB, ScmVendors.GITLAB}:
            try:
                repos = self.scm_clients[scm_type].get_repos(username, password)
            except Exception as e:
                logger.warning('Unable to connect to %s: %s', scm_type, e)
                detail = {
                    'type': scm_type,
                    'username': username,
                    'error': str(e.detail)
                }
                self.event_notification_client.send_message_to_notification_center(CODE_CONFIGURATION_SCM_CONNECTION_ERROR,
                                                                                   detail=detail)
                raise AXApiInvalidParam('Cannot connect to %s server' % scm_type)
            else:
                return repos
        elif scm_type == ScmVendors.GIT:
            _, vendor, repo_owner, repo_name = Gateway.parse_repo(url)
            path = '/tmp/{}/{}/{}'.format(vendor, repo_owner, repo_name)
            if os.path.isfile(path):
                os.remove(path)
            if os.path.isdir(path):
                shutil.rmtree(path)
            os.makedirs(path)
            client = GitClient(path=path, repo=url, username=username, password=password)
            try:
                client.list_remote()
            except Exception as e:
                logger.warning('Unable to connect to git server (%s): %s', url, e)
                detail = {
                    'type': scm_type,
                    'url': url,
                    'username': username,
                    'error': str(e)
                }
                self.event_notification_client.send_message_to_notification_center(CODE_CONFIGURATION_SCM_CONNECTION_ERROR,
                                                                                   detail=detail)
                raise AXApiInvalidParam('Cannot connect to git server')
            else:
                return {url: url}
        elif scm_type == ScmVendors.CODECOMMIT:
            repos = {}
            region = 'us-east-1'
            default_url_format = 'https://git-codecommit.{}.amazonaws.com/v1/repos/{}'
            client = boto3.client('codecommit', aws_access_key_id=username, aws_secret_access_key=password,
                                  region_name=region)
            try:
                response = client.list_repositories().get('repositories', [])
                for r in response:
                    repo_url = default_url_format.format(region, r['repositoryName'])
                    repos[repo_url] = repo_url
            except Exception as exc:
                detail = {
                    'type': scm_type,
                    'region': region,
                    'url': default_url_format.format(region, ''),
                    'username': username,
                    'error': 'Cannot connect to CodeCommit' + str(exc)
                }
                self.event_notification_client.send_message_to_notification_center(CODE_CONFIGURATION_SCM_CONNECTION_ERROR,
                                                                                   detail=detail)
                raise AXApiInvalidParam('Cannot connect to CodeCommit: %s' % exc)
            else:
                return repos
        else:
            return {}

    @staticmethod
    def parse_repo(repo):
        """Parse repo url into 4-tuple (protocol, vendor, repo_owner, repo_name).

        :param repo:
        :return:
        """
        parsed_url = urlparse(repo)
        protocol, vendor = parsed_url.scheme, parsed_url.hostname
        m = re.match(r'/([a-zA-Z0-9\-]+)/([a-zA-Z0-9_.\-/]+)', parsed_url.path)
        if not m:
            raise AXScmException('Illegal repo URL', detail='Illegal repo URL ({})'.format(repo))
        repo_owner, repo_name = m.groups()
        return protocol, vendor, repo_owner, repo_name

    def has_webhook(self, repo):
        """Test if there is any repo which uses webhook.

        :param repo:
        :return:
        """
        tools = self.axops_client.get_tools(category='scm')
        for i in range(len(tools)):
            use_webhook = tools[i].get('use_webhook', False)
            repos = set(tools[i].get('repos', []))
            repos -= {repo}
            if use_webhook and repos:
                return True
        return False

    def get_webhook(self, vendor, repo):
        """Get webhook

        :param vendor:
        :param repo:
        :returns:
        """
        logger.info('Retrieving webhook (repo: %s) ...', repo)
        return self.scm_clients[vendor].get_webhook(repo)

    def create_webhook(self, vendor, repo):
        """Create webhook

        :param vendor:
        :param repo:
        :returns:
        """

        @retry(wait_fixed=5000, stop_max_delay=20 * 60 * 1000)
        def _verify_elb(hostname):
            try:
                logger.info('Verifying ELB (%s) ...', hostname)
                ip = socket.gethostbyname(hostname)
                logger.info('Successfully resolved ELB (%s) to IP (%s)', hostname, ip)
            except Exception as e:
                logger.error('ELB not ready: %s', str(e))
                raise AXApiInternalError('ELB not ready', str(e))

        ip_range = self.scm_clients[vendor].get_webhook_whitelist()

        # Create ELB
        payload = {'ip_range': ip_range, 'external_port': 8443, 'internal_port': 8087}
        try:
            logger.info('Creating ELB for webhook ...')
            result = self.axsys_client.create_webhook(**payload)
        except Exception as e:
            logger.error('Failed to create ELB for webhook: %s', str(e))
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_ELB_CREATION_FAILURE,
                                                                               detail=payload)
            raise AXApiInternalError('Failed to create ELB for webhook', str(e))
        else:
            logger.info('Successfully created ELB for webhook')

        # Verify ELB
        hostname = result['hostname']
        try:
            _verify_elb(hostname)
        except Exception as e:
            logger.error('Timed out on waiting for ELB to be available: %s', str(e))
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_ELB_VERIFICATION_TIMEOUT,
                                                                               detail={'hostname': hostname})
            raise AXApiInternalError('Timed out on waiting for ELB to be available: %s' % str(e))

        # Create webhook
        try:
            logger.info('Creating webhook (repo: %s) ...', repo)
            self.scm_clients[vendor].create_webhook(repo)
        except AXApiAuthFailed as e:
            logger.error('Invalid credential supplied')
            detail = {
                'repo': repo,
                'error': 'Invalid credential supplied:' + str(e)
            }
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_CREATION_FAILURE,
                                                                               detail=detail)
            raise AXApiInvalidParam('User authentication failed', detail=str(e))
        except AXApiForbiddenReq as e:
            logger.error('Supplied credential is valid but having insufficient permission')
            detail = {
                'repo': repo,
                'error': 'Supplied credential is valid but having insufficient permission:' + str(e)
            }
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_CREATION_FAILURE,
                                                                               detail=detail)
            raise AXApiInvalidParam('User has insufficient permission', detail=str(e))
        except Exception as e:
            logger.error('Failed to configure webhook: %s', e)
            detail = {
                'repo': repo,
                'error': 'Failed to configure webhook:' + str(e)
            }
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_CREATION_FAILURE,
                                                                               detail=detail)
            raise AXApiInternalError('Failed to configure webhook', str(e))
        else:
            logger.info('Successfully created webhook (repo: %s)', repo)
            return {}

    def delete_webhook(self, vendor, repo):
        """Delete webhook

        :param vendor:
        :param repo:
        :returns:
        """
        # Delete webhook
        try:
            logger.info('Deleting webhook (repo: %s) ...', repo)
            self.scm_clients[vendor].delete_webhook(repo)
        except AXApiAuthFailed as e:
            logger.error('Invalid credential supplied')
            detail = {
                'repo': repo,
                'error': 'Invalid credential supplied:' + str(e)
            }
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_DELETION_FAILURE,
                                                                               detail=detail)
            raise AXApiInvalidParam('User authentication failed', detail=str(e))
        except AXApiForbiddenReq as e:
            logger.error('Supplied credential is valid but having insufficient permission')
            detail = {
                'repo': repo,
                'error': 'Supplied credential is valid but having insufficient permission:' + str(e)
            }
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_DELETION_FAILURE,
                                                                               detail=detail)
            raise AXApiInvalidParam('User has insufficient permission', detail=str(e))
        except Exception as e:
            logger.error('Failed to delete webhook: %s', e)
            detail = {
                'repo': repo,
                'error': 'Failed to delete webhook:' + str(e)
            }
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_DELETION_FAILURE,
                                                                               detail=detail)
            raise AXApiInternalError('Failed to delete webhook', str(e))
        else:
            logger.info('Successfully deleted webhook (repo: %s)', repo)

        # Delete ELB
        try:
            if not self.has_webhook(repo):
                logger.info('Deleting ELB for webhook ...')
                self.axsys_client.delete_webhook()
        except Exception as e:
            logger.error('Failed to delete ELB for webhook: %s', str(e))
            detail = {'repo': repo,
                      'error': 'Failed to delete ELB for webhook' + str(e)
                      }
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_ELB_DELETION_FAILURE,
                                                                               detail=detail)
            raise AXApiInternalError('Failed to delete ELB for webhook', str(e))
        else:
            logger.info('Successfully deleted ELB for webhook')
            return {}

    def purge_branches(self, repo, branch=None):
        """Purge branch heads.

        :param repo:
        :param branch:
        :return:
        """
        if not repo:
            raise AXApiInvalidParam('Missing required parameter', 'Missing required parameter (repo)')
        logger.info('Purging branch heads (repo: %s, branch: %s) ...', repo, branch)

        try:
            if not branch:
                self.axdb_client.purge_branch_heads(repo)
            else:
                self.axdb_client.purge_branch_head(repo, branch)
        except Exception as e:
            message = 'Unable to purge branch heads'
            detail = 'Unable to purge branch heads (repo: {}, branch: {}): {}'.format(repo, branch, str(e))
            logger.error(detail)
            raise AXApiInternalError(message, detail)
        else:
            logger.info('Successfully purged branch heads')

    def get_branches(self, repo=None, branch=None, order_by=None, limit=None):
        """Get branches.

        :param repo:
        :param branch:
        :param order_by:
        :param limit:
        :return:
        """

        def _get_branches(workspace):
            """Retrieve list of remote branches in the workspace.

            :param workspace:
            :return: a list of dictionaries.
            """
            try:
                key = '{}:{}'.format(Gateway.NAMESPACE, workspace)
                if self.redis_client.exists(key):
                    logger.info('Loading cache (workspace: %s) ...', workspace)
                    results = self.redis_client.get(key, decoder=json.loads)
                    return results
                else:
                    logger.info('Scanning workspace (%s) ...', workspace)
                    git_client = GitClient(path=workspace, read_only=True)
                    repo = git_client.get_remote()
                    branches = git_client.get_remote_heads()
                    results = []
                    for i in range(len(branches)):
                        results.append({
                            'repo': repo,
                            'name': branches[i]['reference'],
                            'revision': branches[i]['commit'],
                            'commit_date': branches[i]['commit_date']
                        })
                    logger.info('Saving cache (workspace: %s) ...', workspace)
                    self.redis_client.set(key, results, expire=Gateway.BRANCH_CACHE_TTL, encoder=json.dumps)
                    return results
            except Exception as e:
                logger.warning('Failed to scan workspace (%s): %s', workspace, e)
                return []

        logger.info('Retrieving branches (repo: %s, branch: %s) ...', repo, branch)
        if repo:
            repo = unquote(repo)
            _, vendor, repo_owner, repo_name = self.parse_repo(repo)
            workspaces = ['{}/{}/{}/{}'.format(Gateway.BASE_DIR, vendor, repo_owner, repo_name)]
        else:
            dirs = [dir_name[0] for dir_name in os.walk(Gateway.BASE_DIR) if dir_name[0].endswith('/.git')]
            workspaces = list(map(lambda v: v[:-5], dirs))

        branches = []
        with ThreadPoolExecutor(max_workers=20) as executor:
            futures = []
            for i in range(len(workspaces)):
                futures.append(executor.submit(_get_branches, workspaces[i]))
            for future in as_completed(futures):
                try:
                    data = future.result()
                except Exception as e:
                    logger.warning('Unexpected exception occurred during processing: %s', e)
                else:
                    for i in range(len(data)):
                        branches.append(data[i])
        if branch:
            pattern = '.*{}.*'.format(branch.replace('*', '.*'))
            branches = [branches[i] for i in range(len(branches)) if re.match(pattern, branches[i]['name'])]
        if order_by == 'commit_date':
            branches = sorted(branches, key=lambda v: v['commit_date'])
        elif order_by == '-commit_date':
            branches = sorted(branches, key=lambda v: v['commit_date'], reverse=True)
        elif order_by == '-native':
            branches = sorted(branches, key=lambda v: (v['repo'], v['name']), reverse=True)
        else:
            branches = sorted(branches, key=lambda v: (v['repo'], v['name']))
        if limit:
            branches = branches[:limit]
        logger.info('Successfully retrieved %s branches', len(branches))
        return branches

    @staticmethod
    def _get_commits(workspace, branch=None, since=None, until=None, commit=None, author=None, committer=None,
                     description=None, limit=None):
        """Search for commits in a workspace."""
        try:
            logger.info('Scanning workspace (%s) for commits ...', workspace)
            git_client = GitClient(path=workspace, read_only=True)
            if commit and commit.startswith('~'):
                commit = commit[1:]
            if author and author.startswith('~'):
                author = author[1:]
            if committer and committer.startswith('~'):
                committer = committer[1:]
            if description and description.startswith('~'):
                description = description[1:]
            return git_client.get_commits(branch=branch, commit=commit, since=since, until=until, author=author,
                                          committer=committer, description=description, limit=limit)
        except Exception as e:
            logger.warning('Failed to scan workspace (%s): %s', workspace, e)

    @staticmethod
    def _get_commit(workspace, commit):
        """Get a commit from a workspace."""
        try:
            logger.info('Scanning workspace (%s) for commit (%s) ...', workspace, commit)
            git_client = GitClient(path=workspace, read_only=True)
            return git_client.get_commit(commit)
        except Exception as e:
            logger.warning('Failed to scan workspace (%s): %s', workspace, e)

    @staticmethod
    def _parse_repo_branch(repo, branch, repo_branch):
        """Parse repo / branch / repo_branch."""
        if repo:
            try:
                repo = unquote(repo)
                _, vendor, repo_owner, repo_name = Gateway.parse_repo(repo)
            except Exception as e:
                msg = 'Unable to parse repo: %s', e
                logger.error(msg)
                raise AXApiInvalidParam('Unable to parse repo', msg)
            else:
                dir = '{}/{}/{}/{}'.format(Gateway.BASE_DIR, vendor, repo_owner, repo_name)
                workspaces = {dir: [branch] if branch else []}
        elif repo_branch:
            try:
                repo_branch = json.loads(repo_branch)
                workspaces = {}
                for repo in repo_branch.keys():
                    repo = unquote(repo)
                    _, vendor, repo_owner, repo_name = Gateway.parse_repo(repo)
                    dir = '{}/{}/{}/{}'.format(Gateway.BASE_DIR, vendor, repo_owner, repo_name)
                    if dir not in workspaces:
                        workspaces[dir] = set()
                    for branch in repo_branch[repo]:
                        workspaces[dir].add(branch)
            except Exception as e:
                msg = 'Unable to parse repo_branch: %s' % str(e)
                logger.error(msg)
                raise AXApiInvalidParam('Unable to parse repo_branch', msg)
        else:
            dirs = [dir[0] for dir in os.walk(Gateway.BASE_DIR) if dir[0].endswith('/.git')]
            workspaces = list(map(lambda v: v[:-5], dirs))
            workspaces = dict([(k, [branch] if branch else []) for k in workspaces])
        return workspaces

    @staticmethod
    def _put_file(repo, branch, path):
        """Put a file in s3.

        :param repo:
        :param branch:
        :param path:
        :return:
        """
        _, vendor, repo_owner, repo_name = Gateway.parse_repo(repo)
        workspace = '{}/{}/{}/{}'.format(Gateway.BASE_DIR, vendor, repo_owner, repo_name)
        if not os.path.isdir(workspace):
            raise AXApiInvalidParam('Invalid repository', 'Invalid repository ({})'.format(repo))
        try:
            logger.info('Extracting file content from repository (repo: %s, branch: %s, path: %s) ...',
                        repo, branch, path)
            git_client = GitClient(path=workspace, read_only=True)
            files = git_client.get_files(branch=branch, subdir=path, binary_mode=True)
        except Exception as e:
            message = 'Failed to extract file content'
            detail = '{}: {}'.format(message, str(e))
            logger.error(detail)
            raise AXApiInternalError(message, detail)
        else:
            if len(files) == 0:
                raise AXApiInvalidParam('Unable to locate file with given information')
            file_content = files[0]['content']
            logger.info('Successfully extracted file content')

        try:
            # Cluster name id always has the form <cluster_name>-<36_bytes_long_cluster_id>
            cluster_name, cluster_id = Gateway.CLUSTER_NAME_ID[:-37], Gateway.CLUSTER_NAME_ID[-36:]
            key = '{cluster_name}/{cluster_id}/{vendor}/{repo_owner}/{repo_name}/{branch}/{path}'.format(
                cluster_name=cluster_name, cluster_id=cluster_id, vendor=vendor,
                repo_owner=repo_owner, repo_name=repo_name, branch=branch, path=path)
            logger.info('Uploading file content to s3 (bucket: %s, key: %s) ...', Gateway.S3_BUCKET_NAME, key)
            response = Gateway.s3_bucket.Object(key).put(Body=file_content)
            etag = response.get('ETag')
            if etag:
                etag = json.loads(etag)
        except Exception as e:
            message = 'Failed to upload file content'
            detail = '{}: {}'.format(message, str(e))
            logger.error(detail)
            raise AXApiInternalError(message, detail)
        else:
            logger.info('Successfully uploaded file content')
            return {'bucket': Gateway.S3_BUCKET_NAME, 'key': key, 'etag': etag}

    @staticmethod
    def _delete_file(repo, branch, path):
        """Delete a file from s3.

        :param repo:
        :param branch:
        :param path:
        :return:
        """
        _, vendor, repo_owner, repo_name = Gateway.parse_repo(repo)
        try:
            cluster_name, cluster_id = Gateway.CLUSTER_NAME_ID[:-37], Gateway.CLUSTER_NAME_ID[-36:]
            key = '{cluster_name}/{cluster_id}/{vendor}/{repo_owner}/{repo_name}/{branch}/{path}'.format(
                cluster_name=cluster_name, cluster_id=cluster_id, vendor=vendor,
                repo_owner=repo_owner, repo_name=repo_name, branch=branch, path=path)
            logger.info('Deleting file from s3 (bucket: %s, key: %s) ...', Gateway.S3_BUCKET_NAME, key)
            Gateway.s3_bucket.Object(key).delete()
        except Exception as e:
            message = 'Failed to delete file'
            detail = '{}: {}'.format(message, str(e))
            logger.error(detail)
            raise AXApiInternalError(message, detail)
        else:
            logger.info('Successfully deleted file')
            return {'bucket': Gateway.S3_BUCKET_NAME, 'key': key}

    @staticmethod
    def init_jira_client(axops_client, url=None, username=None, password=None):
        """Initialize an Jira client"""

        def get_jira_configuration():
            js = axops_client.get_tools(category='issue_management', type='jira')
            if js:
                return {'url': js[0]['url'],
                        'username': js[0]['username'],
                        'password': js[0]['password']
                        }
            else:
                return dict()

        if url is None or username is None or password is None:
            conf = get_jira_configuration()
            if not conf:
                raise AXApiInvalidParam('No JIRA configured')
            else:
                url, username, password = conf['url'], conf['username'], conf['password']
        return JiraClient(url, username, password)

    # Verify whether this function is still needed
    def check_github_whitelist(self):
        if not self.is_github_webhook_enabled():
            logger.info('No GitHub webhook configured')
            return
        configured = self.get_from_cache()
        logger.info('The configured GitHub webhook whitelist is %s', configured)
        advertised = self.scm_clients[ScmVendors.GITHUB].get_webhook_whitelist()
        logger.info('The GitHub webhook whitelist is %s', advertised)
        if set(configured) == set(advertised):
            logger.info('No update needed')
        else:
            # Create ELB
            payload = {'ip_range': advertised, 'external_port': 8443, 'internal_port': 8087}
            try:
                logger.info('Creating ELB for webhook ...')
                self.axsys_client.create_webhook(**payload)
            except Exception as exc:
                logger.error('Failed to create ELB for webhook: %s', str(exc))
                self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_ELB_CREATION_FAILURE,
                                                                                   detail=payload)
            else:
                # Update cache
                self.write_to_cache(advertised)
                logger.info('Successfully updated ELB for webhook')

    def is_github_webhook_enabled(self):
        """ Check whether the webhook is configured or not"""
        github_data = self.axops_client.get_tools(type='github')
        use_webhook = [each for each in github_data if each['use_webhook']]
        return bool(use_webhook)

    @staticmethod
    def write_to_cache(ip_range):
        """ Store the webhook whitelist info"""
        cache_file = '/tmp/github_webhook_whitelist'
        with open(cache_file, 'w+') as f:
            f.write(json.dumps(ip_range))

    def get_from_cache(self):
        """ Get cached webhook whitelist info, otherwise get from axmon"""
        cache_file = '/tmp/github_webhook_whitelist'
        ip_range = list()
        if os.path.exists(cache_file):
            with open(cache_file, 'r+') as f:
                data = f.readlines()
                ip_range = json.loads(data[0])
        else:
            logger.debug('No cache file')
            try:
                data = self.axsys_client.get_webhook()
            except Exception as exc:
                logger.warning(exc)
            else:
                logger.info('Write whitelist info to cache file')
                ip_range = data['ip_ranges']
                self.write_to_cache(ip_range)
        return ip_range
