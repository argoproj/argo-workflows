#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import copy
import json
import logging
import sys
import time
import uuid

import requests
from requests.exceptions import HTTPError

from ax.devops.axrequests.axrequests import AxRequests
from ax.devops.settings import AxSettings

logger = logging.getLogger(__name__)


class AxopsClient(object):
    """AX ops client."""

    def __init__(self, host=None, port=None, protocol=None, version=None, url=None, username=None, password=None, timeout=60, ssl_verify=True):
        """Initialize the AX ops client.

        :param host:
        :param port:
        :param protocol:
        :param version:
        :param url:
        :param username:
        :param password:
        :param timeout:
        :param ssl_verify:
        :return:
        """
        self.host = host or AxSettings.AXOPS_HOSTNAME
        self.port = port or AxSettings.AXOPS_PORT
        self.protocol = protocol or AxSettings.AXOPS_PROTOCOL
        self.version = version or AxSettings.AXOPS_VERSION
        self.username = username or AxSettings.AXOPS_USERNAME
        self.password = password or ""
        self.session_id = None
        self.timeout = timeout
        self.ssl_verify = ssl_verify
        assert self.port == AxSettings.AXOPS_PORT or (self.username and self.password), 'Use port {} or specify username/password'.format(AxSettings.AXOPS_PORT)
        self.ax_request = AxRequests(self.host, port=self.port, version=self.version, protocol=self.protocol, url=url, timeout=self.timeout, ssl_verify=self.ssl_verify)

    def login(self):
        """Login into AX ops server.

        :return:
        """
        output = self.ax_request.post(path='/login', data=json.dumps({'username': self.username, 'password': self.password}), value_only=True)
        self.session_id = output['session']

    def ping(self):
        """Ping api call to verify axops is up and running"""
        try:
            self.ax_request.get(path='/ping')
            return True
        except Exception:
            return False

    def _request(self, method, path, params=None, data=None, **kwargs):
        """Make a AXOPS request.

        :param method:
        :param path:
        :param params:
        :param data:
        :param kwargs:
        :return:
        """
        if self.port != AxSettings.AXOPS_PORT:
            if not self.session_id:
                self.login()
            params = params or {}
            params['session'] = self.session_id
        f = getattr(self.ax_request, method.lower(), None)
        if f is None:
            raise HTTPError('Method (%s) not allowed', method)
        return f(path=path, params=params, data=data, **kwargs)

    def get_all_commit_policies(self):
        """Get all commit policies."""
        logger.info('Retrieving all commit policies ...')
        return self._request('get', path='/commit_policies', value_only=True)['data']

    def get_tools(self, **kwargs):
        """Get all tool configurations.

        :return:
        """
        logger.info('Retrieving all tool configurations ...')
        return self._request('get', path='/tools', params=kwargs, value_only=True)['data']

    def get_tool(self, repo):
        """Get tool configuration of repo.

        :param repo:
        :return:
        """
        logger.info('Retrieving tool configuration (repo: %s) ...', repo)
        all_tool_configs = self.get_tools()
        for config in all_tool_configs:
            repos = set(config.get('repos', []))
            if repo in repos:
                return {
                    'id': config['id'],
                    'category': config['category'],
                    'type': config['type'],
                    'username': config.get('username'),
                    'password': config.get('password'),
                    'repos': config['repos'],
                    'url': config['url']
                }

    def create_service(self, payload):
        """Create a service from a service template.

        :param payload:
        :return:
        """
        return self._request('post', path='/services', data=json.dumps(payload), value_only=True)

    def get_service(self, id):
        """Get service info by id.

        :param id:
        :return:
        """
        return self._request('get', path='/services/{}'.format(id), value_only=True)

    def get_services(self, **kwargs):
        """Get list of services by filters.

        :param tasks_only: only return jobs
        :param is_active: only return active jobs
        :return: list of services
        """
        return self._request('get', path='/services', params=kwargs, value_only=True)['data']

    def get_deployments(self, **kwargs):
        """Get list of deployments by filters.
        :return: list of deployments
        """
        return self._request('get', path='/deployments', params=kwargs, value_only=True)['data']

    def get_most_recent_service(self, repo, commit, policy_id):
        """Get the most recent service with given repo, commit, and policy_id.

        Currently, axops does not support query by repo, commit, and/or policy_id.
        For now, we need to retrieve all services, and filter in memory.

        :param repo:
        :param commit:
        :param policy_id:
        :return:
        """
        services = self._request('get', path='/services', value_only=True)['data']
        most_recent_service = None
        largest_creation_time = -sys.maxsize - 1
        for service in services:
            service_commit_info = service.get('commit') or {}
            service_repo = service_commit_info.get('repo')
            service_commit = service_commit_info.get('revision')
            service_policy_id = service.get('policy_id')
            service_create_time = service.get('create_time')
            if not all([service_repo, service_commit, service_policy_id, service_create_time]):
                continue
            if service_repo != repo or service_commit != commit or service_policy_id != policy_id:
                continue
            if service_create_time > largest_creation_time:
                largest_creation_time = service_create_time
                most_recent_service = service
        return most_recent_service

    def create_service_template(self, payload):
        """Create a service template.

        :param payload:
        :return:
        """
        return self._request('post', path='/templates', data=json.dumps(payload), value_only=True)

    def delete_service_template(self, id):
        """Delete a service template.

        :param id:
        :return:
        """
        return self._request('delete', path='/templates/{}'.format(id), value_only=True)

    def get_fixture_templates(self, params):
        return self._request('get', path='/fixture/templates', params=params, value_only=True)['data']

    def get_fixture_template(self, template_id):
        return self._request('get', path='/fixture/templates/{}'.format(template_id), value_only=True)

    def get_fixture_template_by_repo(self, repo, branch, name):
        params = {
            'repo': repo,
            'branch': branch,
            'name': name
        }
        templates = self.get_fixture_templates(params=params)
        if templates:
            return templates[0]
        else:
            return None

    def get_fixture_classes(self):
        return self._request('get', path='/fixture/classes', value_only=True)['data']

    def update_fixture_class(self, payload):
        class_id = payload['id']
        return self._request('put', path='/fixture/classes/{}'.format(class_id), data=json.dumps(payload), value_only=True)

    def get_fixture_instance(self, id):
        return self._request('get', path='/fixture/instances/{}'.format(id), value_only=True)

    def get_fixture_instances(self):
        return self._request('get', path='/fixture/instances', value_only=True)['data']

    def update_fixture_instance(self, payload):
        fixture_id = payload['id']
        return self._request('put', path='/fixture/instances/{}'.format(fixture_id), data=json.dumps(payload), value_only=True)

    def delete_fixture_instance(self, id):
        return self._request('delete', path='/fixture/instances/{}'.format(id), value_only=True)

    def get_templates(self, repo, branch, name=None):
        """Get service templates.

        :param repo:
        :param branch:
        :param name:
        :return:
        """
        condition = {
            'repo': repo,
            'branch': branch
        }
        if name is not None:
            condition['name'] = name
        params = '&'.join(['{}={}'.format(k, condition[k]) for k in condition])
        return self._request('get', path='/templates?{}'.format(params), value_only=True)['data']

    def get_template(self, repo, branch, name):
        """Get service templates.

        :param repo:
        :param branch:
        :param name:
        :return:
        """
        templates = self.get_templates(repo, branch, name=name)
        if len(templates) == 0:
            return None
        return templates[0]

    def get_policies(self, repo, branch, name=None, enabled=None):
        """Get policies."""
        condition = {
            'repo': repo,
            'branch': branch
        }
        if name:
            condition['name'] = name
        if enabled:
            condition['enabled'] = enabled

        params = '&'.join(['{}={}'.format(k, condition[k]) for k in condition])
        return self._request('get', path='/policies?{}'.format(params), value_only=True)['data']

    def get_policy(self, enabled=None):
        """Get all policies"""
        params = ""
        if enabled is not None:
            params = "?enabled=true" if enabled else "?enabled=false"

        return self._request('get', path='/policies{}'.format(params), value_only=True)['data']

    def get_commit_info(self, repo=None, branch=None, limit=None):
        """
        Get commit info.

        :param repo:
        :param branch:
        :param limit:
        :return:
        """
        condition = dict()
        if repo:
            condition['repo'] = repo
        if branch:
            condition['branch'] = branch
        if limit:
            condition['limit'] = limit

        params = '&'.join(['{}={}'.format(k, condition[k]) for k in condition])
        return self._request('get', path='/commits?{}'.format(params), value_only=True)['data']

    def update_yamls(self, repo, branch, commit, files):
        """Update YAML contents.

        :param repo:
        :param branch:
        :param commit:
        :param files: A list of dictionaries containing keys `path` and `content`.
        :return:
        """
        data = {
            'repo': repo,
            'branch': branch,
            'revision': commit,
            'files': files
        }
        return self._request('post', path='/yamls', data=json.dumps(data), value_only=True)

    def get_branches(self, repo=None, name=None, project=None, search=None):
        """Get all branches"""
        condition = dict()
        if repo:
            condition['repo'] = repo
        if name:
            condition['name'] = name
        if project:
            condition['project'] = project
        if search:
            condition['search'] = search
        params = '&'.join(['{}={}'.format(k, condition[k]) for k in condition])
        return self._request('get', path='/branches?{}'.format(params), value_only=True)['data']

    def get_branch(self, branch_id):
        """Get branch with ID"""
        return self._request('get', path='/branches/{}'.format(branch_id), value_only=True)['data']

    def get_dns(self):
        """Get AxOps DNS name."""
        return self._request('get', path='/system/settings/dnsname', value_only=True)['dnsname']

    def get_user(self, username):
        """Get user with username"""
        return self._request('get', path='/users/{}'.format(username), value_only=True)

    def decrypt(self, token, repo_name):
        """Decrypt secret"""
        data = {
            'cipher_text': {
                'ciphertext': token,
                'repo': repo_name,
            }
        }
        result = self._request('post', path='/secret/decrypt', data=json.dumps(data), value_only=True)
        return result['plain_text']['decrypted_content']

    def search_artifacts(self, params):
        """List Artifacts satisfying some query condition"""
        params = copy.deepcopy(params)
        params['action'] = 'search'
        return self._request('get', path='/artifacts', params=params, value_only=True)['data']

    def download_artifact(self, artifact_id, retries=5):
        """Return a Response object which can stream contents of a file"""
        params = {
            'action': 'download',
            'artifact_id': artifact_id,
        }
        for _ in range(retries):
            # HACK: fixture out why native requests work but no axrequests
            #resp = self._request('get', path='/artifacts', params=params, value_only=False, stream=True, raise_exception=False)
            url = '{}://{}:{}/v1/artifacts'.format(self.protocol, self.host, self.port)
            resp = requests.get(url, params=params, stream=True)
            if resp.status_code >= 400:
                logger.warning("Failed to download artifact. Response: %s", resp.status_code)
                time.sleep(10)
            else:
                return resp
        return resp.raise_for_status()

    def update_jira_issue(self, payload):
        """Update Jira content
        :param payload:
        :return:
        """
        return self._request('put', path='/jira/issues', data=json.dumps(payload), value_only=True)

    def delete_jira_issue(self, id):
        """Delete a Jira issue
        :param id:
        :return:
        """
        return self._request('delete', path='/jira/issues/{}'.format(id), value_only=True)
