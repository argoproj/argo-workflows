#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
from axjira.models import AXJira

from rest_framework import serializers


class AXJiraSerializer(serializers.ModelSerializer):
    """Serializer for jira."""

    class Meta:
        model = AXJira
