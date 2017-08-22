import datetime
import json
import re
import requests
import uuid
from dateutil.tz import tzlocal

from ax.devops.redis.redis_client import RedisClient, DB_REPORTING
from ax.devops.scm_rest import BaseScmRestClient
from . import TEMPLATE_DIR

redis_client = RedisClient('redis', db=DB_REPORTING)


class BitBucketClient(BaseScmRestClient):
    """REST API wrapper for BitBucket."""

    STATUS_MAPPING = {
        -1: 'FAILED',
        0: 'SUCCESSFUL',
        1: 'INPROGRESS'
    }

    def __init__(self):
        """Initializer."""
        super().__init__()
        self.root_urls = {
            '1.0': 'https://api.bitbucket.org/1.0',
            '2.0': 'https://api.bitbucket.org/2.0'
        }
        self.urls = {
            'branches': '%s/repositories/{owner}/{repo_name}/refs/branches/{branch}' % self.root_urls['2.0'],
            'commit': '%s/repositories/{owner}/{repo_name}/commit/{commit}' % self.root_urls['2.0'],
            'content': '%s/repositories/{owner}/{repo_name}/src/{commit}/{path}' % self.root_urls['1.0'],
            'hook': '%s/repositories/{owner}/{repo_name}/hooks/{id}' % self.root_urls['2.0'],
            'hooks': '%s/repositories/{owner}/{repo_name}/hooks' % self.root_urls['2.0'],
            'repos': '%s/repositories/?role=member' % self.root_urls['2.0'],
            'status': '%s/repositories/{owner}/{repo_name}/commit/{commit}/statuses/build' % self.root_urls['2.0']
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
            data = self.make_request(requests.get, url, auth=(username, password), value_only=True)
            url = data.get('next')
            paginated_repos = data.get('values', [])
            for repo in paginated_repos:
                repo_urls = [v['href'] for v in repo['links']['clone'] if v['name'] == 'https']
                for repo_url in repo_urls:
                    m = re.match(r'(https)://(.*)@(.*)', repo_url)
                    if m:
                        _repo_url = '{}://{}'.format(m.groups()[0], m.groups()[-1])
                        repos[_repo_url] = _repo_url
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
        return self.make_request(requests.get, url, auth=auth, value_only=True)['target']['hash']

    def get_webhook(self, repo):
        """Get webhook.

        :param repo:
        :return:
        """
        webhooks = self.get_webhooks(repo)
        webhooks = [webhook for webhook in webhooks if webhook['url'] == self.construct_webhook_url()]
        return webhooks[0] if webhooks else None

    def get_webhooks(self, repo):
        """Get all webhooks.

        :param repo:
        :return:
        """
        owner, repo_name, username, password = self.get_repo_info(repo)
        auth = (username, password) if username and password else None
        url = self.urls['hooks'].format(owner=owner, repo_name=repo_name)
        resp = self.make_request(requests.get, url, auth=auth, value_only=True)
        if resp['size'] > resp['pagelen']:
            resp = self.make_request(requests.get, url, params={'pagelen': resp['size']}, auth=auth, value_only=True)
        webhooks = resp['values']
        return webhooks

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
            'description': self.WEBHOOK_TITLE,
            'url': self.construct_webhook_url(),
            'active': True,
            'events': [
                'repo:push',
                'pullrequest:created',
                'pullrequest:updated',
                'pullrequest:fulfilled',
                'pullrequest:comment_created'
            ],
            'skip_cert_verification': True
        }
        return self.make_request(requests.post, url, auth=auth, json=payload, value_only=True)

    def delete_webhook(self, repo):
        """Delete webhook.

        :param repo:
        :return:
        """
        webhook = self.get_webhook(repo)
        if webhook:
            webhook_id = webhook['uuid']
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
            'name': job_result['name'],
            'description': job_result['description'] or '',  # Make sure that description is not null
            'key': job_result['key'],
            'url': job_result['url'],
            'state': job_result['state']
        }
        owner, repo_name, username, password = self.get_repo_info(job_result['repo'])
        auth = (username, password) if username and password else None
        url = self.urls['status'].format(owner=owner, repo_name=repo_name, commit=job_result['commit'])
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
        status = payload.get('status', 255)
        if status != 0:
            status /= abs(status)
        status = self.STATUS_MAPPING.get(int(status))
        # Extract name
        if 'name' in payload:
            name = 'ax service "{}" ({})'.format(
                payload['name'], datetime.datetime.now(tz=tzlocal()).strftime('%Y-%m-%dT%H:%M:%S%z')
            )
        elif 'name' in old_cache:
            name = old_cache['name']
        else:
            name = 'ax service "unknown" ({})'.format(
                datetime.datetime.now(tz=tzlocal()).strftime('%Y-%m-%dT%H:%M:%S%z')
            )
        # Create payload
        new_cache = {
            'id': service_id,
            'name': name,
            'repo': payload.get('repo') or old_cache.get('repo'),
            'commit': payload.get('commit') or old_cache.get('commit'),
            'description': payload.get('description') or old_cache.get('description'),
            'state': status,
            'key': payload.get('key') or old_cache.get('key') or 'ax-{}'.format(str(uuid.uuid1())),
            'url': '{}/{}'.format(self.WEBHOOK_JOB_URL, service_id)
        }
        redis_client.set(service_id, new_cache, encoder=json.dumps)
        return new_cache

    def get_yamls(self, repo, commit):
        """Get all YAML files in .argo folder.

        :param repo:
        :param commit:
        :return:
        """
        owner, repo_name, username, password = self.get_repo_info(repo)
        auth = (username, password) if username and password else None
        fnames = self._scan_yamls_recursively(repo, commit)
        yamls = []
        for fname in fnames:
            url = self.urls['content'].format(owner=owner, repo_name=repo_name, commit=commit, path=fname)
            data = self.make_request(requests.get, url, auth=auth, value_only=True)
            yamls.append({
                'path': data['path'],
                'content': data['data']
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
            data = self.make_request(requests.get, url, auth=auth, value_only=True)
            path, subdirs, files = data['path'], data['directories'], data['files']
            for file in files:
                if file['path'].endswith('.yaml') or file['path'].endswith('.yml'):
                    yamls.append(file['path'])
            for dir in subdirs:
                dirs.append('{}{}'.format(path, dir))
        return yamls

    def get_webhook_whitelist(self):
        """Get a list of webhook whitelist
        :return: a list
        """
        return ['104.192.143.0/24']
