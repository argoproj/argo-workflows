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
import string
import sys
import time
import unittest
import uuid

from ax.devops.axdb.axops_client import AxopsClient

logger = logging.getLogger('ax.devops.test.nested_workflow')

# In most simple case, a non-nested workflow is embedded in another workflow, in this case, the layer is 2
# As we are testing nested workflow, the minimum is 2; also, minimum is not customizable
DEFAULT_MAX_LAYER = 5
DEFAULT_MIN_LAYER = 2

# The length of individual workflow
# As we need to cover the case of simple group, the minimum is 1; also, minimum is not customizable
DEFAULT_MAX_LENGTH = 5
DEFAULT_MIN_LENGTH = 1

# The width of individual workflow
# As we need to cover the case of simple chain, the minimum is 1; also, minimum is not customizable
DEFAULT_MAX_WIDTH = 5
DEFAULT_MIN_WIDTH = 1

# Default length of names
DEFAULT_NAME_LENGTH = 16

# Timeout is needed as we need to cover the case that a task never returns
DEFAULT_TIMEOUT = 1 * 60 * 60


class NestedWorkflowTest(unittest.TestCase):
    """Nested workflow test."""

    # Fixture configuration
    target_cluster = None
    axops_client = None
    username = None
    password = None

    # Test configuration
    service_template = None
    timeout = DEFAULT_TIMEOUT

    # Scale configuration
    max_layer = DEFAULT_MAX_LAYER
    min_layer = DEFAULT_MIN_LAYER
    max_length = DEFAULT_MAX_LENGTH
    min_length = DEFAULT_MIN_LENGTH
    max_width = DEFAULT_MAX_WIDTH
    min_width = DEFAULT_MIN_WIDTH
    name_length = DEFAULT_NAME_LENGTH

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
        cls.axops_client = AxopsClient(host=hostname, port=port, protocol='https', ssl_verify=False)

        # Create service template to be used in test
        service_template_file = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'service_template.json')
        with open(service_template_file) as f:
            logger.info('Loading service template for test ...')
            cls.service_template = json.load(f)
            logger.info('Successfully loaded service template (%s) for test', cls.service_template['name'])

    def test_nested_workflow(self):
        """Test a randomly generated nested workflow.

        :return:
        """
        logger.info('Creating a service request with a nested workflow ... ')
        service_request = self._create_nested_workflow()
        logger.info('Successfully created a service request (payload: %s)', json.dumps(service_request))
        logger.info('Start executing service ...')
        service = self.axops_client.create_service(service_request)
        result = self._wait_for_result(service['id'], self.timeout)
        self.assertTrue(result, 'Failed to execute service ({})'.format(service['id']))
        logger.info('Successfully executed service (%s)', service['id'])

    def _create_nested_workflow(self):
        """Create a nested workflow.

        :return:
        """
        layer = random.randint(self.min_layer, self.max_layer)
        logger.info('Creating a nested workflow (layer: %s) ...', layer)
        return {'template': self._create_nested_service_template(layer=layer)}

    def _create_nested_service_template(self, layer):
        """Create a nested service template.

        :param layer: If layer is 1, not nesting, otherwise nesting.
        :return:
        """
        workflow_name, step_widths, nested_point = self._generate_random_workflow_skeleton(layer > 1)
        logger.info('Creating workflow from skeleton ...')
        service_template = {
            'id': str(uuid.uuid1()),
            'type': 'service_template',
            'subtype': 'workflow',
            'version': '',
            'name': 'nested_workflow',
            'dns_name': '',
            'description': 'Workflow created for nested workflow test',
            'cost': 0,
            'inputs': {},
            'outputs': {},
        }
        steps = []
        for i in range(len(step_widths)):
            step = {}
            step_width = step_widths[i]
            for j in range(step_width):
                k = '{}.{}.{}'.format(workflow_name, i + 1, j + 1)
                if (i, j) == nested_point:
                    embedded_service_template = self._create_nested_service_template(layer - 1)
                else:
                    embedded_service_template = self._create_leaf_service_template()
                v = {
                    'template': embedded_service_template,
                    'status': 0,
                    'cost': 0,
                    'launch_time': 0,
                    'run_time': 0,
                    'average_runtime': 0
                }
                step[k] = v
            steps.append(step)
        service_template['steps'] = steps
        return service_template

    def _create_leaf_service_template(self):
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

    def _generate_random_workflow_skeleton(self, nested):
        """Generate a random workflow skeleton.

        :param nested:
        :return:
        """
        logger.info('Creating skeleton of a random workflow (nested: %s) ...', nested)
        workflow_name = self._generate_random_name()
        workflow_length = self._generate_random_length()
        step_widths = []
        for _ in range(workflow_length):
            step_width = self._generate_random_width()
            step_widths.append(step_width)
        if nested:
            x = random.randint(0, workflow_length - 1)
            y = random.randint(0, step_widths[x] - 1)
            nested_point = (x, y)
        else:
            nested_point = None
        logger.info('Successfully created workflow skeleton (name: %s, steps: %s, nested_point: %s)', workflow_name, step_widths, nested_point)
        return workflow_name, step_widths, nested_point

    def _generate_random_length(self):
        """Generate random length.

        :return:
        """
        return random.randint(self.min_length, self.max_length)

    def _generate_random_width(self):
        """Generate random width.

        :return:
        """
        return random.randint(self.min_width, self.max_width)

    def _generate_random_name(self):
        """Generate random name.

        :return:
        """
        return ''.join(random.SystemRandom().choice(string.ascii_uppercase + string.digits + string.ascii_lowercase) for _ in range(self.name_length))

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
    parser.add_argument('--max-layer', action='store', dest='max_layer', default=None, type=int, help='specify the maximal layer of nesting')
    parser.add_argument('--max-length', action='store', dest='max_length', default=None, type=int, help='specify the maximal length of workflow')
    parser.add_argument('--max-width', action='store', dest='max_width', default=None, type=int, help='specify the maximal width of workflow')
    args = parser.parse_args()

    logging.basicConfig(format='%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s',
                        datefmt='%Y-%m-%dT%H:%M:%S',
                        stream=sys.stdout,
                        level=logging.INFO)
    logging.getLogger('axdevops.ax_request').setLevel(logging.INFO)

    NestedWorkflowTest.target_cluster = args.target_cluster
    NestedWorkflowTest.username = args.username
    NestedWorkflowTest.password = args.password
    NestedWorkflowTest.timeout = args.timeout or DEFAULT_TIMEOUT
    NestedWorkflowTest.max_layer = args.max_layer or DEFAULT_MAX_LAYER
    NestedWorkflowTest.max_length = args.max_length or DEFAULT_MAX_LENGTH
    NestedWorkflowTest.max_width = args.max_width or DEFAULT_MAX_WIDTH
    NestedWorkflowTest.negative_test = args.negative_test

    sys.argv = sys.argv[:1]
    if args.test_methods:
        sys.argv += list(args.test_methods)

    unittest.main()
