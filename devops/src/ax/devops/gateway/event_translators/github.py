import copy
import logging
import random

from ax.devops.ci.constants import AxEventTypes, GitHubEventTypes, ScmVendors
from .base import BaseEventTranslator
from ax.devops.exceptions import UnrecognizableEventType
from ax.devops.gateway.scm_rest.github_client import GitHubClient
from ax.devops.utility.utilities import utc

logger = logging.getLogger(__name__)


class GitHubEventTranslator(BaseEventTranslator):
    """GitHub event translator."""

    vendor = ScmVendors.GITHUB
    client = GitHubClient()
    greetings = [
        'Hello!',
        'Greetings!',
        'Nice to see you!',
        'Welcome!'
    ]

    @classmethod
    def translate(cls, payload, headers=None):
        """Translate event.

        :param payload:
        :param headers:
        :return:
        """
        event_key = headers.get('X-GitHub-Event')
        if event_key not in GitHubEventTypes.values():
            raise UnrecognizableEventType('Unrecognizable event type', detail='Unrecognizable event type ({})'.format(event_key))
        if event_key == GitHubEventTypes.PING:
            return cls._translate_ping()
        elif event_key == GitHubEventTypes.CREATE:
            return cls._translate_create(payload)
        elif event_key == GitHubEventTypes.PULL_REQUEST:
            return cls._translate_pull_request(payload)
        elif event_key == GitHubEventTypes.ISSUE_COMMENT:
            return cls._translate_pull_request_comment(payload)
        else:
            return cls._translate_push(payload)

    @classmethod
    def _translate_ping(cls):
        """Translate ping event.

        :return:
        """
        message = random.choice(cls.greetings)
        event = {
            'vendor': cls.vendor,
            'type': AxEventTypes.PING,
            'message': message
        }
        return [event]

    @classmethod
    def _translate_push(cls, payload):
        """Translate push event.

        :param payload:
        :return:
        """
        if payload['deleted']:
            logger.info('It is delete event, ignore it')
            return []
        event = {
            'vendor': cls.vendor,
            'type': AxEventTypes.PUSH,
            'repo': payload['repository']['clone_url'],
            'branch': '/'.join(payload['ref'].split('/')[2:]),  # Branch is extracted from reference having form '/refs/heads/<branch_name>'
            'commit': payload['head_commit']['id'],
            'author': '{} {}'.format(payload['head_commit']['author']['name'],
                                     payload['head_commit']['author']['email']),
            'committer': '{} {}'.format(payload['head_commit']['committer']['name'],
                                        payload['head_commit']['committer']['email']),
            'description': payload['head_commit']['message'],
            'date': utc(payload['head_commit']['timestamp'])
        }
        return [event]

    @classmethod
    def _translate_create(cls, payload):
        """Translate create event.

        :param payload:
        :return:
        """
        if payload.get('ref_type', '') != 'tag':
            logger.info('It is NOT tag creation event, ignore it')
            return []

        event = {
            'vendor': cls.vendor,
            'type': AxEventTypes.CREATE,
            'repo': payload['repository']['clone_url'],
            'branch': payload['master_branch'],
            'commit': payload['ref'],
            'author': payload['sender']['login'],
            'committer': payload['sender']['login'],
            'description': payload.get('description', ''),
            'date': utc(payload['repository']['pushed_at'])
        }
        return [event]

    @classmethod
    def _translate_pull_request(cls, payload):
        """Translate pull request event.

        :param payload:
        :return:
        """
        event_type = AxEventTypes.PULL_REQUEST_MERGE if payload['action'] == 'closed' else AxEventTypes.PULL_REQUEST
        commit_obj = cls.client.get_commit(payload['pull_request']['head']['repo']['clone_url'],
                                           payload['pull_request']['head']['sha'])
        author = '{} {}'.format(commit_obj['commit']['author']['name'], commit_obj['commit']['author']['email'])
        committer = '{} {}'.format(commit_obj['commit']['committer']['name'], commit_obj['commit']['committer']['email'])

        if payload['pull_request'].get('merged', False):
            _commit = payload['pull_request']['merge_commit_sha']
        else:
            _commit = payload['pull_request']['head']['sha']
        event = {
            'vendor': cls.vendor,
            'type': event_type,
            'commit': _commit,
            'author': author,
            'committer': committer,
            'title': payload['pull_request']['title'],
            'description': payload['pull_request']['body'],
            'date': utc(payload['pull_request']['updated_at'])
        }
        if event['type'] == AxEventTypes.PULL_REQUEST_MERGE:
            extra_info = {
                'repo': payload['pull_request']['base']['repo']['clone_url'],
                'branch': payload['pull_request']['base']['ref'],
                'source_repo': payload['pull_request']['head']['repo']['clone_url'],
                'source_branch': payload['pull_request']['head']['ref']
            }
        else:
            extra_info = {
                'repo': payload['pull_request']['head']['repo']['clone_url'],
                'branch': payload['pull_request']['head']['ref'],
                'target_repo': payload['pull_request']['base']['repo']['clone_url'],
                'target_branch': payload['pull_request']['base']['ref']
            }
        event.update(extra_info)
        return [event]

    @classmethod
    def _translate_pull_request_comment(cls, payload):
        """Translate pull request comment.

        :param payload:
        :return:
        """
        _pull_request = payload['issue'].get('pull_request', None)
        if _pull_request is None:
            logger.warn('It is a general issue_comment event rather than pull_request_comment, ignore it')
            return []
        pull_request_url = _pull_request['url']
        repo = cls.client.url_to_repo(pull_request_url)
        pull_request_id = pull_request_url.split('/')[-1]
        pull_request = cls.client.get_pull_request(repo, pull_request_id)
        commit_obj = cls.client.get_commit(pull_request['head']['repo']['clone_url'], pull_request['head']['sha'])
        author = '{} {}'.format(commit_obj['commit']['author']['name'], commit_obj['commit']['author']['email'])
        committer = '{} {}'.format(commit_obj['commit']['committer']['name'], commit_obj['commit']['committer']['email'])
        event_template = {
            'vendor': cls.vendor,
            'type': AxEventTypes.PULL_REQUEST,
            'repo': pull_request['head']['repo']['clone_url'],
            'branch': pull_request['head']['ref'],
            'commit': pull_request['head']['sha'],
            'target_repo': pull_request['base']['repo']['clone_url'],
            'target_branch': pull_request['base']['ref'],
            'author': author,
            'committer': committer,
            'title': pull_request['title'],
            'description': pull_request['body'],
            'date': utc(pull_request['updated_at'])
        }
        commands = cls._parse_command(payload['comment']['body'])
        events = []
        for command in commands:
            event = copy.copy(event_template)
            for k in command:
                event[k] = command[k]
            events.append(event)
        return events
