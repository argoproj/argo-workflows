import datetime
import json
import logging
import requests
from dateutil.tz import tzlocal

from ax.devops.exceptions import UnknownRepository
from ax.devops.redis.redis_client import RedisClient, DB_REPORTING
from . import BaseScmRestClient
from . import TEMPLATE_DIR

logger = logging.getLogger(__name__)

redis_client = RedisClient('redis', db=DB_REPORTING)


class GitLabClient(BaseScmRestClient):
    """REST API wrapper for GitLab.

    In GitLab, the structure of endpoint is fundamentally different from GitHub and BitBucket.
    GitLab has a project in all its rest endpoints. Though currently, it assumes there should
    be only 1 repository under each project; the situation may change in future. Currently, we
    assume project = repository.

    """

    STATUS_MAPPING = {
        -1: 'failed',
        0: 'success',
        1: 'running'
    }

    def __init__(self):
        """Initializer."""
        super().__init__()
        self.root_url = 'https://gitlab.com/api/v3'
        self.repo_root_url = 'https://github.com/{owner}/{repo_name}.git'
        self.urls = {
            'login': '%s/session?login={username}&password={password}' % self.root_url,
            'branches': '%s/projects/{owner}%%2F{repo_name}/repository/branches/{branch}' % self.root_url,
            'commit': '%s/projects/{owner}%%2F{repo_name}/repository/commits/{commit}' % self.root_url,
            'hook': '%s/projects/{owner}%%2F{repo_name}/hooks/{id}' % self.root_url,
            'hooks': '%s/projects/{owner}%%2F{repo_name}/hooks' % self.root_url,
            'repos': '%s/projects/owned' % self.root_url,
            'repo_blob': '%s/projects/{owner}%%2F{repo_name}/repository/blobs/{commit}?filepath={path}' % self.root_url,
            'repo_tree': '%s/projects/{owner}%%2F{repo_name}/repository/tree?path={path}&ref_name={commit}' % self.root_url,
            'status': '%s/projects/{owner}%%2F{repo_name}/statuses/{commit}' % self.root_url
        }

    def get_repos(self, username, password):
        """Get repos that an account can see.

        :param username:
        :param password:
        :return:
        """
        private_token = self.login(username, password)
        url = self.urls['repos']
        repos = {}
        while url is not None:
            resp = self.make_request(requests.get, url, headers={'PRIVATE-TOKEN': private_token})
            url = resp.links.get('next', {}).get('url')
            paginated_repos = resp.json()
            for repo in paginated_repos:
                repo_url = repo['http_url_to_repo']
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
        owner, repo_name, private_token = self.get_repo_info(repo)
        url = self.urls['commit'].format(owner=owner, repo_name=repo_name, commit=commit)
        return self.make_request(requests.get, url, headers={'PRIVATE-TOKEN': private_token}, value_only=True)

    def get_branch_head(self, repo, branch):
        """Get HEAD of a branch.

        :param repo:
        :param branch:
        :return:
        """
        owner, repo_name, private_token = self.get_repo_info(repo)
        url = self.urls['branches'].format(owner=owner, repo_name=repo_name, branch=branch)
        return self.make_request(requests.get, url, headers={'PRIVATE-TOKEN': private_token}, value_only=True)['commit']['id']

    def get_webhook(self, repo):
        """Get webhook.

        :param repo:
        :return:
        """
        webhooks = self.get_webhooks(repo)
        webhooks = [webhook for webhook in webhooks if webhook.get('url') == self.construct_webhook_url()]
        return webhooks[0] if webhooks else None

    def get_webhooks(self, repo):
        """Get all webhooks.

        :param repo:
        :return:
        """
        owner, repo_name, private_token = self.get_repo_info(repo)
        url = self.urls['hooks'].format(owner=owner, repo_name=repo_name)
        return self.make_request(requests.get, url, headers={'PRIVATE-TOKEN': private_token}, value_only=True)

    def create_webhook(self, repo):
        """Create webhook.

        :param repo:
        :return:
        """
        if self.has_webhook(repo):
            return {}
        owner, repo_name, private_token = self.get_repo_info(repo)
        url = self.urls['hooks'].format(owner=owner, repo_name=repo_name)
        payload = {
            'url': self.construct_webhook_url(),
            'enable_ssl_verification': False,
            'push_events': True,
            'issues_events': False,
            'merge_requests_events': True,
            'tag_push_events': False,
            'note_events': True,
            'build_events': False,
            'pipeline_events': False,
            'wiki_events': False
        }
        return self.make_request(requests.post, url, headers={'PRIVATE-TOKEN': private_token}, json=payload, value_only=True)

    def delete_webhook(self, repo):
        """Delete webhook.

        :param repo:
        :return:
        """
        webhook = self.get_webhook(repo)
        if webhook:
            webhook_id = webhook['id']
            owner, repo_name, private_token = self.get_repo_info(repo)
            url = self.urls['hook'].format(owner=owner, repo_name=repo_name, id=webhook_id)
            self.make_request(requests.delete, url, headers={'PRIVATE-TOKEN': private_token})
        return {}

    def upload_job_result(self, payload):
        """Upload job result to bitbucket.

        :param payload:
        :return:
        """
        job_result = self._cache_job_result(payload)
        report_payload = {
            'name': job_result['name'],
            'description': job_result['description'] or '',
            'state': job_result['state'],
            'target_url': job_result['target_url']
        }
        owner, repo_name, private_token = self.get_repo_info(job_result['repo'])
        url = self.urls['status'].format(owner=owner, repo_name=repo_name, commit=job_result['commit'])
        self.make_request(requests.post, url, headers={'PRIVATE-TOKEN': private_token}, json=report_payload)
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
            'repo': payload.get('repo') or old_cache.get('repo'),
            'commit': payload.get('commit') or old_cache.get('commit'),
            'name': name,
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
        owner, repo_name, private_token = self.get_repo_info(repo)
        paths = self._scan_yamls_recursively(repo, commit)
        yamls = []
        for i in range(len(paths)):
            url = self.urls['repo_blob'].format(owner=owner, repo_name=repo_name, commit=commit, path=paths[i])
            resp = self.make_request(requests.get, url, headers={'PRIVATE-TOKEN': private_token})
            yamls.append({
                'path': paths[i],
                'content': resp.text
            })
        return yamls

    def _scan_yamls_recursively(self, repo, commit):
        """Recursively scan all yaml files under .argo folder.

        :param repo:
        :param commit:
        :return: List of file names.
        """
        owner, repo_name, private_token = self.get_repo_info(repo)
        yamls = []
        dirs = [TEMPLATE_DIR]
        while dirs:
            dir = dirs.pop()
            url = self.urls['repo_tree'].format(owner=owner, repo_name=repo_name, commit=commit, path=dir)
            contents = self.make_request(requests.get, url, headers={'PRIVATE-TOKEN': private_token}, value_only=True)
            for content in contents:
                if content['type'] == 'tree':
                    dirs.append('{}/{}'.format(dir, content['name']))
                elif content['path'].endswith('.yaml') or content['path'].endswith('.yml'):
                    yamls.append('{}/{}'.format(dir, content['name']))
        return yamls

    def login(self, username, password):
        """Login into GitLab to get a session.

        :param username:
        :param password:
        :return:
        """
        url = self.urls['login'].format(username=username, password=password)
        return self.make_request(requests.post, url, value_only=True)['private_token']

    def get_repo_info(self, repo):
        """Get owner, name, and credential of repo.

        :param repo:
        :return:
        """
        if repo not in self.repos:
            tool_config = self.axops_client.get_tool(repo)
            if not tool_config:
                raise UnknownRepository('Unable to find configuration for repo ({})'.format(repo))
            self.update_repo_info(repo, tool_config['type'], tool_config['username'], tool_config['password'])
        return self.repos[repo]['owner'], self.repos[repo]['name'], self.repos[repo]['private_token']

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
        private_token = self.login(username, password)
        self.repos[repo] = {
            'type': type,
            'name': name,
            'owner': owner,
            'username': username,
            'password': password,
            'private_token': private_token
        }

    def get_webhook_whitelist(self):
        """Get a list of webhook whitelist
        :return: a list
        """
        default_whitelist = ['0.0.0.0/0']

        return default_whitelist
