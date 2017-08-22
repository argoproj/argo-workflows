# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Let's just make sure all applatix config files can be parsed correctly
"""

import os
import json
import uuid

import ax.platform.cluster_config
ax.platform.cluster_config.AXClusterConfig = ax.platform.cluster_config.AXClusterConfigMock
from ax.kubernetes.kube_object import KubeObjectConfigFile, KUBE_OBJ_WAIT_POLL, KUBE_OBJ_WAIT_MONITOR


# Initialize cluster config
test_cluster_config = """
{"cloud": {"kube_up_env": {"KUBE_ENABLE_CLUSTER_LOGGING": "false",
"AWS_SSH_KEY": "/root/.ssh/kube_id_argo-ff83e264-d75c-11e6-98a7-c0ffeec0ffee","KUBE_RESCHED_CPU_REQ": "1",
"RC_CACHE": "15", "AX_CUSTOMER_ID": "b17d6218-8d36-42ed-8108-bc36d152ffbf", "AX_COMPUTE_UNIT_MAX": "300",
"AX_USE_SPOT_INSTANCES": "true", "MASTER_ROOT_DISK_SIZE": "24", "CLUSTER_IP_RANGE": "100.66.0.0/16",
"DNS_SERVER_IP": "100.64.0.10", "HTTP_API_CIDR": "54.149.149.230/32", "NODE_SPOT_PRICE": "0.1512",
"ENABLE_RESCHEDULER": "true", "AWS_S3_STAGING_PATH": "kubernetes-staging/ax-v1.4.3-35300-9a3425f", "SUBNET_SIZE": "24",
"AUTO_UPGRADE": "false", "KUBE_SCHED_CPU_REQ": "3", "AX_NUM_NODES_MAX": "30",
"AX_CLUSTER_NAME_ID": "argo-ff83e264-d75c-11e6-98a7-c0ffeec0ffee",
"KUBECONFIG": "/tmp/ax_kube/cluster_argo-ff83e264-d75c-11e6-98a7-c0ffeec0ffee.conf", "NODE_SIZE": "m3.2xlarge",
"ETCD_SNAPSHOT_COUNT": "1000", "AWS_S3_BUCKET": "applatix-cluster-12345678-c7de-11e6-a65f-0234d974d1bf-0",
"NUM_NODES": "3", "MASTER_DISK_SIZE": "60", "REPLICASET_CACHE": "15", "KUBE_SCHED_API_BURST": "4",
"KUBE_SCHED_MEM_REQ": "15", "KUBE_SCHED_API_QPS": "2", "AWS_DEFAULT_PROFILE": "sb3",
"NON_MASQUERADE_CIDR": "100.64.0.0/10", "KUBE_RESCHED_MEM_REQ": "5",
"API_SERVER_CPU_REQ": "4", "KUBE_AWS_ZONE": "us-west-2c",
"KUBE_AWS_INSTANCE_PREFIX": "argo-ff83e264-d75c-11e6-98a7-c0ffeec0ffee", "API_SERVER_MEM_REQ": "40",
"KUBE_ENABLE_CLUSTER_MONITORING": "false", "DAEMONSET_CACHE": "4", "KUBE_CONTROLLER_CPU_REQ": "4",
"VPC_ID": "vpc-12345678", "MASTER_IP_RANGE": "100.65.0.0/24", "KUBE_ENABLE_NODE_LOGGING": "false",
"API_SERVER_THROTTLING": "10", "AXSYS_NODES": "2", "VPC_NAME": "ax-vpc", "SSH_CIDR": "54.149.149.230/32",
"AXUSER_ON_DEMAND_NODES": "0", "SERVICE_CLUSTER_IP_RANGE": "100.64.0.0/16", "MASTER_SIZE": "r3.large",
"KUBE_VPC_CIDR_BASE": "172.20", "KUBERNETES_PROVIDER": "aws"}, "configure": {"sandbox_enabled": "False",
"placement": "us-west-2c", "axsys_node_type": "m3.large", "axuser_node_type": "m3.large", "axsys_node_count": 3,
"region": "us-west-2", "axuser_placement": "us-west-2c", "max_node_count": 21, "axuser_on_demand_nodes": 0,
"node_tiers": "applatix/user", "min_node_count": 4, "master_type": "r3.large", "cluster_type": "standard"},
"vpc_cidr_base": "172.20", "version": 1,
"trusted_cidr": ["54.149.149.230/32", "73.70.250.25/32", "104.10.248.90/32", "54.200.77.5/32"], "provider": "aws"}}
"""

PWD = os.path.dirname(__file__)
config = json.loads(test_cluster_config)
cluster_config = ax.platform.cluster_config.AXClusterConfig(config=config)


os.environ["AX_CUSTOMER_ID"] = str(uuid.uuid4())
os.environ["AX_CLUSTER_NAME_ID"] = "test-cluster-" + str(uuid.uuid4())


def test_get_axmon_svc_info():
    obj_name = "axmon-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_axdb_svc_info():
    obj_name = "axdb-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_axopsbootstrap_info():
    obj_name = "axopsbootstrap"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_redis_svc_info():
    obj_name = "redis-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_kafka_zk_svc_info():
    obj_name = "kafka-zk-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 7
    assert len(kcf.ping_info) == 7
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 7
    for i in range(0, 4):
        assert kcf.status_info[i].validator == kcf.status_info[i]._validators["poll-for-existence"]
    for i in range(4, 7):
        assert kcf.status_info[i].validator == kcf.status_info[i]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_axnotification_svc_info():
    obj_name = "axnotification-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_fixturemanager_svc_info():
    obj_name = "fixturemanager-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_axworkflowadc_svc_info():
    obj_name = "axworkflowadc-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_gateway_svc_info():
    obj_name = "gateway-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_axops_info():
    obj_name = "axops"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_axstats_info():
    obj_name = "axstats"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_elasticsearch_svc_info():
    obj_name = "elasticsearch-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_kibana_svc_info():
    obj_name = "kibana-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_fluentd_info():
    obj_name = "fluentd"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_cron_info():
    obj_name = "cron"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_axconsole_svc_info():
    obj_name = "axconsole-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_axscheduler_svc_info():
    obj_name = "axscheduler-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 2
    assert len(kcf.ping_info) == 2
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 2
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.status_info[1].validator == kcf.status_info[1]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_autoscaler_info():
    obj_name = "autoscaler"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-pod-healthy"]

    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_vmf_info():
    obj_name = "volume-mounts-fixer"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-pod-healthy"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_registry_secrets_info():
    obj_name = "registry-secrets"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-for-existence"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_POLL
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL


def test_redis_pvc_info():
    obj_name = "redis-pvc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-pvc-bound"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_MONITOR


def test_elasticsearch_pvc_info():
    obj_name = "redis-pvc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-pvc-bound"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_MONITOR


def test_kafka_zk_pvc_info():
    obj_name = "kafka-zk-pvc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 3
    assert len(kcf.ping_info) == 3
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 3
    for obj in kcf.status_info:
        assert obj.validator == obj._validators["poll-pvc-bound"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_MONITOR


def test_axops_svc_info():
    obj_name = "axops-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))

    kcf = KubeObjectConfigFile(config_file=config_file, replacing=None)
    assert len(kcf.kube_objects) == 1
    assert len(kcf.ping_info) == 1
    for obj in kcf.ping_info:
        assert obj.validator == obj._validators["poll-for-existence"]
    assert len(kcf.status_info) == 1
    assert kcf.status_info[0].validator == kcf.status_info[0]._validators["poll-elb-exists"]
    assert kcf.create_monitor_method == KUBE_OBJ_WAIT_MONITOR
    assert kcf.delete_monitor_method == KUBE_OBJ_WAIT_POLL
