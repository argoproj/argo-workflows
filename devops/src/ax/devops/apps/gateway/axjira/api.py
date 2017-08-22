import copy
import json
import logging

import jira
import requests

from concurrent.futures import ThreadPoolExecutor, as_completed
from urllib.parse import urlparse

from rest_framework.decorators import api_view, detail_route, list_route
from rest_framework.response import Response
from rest_framework.viewsets import GenericViewSet

from axjira.serializers import AXJiraSerializer
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.jira.jira_client import JiraClient
from ax.devops.jira.jira_utils import translate_jira_issue_event
from ax.exceptions import AXApiInvalidParam, AXApiAuthFailed, AXApiForbiddenReq, AXApiInternalError
from gateway.settings import LOGGER_NAME

logger = logging.getLogger('{}.{}'.format(LOGGER_NAME, 'jira'))

DELETE_EVENT = 'jira:issue_deleted'
UPDATE_EVENT = 'jira:issue_updated'

STATUS_OK = {'status': 'OK'}

axsys_client = AxsysClient()
axops_client = AxopsClient()

def init_jira_client(url=None, username=None, password=None):
    """
    :param url:
    :param username:
    :param password:
    :return:
    """
    def get_jira_configuration():
        js = axops_client.get_tools(category='issue_management', type='jira')
        if js:
            return {'url':       js[0]['url'],
                    'username':  js[0]['username'],
                    'password':  js[0]['password']
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

def _query_match(data, query_dict):
    """
    :param data:
    :param query_dict:
    :return:
    """
    for k, v in query_dict.items():
        if data.get(k, None) != v:
            return False
    return True


class JiraUserViewSet(GenericViewSet):

    queryset = None
    serializer_class = AXJiraSerializer
    filtered_users_keys = ('key', 'active', 'fullname', 'email')
    jira_client = None

    def list(self, request):
        """
        :param request
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()
        query_dict = dict()
        for pk in self.filtered_users_keys:
            pv = request.query_params.get(pk, None)
            if pk == 'active':
                if pv == 'true':
                    pv = True
                elif pv == 'false':
                    pv = False
            if pv is not None:
                query_dict[pk] = pv

        users = self.jira_client.users()
        users = [u for u in users if _query_match(u, query_dict)]
        return Response({'data': users})


class JiraProjectViewSet(GenericViewSet):

    queryset = None
    serializer_class = AXJiraSerializer
    filtered_project_keys = ('id', 'key', 'name', 'projectTypeKey')
    jira_client = None

    def _normalize_data(self, proj_dict):
        """
        :param proj_dict:
        :return:
        """
        return dict([(k, proj_dict.get(k, None)) for k in self.filtered_project_keys])

    def list(self, request):
        """
        :param request
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()
        query_dict = dict()
        for pk in self.filtered_project_keys:
            pv = request.query_params.get(pk, None)
            if pv is not None:
                query_dict[pk] = pv

        ps = self.jira_client.get_projects(json_result=True)
        ps = [p for p in ps if _query_match(p, query_dict)]
        ps = [self._normalize_data(p) for p in ps]
        return Response({'data': ps})

    def retrieve(self, request, pk=None):
        """
        :param request:
        :param pk:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()
        proj = self.jira_client.get_project(pk, json_result=True)
        return Response(self._normalize_data(proj))

    @list_route(methods=['POST',])
    def test(self, request):
        """Test connection to Jira server.

        :param request:
        :return:
        """
        url = request.data.get('url', '').lower()
        username = request.data.get('username', None)
        password = request.data.get('password', None)
        logger.info('Received request (url: %s, username: %s, password: ******)', url, username)

        assert all([url, username, password]), \
            AXApiInvalidParam('Missing required parameters', detail='Required parameters (username, password, url)')

        try:
            init_jira_client(url, username, password)
        except requests.exceptions.ConnectionError as exc:
            raise AXApiInternalError('Invalid URL', detail=str(exc))
        except jira.exceptions.JIRAError as exc:
            raise AXApiInternalError('Invalid authentication', detail=str(exc))
        except Exception as exc:
            raise AXApiInternalError('Failed to connect to JIRA', detail=str(exc))
        else:
            return Response(STATUS_OK)


class JiraIssueViewSet(GenericViewSet):
    """View set for JIRA issue."""

    queryset = None
    serializer_class = AXJiraSerializer
    default_max_results = 3
    filtered_issue_keys = ('project', 'status', 'component', 'labels', 'issuetype', 'priority',
                           'creator', 'assignee', 'reporter', 'fixversion', 'affectedversion')
    jira_client = None

    def create(self, request):
        """
        :param request:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()

        logger.info('Received jira issue creation request (%s)', request.data)
        project = request.data.get('project', None)
        summary = request.data.get('summary', None)
        issuetype = request.data.get('issuetype', None)
        reporter = request.data.get('reporter', None)

        description = request.data.get('description', None)  # optional

        if project is None:
            raise AXApiInvalidParam('Missing required parameters: Project',
                                    detail='Missing required parameters, Project')
        if summary is None:
            raise AXApiInvalidParam('Missing required parameters: Summary',
                                    detail='Missing required parameters, Summary')
        if issuetype is None:
            raise AXApiInvalidParam('Missing required parameters: Issuetype',
                                    detail='Missing required parameters, Issuetype')
        if reporter is None:
            raise AXApiInvalidParam('Missing required parameters: Reporter',
                                    detail='Missing required parameters, Reporter')

        try:
            issue_obj = self.jira_client.create_issue(project,
                                                      summary,
                                                      issuetype=issuetype,
                                                      reporter=reporter,
                                                      description=description)
        except jira.exceptions.JIRAError as exc:
            raise AXApiInternalError('Invalid Parameters', detail=str(exc))
        else:
            issue_dict = copy.deepcopy(issue_obj.raw['fields'])
            issue_dict['url'] = issue_obj.self
            issue_dict['id'] = issue_obj.id
            issue_dict['key'] = issue_obj.key
            return Response(issue_dict)

    def list(self, request):
        """
        :param request:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()

        query_ids = request.query_params.get('ids', None)
        if query_ids is not None:
            issues = []
            with ThreadPoolExecutor(max_workers=5) as executor:
                futures = []
                id_list = query_ids.strip().split(',')
                logger.info('Query the following Jira issues: %s', id_list)
                for id in id_list:
                    futures.append(executor.submit(self.jira_client.get_issue, id.strip(), json_result=True))
                for future in as_completed(futures):
                    try:
                        issues.append(future.result())
                    except Exception as exc:
                        logger.warn('Unexpected exception %s', exc)
        else:
            kwargs = dict()
            for key in request.query_params.keys():
                if key.lower() in self.filtered_issue_keys:
                    kwargs[key.lower()] = request.query_params.get(key)
            logger.info('Query kwargs: %s:', kwargs)
            issues = self.jira_client.query_issues(json_result=True, **kwargs)

        return Response(issues)

    def retrieve(self, request, pk=None):
        """
        :param request:
        :param pk:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()
        issue = self.jira_client.get_issue(pk, json_result=True)
        return Response(issue)

    @detail_route(methods=['GET',])
    def getcomments(self, request, pk=None):
        """
        :param request:
        :param pk:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()
        max_results = int(request.query_params.get('max_results', self.default_max_results))
        comments = self.jira_client.get_issue_comments(pk,
                                                       latest_num=max_results,
                                                       json_result=True)
        return Response(comments)

    @detail_route(methods=['POST',])
    def addcomment(self, request, pk=None):
        """
        :param request:
        :param pk:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()
        comment = request.data.get('comment', None)
        user = request.data.get('user', None)

        if not comment:
            raise AXApiInvalidParam('Require Comment message info')
        if not user:
            raise AXApiInvalidParam('Require Commenter info')

        try:
            self.jira_client.add_issue_comment(pk, comment, commenter=user)
        except Exception as exc:
            raise AXApiInternalError('Failed to add comment', detail=str(exc))
        return Response(STATUS_OK)


class JiraIssueTypeViewSet(GenericViewSet):
    """View set for JIRA issue types."""

    queryset = None
    serializer_class = AXJiraSerializer
    jira_client = None

    def list(self, request):
        """
        :param request:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()
        issue_types = self.jira_client.get_issue_types(json_result=True)
        return Response({'data': issue_types})

    def retrieve(self, request, pk=None):
        """
        :param request:
        :param pk:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()
        issue_type = self.jira_client.get_issue_type_by_name(pk, json_result=True)
        return Response(issue_type)


class JiraWebhookViewSet(GenericViewSet):
    """View set for JIRA webhook"""

    queryset = None
    serializer_class = AXJiraSerializer
    jira_client = None

    def list(self, request):
        """
        :param request:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()

        ax_webhooks = self.jira_client.get_ax_webhooks()
        return Response({'data': ax_webhooks})

    def create(self, request):
        """
        :param request:
        :return:
        """
        logger.info('Received jira webhook creation request')
        url = request.data.get('url', None)
        username = request.data.get('username', None)
        password = request.data.get('password', None)
        webhook = request.data.get('webhook', None)
        projects = request.data.get('projects', None)

        # Create ingress
        try:
            dnsname = urlparse(webhook).netloc
            logger.info('Creating ingress for Jira webhook %s', dnsname)
            axsys_client.create_ingress(dnsname)
        except Exception as exc:
            logger.error('Failed to create ingress for webhook: %s', str(exc))
            raise AXApiInternalError('Failed to create ingress for webhook', str(exc))
        else:
            logger.info('Successfully created ingress for webhook')

        # Create webhook
        self.jira_client = init_jira_client(url=url, username=username, password=password)
        try:
            if projects:
                logger.info('Filtered projects are: %s', projects)
                if type(projects) == str:
                    projects = json.loads(projects)
            else:
                logger.info('No project filter')
                projects = None
            wh = self.jira_client.create_ax_webhook(webhook, projects=projects)
        except Exception as exc:
            logger.exception(exc)
            raise AXApiInternalError('Fail to create jira webhooks', detail=str(exc))
        return Response(wh.json())

    def put(self, request, pk=None):
        """
        :param request:
        :param pk:
        :return:
        """
        if self.jira_client is None:
            self.jira_client = init_jira_client()

        projects = request.data.get('projects', None)
        logger.info('Received jira webhook update request ...')
        # Update webhook
        try:
            if projects:
                logger.info('Filtered projects are: %s', projects)
                if type(projects) == str:
                    projects = json.loads(projects)
            else:
                logger.info('No project filter')
                projects = None
            self.jira_client.update_ax_webhook(projects)
        except Exception as exc:
            logger.exception(exc)
            raise AXApiInternalError('Fail to update jira webhooks', detail=str(exc))
        else:
            logger.info('Successfully updated Jira webhook')
        return Response(STATUS_OK)

    def delete(self, request):
        """
        :param request:
        :return:
        """
        if self.jira_client is None:
            try:
                self.jira_client = init_jira_client()
            except Exception as exc:
                logger.warn('Could not log into Jira, skip it')
                return Response(STATUS_OK)

        wh = self.jira_client.get_ax_webhook()
        if not wh:
            logger.warn('No webhook on Jira server, ignore it')
            return Response(STATUS_OK)

        # Delete ingress
        try:
            logger.info('Deleting ingress for Jira webhook %s', wh['url'])
            axsys_client.delete_ingress(urlparse(wh['url']).netloc)
        except Exception as exc:
            logger.error('Failed to delete ingress for webhook: %s', str(exc))
            raise AXApiInternalError('Failed to delete ingress for webhook', str(exc))
        else:
            logger.info('Successfully deleted ingress for webhook')
        # Delete webhook
        try:
            self.jira_client.delete_ax_webhook()
        except Exception as exc:
            logger.exception(exc)
            raise AXApiInternalError('Fail to delete jira webhooks', detail=str(exc))
        return Response(STATUS_OK)


@api_view(['POST'])
def events(request):
    """Create a JIRA webhook event.

    :param request:
    :return:
    """
    checked_fields =('description', 'project', 'status', 'summary', 'Key')

    payload = request.data
    try:
        logger.info('Translating JIRA event ...')
        event = translate_jira_issue_event(payload)
    except Exception as exc:
        logger.error('Failed to translate event: %s', exc)
        raise AXApiInternalError('Failed to translate event', detail=str(exc))
    else:
        logger.info('Successfully translated event: %s', event)

    try:
        if event['type'] == UPDATE_EVENT:
            logger.info('The following Jira field(s) get updated: %s', event['changed_fields'])
            if event['status_category_id'] == 3:
                logger.info('Jira issue %s is closed', event['id'])
                logger.info('Delete Jira on AXDB %s', event['id'])
                axops_client.delete_jira_issue(event['id'])
            elif event['changed_fields'] and any(f in event['changed_fields'] for f in checked_fields):
                logger.info('Update Jira content on AXDB ...')
                axops_client.update_jira_issue(event['axdb_content'])
            else:
                logger.info('No Jira content need to be updated')
        elif event['type'] == DELETE_EVENT:
            logger.info('Delete Jira on AXDB %s', event['id'])
            axops_client.delete_jira_issue(event['id'])
        else:
            logger.warn('Not supported event: (%s), ignore it', event['type'])
    except Exception as exc:
        raise AXApiInternalError('Failed to update JIRA content on AXDB', detail=str(exc))
    else:
        return Response(STATUS_OK)
