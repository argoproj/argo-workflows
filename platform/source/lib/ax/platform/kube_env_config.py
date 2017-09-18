#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Module for kube-up interface.
"""

import logging
import os
import StringIO
from parse import parse

from ax.cloud.aws import AWS_DEFAULT_PROFILE
from ax.meta import AXClusterConfigPath, AXCustomerId

logger = logging.getLogger(__name__)


default_kube_up_env = {
    "KUBE_ENABLE_NODE_LOGGING": "false",
    "KUBE_ENABLE_CLUSTER_LOGGING": "false",
    "KUBE_ENABLE_CLUSTER_MONITORING": "false",
    "ENABLE_RESCHEDULER": "true",
    "ETCD_SNAPSHOT_COUNT": "1000",
    "AUTO_UPGRADE": "false",
    "NON_MASQUERADE_CIDR": "192.168.0.0/16",
    "SERVICE_CLUSTER_IP_RANGE": "192.168.0.0/18",
    "DNS_SERVER_IP": "192.168.0.10",
    "MASTER_IP_RANGE": "192.168.64.0/24",
    "CLUSTER_IP_RANGE": "192.168.128.0/18",
    "VPC_NAME": "ax-vpc",
    "DOCKER_STORAGE": "overlay2",
    "REGISTER_MASTER_KUBELET": "true",
    "AX_ENABLE_MASTER_FLUENTD": "false",
    "SERVER_BINARY_TAR_URL": "https://storage.googleapis.com/kubernetes-release/release/v{version}/kubernetes-server-linux-amd64.tar.gz".format(version=os.getenv("KUBERNETES_VERSION")),
    "SERVER_BINARY_TAR_HASH": os.getenv("KUBERNETES_SERVER_HASH"),
}


added_kube_up_envs = {
    "MIN_CONTAINER_GC_TTL": "180s",
    "MAX_DEAD_CONTAINERS": "",
    "MAX_DEAD_CONTAINERS_PER_CONTAINER": "",
    "DOCKER_LOG_OPTS": "--log-driver json-file --log-opt max-size=10m --log-opt max-file=5",
    "ENABLE_DOCKER_DEBUG": "true"
}


def kube_env_update(input_string, updates):
    """
    Give an input user data string, replace everything we know about.
    This includes some part of bootstrap script for new kubernetes version and hashes,
    and kube_env definitions.

    Sample format from kube_env:
        ENV_TIMESTAMP: '2017-02-14T03:24:31+0000'
        INSTANCE_PREFIX: 'dev-008ab2b4-f265-11e6-a5d8-02127c5241cd'
        NODE_INSTANCE_PREFIX: 'dev-008ab2b4-f265-11e6-a5d8-02127c5241cd-minion'
        NODE_TAGS: ''
    """
    kube_version = updates.get("new_kube_version")
    cluster_install_version = updates.get("new_cluster_install_version")
    server_binary_tar_hash = updates.get("new_kube_server_hash")
    salt_tar_hash = updates.get("new_kube_salt_hash")
    api_servers = updates.get("new_api_servers")

    # When the master-manager is started as part of a newly deployed cluster, the SERVER_BINARY_TAR_HASH
    # and SALT_TAR_HASH are empty. The user-data stored in s3 already has all the correct info.
    if not server_binary_tar_hash and not salt_tar_hash:
        logger.info("Server binary hash and/or salt tar hash are not set. No user-data fixup.")
        return input_string

    ax_vol_installer_present = False
    output = StringIO.StringIO()
    buf = StringIO.StringIO(input_string)
    for line in buf.readlines():
        if "SERVER_BINARY_TAR_URL: '" in line:
            line = "SERVER_BINARY_TAR_URL: 'https://storage.googleapis.com/kubernetes-release/release/v{kube_version}/kubernetes-server-linux-amd64.tar.gz'\n".format(kube_version=kube_version)
        elif "SALT_TAR_URL" in line:
            format = 'SALT_TAR_URL: \'https://{aws_s3_prefix}.amazonaws.com/applatix-cluster-{bucket_suffix}/kubernetes-staging/{ax_version}/kubernetes-salt.tar.gz\''
            result = parse(format, line)
            assert result is not None, "Failed to parse SALT_TAR_URL"
            line = "SALT_TAR_URL: 'https://" + result['aws_s3_prefix'] + ".amazonaws.com/applatix-cluster-" + result[
                'bucket_suffix'] + "/kubernetes-staging/v" + kube_version + "/installer/" + cluster_install_version + "/kubernetes-salt.tar.gz'\n"
        elif "wget" in line and "bootstrap" in line:
            # Update bootstrap script download version. Use strict parse to make sure any deviation from standard format would fail.
            line = line.strip()
            format = "wget {options} https://{aws_s3_prefix}.amazonaws.com/applatix-cluster-{bucket_suffix}/kubernetes-staging/{kube_version}/installer/{install_version}/bootstrap-script{trailing}"
            result = parse(format, line)
            assert result is not None, "Failed to parse wget command, line is [{}]".format(line)
            # Reuse most other compoenents but change kube_version and install_version.
            line = format.format(options=result["options"],
                                 aws_s3_prefix=result["aws_s3_prefix"],
                                 bucket_suffix=result["bucket_suffix"],
                                 kube_version="v" + kube_version,
                                 install_version=cluster_install_version,
                                 trailing=result["trailing"])
            line = "  " + line + "\n"
        elif "SERVER_BINARY_TAR_HASH" in line:
            line = "SERVER_BINARY_TAR_HASH: '" + server_binary_tar_hash + "'\n"
        elif "SALT_TAR_HASH" in line:
            line = "SALT_TAR_HASH: '" + salt_tar_hash + "'\n"
        elif "KUBELET_APISERVER" in line and api_servers:
            line = "KUBELET_APISERVER: '" + api_servers +"'\n"
        elif "API_SERVERS" in line and api_servers:
            line = "API_SERVERS: '" + api_servers +"'\n"
        elif "ax_vol_plugin.tar.gz" in line:
            ax_vol_installer_present = True
        elif line.startswith("__EOF_KUBE_ENV_YAML"):
            line = _add_env_from_dict(added_kube_up_envs) + line
        elif line.startswith("__EOF_MASTER_KUBE_ENV_YAML"):
            line = _add_env_from_dict(added_kube_up_envs) + line

        env_name = line.split(":")[0]
        if env_name in added_kube_up_envs:
            added_kube_up_envs.pop(env_name)

        for e in default_kube_up_env:
            if line.startswith(e + ": '"):
                line = e + ": '" + default_kube_up_env[e] + "'\n"
        output.write(line)

    if not ax_vol_installer_present:
        ax_vol_installer_script = "\n" + \
        "AX_VOL_LOCAL_PATH=/tmp/ax_vol_plugin.tar.gz\n" + \
        "AX_VOL_INSTALLER_VERSION=${AX_VOL_INSTALLER_VERSION:-1.1.0}\n" + \
        "wget https://s3-us-west-1.amazonaws.com/ax-public/ax_vol_plugin/installer/${AX_VOL_INSTALLER_VERSION}/ax_vol_plugin.tar.gz -O ${AX_VOL_LOCAL_PATH}\n" + \
        "tar -zxvf ${AX_VOL_LOCAL_PATH}\n" + \
        "ORIG_WD=${PWD}\n" + \
        "cd ax_vol_plugin\n" + \
        "chmod u+x vol_plugin_installer.sh\n" + \
        "./vol_plugin_installer.sh\n" + \
        "cd ${ORIG_WD}\n" + \
        "echo \"AX: Volume plugin installer ran successfully\"\n"
        output.write(ax_vol_installer_script)
    return output.getvalue()


def validate_cluster_node_counts(cluster_config):
    """
    Verifies that the total count of nodes expected in the cluster is greater than the
    on-demand nodes requests.
    """
    num_nodes = int(cluster_config.get_min_node_count())
    on_demand_nodes = int(cluster_config.get_asxys_node_count()) + int(cluster_config.get_axuser_on_demand_count())
    max_nodes = int(cluster_config.get_max_node_count())

    assert max_nodes >= num_nodes >= int(cluster_config.get_asxys_node_count()), "Not enough nodes for running user jobs: " + str(num_nodes) + ":" + cluster_config.get_asxys_node_count()
    assert max_nodes >= num_nodes >= on_demand_nodes, "Total nodes in cluster less than on-demand nodes. " + str(num_nodes) + ":" + str(on_demand_nodes)


def prepare_kube_install_config(name_id, aws_profile, cluster_info, cluster_config):
    """
    This function generates kube-up envs. It also add those envs to cluster config
    :param name_id:
    :param aws_profile:
    :param cluster_info: AXClusterInfo object
    :param cluster_config: AXClusterConfig object
    :return:
    """
    logger.info("Preparing env for kube-up ...")
    validate_cluster_node_counts(cluster_config)
    master_config_env = cluster_config.get_master_config_env()

    # Need to pass in without env.
    customer_id = AXCustomerId().get_customer_id()
    kube_version = os.getenv("AX_KUBE_VERSION")

    env = {
        # AWS environments
        "AWS_IMAGE": cluster_config.get_ami_id(),
        "KUBERNETES_PROVIDER": cluster_config.get_provider(),
        "KUBE_AWS_ZONE": cluster_config.get_zone(),
        "KUBE_AWS_INSTANCE_PREFIX": name_id,
        "AWS_S3_BUCKET": AXClusterConfigPath(name_id).bucket(),
        "AWS_S3_REGION": cluster_config.get_region(),
        "AWS_S3_STAGING_PATH": "kubernetes-staging/v{}".format(kube_version),

        # Node Configs
        "AX_CLUSTER_NUM_NODES_MIN": cluster_config.get_min_node_count(),
        "AXUSER_ON_DEMAND_NUM_NODES": cluster_config.get_axuser_on_demand_count(),
        "AXUSER_NODE_TYPE": cluster_config.get_axuser_node_type(),
        "AXSYS_NUM_NODES": cluster_config.get_asxys_node_count(),
        "AXSYS_NODE_TYPE": cluster_config.get_axsys_node_type(),
        "MASTER_SIZE": cluster_config.get_master_type(),
        "AX_VOL_DISK_SIZE": str(cluster_config.get_ax_vol_size()),

        # Network
        "KUBE_VPC_CIDR_BASE": cluster_config.get_vpc_cidr_base(),

        # Cluster identity
        "AX_CUSTOMER_ID": customer_id,
        "AX_CLUSTER_NAME_ID": name_id,
        "AWS_SSH_KEY": cluster_info.get_key_file_path(),
        "KUBECONFIG": cluster_info.get_kube_config_file_path(),
        "KUBECTL_PATH": "/opt/google-cloud-sdk/bin/kubectl",
    }

    if aws_profile:
        env["AWS_DEFAULT_PROFILE"] = aws_profile

    optional_env = {
        # Start off directly with all spot instances only for dev clusters.
        "AX_USE_SPOT_INSTANCES": cluster_config.get_spot_instances_option() != "none",
        "NODE_SPOT_PRICE": cluster_config.get_node_spot_price(),
        "NODE_SPOT_OPTION": cluster_config.get_spot_instances_option(),
        "SUBNET_SIZE": cluster_config.get_subnet_size(),
        "VPC_ID": cluster_config.get_vpc_id() if cluster_config.get_vpc_id() else "",
    }
    env.update(default_kube_up_env)

    # For optional env, set it only if cluster_config has it set.
    for e in optional_env:
        val = optional_env[e]
        if val is not None:
            if isinstance(val, bool):
                env.update({e: str(val).lower()})
            else:
                env.update({e: str(val)})

    env.update(master_config_env)
    cluster_config.set_kube_installer_config(config=env)
    logger.info("Preparing env for kube-up ... DONE")
    return env


# This is absolutely hacky but we need to make it work
# by v2.0. Theoretically all envs should be re-generated
def _add_env_from_dict(env_dict):
    env_str = "\n"
    for e in env_dict.keys():
        env_str += "{env}: '{val}'\n".format(env=e, val=env_dict[e])
    return env_str
