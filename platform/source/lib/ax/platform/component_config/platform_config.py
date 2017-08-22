#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import yaml

from ax.util.const import SECONDS_PER_MINUTE


VALID_VERSIONS = ["v1"]


class AXPlatformConfigDefaults:
    # Default platform manifest / config file info
    DefaultManifestRoot = "/ax/config/service/standard/"
    DefaultPlatformConfigFile = "/ax/config/service/config/platform-bootstrap.cfg"

    # Create timeouts
    ObjCreateWaitTimeout = 25 * SECONDS_PER_MINUTE
    ObjCreatePollInterval = 3
    ObjCreatePollMaxRetry = ObjCreateWaitTimeout / ObjCreatePollInterval

    # Create extra poll timeouts
    ObjCreateExtraPollTimeout = 15 * SECONDS_PER_MINUTE
    ObjCreateExtraPollInterval = 3
    ObjCreateExtraPollMaxRetry = ObjCreateExtraPollTimeout / ObjCreateExtraPollInterval

    # Delete timeouts
    ObjDeleteWaitTimeout = 2 * SECONDS_PER_MINUTE
    ObjDeletePollInterval = 3
    ObjDeletePollMaxRetry = ObjDeleteWaitTimeout / ObjDeletePollInterval

    # Jitters
    ObjectOperationJitter = 5


class ObjectGroupPolicy:
    # CreateOnce means it should not be recreated or upgraded. i.e. it can only be
    # created once during the cluster's entire life cycle
    # Example would be volume, cluster ELB, namespace, etc
    CreateOnce = "CreateOnce"

    # CreateMany means it can be created multiple times during cluster's life cycle
    # These object groups will be teared down / brought up again during pause/restart/upgrade
    # Example would be all Argo micro-services
    CreateMany = "CreateMany"


class ObjectGroupPolicyPredicate:
    """
    ObjectGroupPolicy can be attached with a predicate.
    e.g. CreateOnce:PrivateRegistryOnly
    """

    NoPredicate = ""
    PrivateRegistryOnly = "PrivateRegistryOnly"


class ObjectGroupConsistency:
    # CreateIfNotExist means during platform start, we will check if the
    # object is there or not. We only create it if the object does not exist
    # Note that if the object is not healthy, i.e. not all Pods are in "Running"
    # state, we delete and recreate
    CreateIfNotExist = "CreateIfNotExist"


class AXPlatformConfig(object):
    def __init__(self, config_file):
        self._config_file = config_file
        self.version = ""
        self.name = ""
        self.steps = []
        self._load_config()

    def _load_config(self):
        with open(self._config_file, "r") as f:
            config_raw = yaml.load(f.read())

        self.version = config_raw["version"]
        if self.version not in VALID_VERSIONS:
            raise ValueError("Invalid platform config version: {}".format(self.version))
        self.name = config_raw["name"]
        for s in config_raw["spec"].get("steps", []):
            self.steps.append(AXPlatformObjectGroup(s))


class AXPlatformObjectGroup(object):
    def __init__(self, object_group):
        self.name = object_group["name"]
        self.policy = object_group.get("policy", ObjectGroupPolicy.CreateMany)
        self.policy_predicate = object_group.get("policy_predicate", ObjectGroupPolicyPredicate.NoPredicate)
        self.consistency = object_group.get("consistency", ObjectGroupConsistency.CreateIfNotExist)

        if self.policy not in [ObjectGroupPolicy.CreateOnce, ObjectGroupPolicy.CreateMany]:
            raise ValueError("Invalid object group policy: {}.".format(self.policy))

        if self.policy_predicate not in [ObjectGroupPolicyPredicate.NoPredicate, ObjectGroupPolicyPredicate.PrivateRegistryOnly]:
            raise ValueError("Invalid object group policy predicate {}".format(self.policy_predicate))

        if self.consistency not in [ObjectGroupConsistency.CreateIfNotExist]:
            raise ValueError("Invalid object group consistency: {}".format(self.consistency))

        self.object_set = set()
        for o in object_group.get("objects", []):
            self.object_set.add(AXPlatformObject(o))


class AXPlatformObject(object):
    def __init__(self, object_input):
        # name, file are must have fields, namespace is not enforced (we might be creating namespace) itself
        self.name = object_input["name"]
        self.manifest = object_input["file"]
        self.namespace = object_input.get("namespace", None)
