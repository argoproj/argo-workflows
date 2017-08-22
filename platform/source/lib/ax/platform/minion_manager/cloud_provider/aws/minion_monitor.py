#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

""" Monitors minions and terminates them if not Ready for > 15 minutes. """

from datetime import datetime
import logging
import sys
from threading import Thread
import time

from ax.kubernetes.client import KubernetesApiClient
from ax.util.const import SECONDS_PER_MINUTE

logger = logging.getLogger("aws.minion-manager.minion-monitor")
logging.getLogger('requests').setLevel(logging.WARNING)


class AWSMinionMonitor(object):
    """
    This class monitors the minions in the Kubernetes cluster and terminates
    them if they're not Ready for > 15 minutes.
    """

    def __init__(self, ec2_client):
        self.ec2_client = ec2_client

        self.instances_not_ready = []

        # The minion_monitor thread actually does periodic checks.
        self.minion_monitor = Thread(target=self.minion_monitor_work,
                                 name="PriceReporterAPI")
        self.minion_monitor.setDaemon(True)

    def minion_monitor_work(self):
        """ Main method of the AWSMinionMonitor object. """
        logger.info("Running Kubernetes client liveness checks ...")
        while True:
            try:
                nodes = KubernetesApiClient().api.list_node().items
                for n in nodes or []:
                    for condition in n.status.conditions:
                        if condition.type == "Ready" and condition.status != "True":
                            ts = datetime.strptime(n.metadata.creation_timestamp, "%Y-%m-%dT%H:%M:%SZ")
                            uptime_sec = (datetime.now() - ts).total_seconds()
                            if uptime_sec > 25 * SECONDS_PER_MINUTE:
                                if aws_instances_id in self.instances_not_ready:
                                    # Node is confirmed to *not* be ready. Terminate it.
                                    # Format in spec: aws:///us-west-2b/i-0a606fb4bc9f8bbeb
                                    aws_instances_id = n.spec.provider_id.split("/")[-1]
                                    logger.info("Uptime: %s, Instance: %s, State: %s. Terminating ...", uptime_sec, aws_instances_id, condition.status)
                                    self.ec2_client.terminate_instances(InstanceIds=[aws_instances_id])
                                else:
                                    self.instances_not_ready.append(
                                        aws_instances_id)
                                    # Check the next node.
                                    break
            except Exception as e:
                # Log an error and swallow the exception.
                logger.error("Failed while checking kubernetes nodes: " + str(e))
            finally:
                time.sleep(15 * SECONDS_PER_MINUTE)

    def run(self):
        self.minion_monitor.start()
