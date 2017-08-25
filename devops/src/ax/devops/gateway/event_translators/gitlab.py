import copy
import logging

from ax.devops.ci.constants import AxEventTypes, GitLabEventTypes, ScmVendors
from .base import BaseEventTranslator
from ax.devops.exceptions import UnrecognizableEventType
from ax.devops.gateway.scm_rest.gitlab_client import GitLabClient
from ax.devops.utility.utilities import utc

logger = logging.getLogger(__name__)


class GitLabEventTranslator(BaseEventTranslator):
    """GitLab event translator."""

    vendor = ScmVendors.GITLAB
    client = GitLabClient()

    @classmethod
    def translate(cls, payload, headers=None):
        """Translate event.

        :param payload:
        :param headers:
        :return:
        """
        event_key = headers.get('HTTP_X_GITLAB_EVENT')
        if event_key not in GitLabEventTypes.values():
            raise UnrecognizableEventType('Unrecognizable event type', detail='Unrecognizable event type ({})'.format(event_key))
        if event_key == GitLabEventTypes.MERGE_REQUEST:
            return cls._translate_pull_request(payload)
        elif event_key == GitLabEventTypes.NOTES:
            return cls._translate_pull_request_comment(payload)
        else:
            return cls._translate_push(payload)

    @classmethod
    def _translate_push(cls, payload):
        """Translate push event.

        :param payload:
        :return:
        """
        event = {
            'vendor': cls.vendor,
            'type': AxEventTypes.PUSH,
            'repo': payload['project']['git_http_url'],
            'branch': '/'.join(payload['ref'].split('/')[2:]),
            'commit': payload['commits'][0]['id'],
            'author': '{} {}'.format(payload['commits'][0]['author']['name'],
                                     payload['commits'][0]['author']['email']),
            'committer': '{} {}'.format(payload['commits'][0]['author']['name'],
                                        payload['commits'][0]['author']['email']),
            'description': payload['commits'][0]['message'],
            'date': utc(payload['commits'][0]['timestamp'])
        }
        return [event]

    @classmethod
    def _translate_pull_request(cls, payload):
        """Translate pull request event.

        :param payload:
        :return:
        """
        if payload['object_attributes']['state'] == 'merged':
            event_type = AxEventTypes.PULL_REQUEST_MERGE
        else:
            event_type = AxEventTypes.PULL_REQUEST
        event = {
            'vendor': cls.vendor,
            'type': event_type,
            'commit': payload['object_attributes']['last_commit']['id'],
            'author': '{} {}'.format(payload['object_attributes']['last_commit']['author']['name'],
                                     payload['object_attributes']['last_commit']['author']['email']),
            'committer': '{} {}'.format(payload['object_attributes']['last_commit']['author']['name'],
                                        payload['object_attributes']['last_commit']['author']['email']),
            'title': payload['object_attributes']['title'],
            'description': payload['object_attributes']['description'],
            'date': utc(payload['object_attributes']['updated_at'])
        }
        if event['type'] == AxEventTypes.PULL_REQUEST_MERGE:
            extra_info = {
                'repo': payload['object_attributes']['target']['git_http_url'],
                'branch': payload['object_attributes']['target_branch'],
                'source_repo': payload['object_attributes']['source']['git_http_url'],
                'source_branch': payload['object_attributes']['source_branch']
            }
        else:
            extra_info = {
                'repo': payload['object_attributes']['source']['git_http_url'],
                'branch': payload['object_attributes']['source_branch'],
                'target_repo': payload['object_attributes']['target']['git_http_url'],
                'target_branch': payload['object_attributes']['target_branch']
            }
        event.update(extra_info)
        return [event]

    @classmethod
    def _translate_pull_request_comment(cls, payload):
        """Translate pull request comment.

        :param payload:
        :return:
        """
        if payload['object_attributes']['noteable_type'] != 'MergeRequest':
            logger.warning('Currently, non-merge-request based comments are not supported, skip')
            return []
        event_template = {
            'vendor': cls.vendor,
            'type': AxEventTypes.PULL_REQUEST,
            'repo': payload['merge_request']['source']['git_http_url'],
            'branch': payload['merge_request']['source_branch'],
            'commit': payload['merge_request']['last_commit']['id'],
            'target_repo': payload['merge_request']['target']['git_http_url'],
            'target_branch': payload['merge_request']['target_branch'],
            'author': '{} {}'.format(payload['merge_request']['last_commit']['author']['name'],
                                     payload['merge_request']['last_commit']['author']['email']),
            'committer': '{} {}'.format(payload['merge_request']['last_commit']['author']['name'],
                                        payload['merge_request']['last_commit']['author']['email']),
            'title': payload['merge_request']['title'],
            'description': payload['merge_request']['description'],
            'date': utc(payload['object_attributes']['updated_at'])
        }
        commands = cls._parse_command(payload['object_attributes']['note'])
        events = []
        for command in commands:
            event = copy.copy(event_template)
            for k in command:
                event[k] = command[k]
            events.append(event)
        return events
