#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import copy
import json
import logging
import os
import requests

from jira import JIRA
from jira.exceptions import JIRAError

logger = logging.getLogger(__name__)

UNKNOWN_CLUSTER = 'unknown_cluster'
AX_JIRA_WEBHOOK_PAYLOAD = {'name': 'Argo JIRA integration:',
                           'url': None,
                           'events': [
                               'jira:issue_deleted',
                               'jira:issue_updated'
                           ],
                           'jqlFilter': '',
                           'excludeIssueDetails': True
                           }


class JiraClient(object):
    """
    Helper class for the JIRA
    """

    def __init__(self, url, username, password):
        """
        :param url:
        :param username:
        :param password:
        :return:
        """
        self.url = url
        self.webhook_url = self.url.strip('/') + '/rest/webhooks/1.0/webhook'
        self.basic_auth = (username, password)
        self.client = JIRA(url, basic_auth=self.basic_auth)

    def __str__(self):
        return '{} {}'.format(self.__class__.__name__, self.url)

################
#   Accounts   #
################

    def groups(self):
        """
        :return:
        """
        return self.client.groups()

    def users(self, json_result=True):
        """
        :param json_result:
        :return:
        """
        return self._search_users('_', json_result=json_result)

    def users_by_email(self, email, json_result=True):
        """
        :param email:
        :param json_result:
        :return:
        """
        return self._search_users(email, json_result=json_result)

    def users_by_group(self, name):
        """
        :param name:
        :return:
        """
        try:
            return self.client.group_members(name)
        except JIRAError as exc:
            logger.warning(exc)
            return dict()

    def _search_users(self, qstr, json_result=True):
        """
        :param qstr:
        :param json_result:
        :return:
        """
        def _user_dict_format(user):  # Note: Keep the consistent return format with the users_by_group method
            return {'key':      user.key,
                    'active':   user.active,
                    'fullname': user.displayName,
                    'email':    user.emailAddress
                    }

        users = self.client.search_users(qstr)

        if json_result:  # Note: Keep the consistent return format with the users_by_group method
            return [_user_dict_format(user) for user in users]
        else:
            return users

#############
#  Project  #
#############

    def create_project(self, key):
        """
        :param key:
        :return:
        """
        return self.client.create_project(key)

    def get_project(self, key, json_result=True):
        """
        :param key:
        :param json_result:
        :return:
        """
        if json_result:
            return self.client.project(key).raw
        else:
            return self.client.project(key)

    def get_projects(self, json_result=True):
        """
        :param json_result:
        :return:
        """
        project_objs = self.client.projects()
        if json_result:
            return [_each.raw for _each in project_objs]
        else:
            return project_objs

    def delete_project(self, key):
        """
        :param key:
        :return:
        """
        return self.client.delete_project(key)

#############
#  Version  #
#############

    def get_project_versions(self, name, json_result=True):
        """
        :param name: project name
        :param json_result:
        :return:
        """
        try:
            version_objs = self.client.project_versions(name)
            if json_result:
                return [_each.name for _each in version_objs]
            else:
                return version_objs
        except Exception as exc:
            logger.warn(exc)
            return []

    def create_project_version(self, name, project_name, **kwargs):
        """
        :param name: version name
        :param project_name: project name
        :param kwargs:
        :return:
        """
        return self.client.create_version(name, project_name, **kwargs)

#############
#   fields  #
########### #

    def get_fields(self):
        """
        :return:
        """
        return self.client.fields()

    def get_non_custom_fields(self):
        """
        :return:
        """
        return [each for each in self.client.fields() if not each.get('custom', True)]

    def get_custom_fields(self):
        """
        :return:
        """
        return [each for each in self.client.fields() if each.get('custom', True)]

    def get_field_id_by_name(self, name):
        """
        :param name:
        :return:
        """
        ids = [each['id'] for each in self.client.fields() if each.get('name', '') == name]
        if ids:
            return ids[0]
        else:
            return None

    def get_field_id_for_hours_left(self):
        """
        :return:
        """
        # For Argo customized field
        name = 'Hrs Left'
        return self.get_field_id_by_name(name)

############
#  issues  #
############

    def get_issue(self, name, json_result=True):
        """
        :param name:
        :param json_result:
        :return:
        """
        try:
            issue_obj = self.client.issue(name)
        except JIRAError as exc:
            logger.warn('Not found: %s', exc)
            return None

        if json_result:
            issue_dict = copy.deepcopy(issue_obj.raw['fields'])
            issue_dict['url'] = issue_obj.self
            issue_dict['id'] = issue_obj.id
            issue_dict['key'] = issue_obj.key
            # Replace custom field name
            return issue_dict
        else:
            return issue_obj

    def add_fix_version_to_issue(self, issue_name, version_name, issuetype=None):
        """
        :param issue_name:
        :param version_name:
        :param issuetype:
        :return:
        """
        return self._add_versions_from_issue(issue_name, version_name, issuetype=issuetype, _version_type='fixVersions')

    def add_affected_version_to_issue(self, issue_name, version_name, issuetype=None):
        """
        :param issue_name:
        :param version_name:
        :param issuetype:
        :return:
        """
        return self._add_versions_from_issue(issue_name, version_name, issuetype=issuetype, _version_type='versions')

    def _add_versions_from_issue(self, issue_name, version_name, issuetype=None, _version_type='fixVersions'):
        """
        :param issue_name:
        :param version_name:
        :param issuetype:
        :param _version_type:
        :return:
        """
        assert _version_type in ['fixVersions', 'versions'], 'Unsupported version type'

        issue_obj = self.get_issue(issue_name, json_result=False)

        if not issue_obj:
            return None

        if issuetype is not None and issue_obj.fields.issuetype.name.lower() != issuetype.lower():
            logger.info('SKIP. The issue type is %s, expected issue type is %s',
                        issue_obj.fields.issuetype.name.lower(),
                        issuetype.lower())
            return None

        logger.info('Update issue %s, with %s %s', issue_obj.key, _version_type, version_name)
        ret = issue_obj.add_field_value(_version_type, {'name': version_name})
        issue_obj.update()
        return ret

    def remove_affected_versions_from_issue(self, issue_name, versions_to_remove):
        """
        :param issue_name:
        :param versions_to_remove:
        :return:
        """
        return self._remove_versions_from_issue(issue_name, versions_to_remove, _version_type='versions')

    def remove_fix_versions_from_issue(self, issue_name, versions_to_remove):
        """
        :param issue_name:
        :param versions_to_remove:
        :return:
        """
        return self._remove_versions_from_issue(issue_name, versions_to_remove, _version_type='fixVersions')

    def _remove_versions_from_issue(self, issue_name, versions_to_remove, _version_type='fixVersions'):
        """
        :param issue_name:
        :param versions_to_remove:
        :param _version_type:
        :return:
        """
        assert _version_type in ['fixVersions', 'versions'], 'Unsupported version type'
        if type(versions_to_remove) not in [list, tuple, set]:
            versions_to_remove = [versions_to_remove]
        versions = []
        issue_obj = self.get_issue(issue_name, json_result=False)
        for ver in getattr(issue_obj.fields, _version_type):
            if ver.name not in versions_to_remove:
                versions.append({'name': ver.name})
        issue_obj.update(fields={_version_type: versions})

    def create_issue(self, project, summary, description=None, issuetype='Bug', reporter=None, **kwargs):
        """
        :param project:
        :param summary:
        :param description:
        :param issuetype:
        :param reporter:
        :param kwargs:
        :return:
        """
        # {
        #     "fields": {
        #        "project":
        #        {
        #           "key": "TEST"
        #        },
        #        "summary": "Always do right. This will gratify some people and astonish the REST.",
        #        "description": "Creating an issue while setting custom field values",
        #        "issuetype": {
        #           "name": "Bug"
        #        },
        #        "customfield_11050" : {"Value that we're putting into a Free Text Field."}
        #    }
        # }

        fields_dict = dict()

        for k, v in kwargs.items():
            fields_dict[k] = v

        fields_dict['project'] = {'key': project}
        fields_dict['description'] = description or ''
        fields_dict['summary'] = summary
        fields_dict['issuetype'] = {'name': issuetype}

        if reporter:
            users = self.users_by_email(reporter, json_result=False)
            if users:
                fields_dict['reporter'] = {'name': users[0].name}

        return self.client.create_issue(fields=fields_dict)

    def update_issue(self, name, **kwargs):
        """
        :param name:
        :param kwargs:
        :return:
        """
        issue_obj = self.get_issue(name, json_result=False)
        issue_obj.update(**kwargs)

######################
#   issue comments   #
######################

    def get_issue_comments(self, issue_name, latest_num=5, json_result=True):
        """
        :param issue_name:
        :param latest_num:
        :param json_result:
        :return:
        """
        try:
            comments = self.client.comments(issue_name)
        except JIRAError as exc:
            logger.warn(exc)
            return []
        comments = comments[::-1][:latest_num]
        if json_result:
            return [each.raw for each in comments]
        else:
            return comments

    def add_issue_comment(self, issue_name, msg, commenter=None):
        """
        :param issue_name:
        :param msg:
        :param commenter:
        :return:
        """
        if commenter:
            users = self.users_by_email(commenter, json_result=False)
            if users:
                msg_header = 'The comment is created by {}({}) from AX system. \n\n'.\
                    format(users[0].displayName, users[0].emailAddress)
                msg = msg_header + msg
        return self.client.add_comment(issue_name, msg)

###############
#  issue type #
###############

    def get_issue_types(self, json_result=True):
        """
        :param json_result:
        :return:
        """
        objs = self.client.issue_types()
        if json_result:
            return [obj.raw for obj in objs]
        else:
            return objs

    def get_issue_type_by_name(self, name, json_result=True):
        """
        :param name:
        :param json_result:
        :return:
        """
        try:
            obj = self.client.issue_type_by_name(name)
        except KeyError as exc:
            logger.warn(exc)
            return None
        else:
            if json_result:
                return obj.raw
            else:
                return obj

#############
#   Query   #
#############

    def query_issues(self, **kwargs):
        """
        :param kwargs:
        :return:

        max_results: maximum number of issues to return. Total number of results
                     If max_results evaluates as False, it will try to get all issues in batches.
        json_result: JSON response will be returned when this parameter is set to True.
                     Otherwise, ResultList will be returned
        """
        SUPPORTED_KEYS = ('project', 'status', 'component', 'labels', 'issuetype', 'priority',
                          'creator', 'assignee', 'reporter', 'fixversion', 'affectedversion')
        max_results = kwargs.pop('max_results', 100)
        _json_result = kwargs.pop('json_result', False)
        jql_str_list = []
        for k, v in kwargs.items():
            if k not in SUPPORTED_KEYS:
                continue
            jql_str_list.append('{} = "{}"'.format(k.strip(), v.strip()))
        if jql_str_list:
            jql_str = ' AND '.join(jql_str_list)
        else:
            jql_str = ''  # Fetch ALL issues
        try:
            ret = self.client.search_issues(jql_str, maxResults=max_results, json_result=_json_result)
        except Exception as exc:
            logger.warn(exc)
            ret = {"issues": []}
        return ret

################
# Query Issues #
################

    def get_issues_by_project(self, project_name, **kwargs):
        """
        :param project_name:
        :param kwargs:
        :return:
        """
        return self.query_issues(project=project_name, **kwargs)

    def get_issues_by_component(self, component, **kwargs):
        """
        :param component:
        :param kwargs:
        :return:
        """
        return self.query_issues(component=component, **kwargs)

    def get_issues_by_assignee(self, assignee, **kwargs):
        """
        :param assignee:
        :param kwargs:
        :return:
        """
        return self.query_issues(assignee=assignee, **kwargs)

    def get_issues_by_status(self, status, **kwargs):
        """
        :param status:
        :param kwargs:
        :return:
        """
        return self.query_issues(status=status, **kwargs)

    def get_issues_by_label(self, labels, **kwargs):
        """
        :param labels:
        :param kwargs:
        :return:
        """
        return self.query_issues(labels=labels, **kwargs)

    def get_issues_by_fixversion(self, fix_version, **kwargs):
        """
        :param fix_version:
        :param kwargs:
        :return:
        """
        return self.query_issues(fixverion=fix_version, **kwargs)

    def get_issues_by_affectedversion(self, affected_version, **kwargs):
        """
        :param affected_version:
        :param kwargs:
        :return:
        """
        return self.query_issues(affectedversion=affected_version, **kwargs)

################
#  Query Hours #
################

    def get_total_hours(self, **kwargs):
        """
        :param kwargs:
        :return:
        """
        all_issues = self.query_issues(**kwargs)
        field_id = self.get_field_id_for_hours_left()
        hours = [getattr(iss_obj.fields, field_id) for iss_obj in all_issues]
        return sum([float(each) for each in hours if each])

    def get_total_hours_by_project(self, project_name):
        """
        :param project_name:
        :return:
        """
        return self.get_total_hours(project=project_name)

    def get_total_hours_by_component(self, component):
        """
        :param component:
        :return:
        """
        return self.get_total_hours(component=component)

    def get_total_hours_by_assignee(self, assignee):
        """
        :param assignee:
        :return:
        """
        return self.get_total_hours(assignee=assignee)

    def get_total_hours_by_label(self, label):
        """
        :param label:
        :return:
        """
        return self.get_total_hours(labels=label)

##############
#   webhook  #
##############

    def create_ax_webhook(self, url, projects=None):
        """Create AX Jira webhook
        :param url:
        :param projects:
        :return:
        """
        payload = copy.deepcopy(AX_JIRA_WEBHOOK_PAYLOAD)
        payload['name'] = payload['name'] + self._get_cluster_name()
        payload['url'] = url
        filter_dict = self._generate_project_filter(projects)
        logger.info('Webhook project filter is: %s', filter_dict)
        payload.update(filter_dict)
        return self._requests(self.webhook_url, 'post', data=payload)

    def get_ax_webhook(self):
        """Get AX Jira webhook
        :return:
        """
        response = self._requests(self.webhook_url, 'get')
        wh_name = AX_JIRA_WEBHOOK_PAYLOAD['name'] + self._get_cluster_name()
        ax_whs = [wh for wh in response.json() if wh['name'] == wh_name]
        if not ax_whs:
            logger.error('Could not get Jira webhook for this cluster: %s, ignore it', wh_name)
        else:
            return ax_whs[0]

    def update_ax_webhook(self, projects=None):
        """Update AX Jira webhook
        :param projects:
        :return:
        """
        ax_wh = self.get_ax_webhook()
        if ax_wh:
            filter_dict = self._generate_project_filter(projects)
            logger.info('Webhook project filter is: %s', filter_dict)
            logger.info('Update the webhook %s', ax_wh['self'])
            return self._requests(ax_wh['self'], 'put', data=filter_dict)
        else:
            logger.warn('Skip the webhook update')

    def delete_ax_webhook(self):
        """Delete AX Jira webhook
        :return: 
        """
        response = self._requests(self.webhook_url, 'get')
        wh_name = AX_JIRA_WEBHOOK_PAYLOAD['name'] + self._get_cluster_name()

        ax_whs = [wh for wh in response.json() if wh['name'] == wh_name]
        for wh in ax_whs:
            logger.info('Delete webhook %s', wh['self'])
            self._delete_webhook(url=wh['self'])

    def get_ax_webhooks(self):
         """Get all AX webhooks
         :return:
         """
         response = self._requests(self.webhook_url, 'get')
         webhooks = response.json()
         # filter out non-ax webhooks
         return [wh for wh in webhooks if wh['name'].startswith(AX_JIRA_WEBHOOK_PAYLOAD['name'])]

    def delete_ax_webhooks(self):
        """Delete all AX Jira webhooks
        :return:
        """
        ax_whs = self.get_ax_webhooks()
        for wh in ax_whs:
            logger.info('Delete webhook %s', wh['self'])
            self._delete_webhook(url=wh['self'])

    def _generate_project_filter(self, projects):
        """
        :param projects:
        :return:
        """
        if not projects:
            filter_str = ''
        else:
            project_filter_list = []
            project_objs = self.get_projects(json_result=False)
            for pkey in projects:
                ps = [p for p in project_objs if p.key == pkey]
                if not ps:
                    logger.error('Could not get project %s, ignore it', pkey)
                else:
                    project_filter_list.append('Project = {}'.format(ps[0].name))
            filter_str = ' AND '.join(project_filter_list)

        return {'filters':
                    {'issue-related-events-section': filter_str
                     }
                }

    def _delete_webhook(self, url=None, id=None):
        """Delete webhook
        :param url:
        :param id:
        :return:
        """
        if url is None:
            url = self.webhook_url + '/' + str(id)
        return self._requests(url, 'delete')

    def _get_cluster_name(self):
        """
        :return:
        """
        return os.environ.get('AX_CLUSTER', UNKNOWN_CLUSTER)

    def _requests(self, url, method, data=None, headers=None, auth=None, raise_exception=True, timeout=30):
        """
        :param url:
        :param method:
        :param data:
        :param headers:
        :param auth:
        :param raise_exception:
        :param timeout:
        :return:
        """
        headers = {'Content-Type': 'application/json'} if headers is None else headers
        auth = self.basic_auth if auth is None else auth
        try:
            response = requests.request(method, url, data=json.dumps(data), headers=headers, auth=auth, timeout=timeout)
        except requests.exceptions.RequestException as exc:
            logger.error('Unexpected exception occurred during request: %s', exc)
            raise
        logger.debug('Response status: %s (%s %s)', response.status_code, response.request.method, response.url)
        # Raise exception if status code indicates a failure
        if response.status_code >= 400:
            logger.error('Request failed (status: %s, reason: %s)', response.status_code, response.text)
        if raise_exception:
            response.raise_for_status()
        return response
