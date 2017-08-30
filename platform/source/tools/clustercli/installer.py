# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import os
import re

from anytree import Node
from ax.cloud.aws import EC2
from .common import CommonPrompts
from prompt_toolkit.history import FileHistory
from ax.platform.cluster_config import AXClusterSize, AXClusterType
from ax.cluster_management.app.options.install_options import ClusterInstallDefaults
from ax.platform.cluster_config import SpotInstanceOption


class InstallPrompts(CommonPrompts):

    INSTALLCMD = "install"
    UNINSTALLCMD = "uninstall"

    def __init__(self, cmd=INSTALLCMD):
        super(InstallPrompts, self).__init__()
        shell_dir = os.path.expanduser("~/.argo/")
        self.history = FileHistory(os.path.join(shell_dir, ".history_install"))

        sizeNode = Node("size",
                        prompt=u'Enter the size of the cluster',
                        values=AXClusterSize.VALID_CLUSTER_SIZES,
                        help=u'Choose a size for the cluster. See argo documentation for what these sizes mean',
                        default=ClusterInstallDefaults.CLUSTER_SIZE,
                        parent=self.get_root())

        typeNode = Node("type",
                        prompt=u'Enter type of node in the cluster',
                        values=AXClusterType.VALID_CLUSTER_TYPES,
                        default=ClusterInstallDefaults.CLUSTER_TYPE,
                        help=u'Type of node in cluster.',
                        parent=sizeNode
                        )

        regionNode = Node("region",
                          prompt=u'Enter the region to install the cluster in',
                          default=self._get_region_from_profile,
                          values=self._get_all_regions,
                          parent=self.get_node("profile")
                          )

        placementNode = Node("placement",
                             prompt=u'Enter the placement in the region',
                             values=self._get_placement,
                             parent=regionNode,
                             help=u'Select the placement in the region, Press tab to see list of possible values'
                            )

        vpcCidrNode = Node("vpccidr",
                           prompt=u'Enter the VPC CIDR base',
                           help=u'Please provide a /16 CIDR base for your new VPC. For example, if you want your VPC CIDR to be \"172.20.0.0/16\", enter 172.20',
                           validator=self._vpc_validator,
                           default=ClusterInstallDefaults.VPC_CIDR_BASE,
                           parent=typeNode
                           )

        subnetMaskNode = Node("subnet_mask",
                              prompt=u'Enter a subnet mask (Cannot be greater than 25)',
                              help=u'An integer value that is in the range [0-25]',
                              validator=self._subnet_validator,
                              default=ClusterInstallDefaults.SUBNET_MASK_SIZE,
                              parent=vpcCidrNode
                              )

        trustedCidrsNode = Node("trusted_cidrs",
                                prompt=u'Enter a list of trusted CIDRs separted by space',
                                default=u' '.join(ClusterInstallDefaults.TRUSTED_CIDR),
                                help=u'E.g. 10.0.0.1/32 10.1.1.0/32',
                                validator=self._trusted_cidr_validator,
                                parent=typeNode
                                )

        spotOption = Node("spot_type",
                          prompt=u'Enter the configuration of spot instances (see toolbar for options)',
                          help=u'all: Every instance is a spot instance (cost effective), none: Every instance is on-demand (stable node), partial: argo services run on on-demand nodes other run on spot (compromise)',
                          default=ClusterInstallDefaults.SPOT_INSTANCE_OPTION,
                          values=SpotInstanceOption.VALID_SPOT_INSTANCE_OPTIONS,
                          parent=typeNode,
                          function=self._should_ask_user_on_demand
                        )

        numberOnDemandNode = Node("num_on_demand",
                                  prompt=u'Number of on-demand nodes for running workflows',
                                  default=unicode(ClusterInstallDefaults.USER_ON_DEMAND_NODE_COUNT),
                                  parent=spotOption,
                                  validator=self._on_demand_validator
                                  )

    def get_argocluster_command(self):
        command = "argocluster install --cluster-name {name} --cloud-provider aws --cloud-profile {profile} --cluster-size {size} --cluster-type {type} " \
                      "--cloud-region {region} --cloud-placement {placement} --vpc-cidr-base {vpc_base} --subnet-mask-size {subnet_mask} --trusted-cidrs {cidrs} " \
                      "--spot-instances-option {spot_option} --user-on-demand-nodes={demand_nodes} --silent".format(
                name=self.get_value("name"),
                profile=self.get_value("profile"),
                size=self.get_value("size"),
                type=self.get_value("type"),
                region=self.get_value("region"),
                placement=self.get_value("placement"),
                vpc_base=self.get_value("vpccidr"),
                subnet_mask=self.get_value("subnet_mask"),
                cidrs=self.get_value("trusted_cidrs"),
                spot_option=self.get_value("spot_type"),
                demand_nodes=self.get_value("num_on_demand", default=0)
            )
        return command

    def get_root(self):
        return self.root

    def get_history(self):
        return self.history

    def get_header(self):
        return u'Interactive Cluster Installation'

    @staticmethod
    def _get_region_from_profile(node):
        profiles = InstallPrompts._get_profiles()
        profile_name = node.parent.value
        region = profiles[profile_name].get("region", None)
        if not region:
            return u''
        return region

    @staticmethod
    def _get_all_regions(node):
        try:
            import boto3
            ec2 = boto3.client("ec2")
            regions = ec2.describe_regions()
            return [x['RegionName'] for x in regions['Regions']]
        except Exception as e:
            print("Could not get Regions due to error {}".format(e))
        return []

    @staticmethod
    def _get_placement(node):
        # node's parent is region and region's parent is profile
        region = node.parent.value
        profile = node.parent.parent.value
        try:
            ec2 = EC2(profile=profile, region=region)
            zones = ec2.get_availability_zones()
            return zones
        except Exception as e:
            print ("Could not get availability zones for profile {}  region {}. Are you sure, that the region exists?".format(profile, region))
        return []

    @staticmethod
    def _vpc_validator(input):
        match = re.match(r"^([0-9]{1,3})\.([0-9]{1,3})$", input)
        if not match:
            raise ValueError("Not a valid CIDR/16")

        if int(match.group(1)) >= 256 or int(match.group(2)) >= 256:
            raise ValueError("CIDR entries need to be in 0-255 range")

    @staticmethod
    def _subnet_validator(input):
        if int(input) < 0 or int(input) > 25:
            raise ValueError("Subnet mask needs to be in range of [0-25]")

    @staticmethod
    def _should_ask_user_on_demand(user_input):
        if user_input == "partial":
            return user_input, True
        else:
            return user_input, False

    @staticmethod
    def _trusted_cidr_validator(input):
        from netaddr import IPAddress
        ret = []
        for cidr in input.split(" "):
            cidr = cidr.strip()
            if not cidr:
                # skip whitespace
                continue

            ip, mask = cidr.split("/")
            if int(mask) < 0 or int(mask) > 32:
                raise ValueError("CIDR {} is not valid as mask {} is not in range [0-32]".format(cidr, mask))

            if ip != "0.0.0.0" or mask != '0':
                ipaddr = IPAddress(ip)
                if ipaddr.is_netmask():
                    raise ValueError("Trusted CIDR {} should not be a net mask".format(ip))
                if ipaddr.is_hostmask():
                    raise ValueError("Trusted CIDR {} should not be a host mask".format(ip))
                if ipaddr.is_reserved():
                    raise ValueError("Trusted CIDR {} should not be in reserved range".format(ip))
                if ipaddr.is_loopback():
                    raise ValueError("Trusted CIDR {} should not be a loop back address".format(ip))

                # Currently we don't support private VPC
                if ipaddr.is_private():
                    raise ValueError("Trusted CIDR {} should not be a private address".format(ip))

            ret.append(cidr)

        return ret

    @staticmethod
    def _on_demand_validator(input):
        val = int(input)
        # TODO: Figure out how to use value of cluster size
        if val < 0 or val > 30:
            raise ValueError("Need to have a value between 0-30")


class UninstallPrompts(CommonPrompts):

    def __init__(self):
        super(UninstallPrompts, self).__init__()
        shell_dir = os.path.expanduser("~/.argo/")
        self.history = FileHistory(os.path.join(shell_dir, ".history_uninstall"))

    def get_history(self):
        return self.history

    def get_header(self):
        return u'Options for deleting your cluster'

    def get_argocluster_command(self):
        command = "argocluster uninstall --cluster-name {name} --cloud-provider aws --cloud-profile {profile} --silent".format(
                    name=self.get_value("name"),
                    profile=self.get_value("profile")
        )
        return command


class PausePrompts(CommonPrompts):

    def __init__(self):
        super(PausePrompts, self).__init__()
        shell_dir = os.path.expanduser("~/.argo/")
        self.history = FileHistory(os.path.join(shell_dir, ".history_pause"))

    def get_history(self):
        return self.history

    def get_header(self):
        return u'Options for pausing your cluster'

    def get_argocluster_command(self):
        command = "argocluster pause --cluster-name {name} --cloud-provider aws --cloud-profile {profile} --silent".format(
                    name=self.get_value("name"),
                    profile=self.get_value("profile")
        )
        return command


class ResumePrompts(CommonPrompts):
    def __init__(self):
        super(ResumePrompts, self).__init__()
        shell_dir = os.path.expanduser("~/.argo/")
        self.history = FileHistory(os.path.join(shell_dir, ".history_resume"))

    def get_history(self):
        return self.history

    def get_header(self):
        return u'Options for resuming your cluster'

    def get_argocluster_command(self):
        command = "argocluster resume --cluster-name {name} --cloud-provider aws --cloud-profile {profile} --silent".format(
            name=self.get_value("name"),
            profile=self.get_value("profile")
        )
        return command
