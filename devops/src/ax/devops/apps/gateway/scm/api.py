import boto3
import copy
import datetime
import json
import logging
import os
import re
import shutil
import socket
from concurrent.futures import ThreadPoolExecutor, as_completed
from retrying import retry
from urllib.parse import unquote

from rest_framework.decorators import list_route, api_view
from rest_framework.response import Response
from rest_framework.viewsets import GenericViewSet

from gateway.kafka import event_notification_client
from gateway.settings import LOGGER_NAME
from scm.models import SCM
from scm.serializers import SCMSerializer

from ax.devops.apps.workers.repo_manager import BASE_DIR
from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.ci.constants import AxEventTypes, ScmVendors
from ax.devops.ci.event_translators import EventTranslator
from ax.devops.kafka.kafka_client import ProducerClient
from ax.devops.redis.redis_client import RedisClient, DB_REPORTING
from ax.devops.scm.scm import GitClient
from ax.devops.scm_rest.bitbucket_client import BitBucketClient
from ax.devops.scm_rest.github_client import GitHubClient
from ax.devops.scm_rest.gitlab_client import GitLabClient
from ax.devops.settings import AxSettings
from ax.devops.utility.utilities import AxPrettyPrinter, parse_repo, top_k, sort_str_dictionaries
from ax.exceptions import AXApiInvalidParam, AXApiAuthFailed, AXApiForbiddenReq, AXApiInternalError
from ax.notification_center import CODE_JOB_CI_STATUS_REPORTING_FAILURE, CODE_JOB_CI_ELB_CREATION_FAILURE, \
    CODE_JOB_CI_ELB_VERIFICATION_TIMEOUT, CODE_JOB_CI_WEBHOOK_CREATION_FAILURE, CODE_JOB_CI_ELB_DELETION_FAILURE, \
    CODE_JOB_CI_WEBHOOK_DELETION_FAILURE, CODE_JOB_CI_EVENT_CREATION_FAILURE, CODE_JOB_CI_EVENT_TRANSLATE_FAILURE, \
    CODE_JOB_CI_YAML_UPDATE_FAILURE, CODE_CONFIGURATION_SCM_CONNECTION_ERROR

logger = logging.getLogger('{}.{}'.format(LOGGER_NAME, 'scm'))

TYPE_BITBUCKET = ScmVendors.BITBUCKET
TYPE_GITHUB = ScmVendors.GITHUB
TYPE_GITLAB = ScmVendors.GITLAB
TYPE_GIT = ScmVendors.GIT
TYPE_CODECOMMIT = ScmVendors.CODECOMMIT
SUPPORTED_TYPES = {
    TYPE_BITBUCKET,
    TYPE_GITHUB,
    TYPE_GITLAB,
    TYPE_GIT,
    TYPE_CODECOMMIT
}
NAMESPACE = 'gateway'
BRANCH_CACHE_TTL = 5 * 60  # 5 minutes TTL as we expect we won't finish upgrade within 5 minutes

CLUSTER_NAME_ID = os.environ.get('AX_CLUSTER')
CUSTOMER_ID = os.environ.get('AX_CUSTOMER_ID')
S3_BUCKET_NAME = 'applatix-cluster-{account}-{seq}'.format(account=CUSTOMER_ID, seq=0)
s3_bucket = boto3.resource('s3').Bucket(S3_BUCKET_NAME)

axdb_client = AxdbClient()
axops_client = AxopsClient()
axsys_client = AxsysClient()
redis_client = RedisClient('redis', db=DB_REPORTING)


class SCMViewSet(GenericViewSet):
    """View set for SCM."""

    queryset = SCM.objects.all()
    serializer_class = SCMSerializer

    scm_clients = {
        ScmVendors.BITBUCKET: BitBucketClient(),
        ScmVendors.GITHUB: GitHubClient(),
        ScmVendors.GITLAB: GitLabClient()
    }
    supported_protocols = {'https'}

    @list_route(methods=['POST', ])
    def test(self, request):
        """Test connection to SCM server.

        :param request:
        :return:
        """
        scm_type = request.data.get('type', '').lower()
        url = request.data.get('url', '').lower()
        username = request.data.get('username', None)
        password = request.data.get('password', None)
        logger.info('Received request (type: %s, url: %s, username: %s, password: ******)', scm_type, url, username)
        if not scm_type:
            raise AXApiInvalidParam('Missing required parameters', detail='Required parameters (type)')
        if scm_type not in SUPPORTED_TYPES:
            raise AXApiInvalidParam('Invalid parameter values', detail='Unsupported type ({})'.format(scm_type))
        if scm_type == TYPE_GIT:
            assert url, AXApiInvalidParam('Missing required parameters',
                                          detail='Require parameter (url) when type is {}'.format(TYPE_GIT))
        else:
            assert all([username, password]), AXApiInvalidParam('Missing required parameters',
                                                                detail='Required parameters (username, password, url)')
        try:
            repos = self.get_repos(scm_type, url, username, password)
        except Exception as e:
            logger.warning('Failed to get repositories: %s', e)
            raise AXApiInternalError('Failed to get repositories', detail=str(e))
        else:
            return Response({'repos': repos})

    def get_repos(self, scm_type, url, username, password):
        """Get all repos owned by the user.

        :param scm_type:
        :param url:
        :param username:
        :param password:
        :return:
        """
        if scm_type in {TYPE_BITBUCKET, TYPE_GITHUB, TYPE_GITLAB}:
            try:
                repos = self.scm_clients[scm_type].get_repos(username, password)
            except Exception as e:
                logger.warning('Unable to connect to %s: %s', scm_type, e)
                detail = {
                    'type': scm_type,
                    'username': username,
                    'error': str(e.detail)
                }
                event_notification_client.send_message_to_notification_center(CODE_CONFIGURATION_SCM_CONNECTION_ERROR,
                                                                              detail=detail)
                raise AXApiInvalidParam('Cannot connect to %s server' % scm_type)
            else:
                return repos
        elif scm_type == TYPE_GIT:
            _, vendor, repo_owner, repo_name = parse_repo(url)
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
                event_notification_client.send_message_to_notification_center(CODE_CONFIGURATION_SCM_CONNECTION_ERROR,
                                                                              detail=detail)
                raise AXApiInvalidParam('Cannot connect to git server')
            else:
                return {url: url}
        elif scm_type == TYPE_CODECOMMIT:
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
                event_notification_client.send_message_to_notification_center(CODE_CONFIGURATION_SCM_CONNECTION_ERROR,
                                                                              detail=detail)
                raise AXApiInvalidParam('Cannot connect to CodeCommit: %s' % exc)
            else:
                return repos
        else:
            return {}

    @list_route(methods=['POST', ])
    def events(self, request):
        """Create a DevOps event.

        :param request:
        :return:
        """
        payload, headers = request.data, request.META
        try:
            logger.info('Translating SCM event ...')
            events = EventTranslator.translate(payload, headers)
        except Exception as e:
            logger.error('Failed to translate event: %s', e)
            # Todo Tianhe Issue: #330 comment out for now because it is distracting
            # event_notification_client.send_message_to_notification_center(CODE_JOB_CI_EVENT_TRANSLATE_FAILURE,
            #                                                               detail={
            #                                                                   'payload': payload,
            #                                                                   'error': str(e)
            #                                                               })
            raise AXApiInternalError('Failed to translate event', detail=str(e))
        else:
            logger.info('Successfully translated event')

        kafka_client = ProducerClient()
        successful_events = []
        for event in events:
            if event['type'] == AxEventTypes.PING:
                logger.info('Received a PING event, skipping service creation ...')
                continue
            else:
                try:
                    logger.info('Creating AX event ...\n%s', AxPrettyPrinter().pformat(event))
                    key = '{}_{}_{}'.format(event['repo'], event['branch'], event['commit'])
                    kafka_client.send(AxSettings.TOPIC_DEVOPS_CI_EVENT, key=key, value=event, timeout=120)
                except Exception as e:
                    event_notification_client.send_message_to_notification_center(CODE_JOB_CI_EVENT_CREATION_FAILURE,
                                                                                  detail={
                                                                                      'event_type': event.get('type', 'UNKNOWN'),
                                                                                      'error': str(e)
                                                                                  })
                    logger.warning('Failed to create AX event: %s', e)
                else:
                    logger.info('Successfully created AX event')
                    successful_events.append(event)
        kafka_client.close()
        return Response(successful_events)

    @list_route(methods=['POST'])
    def reports(self, request):
        """Report build/test status to source control tool.

        :param request:
        :return:
        """
        logger.info('Received reporting request (payload: %s)', request.data)
        id = request.data.get('id')
        repo = request.data.get('repo')
        if not id:
            raise AXApiInvalidParam('Missing required parameters', detail='Required parameters (id)')

        try:
            if not repo:
                cache = redis_client.get(request.data['id'], decoder=json.loads)
                repo = cache['repo']
            vendor = axops_client.get_tool(repo)['type']
            if vendor not in self.scm_clients.keys():
                raise AXApiInvalidParam('Invalid parameter values', detail='Unsupported type ({})'.format(vendor))
            result = self.scm_clients[vendor].upload_job_result(request.data)
            if result == -1:
                logger.info('GitHub does not support status report for the non-sha commits. Skip.')
        except Exception as e:
            logger.error('Failed to report status: %s', e)
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_STATUS_REPORTING_FAILURE,
                                                                          detail=request.data)
            raise AXApiInternalError('Failed to report status', detail=str(e))
        else:
            logger.info('Successfully reported status')
            return Response(result)

    @staticmethod
    def _has_webhook(repo):
        """Test if there is any repo which uses webhook.

        :param repo:
        :return:
        """
        tools = axops_client.get_tools(category='scm')
        for i in range(len(tools)):
            use_webhook = tools[i].get('use_webhook', False)
            repos = set(tools[i].get('repos', []))
            repos -= {repo}
            if use_webhook and repos:
                return True
        return False

    def _get_webhook(self, vendor, repo):
        """Get webhook

        :param vendor:
        :param repo:
        :returns:
        """
        logger.info('Retrieving webhook (repo: %s) ...', repo)
        return self.scm_clients[vendor].get_webhook(repo)

    def _create_webhook(self, vendor, repo):
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
            result = axsys_client.create_webhook(**payload)
        except Exception as e:
            logger.error('Failed to create ELB for webhook: %s', str(e))
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_ELB_CREATION_FAILURE,
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
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_ELB_VERIFICATION_TIMEOUT,
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
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_CREATION_FAILURE,
                                                                          detail=detail)
            raise AXApiInvalidParam('User authentication failed', detail=str(e))
        except AXApiForbiddenReq as e:
            logger.error('Supplied credential is valid but having insufficient permission')
            detail = {
                'repo': repo,
                'error': 'Supplied credential is valid but having insufficient permission:' + str(e)
            }
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_CREATION_FAILURE,
                                                                          detail=detail)
            raise AXApiInvalidParam('User has insufficient permission', detail=str(e))
        except Exception as e:
            logger.error('Failed to configure webhook: %s', e)
            detail = {
                'repo': repo,
                'error': 'Failed to configure webhook:' + str(e)
            }
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_CREATION_FAILURE,
                                                                          detail=detail)
            raise AXApiInternalError('Failed to configure webhook', str(e))
        else:
            logger.info('Successfully created webhook (repo: %s)', repo)
            return {}

    def _delete_webhook(self, vendor, repo):
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
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_DELETION_FAILURE,
                                                                          detail=detail)
            raise AXApiInvalidParam('User authentication failed', detail=str(e))
        except AXApiForbiddenReq as e:
            logger.error('Supplied credential is valid but having insufficient permission')
            detail = {
                'repo': repo,
                'error': 'Supplied credential is valid but having insufficient permission:' + str(e)
            }
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_DELETION_FAILURE,
                                                                          detail=detail)
            raise AXApiInvalidParam('User has insufficient permission', detail=str(e))
        except Exception as e:
            logger.error('Failed to delete webhook: %s', e)
            detail = {
                'repo': repo,
                'error': 'Failed to delete webhook:' + str(e)
            }
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_WEBHOOK_DELETION_FAILURE,
                                                                          detail=detail)
            raise AXApiInternalError('Failed to delete webhook', str(e))
        else:
            logger.info('Successfully deleted webhook (repo: %s)', repo)

        # Delete ELB
        try:
            if not self._has_webhook(repo):
                logger.info('Deleting ELB for webhook ...')
                axsys_client.delete_webhook()
        except Exception as e:
            logger.error('Failed to delete ELB for webhook: %s', str(e))
            detail = {'repo': repo,
                      'error': 'Failed to delete ELB for webhook' + str(e)
                      }
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_ELB_DELETION_FAILURE,
                                                                          detail=detail)
            raise AXApiInternalError('Failed to delete ELB for webhook', str(e))
        else:
            logger.info('Successfully deleted ELB for webhook')
            return {}

    @list_route(methods=['GET', 'POST', 'DELETE'])
    def webhooks(self, request):
        """Create / delete a webhook.

        :param request:Translating
        :return:
        """
        repo = request.data.get('repo')
        vendor = request.data.get('type')
        username = request.data.get('username')
        password = request.data.get('password')
        if not all([repo, vendor]):
            raise AXApiInvalidParam('Missing required parameters', detail='Required parameters (repo, type)')
        if vendor not in self.scm_clients.keys():
            raise AXApiInvalidParam('Invalid parameter values', detail='Unsupported type ({})'.format(vendor))

        if username and password:
            self.scm_clients[vendor].update_repo_info(repo, vendor, username, password)
        if request.method == 'GET':
            result = self._get_webhook(vendor, repo)
        elif request.method == 'POST':
            result = self._create_webhook(vendor, repo)
        else:
            result = self._delete_webhook(vendor, repo)
        return Response(result)

    @list_route(methods=['POST'])
    def yamls(self, request):
        """Update YAML contents (i.e. policy, template).

        :param request:
        :return:
        """
        vendor = request.data.get('type')
        repo = request.data.get('repo')
        branch = request.data.get('branch')
        if not all([vendor, repo, branch]):
            raise AXApiInvalidParam('Missing required parameters', detail='Required parameters (type, repo, branch)')
        if vendor not in self.scm_clients.keys():
            raise AXApiInvalidParam('Invalid parameter values', detail='Unsupported type ({})'.format(vendor))

        try:
            # The arrival of events may not always be in the natural order of commits. For
            # example, the user may resent an old event from UI of source control tool. In
            # this case, we may update the YAML contents to an older version. To avoid this,
            # we guarantee that every YAML update will only update the content to the latest
            # version on a branch. More specifically, whenever we receive an event, we extract
            # the repo and branch information, and find the HEAD of the branch. Then, we use
            # the commit of HEAD to retrieve the YAML content, and update policies/templates
            # correspondingly.
            scm_client = self.scm_clients[vendor]
            commit = scm_client.get_branch_head(repo, branch)
            yaml_files = scm_client.get_yamls(repo, commit)
            logger.info('Updating YAML contents (policy/template) ...')
            axops_client.update_yamls(repo, branch, commit, yaml_files)
        except Exception as e:
            if 'Branch not found' in e.message:
                logger.info('No need to update YAML contents')
                return Response()
            else:
                logger.error('Failed to update YAML contents: %s', e)
                event_notification_client.send_message_to_notification_center(
                    CODE_JOB_CI_YAML_UPDATE_FAILURE, detail={'vendor': vendor,
                                                             'repo': repo,
                                                             'branch': branch,
                                                             'error': str(e)})
                raise AXApiInternalError('Failed to update YAML contents', str(e))
        else:
            logger.info('Successfully updated YAML contents')
            return Response()


def purge_branches(repo, branch=None):
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
            axdb_client.purge_branch_heads(repo)
        else:
            axdb_client.purge_branch_head(repo, branch)
    except Exception as e:
        message = 'Unable to purge branch heads'
        detail = 'Unable to purge branch heads (repo: {}, branch: {}): {}'.format(repo, branch, str(e))
        logger.error(detail)
        raise AXApiInternalError(message, detail)
    else:
        logger.info('Successfully purged branch heads')


def get_branches(repo=None, branch=None, order_by=None, limit=None):
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
            key = '{}:{}'.format(NAMESPACE, workspace)
            if redis_client.exists(key):
                logger.info('Loading cache (workspace: %s) ...', workspace)
                results = redis_client.get(key, decoder=json.loads)
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
                redis_client.set(key, results, expire=BRANCH_CACHE_TTL, encoder=json.dumps)
                return results
        except Exception as e:
            logger.warning('Failed to scan workspace (%s): %s', workspace, e)
            return []

    logger.info('Retrieving branches (repo: %s, branch: %s) ...', repo, branch)
    if repo:
        repo = unquote(repo)
        _, vendor, repo_owner, repo_name = parse_repo(repo)
        workspaces = ['{}/{}/{}/{}'.format(BASE_DIR, vendor, repo_owner, repo_name)]
    else:
        dirs = [dir[0] for dir in os.walk(BASE_DIR) if dir[0].endswith('/.git')]
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


@api_view(['GET', 'DELETE'])
def branches(request):
    """Query branches.

    :param request:
    :return:
    """
    repo = request.query_params.get('repo')
    branch = request.query_params.get('branch') or request.query_params.get('name')
    if request.method == 'DELETE':
        purge_branches(repo, branch)
        return Response({})
    else:
        if branch and branch.startswith('~'):
            branch = branch[1:]
        order_by = request.query_params.get('order_by')
        limit = request.query_params.get('limit')
        if limit:
            limit = int(limit)
        branches = get_branches(repo, branch, order_by, limit)
        return Response({'data': branches})


def _get_commits(workspace, branch=None, since=None, until=None, commit=None, author=None, committer=None,
                 description=None, limit=None):
    """Search for commits in a workspace.

    :param workspace:
    :param branch:
    :param since:
    :param until:
    :param commit:
    :param author:
    :param committer:
    :param description:
    :param limit:
    :return: a list of generators.
    """
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


def _get_commit(workspace, commit):
    """Get a commit from a workspace.

    :param workspace:
    :param commit:
    :return:
    """
    try:
        logger.info('Scanning workspace (%s) for commit (%s) ...', workspace, commit)
        git_client = GitClient(path=workspace, read_only=True)
        return git_client.get_commit(commit)
    except Exception as e:
        logger.warning('Failed to scan workspace (%s): %s', workspace, e)


def _parse_repo_branch(repo, branch, repo_branch):
    """Parse repo / branch / repo_branch.

    :param repo:
    :param branch:
    :param repo_branch:
    :return:
    """
    if repo:
        try:
            repo = unquote(repo)
            _, vendor, repo_owner, repo_name = parse_repo(repo)
        except Exception as e:
            msg = 'Unable to parse repo: %s', e
            logger.error(msg)
            raise AXApiInvalidParam('Unable to parse repo', msg)
        else:
            dir = '{}/{}/{}/{}'.format(BASE_DIR, vendor, repo_owner, repo_name)
            workspaces = {dir: [branch] if branch else []}
    elif repo_branch:
        try:
            repo_branch = json.loads(repo_branch)
            workspaces = {}
            for repo in repo_branch.keys():
                repo = unquote(repo)
                _, vendor, repo_owner, repo_name = parse_repo(repo)
                dir = '{}/{}/{}/{}'.format(BASE_DIR, vendor, repo_owner, repo_name)
                if dir not in workspaces:
                    workspaces[dir] = set()
                for branch in repo_branch[repo]:
                    workspaces[dir].add(branch)
        except Exception as e:
            msg = 'Unable to parse repo_branch: %s' % str(e)
            logger.error(msg)
            raise AXApiInvalidParam('Unable to parse repo_branch', msg)
    else:
        dirs = [dir[0] for dir in os.walk(BASE_DIR) if dir[0].endswith('/.git')]
        workspaces = list(map(lambda v: v[:-5], dirs))
        workspaces = dict([(k, [branch] if branch else []) for k in workspaces])
    return workspaces


@api_view(['GET'])
def commits(request):
    """Query commits.

    :param request:
    :return:
    """
    # Repo and branch are optional parameters that can always be used to reduce
    # search scope. Repo is used to construct the path to the workspace so that
    # the number of commands we issue can be significantly reduced. Branch can
    # be used in every command to filter commits by reference (branch).
    repo = request.query_params.get('repo')
    branch = request.query_params.get('branch')
    repo_branch = request.query_params.get('repo_branch')
    if repo_branch and (repo or branch):
        raise AXApiInvalidParam('Ambiguous query condition', 'It is ambiguous to us to supply both repo_branch and repo/branch')
    workspaces = _parse_repo_branch(repo, branch, repo_branch)

    # If commit / revision is supplied, we will disrespect all other parameters.
    # Also, we no longer use `git log` to issue query but use `git show` to directly
    # show the commit information.
    commit = request.query_params.get('commit') or request.query_params.get('revision')

    # Full-text search can be performed against 3 fields: author, committer, and description.
    # To perform narrow search, specify `author=~<author>&committer=~<committer>&description=~<description>`.
    # To perform broad search, specify `search=~<search>`.
    # Note that, in git, all queries are full-text search already, so we will strip off `~`.
    search = request.query_params.get('search')
    author = request.query_params.get('author', None)
    committer = request.query_params.get('committer', None)
    description = request.query_params.get('description', None)
    if search:
        use_broad_search = True
    else:
        use_broad_search = False
    if author:
        author = author.split(',')
    else:
        author = [None]
    if committer:
        committer = committer.split(',')
    else:
        committer = [None]
    author_committer = []
    for i in range(len(author)):
        for j in range(len(committer)):
            author_committer.append([author[i], committer[j]])

    # We use time-based pagination. min_time is converted to since and max_time is
    # converted to until. Also, the time format seconds since epoch (UTC).
    since = request.query_params.get('min_time')
    until = request.query_params.get('max_time')
    if since:
        since = datetime.datetime.utcfromtimestamp(int(since)).strftime('%Y-%m-%dT%H:%M:%S')
    if until:
        until = datetime.datetime.utcfromtimestamp(int(until)).strftime('%Y-%m-%dT%H:%M:%S')

    # Limit specify the maximal records that we return. Fields specify the fields
    # that we return. Sort allows the sorting of final results.
    limit = request.query_params.get('limit')
    fields = request.query_params.get('fields')
    sorter = request.query_params.get('sort')
    if limit:
        limit = int(limit)
    if fields:
        fields = set(fields.split(','))
    if sorter:
        sorters = sorter.split(',')
        valid_keys = {'repo', 'revision', 'author', 'author_date', 'committer', 'commit_date', 'date', 'description'}
        valid_sorters = []
        for i in range(len(sorters)):
            key = sorters[i][1:] if sorters[i].startswith('-') else sorters[i]
            if key in valid_keys:
                valid_sorters.append(sorters[i])
        sorter = valid_sorters

    logger.info('Retrieving commits (repo: %s, branch: %s, commit: %s, limit: %s) ...', repo, branch, commit, limit)

    # Prepare arguments for workspace scanning
    search_conditions = []
    for key in workspaces.keys():
        if not os.path.isdir(key):  # If the workspace does not exist, we should skip scanning it
            continue
        elif commit:
            search_conditions.append({'workspace': key, 'commit': commit})
        elif use_broad_search:
            for j in range(len(author_committer)):
                _author, _committer = author_committer[j][0], author_committer[j][1]
                _search_dict = {'workspace':   key,
                                'branch':      list(workspaces[key]),
                                'since':       since,
                                'until':       until,
                                'limit':       limit,
                                'author':      _author,
                                'committer':   _committer,
                                'description': description,
                                }
                for field in {'author', 'committer', 'description'}:
                    new_dict = copy.deepcopy(_search_dict)
                    new_dict[field] = search
                    search_conditions.append(new_dict)
        else:
            for j in range(len(author_committer)):
                _author, _committer = author_committer[j][0], author_committer[j][1]
                search_conditions.append({'workspace': key, 'branch': list(workspaces[key]),
                                          'author': _author, 'committer': _committer, 'description': description,
                                          'since': since, 'until': until, 'limit': limit})

    # Scan workspaces
    commits_list = []
    with ThreadPoolExecutor(max_workers=20) as executor:
        futures = []
        for i in range(len(search_conditions)):
            if commit:
                futures.append(executor.submit(_get_commit, **search_conditions[i]))
            else:
                futures.append(executor.submit(_get_commits, **search_conditions[i]))
        for future in as_completed(futures):
            try:
                data = future.result()
                if data:
                    commits_list.append(data)
            except Exception as e:
                logger.warning('Unexpected exception occurred during processing: %s', e)

    if commit:
        # If commit is supplied in the query, the return list is a list of commits, so we do not need to run top_k algorithm
        top_commits = sorted(commits_list, key=lambda v: -v['date'])
    else:
        # Retrieve top k commits
        top_commits = top_k(commits_list, limit, key=lambda v: -v['date'])

    # Sort commits
    if sorter:
        top_commits = sort_str_dictionaries(top_commits, sorter)
    else:
        top_commits = sorted(top_commits, key=lambda v: -v['date'])

    # Filter fields
    for i in range(len(top_commits)):
        for k in list(top_commits[i].keys()):
            if fields is not None and k not in fields:
                del top_commits[i][k]
    logger.info('Successfully retrieved commits')

    return Response({'data': top_commits})


@api_view(['GET'])
def commit(request, pk=None):
    """Get a single commit.

    :param request:
    :param pk:
    :return:
    """

    def get_commits(commit, repo=None):
        """Get commit(s) by commit hash.

        Normally, this function should return only 1 commit object. However, if a repo and its forked repo
        both appear in our workspace, there could be multiple commit objects.

        :param commit:
        :param repo:
        :return:
        """
        # If repo is not supplied, we need to scan all workspaces
        if repo:
            _, vendor, repo_owner, repo_name = parse_repo(repo)
            workspaces = ['{}/{}/{}/{}'.format(BASE_DIR, vendor, repo_owner, repo_name)]
        else:
            dirs = [dir[0] for dir in os.walk(BASE_DIR) if dir[0].endswith('/.git')]
            workspaces = list(map(lambda v: v[:-5], dirs))

        commits = []
        with ThreadPoolExecutor(max_workers=20) as executor:
            futures = []
            for i in range(len(workspaces)):
                futures.append(executor.submit(_get_commit, workspaces[i], commit=commit))
            for future in as_completed(futures):
                try:
                    data = future.result()
                    if data:
                        commits.append(data)
                except Exception as e:
                    logger.warning('Unexpected exception occurred during processing: %s', e)

        return commits

    repo = request.query_params.get('repo')
    if repo:
        repo = unquote(repo)
    logger.info('Retrieving commit (repo: %s, commit: %s) ...', repo, pk)
    commits = get_commits(pk, repo)
    if not commits:
        logger.warning('Failed to retrieve commit')
        raise AXApiInvalidParam('Invalid revision', detail='Invalid revision ({})'.format(pk))
    else:
        if len(commits) > 1:
            logger.warning('Found multiple commits with given sha, returning the first one ...')
        logger.info('Successfully retrieved commit')
        return Response(commits[0])


@api_view(['PUT', 'DELETE'])
def files(request):
    """Get a single file content and upload to s3.

    :param request:
    :return:
    """
    repo = request.query_params.get('repo')
    branch = request.query_params.get('branch')
    path = request.query_params.get('path')
    if not all([repo, branch, path]):
        raise AXApiInvalidParam('Missing required parameters', 'Missing required parameters (repo, branch, path)')
    if path.startswith('/'):
        path = path[1:]

    if request.method == 'PUT':
        resp = _put_file(repo, branch, path)
    else:
        resp = _delete_file(repo, branch, path)
    return Response(resp)


def _put_file(repo, branch, path):
    """Put a file in s3.

    :param repo:
    :param branch:
    :param path:
    :return:
    """
    _, vendor, repo_owner, repo_name = parse_repo(repo)
    workspace = '{}/{}/{}/{}'.format(BASE_DIR, vendor, repo_owner, repo_name)
    if not os.path.isdir(workspace):
        raise AXApiInvalidParam('Invalid repository', 'Invalid repository ({})'.format(repo))
    try:
        logger.info('Extracting file content from repository (repo: %s, branch: %s, path: %s) ...', repo, branch, path)
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
        cluster_name, cluster_id = CLUSTER_NAME_ID[:-37], CLUSTER_NAME_ID[-36:]
        key = '{cluster_name}/{cluster_id}/{vendor}/{repo_owner}/{repo_name}/{branch}/{path}'.format(
            cluster_name=cluster_name, cluster_id=cluster_id, vendor=vendor,
            repo_owner=repo_owner, repo_name=repo_name, branch=branch, path=path)
        logger.info('Uploading file content to s3 (bucket: %s, key: %s) ...', S3_BUCKET_NAME, key)
        response = s3_bucket.Object(key).put(Body=file_content)
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
        return {'bucket': S3_BUCKET_NAME, 'key': key, 'etag': etag}


def _delete_file(repo, branch, path):
    """Delete a file from s3.

    :param repo:
    :param branch:
    :param path:
    :return:
    """
    _, vendor, repo_owner, repo_name = parse_repo(repo)
    try:
        cluster_name, cluster_id = CLUSTER_NAME_ID[:-37], CLUSTER_NAME_ID[-36:]
        key = '{cluster_name}/{cluster_id}/{vendor}/{repo_owner}/{repo_name}/{branch}/{path}'.format(
            cluster_name=cluster_name, cluster_id=cluster_id, vendor=vendor,
            repo_owner=repo_owner, repo_name=repo_name, branch=branch, path=path)
        logger.info('Deleting file from s3 (bucket: %s, key: %s) ...', S3_BUCKET_NAME, key)
        s3_bucket.Object(key).delete()
    except Exception as e:
        message = 'Failed to delete file'
        detail = '{}: {}'.format(message, str(e))
        logger.error(detail)
        raise AXApiInternalError(message, detail)
    else:
        logger.info('Successfully deleted file')
        return {'bucket': S3_BUCKET_NAME, 'key': key}
