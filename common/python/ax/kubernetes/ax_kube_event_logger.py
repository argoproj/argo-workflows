#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Log Kubernetes events.
"""

import logging
import time
import socket
from retrying import retry
from ax.kubernetes.swagger_client.models.v1_event import V1Event
from ax.kubernetes.swagger_client.models.v1_object_reference import V1ObjectReference
from ax.kubernetes.swagger_client.models.v1_object_meta import V1ObjectMeta
from ax.kubernetes.swagger_client.models.v1_event_source import V1EventSource

logger = logging.getLogger(__name__)

class AXKubeEventLogger(object):
    """
    Creates swagger_client objects that will be used while actually logging
    the event.
    """
    def __init__(self, namespace, client):
        event_metadata = V1ObjectMeta()
        event_metadata.namespace = namespace
        event_metadata.generate_name = socket.gethostname()

        involved_object = V1ObjectReference()
        involved_object.kind = "Pod"
        involved_object.name = socket.gethostname()
        involved_object.namespace = namespace

        event_source = V1EventSource()
        event_source.component = "AX"
        event_source.host = socket.gethostname()

        self.event = V1Event()
        self.event.kind = "Event"
        self.event.api_version = "v1"
        self.event.metadata = event_metadata
        self.event.involved_object = involved_object
        self.event.source = event_source
        self.event.type = "ax-event"
        self.event.count = 1

        self.namespace = namespace
        self.client = client

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
    def log_event(self, reason, message):
        logger.debug("Logging event:: %s: %s", reason, message)
        self.event.first_timestamp = time.strftime("%Y-%d-%mT%I:%M:%S-08:00")
        self.event.last_timestamp = time.strftime("%Y-%d-%mT%I:%M:%S-08:00")
        self.event.reason = reason
        self.event.message = message
        self.client.api.create_namespaced_event(body=self.event, namespace=self.namespace)
