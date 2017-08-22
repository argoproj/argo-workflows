#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
from ax.devops.kafka.kafka_client import EventNotificationClient
from ax.notification_center import FACILITY_GATEWAY

event_notification_client = EventNotificationClient(FACILITY_GATEWAY)
