import datetime
import logging
import os
import re
import shutil
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from retrying import retry
from urllib.parse import urlparse

from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.ci.constants import ScmVendors
from ax.devops.kafka.kafka_client import ProducerClient
from ax.devops.redis.redis_client import RedisClient
from ax.devops.scm.scm import GitClient, CodeCommitClient
from ax.devops.settings import AxSettings

logger = logging.getLogger(__name__)

BASE_DIR = '/ax/data/repos'
NAMESPACE = 'gateway'
TEMPLATE_DIR = '.argo'

redis_client = RedisClient('redis', db=10)


class RepoManager(object):
    """Manage all repositories in track."""

    def __init__(self, concurrency, interval):
        self.axdb_client = AxdbClient()
        self.axops_client = AxopsClient()
        self.concurrency = concurrency
        self.interval = interval

    def run(self):
        """Create workspaces and perform initial/incremental fetch."""
        while True:
            logger.info('Start repository scan ...')
            try:
                self.connect()
                repos, has_change = self.synchronize()
                with ThreadPoolExecutor(max_workers=self.concurrency) as executor:
                    futures = []
                    for i in range(len(repos)):
                        futures.append(executor.submit(self.update_repo, **repos[i]))
                    for future in as_completed(futures):
                        try:
                            if not has_change and future.result():
                                has_change = True
                        except Exception as e:
                            logger.warning('Unexpected exception occurred during processing: %s', e)
                # Notify UI backend about a change in repos
                if has_change:
                    key = '{}:repos_updated'.format(NAMESPACE)
                    redis_client.set(key, value=str(int(time.time())))
            except Exception as e:
                logger.warning('Repository scan failed: %s', str(e))
            else:
                logger.info('Repository scan completed\n')
            finally:
                time.sleep(self.interval)

    @retry(wait_fixed=5000)
    def connect(self):
        """Connect to axops."""
        connected = self.axops_client.ping()
        if not connected:
            msg = 'Unable to connect to axops'
            logger.warning(msg)
            raise ConnectionError(msg)

    def synchronize(self):
        """Synchronize all repos."""
        logger.info('Synchronizing repositories ...')

        # Get all repos
        repos = self.get_all_repos()
        logger.info('%s repositories currently in track', len(repos))

        # Get untracked repos currently on disk
        untracked_repos = self.get_untracked_repos(repos)
        logger.info('%s untracked repositories found on disk', len(list(untracked_repos.keys())))

        for repo in untracked_repos:
            # Purge all branch heads
            logger.info('Purging branch heads (repo: %s) ...', repo)
            self.axdb_client.purge_branch_heads(repo)
            # Delete workspace
            logger.info('Deleting workspace (path: %s) ...', untracked_repos[repo])
            shutil.rmtree(untracked_repos[repo])
            # Invalidate caches
            logger.info('Invalidating caches (workspace: %s) ...', untracked_repos[repo])
            key_pattern = '^{}\:{}.*$'.format(NAMESPACE, untracked_repos[repo])
            keys = redis_client.keys(key_pattern)
            for k in keys:
                logger.debug('Invalidating cache (key: %s) ...', k)
                redis_client.delete(k)

        # Send event to trigger garbage collection from axops
        if untracked_repos:
            kafka_client = ProducerClient()
            ci_event = {
                'Op': "gc",
                'Payload': {
                    'details': "Repo or branch get deleted."
                }
            }
            kafka_client.send(AxSettings.TOPIC_GC_EVENT, key=AxSettings.TOPIC_GC_EVENT, value=ci_event, timeout=120)

        return repos, len(untracked_repos) > 0

    def get_all_repos(self):
        """Retrieve all repos from axops."""
        tools = self.axops_client.get_tools(category='scm')
        repos = {}
        for i in range(len(tools)):
            _repos = tools[i].get('repos', [])
            for j in range(len(_repos)):
                parsed_url = urlparse(_repos[j])
                protocol, vendor = parsed_url.scheme, parsed_url.hostname
                m = re.match(r'/([a-zA-Z0-9-]+)/([a-zA-Z0-9_.-]+)', parsed_url.path)
                if not m:
                    logger.warning('Illegal repo URL: %s, skip', parsed_url)
                    continue
                _, repo_owner, repo_name = parsed_url.path.split('/', maxsplit=2)
                key = (vendor, repo_owner, repo_name)
                if key in repos and repos[key]['protocol'] == 'https':
                    continue
                repos[key] = {
                    'repo_type': tools[i].get('type'),
                    'vendor': vendor,
                    'protocol': protocol,
                    'repo_owner': repo_owner,
                    'repo_name': repo_name,
                    'username': tools[i].get('username'),
                    'password': tools[i].get('password'),
                    'use_webhook': tools[i].get('use_webhook', False)
                }
        return list(repos.values())

    @staticmethod
    def get_untracked_repos(repos):
        """Get all untracked repos."""
        # Construct list of expected workspaces
        expected_workspaces = set()
        for i in range(len(repos)):
            expected_workspace = '{}/{}/{}/{}'.format(
                BASE_DIR, repos[i]['vendor'], repos[i]['repo_owner'], repos[i]['repo_name']
            )
            expected_workspaces.add(expected_workspace)
        # Construct list of all workspaces currently on disk
        dirs = [dir[0] for dir in os.walk(BASE_DIR) if dir[0].endswith('/.git')]
        workspaces = list(map(lambda v: v[:-5], dirs))
        # Construct list of untracked repos
        untracked_repos = {}
        for i in range(len(workspaces)):
            if workspaces[i] not in expected_workspaces:
                client = GitClient(path=workspaces[i])
                repo = client.get_remote()
                untracked_repos[repo] = workspaces[i]
        return untracked_repos

    @staticmethod
    def get_repo_workspace(repo_vendor, repo_owner, repo_name):
        return '{}/{}/{}/{}'.format(BASE_DIR, repo_vendor, repo_owner, repo_name)

    @staticmethod
    def get_repo_url(protocol, repo_vendor, repo_owner, repo_name):
        return '{}://{}/{}/{}'.format(protocol, repo_vendor, repo_owner, repo_name)

    @staticmethod
    def update_yaml(repo_client, kafka_client, repo, branch, head):
        """Using Kafka to send a event to axops to update the yamls in the axdb."""
        logger.info("Update yaml %s, %s, %s", repo, branch, head)
        try:
            yaml_contents = repo_client.get_files(commit=head, subdir=TEMPLATE_DIR, filter_yaml=True)
        except Exception as e:
            logger.error("Failed to obtain YAML files: %s", str(e))
            return -1

        if len(yaml_contents) >= 0:
            # This is a partition key defined as RepoName$$$$BranchName.
            # The key is used by Kafka partition, which means it allows concurrency
            #  if the events are for different repo/branch
            key = '{}$$$${}'.format(repo, branch)
            payload = {
                'Op': 'update',
                'Payload': {
                    'Revision': head,
                    'Content': [v['content'] for v in yaml_contents] if yaml_contents else []
                }
            }
            kafka_client.send('devops_template', key=key, value=payload, timeout=120)
            logger.info("Updated YAML %s files (repo: %s, branch: %s)", len(yaml_contents), repo, branch)
        return len(yaml_contents)

    def update_repo(self, repo_type, vendor, protocol, repo_owner, repo_name, username, password, use_webhook):
        """Update a repo."""

        # Examples for the input variables
        # BASE_DIR:   /ax/data/repos
        # Repo_type:  github
        # Vendor:     github.com
        # Protocol:   https
        # Repo_owner: argo
        # Repo_name:  prod.git

        is_first_fetch = False
        do_send_gc_event = False
        workspace = '{}/{}/{}/{}'.format(BASE_DIR, vendor, repo_owner, repo_name)
        url = '{}://{}/{}/{}'.format(protocol, vendor, repo_owner, repo_name)
        kafka_client = ProducerClient()

        if not os.path.isdir(workspace):
            os.makedirs(workspace)
            # If we recreate the workspace, we need to purge all branch heads of this repo
            self.axdb_client.purge_branch_heads(url)

        logger.info("Start scanning repository (%s) ...", url)
        if repo_type == ScmVendors.CODECOMMIT:
            client = CodeCommitClient(path=workspace, repo=url, username=username, password=password)
        else:
            client = GitClient(path=workspace, repo=url, username=username, password=password, use_permanent_credentials=True)

        # Even if there is no change, performing a fetch is harmless but has a benefit
        # that, in case the workspace is destroyed without purging the history, we can
        # still update the workspace to the proper state
        logger.info("Start fetching ...")
        client.fetch()

        # Retrieve all previous branch heads and construct hash table
        prev_heads = self.axdb_client.get_branch_heads(url)
        logger.info("Have %s branch heads (repo: %s) from previous scan", len(prev_heads), url)

        if len(prev_heads) == 0:
            is_first_fetch = True
            logger.debug("This is an initial scan as no previous heads were found")

        prev_heads_map = dict()
        for prev_head in prev_heads:
            key = (prev_head['repo'], prev_head['branch'])
            prev_heads_map[key] = prev_head['head']

        # Retrieve all current branch heads
        current_heads = client.get_remote_heads()
        logger.info("Have %s branch heads (repo: %s) from current scan", len(current_heads), url)
        current_heads = sorted(current_heads, key=lambda v: v['commit_date'], reverse=is_first_fetch)

        # Find out which branch heads need to be updated
        heads_to_update = list()
        heads_for_event = list()
        for current_head in current_heads:
            head, branch = current_head['commit'], current_head['reference'].replace('refs/heads/', '')
            previous_head = prev_heads_map.pop((url, branch), None)
            if head != previous_head:
                event = {
                    'repo': url,
                    'branch': branch,
                    'head': head
                }
                heads_to_update.append(event)

                if previous_head is None:
                    logger.info("New branch detected (branch: %s, current head: %s)", branch, head)
                else:
                    logger.info("Existing ranch head updated (branch: %s, previous: %s, current: %s)", branch, previous_head, head)
                    # Send CI event in case of policy
                    heads_for_event.append(event.copy())

        if prev_heads_map:
            logger.info("There are %s get deleted from repo: %s", prev_heads_map.keys(), url)
            do_send_gc_event = True
            for key in prev_heads_map:
                self.axdb_client.purge_branch_head(repo=key[0], branch=key[1])

        # Invalidate cache if there is head update or branch deleted
        if heads_to_update or prev_heads_map:
            cache_key = '{}:{}'.format(NAMESPACE, workspace)
            logger.info('Invalidating cache (key: %s) ...', cache_key)
            if redis_client.exists(cache_key):
                redis_client.delete(cache_key)

        # Update YAML contents
        count = 0
        for event in heads_to_update:
            res_count = RepoManager.update_yaml(repo_client=client, kafka_client=kafka_client, repo=url,
                                                branch=event['branch'], head=event['head'])
            if res_count >= 0:
                self.axdb_client.set_branch_head(**event)
                count += res_count

        logger.info("Updated %s YAML files (template/policy) for %s branches (repo: %s)", count, len(heads_to_update), url)
        logger.info("Updated %s branch heads (repo: %s)", len(heads_to_update), url)

        # If garbarge collection needed due to branch or repo deletion
        if do_send_gc_event:
            logger.info("Send gc event so that axops can garbage collect deleted branch / repo")
            ci_event = {
                'Op': "gc",
                'Payload': {
                    'details': "Repo or branch get deleted."
                }
            }
            kafka_client.send(AxSettings.TOPIC_GC_EVENT, key=AxSettings.TOPIC_GC_EVENT, value=ci_event, timeout=120)

        # If webhook is disabled, we need to send CI events
        if not use_webhook:
            for event in heads_for_event:
                commit = client.get_commit(event['head'])
                ci_event = {
                    'Op': "ci",
                    'Payload': {
                        'author': commit['author'],
                        'branch': event['branch'],
                        'commit': commit['revision'],
                        'committer': commit['committer'],
                        'date': datetime.datetime.fromtimestamp(commit['date']).strftime('%Y-%m-%dT%H:%M:%S'),
                        'description': commit['description'],
                        'repo': commit['repo'],
                        'type': "push",
                        'vendor': repo_type
                    }
                }
                kafka_client.send("devops_template", key="{}$$$${}".format(event['repo'], event['branch']), value=ci_event, timeout=120)
            logger.info('Webhook not enabled, send %s devops_ci_event events', len(heads_for_event))

        kafka_client.close()
        logger.info('Successfully scanned repository (%s)', url)

        return len(heads_to_update) > 0 or len(prev_heads_map) > 0
