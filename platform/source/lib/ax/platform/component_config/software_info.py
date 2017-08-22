#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Class to get static software information from environment / config. This class, including
currently ARGO framework assumes that all ARGO software comes from same registry with same
secrets
"""

import os
import yaml
from pprint import pformat


KUBE_INSTALLER_VERSION_FILE = "/kubernetes/cluster/version.txt"
SOFTWARE_INFO_VERSION = 2


class RegistrySecretType:
    DEFAULT_DOCKER_REGISTRY_SECRET_TYPE = "DockerConfigFileBase64"


class SoftwareInfo(object):
    def __init__(self, info_dict=None, info_file=None):
        """
        This class can be initialized by
            1. a dictionary including all information
            2. a path to a yaml file containing all information

        It's not necessary to config it with all information and it's up the provisioner to
        decide what information to give to a particular software.
        :param info_dict:
        :param info_file:
        """
        self._attribute_map = ["ami_name", "ami_id", "registry", "registry_secrets", "image_version",
                               "image_namespace", "kube_version", "kube_installer_version"]

        # AMI namd and AMI ID are referring to node AMI. During installation / upgrade
        # we need AMI name in order to get AMI id from cloud provider, in order to
        # create launch configuration
        self.ami_name = None
        self.ami_id = None

        # Image registry address. Currently we assume all Argo software are pulled
        # from same registry, and docker is the only container run time we support
        self.registry = None

        # Registry secret for the image registry to pull all Argo software. This
        # should be a base64 encoded docker config file.
        self.registry_secrets = None

        # A hook to indicate how to interpret registry secret string. Currently
        # this is not used, but this will be used later if we are supporting more
        # types of registry secrets
        self.registry_secret_type = RegistrySecretType.DEFAULT_DOCKER_REGISTRY_SECRET_TYPE

        # All Argo images should have name format: {registry}/{namespace}/{image_name}:{version}
        # TODO: we might want to make image naming more flexible
        self.image_version = None
        self.image_namespace = None

        # Version string for Kubernetes and Kubernetes installer. Kubernetes installer is some
        # software for install / uninstall Kubernetes cluster
        self.kube_version = None
        self.kube_installer_version = None

        self.reset_software_info(info_dict, info_file)

    def registry_is_private(self):
        return bool(self.registry_secrets)

    def reset_software_info(self, info_dict=None, info_file=None):
        """
        Singleton makes this object has all
        :param info_dict:
        :param info_file:
        :return:
        """
        assert not (info_dict and info_file), "SoftwareInfo: Cannot specify info_dict and info_file together"
        if info_dict:
            self._load_info_from_dict(info_dict)
        elif info_file:
            with open(info_file, "r") as f:
                self._load_info_from_dict(yaml.load(f))
        else:
            self._load_info_from_env()

    def _load_info_from_dict(self, info):
        for attr in self._attribute_map:
            setattr(self, attr, info.get(attr, None))

    def _load_info_from_env(self):
        self.ami_name = os.getenv("AX_AWS_IMAGE_NAME", None)
        self.ami_id = os.getenv("AX_AWS_IMAGE_ID", None)

        self.registry = os.getenv("ARGO_DIST_REGISTRY", None)
        self.registry_secrets = os.getenv("ARGO_DIST_REGISTRY_SECRETS", None)

        self.image_version = os.getenv("AX_VERSION", None)
        self.image_namespace = os.getenv("AX_NAMESPACE", None)

        self.kube_version = os.getenv("AX_KUBE_VERSION", None)
        if os.path.isfile(KUBE_INSTALLER_VERSION_FILE):
            with open(KUBE_INSTALLER_VERSION_FILE, "r") as f:
                self.kube_installer_version = f.read().strip()

    def to_dict(self):
        ret = {
            "version": SOFTWARE_INFO_VERSION,
            "registry_secret_type": self.registry_secret_type
        }
        for attr in self._attribute_map:
            ret[attr] = getattr(self, attr)
        return ret

    def __repr__(self):
        return pformat(self.to_dict())
