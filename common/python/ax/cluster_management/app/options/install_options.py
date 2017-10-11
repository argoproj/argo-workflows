# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import argparse
import logging
import os
import random
import re
from netaddr import IPAddress

from ax.cloud import Cloud
from ax.cloud.aws import EC2
from ax.platform.cluster_config import AXClusterSize, AXClusterType, SpotInstanceOption
from ax.platform.component_config import SoftwareInfo
from .common import add_common_flags, add_software_info_flags, validate_software_info, \
    ClusterOperationDefaults, ClusterManagementOperationConfigBase, typed_raw_input_with_default, AWS_NO_PROFILE


logger = logging.getLogger(__name__)


class ClusterInstallDefaults:
    CLUSTER_NAME = "argo-cluster"
    CLUSTER_SIZE = "small"
    CLUSTER_TYPE = "standard"
    CLOUD_REGION = "us-west-2"
    CLOUD_PROFILE = AWS_NO_PROFILE
    CLOUD_PLACEMENT = "us-west-2a"
    VPC_CIDR_BASE = "172.20"
    SUBNET_MASK_SIZE = 22
    TRUSTED_CIDR = ["0.0.0.0/0"]
    USER_ON_DEMAND_NODE_COUNT = 0
    SPOT_INSTANCE_OPTION = "partial"
    CLUSTER_AUTO_SCALING_SCAN_INTERVAL = 10


class ClusterInstallConfig(ClusterManagementOperationConfigBase):
    CLUSTER_NAME_REGEX = "[A-Za-z0-9]([-A-Za-z0-9_]*)?[A-Za-z0-9]$"
    VPC_CIDR_BASE_REGEX = "(^172\.1[6-9]$)|(^172\.2[0-9]$)|(^172\.3[0-1]$)"

    # We are using <name>-<id>-master as IAM role name, which is
    # limited to 64 characters. Max name length is 64 - 36 - 8 = 20
    MIN_CLUSTER_NAME_LENGTH = 2
    MAX_CLUSTER_NAME_LENGTH = 20
    MAX_SUBNET_MASK_SIZE = 25

    def __init__(self, cfg):

        super(ClusterInstallConfig, self).__init__(cfg)

        self.cluster_size = cfg.cluster_size
        self.cluster_type = cfg.cluster_type

        self.cloud_region = cfg.cloud_region
        self.cloud_placement = cfg.cloud_placement

        self.vpc_id = cfg.vpc_id
        self.vpc_cidr_base = cfg.vpc_cidr_base
        self.subnet_mask_size = cfg.subnet_mask_size
        self.trusted_cidrs = cfg.trusted_cidrs

        self.user_on_demand_nodes = cfg.user_on_demand_nodes
        self.spot_instances_option = cfg.spot_instances_option

        self.enable_sandbox = cfg.enable_sandbox
        self.manifest_root = cfg.service_manifest_root
        self.bootstrap_config = cfg.platform_bootstrap_config
        self.autoscaling_interval = cfg.cluster_autoscaling_scan_interval
        self.support_object_store_name = cfg.support_object_store_name

        if cfg.software_version_info:
            # Read software info from config file
            self.software_info = SoftwareInfo(info_file=cfg.software_version_info)
        else:
            # Read software info from envs
            self.software_info = SoftwareInfo()

    def default_or_wizard(self):
        """
        User can specify a basic set of configurations using an interactive mode. Several more advanced
        options such as sandbox, manifest / software config, auto scaling scan interval, object store, etc
        are not exposed through interactive mode.
        :return:
        """
        if self.silent:
            self.cluster_size = ClusterInstallDefaults.CLUSTER_SIZE if not self.cluster_size else self.cluster_size
            self.cluster_type = ClusterInstallDefaults.CLUSTER_TYPE if not self.cluster_type else self.cluster_type
            self.cloud_profile = ClusterInstallDefaults.CLOUD_PROFILE if not self.cloud_profile else self.cloud_profile
            self.cloud_region = ClusterInstallDefaults.CLOUD_REGION if not self.cloud_region else self.cloud_region
            self.vpc_cidr_base = ClusterInstallDefaults.VPC_CIDR_BASE if not self.vpc_cidr_base else self.vpc_cidr_base
            self.subnet_mask_size = ClusterInstallDefaults.SUBNET_MASK_SIZE if not self.subnet_mask_size else self.subnet_mask_size
            self.trusted_cidrs = ClusterInstallDefaults.TRUSTED_CIDR if not self.trusted_cidrs else self.trusted_cidrs
            self.spot_instances_option = ClusterInstallDefaults.SPOT_INSTANCE_OPTION if not self.spot_instances_option else self.spot_instances_option
            self.user_on_demand_nodes = ClusterInstallDefaults.USER_ON_DEMAND_NODE_COUNT if not self.user_on_demand_nodes else self.user_on_demand_nodes
        else:

            print("\n====== Argo Cluster Installation Configuration Wizard ======\n")
            if self.cluster_name is None:
                self.cluster_name = typed_raw_input_with_default(
                    prompt="Please enter a cluster name (cluster name should be between 2 and 20 characters, letters and dash only.)",
                    default=ClusterInstallDefaults.CLUSTER_NAME,
                    type_converter=str
                )

            if self.cluster_size is None:
                self.cluster_size = typed_raw_input_with_default(
                    prompt="Please choose a cluster size from \"small\", \"medium\", \"large\" and \"xlarge\"",
                    default=ClusterInstallDefaults.CLUSTER_SIZE,
                    type_converter=str
                )

            if self.cluster_type is None:
                self.cluster_type = typed_raw_input_with_default(
                    prompt="Please choose a cluster type from \"standard\" and \"compute\"",
                    default=ClusterInstallDefaults.CLUSTER_TYPE,
                    type_converter=str
                )

            if self.cloud_profile is None:
                self.cloud_profile = typed_raw_input_with_default(
                    prompt="Please enter your cloud provider profile. If you don't provide one, we are going to use the default you configured on host.",
                    default=AWS_NO_PROFILE,
                    type_converter=str
                )

            if self.cloud_region is None:
                self.cloud_region = typed_raw_input_with_default(
                    prompt="Please enter a cloud region",
                    default=ClusterInstallDefaults.CLOUD_REGION,
                    type_converter=str
                )

            if self.cloud_placement is None:
                self.cloud_placement = typed_raw_input_with_default(
                    prompt="Please enter a cloud placment (zone). If you don't provide one, we are going to randomly pick one from the region you specified",
                    default="",
                    type_converter=str
                )

            if self.vpc_cidr_base is None:
                self.vpc_cidr_base = typed_raw_input_with_default(
                    prompt="Please provide a /16 CIDR base for your new VPC. For example, if you want your VPC CIDR to be \"172.20.0.0/16\", enter 172.20",
                    default=ClusterInstallDefaults.VPC_CIDR_BASE,
                    type_converter=str
                )

            if self.subnet_mask_size is None:
                self.subnet_mask_size = typed_raw_input_with_default(
                    prompt="Please provide a subnet mask size for the subnet that runs your Argo cluster. Subnet mask size cannot be larger than 25",
                    default=ClusterInstallDefaults.SUBNET_MASK_SIZE,
                    type_converter=int
                )

            if self.trusted_cidrs is None:
                self.trusted_cidrs = typed_raw_input_with_default(
                    prompt="Please provide a list of CIDRs separated by space to be allowed to access the cluster",
                    default="0.0.0.0/0",
                    type_converter=str
                ).split()

            if self.spot_instances_option is None:
                self.spot_instances_option = typed_raw_input_with_default(
                    prompt="Please provide a spot instance option. You can choose from \"none\", \"partial\" and \"all\"",
                    default=ClusterInstallDefaults.SPOT_INSTANCE_OPTION,
                    type_converter=str
                )

            if self.spot_instances_option == SpotInstanceOption.PARTIAL_SPOT:
                if self.user_on_demand_nodes is None:
                    self.user_on_demand_nodes = typed_raw_input_with_default(
                        prompt="You are using spot instance. Please specify the minimum number of nodes for running user jobs that you want to keep on-demand",
                        default=ClusterInstallDefaults.USER_ON_DEMAND_NODE_COUNT,
                        type_converter=int
                    )
            else:
                # Setting on-demand to default (0) with either non-spot or all spot
                # For non-spot, we are not using this at all. For all spot, it should be 0
                self.user_on_demand_nodes = ClusterInstallDefaults.USER_ON_DEMAND_NODE_COUNT

            confirmation = "\n\nYou have configured your cluster with the following parameters for installation:\n"
            confirmation += "Cluster Name:          {}\n".format(self.cluster_name)
            confirmation += "Cluster Size:          {}\n".format(self.cluster_size)
            confirmation += "Cluster Type:          {}\n".format(self.cluster_type)
            confirmation += "Cloud Provider:        {}\n".format(self.cloud_provider)
            confirmation += "Cloud Profile:         {}\n".format(self.cloud_profile)
            confirmation += "Cloud Region:          {}\n".format(self.cloud_region)
            confirmation += "Cloud Placement:       {}\n".format(self.cloud_placement if self.cloud_placement else "<Pick Randomly>")
            confirmation += "VPC CIDR Base:         {}\n".format(self.vpc_cidr_base)
            confirmation += "Subnet Mask Size:      {}\n".format(self.subnet_mask_size)
            confirmation += "Trusted CIDRs:         {}\n".format(self.trusted_cidrs)
            confirmation += "Spot Instance Option:  {}\n".format(self.spot_instances_option)
            confirmation += "User On-Demand Nodes:  {}\n".format(self.user_on_demand_nodes)
            confirmation += "\n\nPlease press ENTER to continue or press Ctrl-C to terminate the program if these configurations are not what you want:"
            raw_input(confirmation)

        # TODO: revise this once we bring GCP into picture
        if self.cloud_profile == AWS_NO_PROFILE:
            self.cloud_profile = None



    def validate(self):
        all_errs = []

        all_errs += self._validate_critical_directories()
        all_errs += self._validate_cluster_meta_info()
        all_errs += self._validate_and_set_cloud_provider()
        all_errs += self._validate_network_information()
        all_errs += self._validate_software_configurations()
        all_errs += validate_software_info(self.software_info)

        return all_errs

    def _validate_cluster_meta_info(self):
        all_errs = []

        if not self.cluster_name:
            all_errs.append("Please provide cluster name")
        else:
            if not self.MIN_CLUSTER_NAME_LENGTH <= len(self.cluster_name) <= self.MAX_CLUSTER_NAME_LENGTH:
                all_errs.append("Invalid cluster name: {}. Cluster name length should between {} and {}".format(
                    self.cluster_name, self.MIN_CLUSTER_NAME_LENGTH, self.MAX_CLUSTER_NAME_LENGTH
                ))

            name_validator = re.compile(self.CLUSTER_NAME_REGEX)
            if not name_validator.match(self.cluster_name):
                all_errs.append("Invalid cluster name format: {}. Cluster name should match regex {}".format(
                    self.cluster_name, self.CLUSTER_NAME_REGEX
                ))

        if self.cluster_size not in AXClusterSize.VALID_CLUSTER_SIZES:
            all_errs.append("Invalid cluster size: {}. Please choose from {}".format(
                self.cluster_size,
                AXClusterSize.VALID_CLUSTER_SIZES)
            )

        if self.cluster_type not in AXClusterType.VALID_CLUSTER_TYPES:
            all_errs.append("Invalid cluster type: {}. Please choose from {}".format(
                self.cluster_type,
                AXClusterType.VALID_CLUSTER_TYPES)
            )

        return all_errs

    def _validate_and_set_cloud_provider(self):
        all_errs = []
        if self.cloud_provider not in Cloud.VALID_TARGET_CLOUD_INPUT:
            all_errs.append("Cloud provider {} not supported. Please choose from {}".format(
                self.cloud_provider, Cloud.VALID_TARGET_CLOUD_INPUT
            ))
        else:
            try:
                # Validate placement only for AWS
                c = Cloud(target_cloud=self.cloud_provider)
                if c.target_cloud_aws():
                    ec2 = EC2(profile=self.cloud_profile, region=self.cloud_region)
                    zones = ec2.get_availability_zones()
                    if self.cloud_placement:
                        if self.cloud_placement not in zones:
                            all_errs.append("Invalid cloud placement {}. Please choose from {}".format(
                                self.cloud_placement, zones
                            ))
                    else:
                        self.cloud_placement = random.choice(zones)
                        logger.info("Cloud placement not provided, setting it to %s from currently available zones %s", self.cloud_placement, zones)

            except Exception as e:
                all_errs.append("Cloud provider validation error: {}".format(e))
        return all_errs

    def _validate_network_information(self):
        all_errs = []

        if self.subnet_mask_size > self.MAX_SUBNET_MASK_SIZE:
            all_errs.append("Subnet size too small. Subnet mask should <= {}, current value: {}".format(
                self.MAX_SUBNET_MASK_SIZE, self.subnet_mask_size
            ))

        cidr_base_validator = re.compile(self.VPC_CIDR_BASE_REGEX)
        if self.vpc_cidr_base and not cidr_base_validator.match(self.vpc_cidr_base):
            all_errs.append("Invalid VPC CIDR base {}. VPC CIDR base should match regex {}".format(
                self.vpc_cidr_base, self.VPC_CIDR_BASE_REGEX
            ))

        if self.trusted_cidrs:
            try:
                for cidr in self.trusted_cidrs:
                    [ip, mask] = cidr.split("/")
                    if ip == "0.0.0.0":
                        if mask != "0":
                            all_errs.append("Trusting traffic from everywhere should specify \"0.0.0.0/0\" as trusted CIDR.")
                    else:
                        if not 0 < int(mask) <= 32:
                            all_errs.append("Subnet mask {} should be greater than 0 but less than 32.".format(mask))
                        ipaddr = IPAddress(ip)
                        if ipaddr.is_netmask():
                            all_errs.append("Trusted CIDR {} should not be a net mask".format(ip))
                        if ipaddr.is_hostmask():
                            all_errs.append("Trusted CIDR {} should not be a host mask".format(ip))
                        if ipaddr.is_reserved():
                            all_errs.append("Trusted CIDR {} should not be in reserved range".format(ip))
                        if ipaddr.is_loopback():
                            all_errs.append("Trusted CIDR {} should not be a loop back address".format(ip))

                        # Currently we don't support private VPC
                        if ipaddr.is_private():
                            all_errs.append("Trusted CIDR {} should not be a private address".format(ip))
            except ValueError as ve:
                all_errs.append("Cannot parse trusted CIDRs ({}). Err: {}".format(self.trusted_cidrs, ve))
        else:
            all_errs.append("Please provide trusted CIDRs through --trusted-cidrs flag")

        return all_errs

    def _validate_software_configurations(self):
        all_errs = []

        if self.spot_instances_option not in SpotInstanceOption.VALID_SPOT_INSTANCE_OPTIONS:
            all_errs.append("Invalid spot instance option: {}. Please choose from {}".format(
                self.spot_instances_option, SpotInstanceOption.VALID_SPOT_INSTANCE_OPTIONS
            ))

        if not os.path.isdir(self.manifest_root):
            all_errs.append("Manifest root {} is not a valid directory".format(self.manifest_root))

        if not os.path.isfile(self.bootstrap_config):
            all_errs.append("Bootstrap config {} is not a valid file".format(self.bootstrap_config))

        if self.autoscaling_interval <= 0:
            all_errs.append("Autoscaling interval should be greater than 0. Currently {}".format(
                self.autoscaling_interval
            ))

        return all_errs


class PlatformOnlyInstallConfig(ClusterManagementOperationConfigBase):
    def __init__(self, cfg):
        cfg.cluster_size = AXClusterSize.CLUSTER_USER_PROVIDED
        cfg.cloud_profile = "default"
        cfg.cluster_type = "standard"
        cfg.vpc_id = None
        cfg.vpc_cidr_base = None
        cfg.subnet_mask_size = None
        cfg.trusted_cidrs = ClusterInstallDefaults.TRUSTED_CIDR
        cfg.user_on_demand_nodes = None
        cfg.spot_instances_option = "none"
        cfg.cluster_autoscaling_scan_interval = None
        cfg.support_object_store_name = ""
        cfg.enable_sandbox = None
        cfg.software_version_info = None

        self.cluster_size = cfg.cluster_size
        if cfg.cloud_provider == "minikube":
            self.service_manifest_root = "/ax/config/service/argo-wfe"
            self.platform_bootstrap_config = "/ax/config/service/config/argo-wfe-platform-bootstrap.cfg"
            Cloud(target_cloud="aws")
        else:
            self.service_manifest_root = "/ax/config/service/argo-all"
            self.platform_bootstrap_config = "/ax/config/service/config/argo-all-platform-bootstrap.cfg"

        super(PlatformOnlyInstallConfig, self).__init__(cfg)
        self.install_config = ClusterInstallConfig(cfg=cfg)
        self.install_config.validate()

        self.cluster_bucket = cfg.cluster_bucket
        self.kube_config = cfg.kubeconfig
        try:
            self.bucket_endpoint = cfg.endpoint
            self.access_key = cfg.access_key
            self.secret_key = cfg.secret_key
        except Exception as ae:
            self.bucket_endpoint = None
            self.access_key = None
            self.secret_key = None

        # Overwrite the manifest_root and bootstrap_config.
        self.install_config.manifest_root = self.service_manifest_root
        self.install_config.bootstrap_config = self.platform_bootstrap_config

        return

    def validate(self):
        return None

    def get_cluster_bucket(self):
        return self.cluster_bucket

    def get_install_config(self):
        return self.install_config

def add_install_flags(parser):
    assert isinstance(parser, argparse.ArgumentParser)

    add_common_flags(parser)

    # Cluster meta config
    parser.add_argument("--cluster-size", default=None, help="Pre-canned cluster types: mvc, small, medium, large, xlarge")
    parser.add_argument("--cluster-type", default=None, help="Pre-canned cluster types: standard or compute")

    # Cloud information
    parser.add_argument("--cloud-region", default=None, help="A valid cloud region")
    parser.add_argument("--cloud-placement", default=None, help="A valid cloud placement")

    # Cloud network config
    parser.add_argument("--vpc-id", default=None, help="Specify a valid VPC id if cluster is to be created in existing VPC. If not specified, a new VPC will be created")
    parser.add_argument("--vpc-cidr-base", default=None, help="A /16 vpc cidr block prefix for new VPC. i.e. if you want vpc cidr to be 192.168.0.0/16, enter \"192.168\"")
    parser.add_argument("--subnet-mask-size", default=None, type=int, help="Subnet size, must be smaller than 25")
    parser.add_argument("--trusted-cidrs", default=None, nargs="+", help="A list of IP cidrs allowed to access cluster.")

    # Node Config
    parser.add_argument("--user-on-demand-nodes", default=None, type=int, help="Number of on-demand nodes to use for user autoscaling group")
    parser.add_argument("--spot-instances-option", default=None, help="Spot instance option: choose from none, partial, and all")

    # Software Config
    add_software_info_flags(parser)
    parser.add_argument("--cluster-autoscaling-scan-interval", default=ClusterInstallDefaults.CLUSTER_AUTO_SCALING_SCAN_INTERVAL, type=int, help="Cluster will scan for autoscaling every given seconds")

    # Support config
    parser.add_argument("--support-object-store-name", default="", help="Object store name (bucket name) for support logs")

    # TODO: consider removing --enable-sandbox due to open source
    parser.add_argument("--enable-sandbox", default=False, action="store_true", help="Install this cluster as a sandbox")

def add_platform_only_flags(parser):
    assert isinstance(parser, argparse.ArgumentParser)

    add_common_flags(parser)
    add_software_info_flags(parser)

    # Cloud information
    parser.add_argument("--cloud-region", default=None, help="A valid cloud region")
    parser.add_argument("--cloud-placement", default=None, help="A valid cloud placement")

    # Add bucket
    parser.add_argument("--cluster-bucket", default=None, required=True, help="S3 complaint bucket to use")
    parser.add_argument("--bucket-endpoint", default=None, help="HTTP Endpoint for the cluster-bucket")
    parser.add_argument("--access-key", default=None, help="Access key for accessing the bucket")
    parser.add_argument("--secret-key", default=None, help="Secret key for accessing the bucket")

    # Add kubeconfig
    parser.add_argument("--kubeconfig", default=None, required=True, help="Kubeconfig file for the cluster")
