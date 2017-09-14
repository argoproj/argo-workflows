import datetime
import logging
import re
import requests

from urllib.parse import urlparse
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.ci.constants import AxCommands, AxEventTypes
from ax.devops.exceptions import YamlUpdateError
from ax.devops.kafka.kafka_client import EventNotificationClient
from ax.exceptions import AXApiInternalError
from ax.notification_center import FACILITY_AX_EVENT_TRIGGER, CODE_JOB_CI_TEMPLATE_NOT_FOUND

logger = logging.getLogger(__name__)


class EventTrigger(object):
    """DevOps event trigger."""

    axops_client = AxopsClient()
    event_notification_client = EventNotificationClient(FACILITY_AX_EVENT_TRIGGER)
    event_keys = {
        AxEventTypes.PUSH: [
            'repo',
            'branch',
            'commit',
            'author'
        ],
        AxEventTypes.CREATE: [
            'repo',
            'branch',
            'commit',
            'author'
        ],
        AxEventTypes.PULL_REQUEST: [
            'repo',
            'branch',
            'commit',
            'target_repo',
            'target_branch',
            'author'
        ],
        AxEventTypes.PULL_REQUEST_MERGE: [
            'repo',
            'branch',
            'commit',
            'source_repo',
            'source_branch',
            'author'
        ]
    }

    def evaluate(self, event):
        """Evaluate an event by enforcing its applicable event policies, and trigger a service instance.

        :param event:
        :return:
        """
        logger.info('Evaluating AX event ...')
        # TODO: Currently, git and codecommit do not support the update of YAML files
        if event['vendor'] not in {'git', 'codecommit'}:
            logger.info('Updating policies/templates (repo: %s, branch: %s) ...', event['repo'], event['branch'])
            payload = {
                'type': event['vendor'],
                'repo': event['repo'],
                'branch': event['branch']
            }
            # Not sure what it does
            if len(event.get('commit', '')) == 36:
                logger.info('It is a commit, try to update the YAML file')
                resp = requests.post('http://gateway:8889/v1/scm/yamls', json=payload)
                if 400 <= resp.status_code < 600:
                    raise YamlUpdateError('Failed to update YAML content', detail='Failed to update policy/template')

        services = []
        # If the event does not have command section or the command is rerun, we need to enforce policies
        if 'command' not in event or event.get('command') == AxCommands.RERUN:
            logger.info('Searching for applicable event policies ...')
            applicable_event_policies = self.get_applicable_event_policies(event)
            if applicable_event_policies:
                logger.info('Found %s applicable event policies', len(applicable_event_policies))
                for i in range(len(applicable_event_policies)):
                    # If we only need to run failed jobs, we need to retrieve the status of last job
                    if event.get('command') == AxCommands.RERUN and not event.get('rerun_all', False):
                        most_recent_service = self.axops_client.get_most_recent_service(
                            event['repo'], event['commit'], applicable_event_policies[i]['id']
                        )
                        if most_recent_service['status'] >= 0:
                            logger.info('Most recent service is successful or still running; since user selects '
                                        'to rerun failed jobs, the creation of this service will be skipped')
                            continue
                    service = self.enforce_policy(applicable_event_policies[i], event)
                    if service:
                        services.append(service)
            else:
                logger.warning('Found 0 applicable event policies, skip processing')
        # If the command is run, we do not need to enforce policies
        else:
            logger.info('Received event with command, kicking off service now ...')
            service = self.run_command(event)
            if service:
                services.append(service)
        logger.info('Evaluation completed')
        return services or []

    def get_applicable_event_policies(self, event):
        """Get applicable event policies.

        :param event:
        :return:
        """
        applicable_policies = []
        if event['type'] == AxEventTypes.PULL_REQUEST:
            # Verify if the source repo of the pull request is configured in the integration page
            if not self.verify_repo_configured(event['repo']):
                return applicable_policies

            key_repo, key_branch = 'target_repo', 'target_branch'
        else:
            key_repo, key_branch = 'repo', 'branch'
        policies = self.axops_client.get_policies(repo=event[key_repo], branch=event[key_branch], enabled=True)
        logger.info("Found enabled policies with repo %s, branch %s, %s", event[key_repo], event[key_branch], policies)

        for policy in policies:
            if self.match(policy, event):
                applicable_policies.append(policy)

        return applicable_policies

    def verify_repo_configured(self, repo_url):
        """Verify is the repo is integrated into argo from the tools integration page"""
        tools = self.axops_client.get_tools(category='scm')
        vendor_owner_name = self.get_vendor_owner_name(repo_url)

        for tool in tools:
            repo_list = tool.get('repos', [])
            for repo in repo_list:
                res = self.get_vendor_owner_name(repo)
                if res and res == vendor_owner_name:
                    return True
        return False

    @staticmethod
    def get_vendor_owner_name(repo_url):
        """Get vendor, repo owner and repo name from the repo url"""
        parsed_url = urlparse(repo_url)
        protocol, vendor = parsed_url.scheme, parsed_url.hostname
        m = re.match(r'/([a-zA-Z0-9-]+)/([a-zA-Z0-9_.-]+)', parsed_url.path)
        if not m:
            logger.warning('Illegal repo URL: %s, skip', parsed_url)
            return []
        _, repo_owner, repo_name = parsed_url.path.split('/', maxsplit=2)
        return [vendor, repo_owner, repo_name]

    def match(self, policy, event):
        """Determine if a policy is applicable to an event.

        :param policy:
        :param event:
        :return:
        """
        conditions = policy['when']
        for condition in conditions:
            if self.match_condition(condition, event):
                return True
        return False

    def match_condition(self, condition, event):
        """Match event with a condition.

        :param condition:
        :param event:
        :return:
        """
        if event['type'] == AxEventTypes.CREATE and not condition['event'].endswith('tag'):
            return False
        elif event['type'] != AxEventTypes.CREATE and not condition['event'].endswith(event['type']):
            return False

        return True

    @staticmethod
    def match_patterns(patterns, string):
        """Match branch.

        :param patterns:
        :param string:
        :return:
        """
        for pattern in patterns:
            try:
                match = re.match(pattern, string)
            except Exception as e:
                logger.error('Failed to match pattern (pattern: %s, string: %s): %s', pattern, string, e)
                raise AXApiInternalError('Failed to match pattern', detail=str(e))
            else:
                if match:
                    return True
        return False

    def enforce_policy(self, policy, event):
        """Enforce policy.

        :param policy:
        :param event:
        :return:
        """
        # Retrieve service template payload
        logger.info('Retrieving service template ...')
        service_template = self.get_service_template_by_policy(policy)
        if not service_template:
            logger.warning('Unable to find service template, skip')
            return
        # Construct parameters from event and policy
        logger.info('Constructing parameters ...')
        parameters = self.construct_parameters(event=event, policy=policy)
        notifications = self.construct_notifications(policy)
        commit = self.construct_commit_info(event)
        # Create service instance
        service = {
            'commit': commit,
            'notifications': notifications,
            'arguments': parameters,
            'policy_id': policy['id'],
            'template': service_template
        }
        service = self.axops_client.create_service(service)
        logger.info('Successfully created service (id: %s)', service['id'])
        return service

    def run_command(self, event):
        """Run command specified in the event.

        :param event:
        :return:
        """
        # Retrieve service template payload
        logger.info('Retrieving service template ...')
        service_template = self.get_service_template_by_event(event)
        if not service_template:
            logger.warning('Unable to find service template, skip')
            self.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_TEMPLATE_NOT_FOUND, detail=event)
            return
        # Construct parameters from event
        logger.info('Constructing parameters ...')
        parameters = self.construct_parameters(event=event)
        for key in event.get('arguments', {}):
            parameters[key] = event['arguments'][key]
        notifications = self.construct_notifications()
        commit = self.construct_commit_info(event)
        # Create service instance
        service = {
            'commit': commit,
            'notifications': notifications,
            'arguments': parameters,
            'template': service_template
        }
        service = self.axops_client.create_service(service)
        logger.info('Successfully created service (id: %s)', service['id'])
        return service

    def get_service_template_by_policy(self, policy):
        """Get service template by policy.

        :param policy:
        :return:
        """
        service_templates = self.axops_client.get_templates(policy['repo'], policy['branch'], policy['template'])
        if service_templates:
            return service_templates[0]

    def get_service_template_by_event(self, event):
        """Get service template by event.

        :param event:
        :return:
        """
        service_templates = self.axops_client.get_templates(event['repo'], event['branch'], event['template'])
        if service_templates:
            return service_templates[0]

    def construct_parameters(self, event, policy=None):
        """Construct parameters.

        :param event:
        :param policy:
        :return:
        """
        parameters_from_event = self.event_to_parameters(event)
        parameters = policy.get('arguments', {}) if policy else {}
        parameters.update(parameters_from_event)
        return parameters

    @staticmethod
    def construct_notifications(policy=None):
        """Construct notifications.

        :param policy:
        :return:
        """
        notifications = [
            {
                "whom": [
                    "scm"
                ],
                "when": [
                    "on_success",
                    "on_failure"
                ]
            }
        ]
        if policy and 'notifications' in policy:
            notifications += policy['notifications']
        return notifications

    @staticmethod
    def construct_commit_info(event):
        """Construct commit info.

        :param event:
        :return:
        """
        return {
            'revision': event['commit'],
            'repo': event['repo'],
            'branch': event['branch'],
            'author': event['author'],
            'committer': event['committer'],
            'description': event['description'],
            'date': int((datetime.datetime.strptime(event['date'], '%Y-%m-%dT%H:%M:%S') -
                         datetime.datetime(1970, 1, 1)).total_seconds())
        }

    def event_to_parameters(self, event):
        """Create parameters from event.

        :param event:
        :return:
        """
        parameters = {}
        for k in self.event_keys[event['type']]:
            key = 'session.{}'.format(k)
            parameters[key] = event[k]
        if event['type'] in [AxEventTypes.PUSH, AxEventTypes.CREATE]:
            parameters['session.target_branch'] = event['branch']
        return parameters
