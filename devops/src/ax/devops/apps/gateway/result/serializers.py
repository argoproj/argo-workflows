#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
from result.models import Result

from rest_framework import serializers


class ResultSerializer(serializers.ModelSerializer):
    """Serializer for result."""

    class Meta:
        model = Result
