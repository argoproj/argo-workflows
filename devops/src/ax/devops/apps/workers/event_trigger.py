import argparse
import logging
import pprint
import requests
import sys
from retrying import retry

from ax.devops.ci.event_trigger import EventTrigger as _EventTrigger
from ax.devops.kafka.kafka_client import ConsumerClient, ProducerClient
from ax.devops.settings import AxSettings
from ax.version import __version__

logger = logging.getLogger(__name__)
event_trigger = _EventTrigger()


class EventTrigger():

    def __init__(self):
        """
        :return:
        """
        self.consumer = ConsumerClient(AxSettings.TOPIC_DEVOPS_CI_EVENT)
        self.consumer.poll()

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=10)
    def run(self):
        """
        :return:
        """
        while True:
            logger.info('Start consume event ...')
            try:
                for message in self.consumer.consumer:
                    logger.debug(message)
                    self._consumer_event(message.value)
            except Exception as exc:
                logger.exception(exc)
            finally:
                self.consumer.close()

    def _consumer_event(self, event):
        """Evaluate an event by enforcing its applicable event policies, and trigger a service instance.
        :param event:
        :return:
        """
        try:
            logger.info('Received AX event\n%s', pprint.pformat(event))
            services = event_trigger.evaluate(event)
        except Exception as e:
            logger.warning('Unexpected exception occurred during processing: %s', e)
        else:
            for service in services:
                self._report_status(service, event)

    @staticmethod
    def _report_status(service, message):
        """
        :param service:
        :param message:
        :return:
        """
        try:
            logger.info('Updating build status (service_id: %s, repo: %s, commit: %s) ...',
                        service['id'], message['repo'], message['commit'])
            payload = {
                'id': service['id'],
                'name': service['name'],
                'repo': message['repo'],
                'commit': message['commit'],
                'description': message['description'],
                'status': 255
            }
            resp = requests.post('http://gateway:8889/v1/scm/reports', json=payload)
            resp.raise_for_status()
        except Exception as e:
            logger.warning('Failed to upload job result: %s', e)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('-l', '--log-level', dest='log_level', type=int, default=logging.INFO, help="specify log level")
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    args = parser.parse_args()

    logging.basicConfig(stream=sys.stdout, level=args.log_level,
                        format="%(asctime)s %(levelname)s %(lineno)d %(name)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S")
    logging.getLogger('ax.devops.scm').setLevel(logging.WARNING)
    logging.getLogger('ax.devops.kafka.kafka_client').setLevel(logging.WARNING)
    logging.getLogger('kafka.conn').setLevel(logging.WARNING)
    logging.getLogger('kafka.producer.kafka').setLevel(logging.WARNING)
    event_trigger = EventTrigger()
    event_trigger.run()
