import logging
import json
import os

from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.scm_rest.github_client import GitHubClient
from ax.notification_center import CODE_JOB_CI_ELB_CREATION_FAILURE
from gateway.settings import LOGGER_NAME
from gateway.kafka import event_notification_client

axops_client = AxopsClient()
axsys_client = AxsysClient()
github_client = GitHubClient()
cache_file = '/tmp/github_webhook_whitelist'

logger = logging.getLogger('{}.{}'.format(LOGGER_NAME, 'jira_cron'))


def check_github_whitelist():
    """
    :return:
    """
    if not is_github_webhook_enabled():
        logger.info('No GitHub webhook configured')
        return

    configured = get_from_cache()
    logger.info('The configured GitHub webhook whitelist is %s', configured)
    advertised = github_client.get_webhook_whitelist()
    logger.info('The GitHub webhook whitelist is %s', advertised)
    if set(configured) == set(advertised):
        logger.info('No update needed')
    else:
        # Create ELB
        payload = {'ip_range': advertised, 'external_port': 8443, 'internal_port': 8087}
        try:
            logger.info('Creating ELB for webhook ...')
            axsys_client.create_webhook(**payload)
        except Exception as exc:
            logger.error('Failed to create ELB for webhook: %s', str(exc))
            event_notification_client.send_message_to_notification_center(CODE_JOB_CI_ELB_CREATION_FAILURE, detail=payload)
        else:
            # Update cache
            write_to_cache(advertised)
            logger.info('Successfully updated ELB for webhook')


def is_github_webhook_enabled():
    """ Check whether the webhook is configured or not
    :return:
    """
    github_data = axops_client.get_tools(type='github')
    use_webhook = [each for each in github_data if each['use_webhook']]
    return bool(use_webhook)


def write_to_cache(ip_range):
    """ Store the webhook whitelist info
    :param ip_range:
    :return:
    """
    with open(cache_file, 'w+') as f:
        f.write(json.dumps((ip_range)))


def get_from_cache():
    """ Get cached webhook whitelist info, otherwise get from axmon
    :return:
    """
    if os.path.exists(cache_file):
        with open(cache_file, 'r+') as f:
            data = f.readlines()
            ip_range = json.loads(data[0])
    else:
        logger.debug('No cache file')
        try:
            data = axsys_client.get_webhook()
        except Exception as exc:
            logger.warn(exc)
        else:
            logger.info('Write whitelist info to cache file')
            ip_range = data['ip_ranges']
            write_to_cache(ip_range)
    return ip_range



