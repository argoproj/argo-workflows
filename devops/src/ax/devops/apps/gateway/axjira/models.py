#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
from django.db import models


class AXJira(models.Model):
    """DevOps jira.

    Currently, this model is only used for enabling REST API.
    """
