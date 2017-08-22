import datetime
import json
import logging
import re
import requests
from dateutil.tz import tzlocal

from ax.devops.redis.redis_client import RedisClient, DB_REPORTING
from ax.devops.scm_rest import BaseScmRestClient
from . import TEMPLATE_DIR

logger = logging.getLogger(__name__)

redis_client = RedisClient('redis', db=DB_REPORTING)


class GitHubClient(BaseScmRestClient):
    """REST API wrapper for GitHub."""

    STATUS_MAPPING = {
        -1: 'failure',
        0: 'success',
        1: 'pending'
    }

    def __init__(self):
        """Initializer."""
        super().__init__()
        self.root_url = 'https://api.github.com'
        self.repo_root_url = 'https://github.com/{owner}/{repo_name}.git'
        self.urls = {
            'branches': '%s/repos/{owner}/{repo_name}/branches/{branch}' % self.root_url,
            'commit': '%s/repos/{owner}/{repo_name}/commits/{commit}' % self.root_url,
            'content': '%s/repos/{owner}/{repo_name}/contents/{path}?ref={commit}' % self.root_url,
            'hook': '%s/repos/{owner}/{repo_name}/hooks/{id}' % self.root_url,
            'hooks': '%s/repos/{owner}/{repo_name}/hooks' % self.root_url,
            'pull_request': '%s/repos/{owner}/{repo_name}/pulls/{id}' % self.root_url,
            'repos': '%s/user/repos' % self.root_url,
            'status': '%s/repos/{owner}/{repo_name}/commits/{commit}/statuses' % self.root_url
        }

    def get_repos(self, username, password):
        """Get repos that an account can see.

        :param username:
        :param password:
        :return:
        """
        url = self.urls['repos']
        repos = {}
        while url is not None:
            resp = self.make_request(requests.get, url, auth=(username, password))
            url = resp.links.get('next', {}).get('url')
            paginated_repos = resp.json()
            for repo in paginated_repos:
                repo_url = repo['clone_url']
                repo_protocol = repo_url.split('://')[0]
                if repo_protocol == 'https':
                    repos[repo_url] = repo_url
        return repos

    def get_commit(self, repo, commit):
        """Get commit info.

        :param repo:
        :param commit:
        :return:
        """
        owner, repo_name, username, password = self.get_repo_info(repo)
        auth = (username, password) if username and password else None
        url = self.urls['commit'].format(owner=owner, repo_name=repo_name, commit=commit)
        return self.make_request(requests.get, url, auth=auth, value_only=True)

    def get_branch_head(self, repo, branch):
        """Get HEAD of a branch.

        :param repo:
        :param branch:
        :return:
        """
        owner, repo_name, username, password = self.get_repo_info(repo)
        auth = (username, password) if username and password else None
        url = self.urls['branches'].format(owner=owner, repo_name=repo_name, branch=branch)
        return self.make_request(requests.get, url, auth=auth, value_only=True)['commit']['sha']

    def get_pull_request(self, repo, id):
        """Get pull request.

        :param repo:
        :param id:
        :return:
        """
        owner, repo_name, username, password = self.get_repo_info(repo)
        auth = (username, password) if username and password else None
        url = self.urls['pull_request'].format(owner=owner, repo_name=repo_name, id=id)
        return self.make_request(requests.get, url, auth=auth, value_only=True)

    def get_webhook(self, repo):
        """Get webhook.

        :param repo:
        :return:
        """
        webhooks = self.get_webhooks(repo)
        webhooks = [webhook for webhook in webhooks if webhook['config'].get('url') == self.construct_webhook_url()]
        return webhooks[0] if webhooks else None

    def get_webhooks(self, repo):
        """Get all webhooks.

        :param repo:
        :return:
        """
        # GitHub webhook rest endpoint does not properly handle pagination.
        # The result is paginated if we pass parameter per_page=<int>, but
        # the links are not properly included in the headers of the response.
        # Thus, if the repository has a large number of webhooks, the endpoint
        # may not return all the webhooks for us to iterate, even if we set
        # a large value for per_page parameter. However, we feel that in
        # practice, having a large number of webhooks is rare.
        owner, repo_name, username, password = self.get_repo_info(repo)
        auth = (username, password) if username and password else None
        url = self.urls['hooks'].format(owner=owner, repo_name=repo_name)
        return self.make_request(requests.get, url, params={'per_page': 1000}, auth=auth, value_only=True)

    def create_webhook(self, repo):
        """Create webhook.

        :param repo:
        :return:
        """
        if self.has_webhook(repo):
            return {}
        owner, repo_name, username, password = self.get_repo_info(repo)
        auth = (username, password) if username and password else None
        url = self.urls['hooks'].format(owner=owner, repo_name=repo_name)
        payload = {
            'name': 'web',  # On GitHub, a webhook must have name 'web'
            'active': True,
            'events': [
                'create',
                'push',
                'pull_request',
                'issue_comment'
            ],
            'config': {
                'content_type': 'json',
                'insecure_ssl': '1',
                'url': self.construct_webhook_url()
            }
        }
        return self.make_request(requests.post, url, auth=auth, json=payload, value_only=True)

    def delete_webhook(self, repo):
        """Delete webhook.

        :param repo:
        :return:
        """
        webhook = self.get_webhook(repo)
        if webhook:
            webhook_id = webhook['id']
            owner, repo_name, username, password = self.get_repo_info(repo)
            auth = (username, password) if username and password else None
            url = self.urls['hook'].format(owner=owner, repo_name=repo_name, id=webhook_id)
            self.make_request(requests.delete, url, auth=auth)
        return {}

    def upload_job_result(self, payload):
        """Upload job result to bitbucket.

        :param payload:
        :return:
        """
        job_result = self._cache_job_result(payload)
        report_payload = {
            'context': job_result['context'],
            'description': job_result['description'] or '',
            'state': job_result['state'],
            'target_url': job_result['target_url']
        }
        # GitHub does not allow description greater than 140 characters
        if report_payload['description'] and len(report_payload['description']) >= 140:
            report_payload['description'] = report_payload['description'][:136] + '...'
        owner, repo_name, username, password = self.get_repo_info(job_result['repo'])
        auth = (username, password) if username and password else None
        url = self.urls['status'].format(owner=owner, repo_name=repo_name, commit=job_result['commit'])
        if len(job_result.get('commit', '')) != 40:
            logger.info('GitHub does not support status report for the non-sha commits. Skip.')
            return -1
        else:
            self.make_request(requests.post, url, auth=auth, json=report_payload)
        return {}

    def _cache_job_result(self, payload):
        """Cache job result.

        :param payload:
        :return:
        """
        service_id = payload.get('id')
        # Retrieve cache
        if service_id in redis_client.keys():
            old_cache = redis_client.get(service_id, decoder=json.loads)
        else:
            old_cache = {}
        # Extract status
        status = int(payload.get('status', 255))
        if status != 0:
            status /= abs(status)
        status = self.STATUS_MAPPING.get(int(status))
        # Extract name
        if 'name' in payload:
            context = 'ax service "{}" ({})'.format(
                payload['name'], datetime.datetime.now(tz=tzlocal()).strftime('%Y-%m-%dT%H:%M:%S%z')
            )
        elif 'context' in old_cache:
            context = old_cache['context']
        else:
            context = 'ax service "unknown" ({})'.format(
                datetime.datetime.now(tz=tzlocal()).strftime('%Y-%m-%dT%H:%M:%S%z')
            )
        # Create payload
        new_cache = {
            'id': service_id,
            'repo': payload.get('repo') or old_cache.get('repo'),
            'commit': payload.get('commit') or old_cache.get('commit'),
            'context': context,
            'description': payload.get('description') or old_cache.get('description'),
            'state': status,
            'target_url': '{}/{}'.format(self.WEBHOOK_JOB_URL, service_id)
        }
        redis_client.set(service_id, new_cache, encoder=json.dumps)
        return new_cache

    def get_yamls(self, repo, commit):
        """Get all YAML files in .argo folder.

        :param repo:
        :param commit:
        :return:
        """
        owner, name, username, password = self.get_repo_info(repo)
        auth = (username, password) if username and password else None
        contents = self._scan_yamls_recursively(repo, commit)
        yamls = []
        for i in range(len(contents)):
            url = contents[i]['download_url']
            resp = self.make_request(requests.get, url, auth=auth)
            content = resp.text
            yamls.append({
                'path': contents[i]['path'],
                'content': content
            })
        return yamls

    def _scan_yamls_recursively(self, repo, commit):
        """Recursively scan all yaml files under .argo folder.

        :param repo:
        :param commit:
        :return: List of file names.
        """
        owner, repo_name, username, password = self.get_repo_info(repo)
        auth = (username, password) if username and password else None
        yamls = []
        dirs = [TEMPLATE_DIR]
        while dirs:
            dir = dirs.pop()
            url = self.urls['content'].format(owner=owner, repo_name=repo_name, commit=commit, path=dir)
            contents = self.make_request(requests.get, url, auth=auth, value_only=True)
            for content in contents:
                if content['type'] == 'dir':
                    dirs.append(content['path'])
                elif content['path'].endswith('.yaml') or content['path'].endswith('.yml'):
                    yamls.append({
                        'path': content['path'],
                        'download_url': content['download_url']
                    })
        return yamls

    def url_to_repo(self, url):
        """Convert a rest api url to its repo url.

        :param url:
        :return:
        """
        tokens = re.split(r'://|/', url)
        owner, repo_name = tokens[3], tokens[4]
        return self.repo_root_url.format(owner=owner, repo_name=repo_name)

    def get_webhook_whitelist(self):
        """Get a list of webhook whitelist
        :return: a list
        """
        default_whitelist = ['0.0.0.0/0']
        meta_url = 'https://api.github.com/meta'
        try:
            resp = self.make_request(requests.get, meta_url, value_only=True)
        except Exception as exc:
            logger.warn(exc)
            return default_whitelist
        else:
            return resp.get('hooks', default_whitelist)

