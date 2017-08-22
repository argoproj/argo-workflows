#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
from django.db import models


class Result(models.Model):
    """DevOps result.

    Currently, this model is only used for enabling REST API.
    """
