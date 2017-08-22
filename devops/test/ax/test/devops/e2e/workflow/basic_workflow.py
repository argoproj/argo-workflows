#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#

from ax.util.az_patch import az_patch
az_patch()

import argparse
import copy
import json
import logging
import os
import random
import sys
import time
import unittest
import uuid

from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.utility.utilities import AxPrettyPrinter

logger = logging.getLogger('ax.devops.test.basic_workflow')

DEFAULT_TIMEOUT = 1 * 60 * 60  # 1 hour timeout
DEFAULT_MAX_CHORD_LENGTH = 5
DEFAULT_MAX_CHORD_WIDTH = 5


class BasicWorkflowTest(unittest.TestCase):
    """Basic workflow test."""

    # Fixture configuration
    target_cluster = None
    axops_client = None
    username = None
    password = None

    # Test configuration
    service_template = None
    timeout = DEFAULT_TIMEOUT
    max_chord_length = DEFAULT_MAX_CHORD_LENGTH
    max_chord_width = DEFAULT_MAX_CHORD_WIDTH

    # For negative test
    negative_test = False
    expected_status = 0

    @classmethod
    def setUpClass(cls):
        """Set up test environment.

        Steps:
            1. Load service template for test.

        :return:
        """
        hostname, port = cls.target_cluster, 443
        logger.info('Connecting AXOPS server (hostname: %s, port: %s) ...', hostname, port)
        cls.axops_client = AxopsClient(host=hostname, port=port, protocol='https', ssl_verify=False, username=cls.username, password=cls.password)

        # Create service template to be used in test
        service_template_file = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'service_template.json')
        with open(service_template_file) as f:
            logger.info('Loading service template for test ...')
            cls.service_template = json.load(f)
            logger.info('Successfully loaded service template (%s) for test', cls.service_template['name'])

    def test_single_workflow(self):
        """Test workflow with a single subtask.

        :return:
        """
        logger.info('Creating a service request with a single subtask ... ')
        service_request = self._create_single_workflow()
        logger.info('Successfully created a service request: \n%s', AxPrettyPrinter().pformat(service_request))
        logger.info('Start executing service ...')
        service = self.axops_client.create_service(service_request)
        result = self._wait_for_result(service['id'], self.timeout)
        self.assertTrue(result, 'Failed to execute service ({})'.format(service['id']))
        logger.info('Successfully executed service (%s)', service['id'])

    def test_chord_workflow(self):
        """Test workflow with a chord.

        :return:
        """
        logger.info('Creating a service request with a chord workflow ... ')
        service_request = self._create_chord_workflow()
        logger.info('Successfully created a service request (payload: %s)', json.dumps(service_request))
        logger.info('Start executing service ...')
        service = self.axops_client.create_service(service_request)
        result = self._wait_for_result(service['id'], self.timeout)
        self.assertTrue(result, 'Failed to execute service ({})'.format(service['id']))
        logger.info('Successfully executed service (%s)', service['id'])

    def _create_single_workflow(self):
        """Create a single workflow.

        :return:
        """
        return {
            'template': self._create_service_template()
        }

    def _create_chord_workflow(self):
        """Create a chord workflow.

        :return:
        """
        length = self._generate_random_chord_length()
        chord_desc = []
        service_request = {
            'template': {
                'id': str(uuid.uuid1()),
                'type': 'service_template',
                'subtype': 'workflow',
                'version': '',
                'name': 'basic_workflow',
                'dns_name': '',
                'description': 'Workflow created for basic workflow test',
                'cost': 0,
                'inputs': {},
                'outputs': {},
            }
        }
        steps = []
        for i in range(length):
            step = {}
            width = self._generate_random_chord_width()
            for j in range(width):
                k = 'step-{}.{}'.format(i + 1, j + 1)
                v = {
                    'template': self._create_service_template(),
                    'status': 0,
                    'cost': 0,
                    'launch_time': 0,
                    'run_time': 0,
                    'average_runtime': 0
                }
                step[k] = v
            steps.append(step)
            chord_desc.append('{%s}' % width)
        service_request['template']['steps'] = steps
        chord_desc = ' -> '.join(chord_desc)
        logger.info('Chord workflow representational diagram: %s', chord_desc)
        return service_request

    def _generate_random_chord_length(self):
        """Generate random chord length.

        :return:
        """
        return random.randint(1, self.max_chord_length)

    def _generate_random_chord_width(self):
        """Generate random chord width.

        :return:
        """
        return random.randint(1, self.max_chord_width)

    def _create_service_template(self):
        """Create a service template with a random sleep command.

        :return:
        """
        cmd = self._generate_cmd_and_expected_status()
        service_template = copy.deepcopy(self.service_template)
        service_template['container']['command'] = '{} {}'.format(cmd, random.randint(10, 30))
        return service_template

    def _generate_cmd_and_expected_status(self):
        """Generate command and expected status.

        :return:
        """
        # If negative test, throw a dice
        if self.negative_test:
            if random.choice([True, False]):
                cmd = 'sleep'
            else:
                cmd = 'sleepy'
                self.expected_status = -1
        else:
            cmd = 'sleep'
        return cmd

    def _wait_for_result(self, service_id, timeout=None):
        """Wait for result to be posted to axdb.

        :param service_id:
        :param timeout:
        :return:
        """
        start_time = time.time()
        while time.time() <= (start_time + timeout):
            logger.info('Waiting for service status to be updated (id: %s: expected_status: %s) ...', service_id, self.expected_status)
            service = self.axops_client.get_service(service_id)
            if service['status'] in {0, -1}:
                logger.info('Service execution completed (id: %s, status: %s)', service_id, service['status'])
                return service['status'] == self.expected_status
            else:
                time.sleep(10)
        logger.warning('Unable to get status updated for service (id: %s, timeout: %s) in time', service_id, timeout)
        return False


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-m', '--test-methods', action='append', dest='test_methods', default=None, help='specify the test methods to run')
    parser.add_argument('-c', '--target-cluster', action='store', dest='target_cluster', required=True, help='specify the public dnsname of the target cluster')
    parser.add_argument('-u', '--username', action='store', dest='username', required=True, help='specify the username for logging into axops')
    parser.add_argument('-p', '--password', action='store', dest='password', required=True, help='specify the password for logging into axops')
    parser.add_argument('--negative-test', action='store_true', dest='negative_test', default=False, help='flag for negative test')
    parser.add_argument('--timeout', action='store', dest='timeout', default=None, type=int, help='specify timeout of test')
    parser.add_argument('--max-chord-length', action='store', dest='max_chord_length', default=None, type=int, help='specify the maximal length of chord')
    parser.add_argument('--max-chord-width', action='store', dest='max_chord_width', default=None, type=int, help='specify the maximal width of chord')
    args = parser.parse_args()

    logging.basicConfig(format='%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s',
                        datefmt='%Y-%m-%dT%H:%M:%S',
                        stream=sys.stdout,
                        level=logging.INFO)
    logging.getLogger('ax.devops.ax_request').setLevel(logging.INFO)

    BasicWorkflowTest.target_cluster = args.target_cluster
    BasicWorkflowTest.username = args.username
    BasicWorkflowTest.password = args.password
    BasicWorkflowTest.timeout = args.timeout or DEFAULT_TIMEOUT
    BasicWorkflowTest.max_chord_length = args.max_chord_length or DEFAULT_MAX_CHORD_LENGTH
    BasicWorkflowTest.max_chord_width = args.max_chord_width or DEFAULT_MAX_CHORD_WIDTH
    BasicWorkflowTest.negative_test = args.negative_test

    sys.argv = sys.argv[:1]
    if args.test_methods:
        sys.argv += list(args.test_methods)

    unittest.main()
