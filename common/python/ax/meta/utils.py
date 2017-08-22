#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import uuid


class AXClusterNameIdParser:
    """
    Parser takes in an input, input can be one of the following format:
        1. <cluster_name>-<cluster_id>
        2. <cluster_name>
    Cluster ID is a UUID for aws clusters, and a 8 character hash for GCP

    Parser returns (cluster_name, cluster_id) tuple
    """
    @staticmethod
    def parse_cluster_name_id_aws(input_name):
        try:
            cid = str(uuid.UUID(input_name[-36:]))
            cname = input_name[:-37]
        except ValueError:
            cid = None
            cname = input_name
        assert cname, "Must provide a non-empty input to parse"
        return cname, cid

    @staticmethod
    def parse_cluster_name_id_gcp(input_name):
        try:
            cid = input_name[-8:]
            cname = input_name[:-9]
        except ValueError:
            cid = None
            cname = input_name
        assert cname, "Must provide a non-empty input to parse"
        return cname, cid
