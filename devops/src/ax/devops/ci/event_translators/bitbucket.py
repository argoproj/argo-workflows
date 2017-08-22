import copy
import logging

from ax.devops.ci.constants import AxEventTypes, BitBucketEventTypes, ScmTypes, ScmVendors
from ax.devops.ci.event_translators.base import BaseEventTranslator
from ax.devops.exceptions import UnrecognizableEventType, UnsupportedSCMType
from ax.devops.scm_rest.bitbucket_client import BitBucketClient
from ax.devops.utility.utilities import utc
from ax.notification_center import CODE_JOB_CI_INVALID_EVENT_TYPE, CODE_JOB_CI_INVALID_SCM_TYPE

logger = logging.getLogger(__name__)


class BitBucketEventTranslator(BaseEventTranslator):
    """BitBucket event translator."""

    vendor = ScmVendors.BITBUCKET
    client = BitBucketClient()

    @classmethod
    def translate(cls, payload, headers=None):
        """Translate event.

        :param payload:
        :param headers:
        :return:
        """
        event_key = headers.get('HTTP_X_EVENT_KEY')
        scm_type = payload.get('repository', {}).get('scm')
        if event_key not in BitBucketEventTypes.values():
            cls.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_INVALID_EVENT_TYPE, detail={'event_type': event_key})
            raise UnrecognizableEventType('Unrecognizable event type', detail='Unrecognizable event type ({})'.format(event_key))
        if scm_type not in ScmTypes.values():
            cls.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_INVALID_SCM_TYPE, detail={'scm_type': scm_type})
            raise UnsupportedSCMType('Unsupported SCM type', detail='Unsupported SCM type ({})'.format(scm_type))
        if event_key in {BitBucketEventTypes.PULL_REQUEST_CREATED,
                         BitBucketEventTypes.PULL_REQUEST_UPDATED}:
            return cls._translate_pull_request(payload)
        elif event_key == BitBucketEventTypes.PULL_REQUEST_FULFILLED:
            return cls._translate_pull_request_merge(payload)
        elif event_key == BitBucketEventTypes.PULL_REQUEST_COMMENT_CREATED:
            return cls._translate_pull_request_comment(payload)
        else:
            return cls._translate_push(payload)

    @classmethod
    def _translate_push(cls, payload):
        """Translate push event.

        BitBucket may put multiple pushes (i.e. when the developer uses push --all) into one event.

        :param payload:
        :return:
        """
        changes = payload['push']['changes']
        events = []
        for change in changes:
            if change.get('new'):
                event = {
                    'vendor': cls.vendor,
                    'type': AxEventTypes.PUSH,
                    'repo': payload['repository']['links']['html']['href'] + '.git',
                    'branch': change['new']['name'],
                    'commit': change['new']['target']['hash'],
                    'author': change['new']['target']['author']['raw'],
                    'committer': change['new']['target']['author']['raw'],
                    'description': change['new']['target']['message'],
                    'date': utc(change['new']['target']['date'])
                }
                events.append(event)
        return events

    @classmethod
    def _translate_pull_request(cls, payload):
        """Translate pull request event.

        :param payload:
        :return:
        """
        pull_request = payload.get('pullrequest')
        repo = pull_request['source']['repository']['links']['html']['href'] + '.git'
        commit_short = pull_request['source']['commit']['hash']
        commit_obj = cls.client.get_commit(repo, commit_short)
        event = {
            'vendor': cls.vendor,
            'type': AxEventTypes.PULL_REQUEST,
            'repo': repo,
            'branch': pull_request['source']['branch']['name'],
            'commit': commit_obj['hash'],
            'target_repo': pull_request['destination']['repository']['links']['html']['href'] + '.git',
            'target_branch': pull_request['destination']['branch']['name'],
            'author': commit_obj['author']['raw'],
            'committer': commit_obj['author']['raw'],  # BitBucket rest api does not have committer section
            'title': pull_request['title'],
            'description': pull_request['description'],
            'date': utc(commit_obj['date'])
        }
        return [event]

    @classmethod
    def _translate_pull_request_merge(cls, payload):
        """Translate pull request merge.

        :param payload:
        :return:
        """
        pull_request = payload.get('pullrequest')
        repo = pull_request['destination']['repository']['links']['html']['href'] + '.git'
        commit_short = pull_request['merge_commit']['hash']
        commit_obj = cls.client.get_commit(repo, commit_short)
        event = {
            'vendor': cls.vendor,
            'type': AxEventTypes.PULL_REQUEST_MERGE,
            'repo': repo,
            'branch': pull_request['destination']['branch']['name'],
            'commit': commit_obj['hash'],
            'source_repo': pull_request['source']['repository']['links']['html']['href'] + '.git',
            'source_branch': pull_request['source']['branch']['name'],
            'author': commit_obj['author']['raw'],
            'committer': commit_obj['author']['raw'],  # BitBucket rest api does not have committer section
            'title': pull_request['title'],
            'description': pull_request['description'],
            'date': utc(commit_obj['date'])
        }
        return [event]

    @classmethod
    def _translate_pull_request_comment(cls, payload):
        """Translate pull request comment.

        :param payload:
        :return:
        """
        event_template = cls._translate_pull_request(payload)[0]
        commands = cls._parse_command(payload['comment']['content']['raw'])
        events = []
        for command in commands:
            event = copy.copy(event_template)
            for k in command:
                event[k] = command[k]
            events.append(event)
        return events
