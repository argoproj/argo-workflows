# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import abc
import argparse
import os
from future.utils import with_metaclass

from ax.platform.component_config import AXPlatformConfigDefaults, SoftwareInfo

# We should set aws profile to None if user does not provide one
# because in python None type is different from str, we convert None
# to string first and then finally it to None
AWS_NO_PROFILE = "None"

CLOUD_PROVIDER_CHOICES = ["aws", "minikube", "gke"]

def typed_raw_input_with_default(prompt, default, type_converter):
    real_prompt = prompt + " (Default: {}): ".format(default)
    ret = raw_input(real_prompt).lstrip().rstrip() or default
    return type_converter(ret)


class ClusterOperationDefaults:
    CLOUD_PROVIDER = "aws"
    CLOUD_PROFILE = AWS_NO_PROFILE
    PLATFORM_SERVICE_MANIFEST_ROOT = AXPlatformConfigDefaults.DefaultManifestRoot
    PLATFORM_BOOTSTRAP_CONFIG_FILE = AXPlatformConfigDefaults.DefaultPlatformConfigFile


class ClusterManagementOperationConfigBase(with_metaclass(abc.ABCMeta, object)):
    def __init__(self, cfg):
        self.cluster_name = cfg.cluster_name
        self.cluster_id = cfg.cluster_id
        self.cloud_provider = cfg.cloud_provider
        self.cloud_profile = cfg.cloud_profile
        self.silent = cfg.silent
        self.dry_run = cfg.dry_run

    @abc.abstractmethod
    def validate(self):
        pass

    @staticmethod
    def _validate_critical_directories():
        home = os.getenv("HOME")
        all_errs = []
        if not os.path.isdir(os.path.join(home, ".ssh")):
            all_errs.append("Missing ssh directory \"{}\". Is your cluster manager set up properly?".format(os.path.join(home, ".ssh")))

        if not os.path.isdir(os.path.join(home, ".argo")):
            all_errs.append("Missing argo config directory \"{}\". Is your cluster manager set up properly?".format(os.path.join(home, ".argo")))

        return all_errs

    def default_or_wizard(self):
        """
        Fill in configurations using default values or using interactive wizard, based on whether user specify
        the "--silent" or "-s" flag
        :return:
        """

        if self.silent:
            self.cloud_profile = ClusterOperationDefaults.CLOUD_PROFILE if not self.cloud_profile else self.cloud_profile
        else:
            print("\n====== Argo Cluster Operation Configuration Wizard ======\n")
            if self.cluster_name is None:
                self.cluster_name = typed_raw_input_with_default(
                    prompt="Please enter cluster name",
                    default="",
                    type_converter=str
                )

            if self.cloud_profile is None:
                self.cloud_profile = typed_raw_input_with_default(
                    prompt="Please enter your cloud provider profile. If you don't provide one, we are going to use the default you configured on host.",
                    default=AWS_NO_PROFILE,
                    type_converter=str
                )

            confirmation = "\n\nYou are using the following parameters for cluster operation:\n"
            confirmation += "Cluster Name:          {}\n".format(self.cluster_name)
            confirmation += "Cloud Provider:        {}\n".format(self.cloud_provider)
            confirmation += "Cloud Profile:         {}\n".format(self.cloud_profile)
            confirmation += "\n\nPlease press ENTER to continue or press Ctrl-C to terminate:"
            raw_input(confirmation)

        # TODO: revise this once we bring GCP into picture
        if self.cloud_profile == AWS_NO_PROFILE:
            self.cloud_profile = None


def validate_software_info(software_info):
    assert isinstance(software_info, SoftwareInfo)
    all_errs = []
    if not software_info.ami_name:
        all_errs.append("Missing AMI name. Please specify it though env AX_AWS_IMAGE_NAME")

    if not software_info.image_namespace:
        all_errs.append("Missing image namespace. Please specify it though env AX_NAMESPACE")

    if not software_info.image_version:
        all_errs.append("Missing image version. Please specify it though env AX_VERSION")

    if not software_info.registry:
        all_errs.append("Missing registry information. Please specify it through env ARGO_DIST_REGISTRY")

    if not software_info.kube_version:
        all_errs.append("Missing Kubernetes version info. Please specify it though env AX_KUBE_VERSION")

    if not software_info.kube_installer_version:
        all_errs.append("Missing Kube installer version. Please make sure you are in cluster manager container")
    return all_errs


def add_common_flags(parser):
    assert isinstance(parser, argparse.ArgumentParser)
    parser.add_argument("--cluster-name", default=None, help="Target Argo cluster name")
    parser.add_argument("--cluster-id", default=None, help="A pre-generated cluster id")
    parser.add_argument("--cloud-provider", default=ClusterOperationDefaults.CLOUD_PROVIDER, help="Cloud type: "+str(CLOUD_PROVIDER_CHOICES), choices=CLOUD_PROVIDER_CHOICES)
    parser.add_argument("--cloud-profile", default=None, help="Cloud profile name (e.g. aws profile)")
    parser.add_argument("--dry-run", default=False, action="store_true", help="Dry run operation")
    parser.add_argument("--silent", "-s", default=False, action="store_true", help="Perform cluster management operation using silent mode (automatically fill in defaults)")


def add_software_info_flags(parser):
    assert isinstance(parser, argparse.ArgumentParser)
    parser.add_argument("--service-manifest-root", default=ClusterOperationDefaults.PLATFORM_SERVICE_MANIFEST_ROOT, help="Root directory for all Argo service manifests")
    parser.add_argument("--platform-bootstrap-config", default=ClusterOperationDefaults.PLATFORM_BOOTSTRAP_CONFIG_FILE, help="Config file indicating how platform should be booted up")

    # TODO: Not used today as software version info still depends on axclustermanager
    parser.add_argument("--software-version-info", default=None, help="A file indicating software configuration. Currently NOT used.")
