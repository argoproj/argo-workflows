#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from kubernetes import client
from kubernetes.client.rest import ApiException


class Namespace(object):
    def __init__(self, name, api=None):
        self._name = name
        if api:
            self._api = api
        else:
            self._api = client.CoreV1Api()

    def create(self):
        ns = client.V1Namespace()
        ns.api_version = "v1"
        ns.kind = "Namespace"
        ns.metadata = client.V1ObjectMeta()
        ns.metadata.name = self._name
        try:
            self._api.create_namespace(ns)
        except ApiException as e:
            if e.status != 409 or "AlreadyExists" not in e.body:
                raise

    def delete(self):
        try:
            self._api.delete_namespace(self._name, client.V1DeleteOptions())
        except ApiException as e:
            if e.status != 404 or "NotFound" not in e.body:
                raise
