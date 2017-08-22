import copy
import json
import threading
import logging
import re
import sys
import time
from apscheduler.schedulers.background import BackgroundScheduler
from voluptuous import Schema, Required, Optional, MultipleInvalid
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.utility.utilities import AxPrettyPrinter
from ax.devops.kafka.kafka_client import EventNotificationClient
from ax.notification_center import FACILITY_AX_SCHEDULER, CODE_JOB_SCHEDULER_INVALID_POLICY_DEFINITION,\
    CODE_JOB_SCHEDULER_INVALID_CRON_EXPRESSION, CODE_JOB_SCHEDULER_CANNOT_ADD_POLICY

logger = logging.getLogger(__name__)

# Schema for policy
policy_schema = Schema({
    Required('id'): str,
    Required('repo'): str,
    Required('branch'): str,
    Required('template'): str,
    Required('enabled'): bool,
    Required('when'): [],
    Optional('arguments', default=lambda: {}): {},
    Optional('notifications', default=lambda: []): [],
}, extra=True)

# Schema for commit
commit_schema = Schema({
    Required('revision'): str,
    Required('repo'): str,
    Required('branch'): str,
    Required('author'): str,
    Required('committer'): str,
    Optional('description', default=""): str,
    Required('date'): int,
}, extra=True)

# Schema for on_cron policy
schedule_schema = Schema({
    Required('event'): 'on_cron',
    Required('schedule'): str,
    Optional('timezone', default='UTC'): str,
}, extra=True)


class JobScheduler(object):
    def __init__(self, axops_host=None):
        self._schedule_lock = threading.Lock()
        self.axops_client = AxopsClient(host=axops_host)
        self.scheduler = BackgroundScheduler()
        self.event_notification_client = EventNotificationClient(FACILITY_AX_SCHEDULER)

    def init(self):
        """
        Init Job Scheduler. Check access to AxOps.
        """
        counter = 0
        while counter < 20:
            if self.axops_client.ping():
                self.refresh_scheduler()
                return
            else:
                counter += 1
                logger.info("JobScheduler cannot ping AxOps. Count: %s", counter)
                time.sleep(10)
        logger.error("[Init] scheduler failed to ping AxOps after 20 tries. Exit.")
        sys.exit(1)

    def refresh_scheduler(self):
        """
        Refresh the job scheduler.

        The major functionality of this service. Read all the cron policies from AxOps, then
        load the schedules into the job scheduler.
        """

        if self._schedule_lock.acquire(timeout=2):  # Try to acquire lock for 2 seconds
            try:
                scheduler = BackgroundScheduler()
                logger.info("Start refreshing the scheduler.")

                for policy in self.axops_client.get_policy(enabled=True):
                    self.add_policy(policy, scheduler)

                # Scheduler swap
                self.stop_scheduler()
                self.scheduler = scheduler
                self.scheduler.start()
                logger.info("Successfully finish refreshing the scheduler. \n%s",
                            AxPrettyPrinter().pformat(self.get_schedules()))
                return {}
            finally:
                self._schedule_lock.release()
        else:
            with self._schedule_lock:
                logger.info("Some other thread is refreshing the scheduler. Instant return.")
            return {'Details': 'Instant return'}

    def add_policy(self, policy, scheduler):
        """
        Add a schedule into scheduler based on policy.

        Ignore exceptions (for now).
        """
        try:
            policy_json = policy_schema(policy)
            policy_id = policy_json['id']
            event_list = policy_json['when']
            logger.info("Processing policy, %s", policy_id)
            for event in event_list:
                if event.get('event', None) != 'on_cron':
                    continue
                event_json = schedule_schema(event)
                cron_str = event_json['schedule'].strip().split(' ')  # Parse the cron string
                assert len(cron_str) == 5, "Invalid cron schedule format"
                logger.info("Adding cron event, \n %s", AxPrettyPrinter().pformat(event_json))
                scheduler.add_job(self.create_service, 'cron',  # Add cron job into scheduler
                                  id='{}-{}'.format(policy_id, cron_str),
                                  args=[policy_json],
                                  minute=cron_str[0],
                                  hour=cron_str[1],
                                  day=cron_str[2],
                                  month=cron_str[3],
                                  day_of_week=cron_str[4],
                                  timezone=event_json['timezone'])
        except MultipleInvalid as e:
            logger.exception("Invalid cron policy format, \n%s. Details: %s", AxPrettyPrinter().pformat(policy), str(e))
            try:
                if 'when' in policy:
                    policy['when'] = json.dumps(policy['when'])
                self.event_notification_client.send_message_to_notification_center(CODE_JOB_SCHEDULER_INVALID_POLICY_DEFINITION, detail=policy)
            except Exception:
                logger.exception("Failed to send out alert to notification center.")
        except AssertionError as e:
            logger.exception("Invalid cron policy format, \n%s, cron string. Details: %s", AxPrettyPrinter().pformat(policy), str(e))
            try:
                if 'when' in policy:
                    policy['when'] = json.dumps(policy['when'])
                self.event_notification_client.send_message_to_notification_center(CODE_JOB_SCHEDULER_INVALID_CRON_EXPRESSION, detail=policy)
            except Exception:
                logger.exception("Failed to send out alert to notification center.")
        except Exception as e:
            logger.exception("Failed to add event, \n%s into scheduler. Details: %s", AxPrettyPrinter().pformat(policy), str(e))
            try:
                if 'when' in policy:
                    policy['when'] = json.dumps(policy['when'])
                self.event_notification_client.send_message_to_notification_center(CODE_JOB_SCHEDULER_CANNOT_ADD_POLICY, detail=policy)
            except Exception:
                logger.exception("Failed to send out alert to notification center.")

    @staticmethod
    def is_matched(target_branches, branch_name):
        """
        Check the regex of target branches can be matched with branch name.
        """
        is_matched = False
        for branch in target_branches:
            try:
                if re.compile(branch).match(branch_name):
                    is_matched = True
                    break
            except Exception as e:
                logger.exception("Failed to compare using regex. %s", str(e))
                pass
        return is_matched

    def create_service(self, policy):
        """
        Create job based on the policy.

        The payload is tailored for the AxOps POST /v1/services. This might get improved in the future.
        """
        logger.info("Start triggering job based on cron schedule. Policy info: \n%s", AxPrettyPrinter().pformat(policy))

        service_template = self.axops_client.get_templates(policy['repo'], policy['branch'], name=policy['template'])[0]
        commit_res = self.axops_client.get_commit_info(repo=policy['repo'], branch=policy['branch'], limit=1)
        if not commit_res or len(commit_res) != 1:
            logger.error("Error retrieving latest commit info for cron job, commit_info: %s. Return", commit_res)
            return
        commit_json = commit_schema(commit_res[0])
        notification_info = policy['notifications']

        commit_info = {
            'revision': commit_json['revision'],
            'repo': commit_json['repo'],
            'branch': commit_json['branch'],
            'author': commit_json['author'],
            'committer': commit_json['committer'],
            'description': commit_json['description'],
            'date': commit_json['date']
        }

        parameters = copy.deepcopy(policy['arguments'])
        parameters['session.commit'] = commit_json['revision']
        parameters['session.branch'] = commit_json['branch']
        parameters['session.repo'] = commit_json['repo']

        service = {
            'template_id': service_template['id'],
            'arguments': parameters,
            'policy_id': policy['id'],
            'commit': commit_info,
        }

        if notification_info:
            service['notifications'] = notification_info

        logger.info("Creating new service with the following payload ...\n%s", AxPrettyPrinter().pformat(service))
        service = self.axops_client.create_service(service)
        logger.info('Successfully created service (id: %s)', service['id'])

    def get_schedules(self):
        """
        Get the scheduled jobs in the current scheduler.
        :return: list of scheduled jobs.
        """
        result = dict()
        if self.scheduler:
            for job in self.scheduler.get_jobs():
                result[job.id] = str(job)
        return result

    def stop_scheduler(self, wait=False):
        """
        Stop the current scheduler.
        :param wait: whether to wait for the current running job.
        :return:
        """
        if self.scheduler.running:
            self.scheduler.shutdown(wait=wait)
