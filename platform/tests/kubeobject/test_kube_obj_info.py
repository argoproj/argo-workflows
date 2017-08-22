# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Let's just make sure all applatix config files can be parsed correctly
"""

import os
import uuid

import yaml
from ax.kubernetes.ax_kube_dict import KubeKind
from ax.kubernetes.kube_object import KubeObjectInfo

PWD = os.path.dirname(__file__)

os.environ["AX_CUSTOMER_ID"] = str(uuid.uuid4())
os.environ["AX_CLUSTER_NAME_ID"] = "test-cluster-" + str(uuid.uuid4())


def test_get_axmon_svc_info():
    obj_name = "axmon-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=axmon"
    assert svc.name == "axmon"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=axmon-deployment"
    assert dep.name == "axmon-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (75, 170, 0, 0)


def test_axdb_svc_info():
    obj_name = "axdb-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=axdb"
    assert svc.name == "axdb"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.STATEFULSET
    assert dep.monitor_label == "app=axdbstatefulset"
    assert dep.name == "axdb"
    assert dep.replica == 3
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (300, 6900, 0, 0)


def test_axopsbootstrap_info():
    obj_name = "axopsbootstrap"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    pod = KubeObjectInfo(components[0])
    assert pod.kube_kind == KubeKind.POD
    assert pod.monitor_label == "app=axops-bootstrap"
    assert pod.name == "axops-bootstrap-pod"
    assert pod.replica == 1
    assert not pod.svc_elb
    assert not pod.extra_poll
    assert pod.usage == (0, 0, 0, 0)


def test_redis_svc_info():
    obj_name = "redis-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=redis"
    assert svc.name == "redis"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=redis-deployment"
    assert dep.name == "redis-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (50, 50, 0, 0)


def test_kafka_zk_svc_info():
    obj_name = "kafka-zk-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 7

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=kafka-zk-svc"
    assert svc.name == "kafka-zk"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    for i in range(1, 4):
        sub_svc = KubeObjectInfo(components[i])
        assert sub_svc.kube_kind == KubeKind.SERVICE
        assert sub_svc.monitor_label == "app=kafka-zk-{}-svc".format(i)
        assert sub_svc.name == "kafka-zk-{}".format(i)
        assert sub_svc.replica == 1
        assert not sub_svc.svc_elb
        assert not sub_svc.extra_poll
        assert sub_svc.usage == (0, 0, 0, 0)

    for i in range(4, 7):
        dep = KubeObjectInfo(components[i])
        assert dep.kube_kind == KubeKind.DEPLOYMENT
        assert dep.monitor_label == "app=kafka-zk-{}".format(i-3)
        assert dep.name == "kafka-zk-{}".format(i-3)
        assert dep.replica == 1
        assert not dep.svc_elb
        assert not dep.extra_poll
        assert dep.usage == (150, 700, 0, 0)


def test_axnotification_svc_info():
    obj_name = "axnotification-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=axnotification"
    assert svc.name == "axnotification"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=axnotification-deployment"
    assert dep.name == "axnotification-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (50, 40, 0, 0)


def test_fixturemanager_svc_info():
    obj_name = "fixturemanager-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=fixturemanager"
    assert svc.name == "fixturemanager"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=fixturemanager-deployment"
    assert dep.name == "fixturemanager-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (100, 200, 0, 0)


def test_axworkflowadc_svc_info():
    obj_name = "axworkflowadc-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=axworkflowadc"
    assert svc.name == "axworkflowadc"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=axworkflowadc-deployment"
    assert dep.name == "axworkflowadc-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (75, 100, 0, 0)


def test_commitdata_info():
    obj_name = "commitdata"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=commitdata-deployment"
    assert dep.name == "commitdata-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (150, 300, 0, 0)


def test_gateway_svc_info():
    obj_name = "gateway-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=gateway"
    assert svc.name == "gateway"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=gateway-deployment"
    assert dep.name == "gateway-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (350, 700, 0, 0)


def test_axops_info():
    obj_name = "axops"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=axops-deployment"
    assert dep.name == "axops-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (310, 520, 0, 0)


def test_axstats_info():
    obj_name = "axstats"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=axstats-deployment"
    assert dep.name == "axstats-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (75, 270, 0, 0)



def test_elasticsearch_svc_info():
    obj_name = "elasticsearch-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=elasticsearch"
    assert svc.name == "elasticsearch"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=elasticsearch"
    assert dep.name == "elasticsearch"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (50, 1400, 0, 0)


def test_kibana_svc_info():
    obj_name = "kibana-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=kibana"
    assert svc.name == "kibana"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=kibana"
    assert dep.name == "kibana"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (100, 400, 0, 0)


def test_fluentd_info():
    obj_name = "fluentd"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.DAEMONSET
    assert dep.monitor_label == "app=fluentd"
    assert dep.name == "fluentd"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert dep.extra_poll
    assert dep.usage == (0, 0, 50, 200)


def test_cron_info():
    obj_name = "cron"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=cron-deployment"
    assert dep.name == "cron-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (50, 80, 0, 0)


def test_axconsole_svc_info():
    obj_name = "axconsole-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=axconsole"
    assert svc.name == "axconsole"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=axconsole-deployment"
    assert dep.name == "axconsole-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (50, 150, 0, 0)


def test_axscheduler_svc_info():
    obj_name = "axscheduler-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 2

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=axscheduler"
    assert svc.name == "axscheduler"
    assert svc.replica == 1
    assert not svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)

    dep = KubeObjectInfo(components[1])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=axscheduler-deployment"
    assert dep.name == "axscheduler-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (75, 40, 0, 0)


def test_autoscaler_info():
    obj_name = "autoscaler"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.DEPLOYMENT
    assert dep.monitor_label == "app=autoscaler"
    assert dep.name == "autoscaler-deployment"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (25, 50, 0, 0)


def test_vmf_info():
    obj_name = "volume-mounts-fixer"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.DAEMONSET
    assert dep.monitor_label == "app=volume-mounts-fixer"
    assert dep.name == "volume-mounts-fixer"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert dep.extra_poll
    assert dep.usage == (0, 0, 25, 60)


def test_registry_secrets_info():
    obj_name = "registry-secrets"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.SECRET
    assert dep.monitor_label == "app=axsecret"
    assert dep.name == "applatix-registry"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (0, 0, 0, 0)


def test_redis_pvc_info():
    obj_name = "redis-pvc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.PVC
    assert dep.monitor_label == "app=redis"
    assert dep.name == "redis-pvc"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (0, 0, 0, 0)


def test_elasticsearch_pvc_info():
    obj_name = "redis-pvc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    dep = KubeObjectInfo(components[0])
    assert dep.kube_kind == KubeKind.PVC
    assert dep.monitor_label == "app=redis"
    assert dep.name == "redis-pvc"
    assert dep.replica == 1
    assert not dep.svc_elb
    assert not dep.extra_poll
    assert dep.usage == (0, 0, 0, 0)


def test_kafka_zk_pvc_info():
    obj_name = "kafka-zk-pvc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 3

    for i in range(0, 3):
        dep = KubeObjectInfo(components[i])
        assert dep.kube_kind == KubeKind.PVC
        assert dep.monitor_label == "app=kafka-zk-{}".format(i+1)
        assert dep.name == "kdata-{}".format(i+1)
        assert dep.replica == 1
        assert not dep.svc_elb
        assert not dep.extra_poll
        assert dep.usage == (0, 0, 0, 0)


def test_axops_svc_info():
    obj_name = "axops-svc"
    config_file = os.path.join(PWD, "testdata/{}.yml.in".format(obj_name))
    with open(config_file) as f:
        data = f.read()
    components = [c for c in yaml.load_all(data)]
    assert len(components) == 1

    svc = KubeObjectInfo(components[0])
    assert svc.kube_kind == KubeKind.SERVICE
    assert svc.monitor_label == "app=axops"
    assert svc.name == "axops"
    assert svc.replica == 1
    assert svc.svc_elb
    assert not svc.extra_poll
    assert svc.usage == (0, 0, 0, 0)
