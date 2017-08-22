import abc
import logging
import os
import requests

from ax import exceptions, notification_center
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.exceptions import UnknownRepository
from ax.devops.kafka.kafka_client import EventNotificationClient

logger = logging.getLogger(__name__)

TEMPLATE_DIR = '.argo'

class BaseScmRestClient(metaclass=abc.ABCMeta):
    """Base REST API wrapper for SCM."""

    event_notification_client = EventNotificationClient(notification_center.FACILITY_GATEWAY)

    WEBHOOK_TITLE = 'AX CI - {}'.format(os.environ.get('AX_CLUSTER'))
    WEBHOOK_JOB_URL = 'https://{}/app/jobs/job-details'.format(os.environ.get('AXOPS_EXT_DNS'))

    EXCEPTION_MAPPING = {
        400: exceptions.AXApiInvalidParam,
        401: exceptions.AXApiAuthFailed,
        403: exceptions.AXApiForbiddenReq,
        404: exceptions.AXApiResourceNotFound,
        405: exceptions.AXApiResourceNotFound
    }

    STATUS_MAPPING = {}

    def __init__(self):
        """Initializer."""
        self.axops_client = AxopsClient()
        self.axsys_client = AxsysClient()
        self.repos = {}
        self.urls = {}

    def construct_webhook_url(self):
        """Construct the URL of webhook"""
        try:
            payload = self.axsys_client.get_webhook()
            dnsname = payload['hostname']
            port = payload['port_spec'][0]['port']
        except Exception as e:
            message = 'Unable to extract dnsname or port of webhook'
            detail = 'Unable to extract dnsname or port of webhook: {}'.format(str(e))
            logger.error(detail)
            raise exceptions.AXApiInternalError(message, detail)
        else:
            return 'https://{}:{}/v1/webhooks/scm'.format(dnsname, port)

    @abc.abstractmethod
    def get_repos(self, username, password):
        """Get repos that an account can see.

        :param username:
        :param password:
        :return:
        """

    @abc.abstractmethod
    def get_commit(self, repo, commit):
        """Get commit info.

        :param repo:
        :param commit:
        :return:
        """

    @abc.abstractmethod
    def get_branch_head(self, repo, branch):
        """Get HEAD of a branch.

        :param repo:
        :param branch:
        :return:
        """

    def has_webhook(self, repo):
        """Test if webhook is configured on this repo.

        :param repo:
        :return:
        """
        return bool(self.get_webhook(repo))

    @abc.abstractmethod
    def get_webhook(self, repo):
        """Get webhook.

        :param repo:
        :return:
        """

    @abc.abstractmethod
    def get_webhooks(self, repo):
        """Get all webhooks.

        :param repo:
        :return:
        """

    @abc.abstractmethod
    def create_webhook(self, repo):
        """Create webhook.

        :param repo:
        :return:
        """

    @abc.abstractmethod
    def delete_webhook(self, repo):
        """Delete webhook.

        :param repo:
        :return:
        """

    @abc.abstractmethod
    def upload_job_result(self, payload):
        """Upload job result to bitbucket.

        :param payload:
        :return:
        """

    @abc.abstractmethod
    def get_yamls(self, repo, commit):
        """Get all YAML files in .argo folder.

        :param repo:
        :param commit:
        :return:
        """

    def get_repo_info(self, repo):
        """Get owner, name, and credential of repo.

        :param repo:
        :return:
        """
        if repo not in self.repos:
            tool_config = self.axops_client.get_tool(repo)
            if not tool_config:
                self.event_notification_client.send_message_to_notification_center(notification_center.CODE_JOB_CI_REPO_NOT_FOUND, detail={'repo': repo})
                raise UnknownRepository('Unable to find configuration for repo ({})'.format(repo))
            self.update_repo_info(repo, tool_config['type'], tool_config['username'], tool_config['password'])
        return (self.repos[repo]['owner'], self.repos[repo]['name'],
                self.repos[repo]['username'], self.repos[repo]['password'])

    def update_repo_info(self, repo, type, username, password):
        """Update repo info.

        :param repo:
        :param type:
        :param username:
        :param password:
        :return:
        """
        strs = repo.split('/')
        owner, name = strs[-2], strs[-1].split('.')[0]
        self.repos[repo] = {
            'type': type,
            'name': name,
            'owner': owner,
            'username': username,
            'password': password
        }

    def make_request(self, f, url, *args, **kwargs):
        """Make an HTTP request.

        :param f:
        :param url:
        :param args:
        :param kwargs:
        :returns:
        """
        logger.info('Request: %s %s', f.__name__, url)
        value_only = kwargs.pop('value_only', False)
        resp = f(url, *args, **kwargs)
        self.handle_exception(resp)
        if value_only:
            return resp.json()
        else:
            return resp

    def handle_exception(self, response):
        """Handle exception in response.

        :param response:
        :return:
        """
        try:
            response.raise_for_status()
        except requests.ConnectionError as e:
            logger.error('Connection error: %s %s', e.response.status_code, e.response.text)
            raise exceptions.AXTimeoutException('Connection timeout', str(e))
        except requests.HTTPError as e:
            logger.error('HTTP error: %s %s', e.response.status_code, e.response.text)
            exception_class = self.EXCEPTION_MAPPING.get(response.status_code) or exceptions.AXApiInternalError
            raise exception_class('HTTP error', str(e))
        else:
            logger.info('Response: %s', response.status_code)
