#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Create and delete an object in kubernetes with a yaml or json config file.

This is more flexible as swagger client doesn't support passing config file.
"""

import logging
import os
import shlex
import subprocess
import tempfile

from ax.cloud import Cloud
from ax.kubernetes.ax_kube_dict import KubeKind, AXNameSpaces
from ax.kubernetes.ax_kube_yaml_update import AXSYSKubeYamlUpdater
from ax.kubernetes.client import KubernetesApiClient, parse_swagger_object, return_swagger_subobject
from ax.kubernetes.swagger_client import ApiClient
from ax.platform.exceptions import AXPlatformException
from ax.util.kube_poll import KubeObjPoll
from ax.util.macro import macro_replace
from jsonpath_rw import parse
import yaml


logger = logging.getLogger(__name__)

DEFAULT_KUBE_OBJECT_NAMESPACE = AXNameSpaces.AXSYS

# All Argo Kubernetes objects should have a label
# with this key for monitoring purpose
AX_MONITOR_LABEL_KEY = "app"

# How to wait for create_delete
KUBE_OBJ_WAIT_POLL = "KUBE_OBJ_WAIT_POLL"
KUBE_OBJ_WAIT_MONITOR = "KUBE_OBJ_WAIT_MONITOR"

# If object kind is in pod_list, we treat it as a pod
POD_LIST = [KubeKind.DEPLOYMENT, KubeKind.STATEFULSET, KubeKind.DAEMONSET, KubeKind.POD]


class KubeObjMonitorInfo(object):
    """
    Defines monitoring methods / validators and necessary parameters
    """
    def __init__(self, monitor_method, validator_key, kube_kind, label=None, name=None):
        if monitor_method == KUBE_OBJ_WAIT_POLL:
            assert label, "Monitoring using polling should provide label information"
        elif monitor_method == KUBE_OBJ_WAIT_MONITOR:
            assert name, "Monitoring using monitor should provide object name information"
        else:
            raise ValueError("Invalid wait method {}".format(monitor_method))
        self.monitor_method = monitor_method
        self.name = name
        self.label = label
        self.kube_kind = kube_kind

        self._validators = {
            "poll-namespace-active": self.poll_for_ns_active,
            "poll-for-existence": self.poll_for_existance,
            "poll-pod-healthy": self.poll_for_pod_healthy,
            "poll-pvc-bound": self.poll_for_pvc_bound,
            "poll-elb-exists": self.poll_for_elb_exists,
            "wait-for-pod": self.wait_for_pod_validator,
            "wait-for-pvc": self.wait_for_pvc_validator,
            "pv-release": self.pv_release_validator,
            "wait-for-elb": self.wait_for_svc_lb_validator
        }
        assert validator_key in self._validators
        self.validator = self._validators[validator_key]

    def __str__(self):
        info_temp = "Name: {}; Label: {}; KubeKind: {}; MonitorMethod: {}; Validator: {}"
        info = info_temp.format(self.name, self.label, self.kube_kind, self.monitor_method, self.validator)
        return info

    def __repr__(self):
        return self.__str__()

    @staticmethod
    def poll_for_ns_active(poll_result):
        if not poll_result:
            return False
        assert isinstance(poll_result, list), "Poll result should be a list of objects"
        # Not considering replicas
        for ns in poll_result:
            if ns.status.phase != "Active":
                return False
        return True

    @staticmethod
    def poll_for_existance(poll_result):
        if not poll_result:
            return False
        assert isinstance(poll_result, list), "Poll result should be a list of objects"
        # Not considering replicas
        return len(poll_result) > 0

    @staticmethod
    def poll_for_pod_healthy(poll_result):
        if not poll_result:
            return False
        assert isinstance(poll_result, list), "Poll result should be a list of objects"
        for po in poll_result:
            if po.status.phase != "Running" and po.status.phase != "Succeeded":
                return False
        return True

    @staticmethod
    def poll_for_pvc_bound(poll_result):
        if not poll_result:
            return False
        assert isinstance(poll_result, list), "Poll result should be a list of objects"
        for pvc in poll_result:
            if pvc.status.phase != "Bound":
                return False
        return True

    @staticmethod
    def poll_for_elb_exists(poll_result):
        if not poll_result:
            return False
        assert isinstance(poll_result, list), "Poll result should be a list of objects"
        for svc in poll_result:
            try:
                if bool(svc.status.load_balancer.ingress and
                        len(svc.status.load_balancer.ingress) == 1):
                    if Cloud().target_cloud_aws():
                        return "elb.amazonaws.com" in svc.status.load_balancer.ingress[0].hostname
                    elif Cloud().target_cloud_gcp():
                        return hasattr(svc.status.load_balancer.ingress[0], "ip")
                return False
            except Exception:
                return False
        return True

    @staticmethod
    def wait_for_pod_validator(status):
        return status["phase"] == "Running" or status["phase"] == "Succeeded"

    @staticmethod
    def wait_for_pvc_validator(status):
        return status["phase"] == "Bound"

    @staticmethod
    def pv_release_validator(status):
        return status["phase"] == "Released"

    @staticmethod
    def wait_for_svc_lb_validator(status):
        if bool(status["loadBalancer"] and
                status["loadBalancer"]["ingress"] and
                len(status["loadBalancer"]["ingress"]) == 1):
            if Cloud().target_cloud_aws():
                return "elb.amazonaws.com" in status["loadBalancer"]["ingress"][0]["hostname"]
            elif Cloud().target_cloud_gcp():
                return "ip" in status["loadBalancer"]["ingress"][0]
        return False


class KubeObjectInfo(object):
    """
    Information of a single Kubernetes object. A Kubernetes config file can have multiple
    Kubernetes objects
    """
    def __init__(self, kube_obj):
        self._kube_kind = None
        self._name = None
        self._monitor_label = None

        # TODO: due to the limitation of AXKubeMonitor, replica of DaemonSet is set to 1
        self._replica = 1
        self._cpu = 0
        self._mem = 0

        # Extra flags
        self._svc_elb = False
        self._extra_poll = False

        self._parse_kube_obj(kube_obj)

    def _parse_kube_obj(self, kube_obj):
        """
        Dictionary of a single kube_obj
        :param kube_obj:
        :return:
        """
        self._kube_kind = kube_obj["kind"]
        self._name = kube_obj["metadata"]["name"]
        self._get_label(kube_obj)
        self._get_usage(kube_obj)
        self._get_elb(kube_obj)

    def _get_label(self, kube_obj):
        # For deployments / daemonset / StatefulSet, we use the label in template,
        # for pod, pvc, svc, we get object from metadata directly
        if "spec" in kube_obj and "template" in kube_obj["spec"]:
            assert "labels" in kube_obj["spec"]["template"]["metadata"], "kube-object template does not have label"
            monitor_label_val = kube_obj["spec"]["template"]["metadata"]["labels"].get(AX_MONITOR_LABEL_KEY, None)
            assert monitor_label_val, "kube-obj does not have label for monitoring"
            self._monitor_label = "{}={}".format(AX_MONITOR_LABEL_KEY, monitor_label_val)
        else:
            assert "labels" in kube_obj["metadata"], "kube-object does not have label"
            monitor_label_val = kube_obj["metadata"]["labels"].get(AX_MONITOR_LABEL_KEY, None)
            assert monitor_label_val, "kube-obj does not have label for monitoring"
            self._monitor_label = "{}={}".format(AX_MONITOR_LABEL_KEY, monitor_label_val)

    def _get_usage(self, kube_obj):
        # We only count usage information for long running pods
        if self._kube_kind in [KubeKind.DEPLOYMENT, KubeKind.STATEFULSET, KubeKind.DAEMONSET]:
            if self._kube_kind == KubeKind.DAEMONSET:
                self._extra_poll = True
            multiplier = kube_obj["spec"].get("replicas", 1)
            if multiplier > 1:
                self._replica = multiplier
                if self._kube_kind == KubeKind.DEPLOYMENT:
                    self._extra_poll = True
            containers = kube_obj["spec"]["template"]["spec"]["containers"]
            for container in containers:
                # All Argo yamls MUST have these fields
                # If not, exception will be thrown to the very out side
                assert container["resources"]["requests"]["cpu"][-1] == "m", "CPU has to use unit \"milicore\""
                cpu = multiplier * int(container["resources"]["requests"]["cpu"][:-1])
                mem = multiplier * int(container["resources"]["requests"]["memory"][:-2])
                self._cpu += cpu
                self._mem += mem

    def _get_elb(self, kube_obj):
        if self._kube_kind == KubeKind.SERVICE:
            if "type" in kube_obj["spec"] and kube_obj["spec"]["type"] == "LoadBalancer":
                self._svc_elb = True

    @property
    def kube_kind(self):
        return self._kube_kind

    @property
    def name(self):
        return self._name

    @property
    def monitor_label(self):
        return self._monitor_label

    @property
    def replica(self):
        return self._replica

    @property
    def svc_elb(self):
        return self._svc_elb

    @property
    def extra_poll(self):
        return self._extra_poll

    @property
    def usage(self):
        """ (axsys_cpu, axsys_mem, daemon_cpu, daemon_mem) """
        if self._kube_kind == KubeKind.DAEMONSET:
            return 0, 0, self._cpu, self._mem
        else:
            return self._cpu, self._mem, 0, 0


class KubeObjectConfigFile(object):
    """
    KubeObject information fetched from kubernetes yaml config file template
    """
    def __init__(self, config_file, replacing):
        self._config_file = config_file
        self._replacing = replacing
        self._raw = None
        self._tmp_file = None
        self._kube_objects = []
        self._axsys_cpu = 0
        self._axsys_mem = 0
        self._daemon_cpu = 0
        self._daemon_mem = 0

        # A single config file can consist of one or more kubernetes objects
        # ping_info is used for pinging the existence of objects
        self._ping_info = []

        # status_info is used for querying the status of objects
        self._status_info = []

        # Default monitor methods. Currently a single kube_object can only have one
        # monitor method during create/delete. I.e. if you put a deployment (should use poll
        # during deletion) and a pvc (should use monitor during deletion), we can only use polling
        self._create_monitor_method = KUBE_OBJ_WAIT_MONITOR
        self._delete_monitor_method = KUBE_OBJ_WAIT_MONITOR

        self._parse_config()
        self._get_config_file_usage()
        self._generate_monitoring_info()

    def _parse_config(self):
        resource_updater = AXSYSKubeYamlUpdater(config_file_path=self._config_file)
        for c in resource_updater.components_in_dict:
            self._kube_objects.append(KubeObjectInfo(c))
        self._raw = resource_updater.updated_raw

    def _get_config_file_usage(self):
        """
        Total usage of objects in this config file
        """
        for ko in self._kube_objects:
            cpu, mem, dcpu, dmem = ko.usage
            self._axsys_cpu += cpu
            self._axsys_mem += mem
            self._daemon_cpu += dcpu
            self._daemon_mem += dmem

    def _generate_monitoring_info(self):
        """
        Generate create / delete monitor method and ping info
        :return:
        """
        for ko in self._kube_objects:
            assert isinstance(ko, KubeObjectInfo), "Invalid kube object info type"
            if ko.kube_kind in POD_LIST:
                validator_key = "poll-pod-healthy"
                kube_kind = KubeKind.POD
            elif ko.kube_kind == KubeKind.PVC:
                validator_key = "poll-pvc-bound"
                kube_kind = ko.kube_kind
            elif ko.kube_kind == KubeKind.SERVICE and ko.svc_elb:
                validator_key = "poll-elb-exists"
                kube_kind = ko.kube_kind
            elif ko.kube_kind == KubeKind.NAMESPACE:
                validator_key = "poll-namespace-active"
                kube_kind = ko.kube_kind
            else:
                validator_key = "poll-for-existence"
                kube_kind = ko.kube_kind

            self._status_info.append(
                KubeObjMonitorInfo(monitor_method=KUBE_OBJ_WAIT_POLL,
                                   validator_key=validator_key,
                                   kube_kind=kube_kind,
                                   label=ko.monitor_label
                                   )
            )

            self._ping_info.append(
                KubeObjMonitorInfo(monitor_method=KUBE_OBJ_WAIT_POLL,
                                   validator_key="poll-for-existence",
                                   kube_kind=kube_kind,
                                   label=ko.monitor_label
                                   )
            )
            if ko.kube_kind == KubeKind.SECRET or ko.kube_kind == KubeKind.NAMESPACE:
                # We use polling during creation / deletion for secret and namespace
                self._create_monitor_method = KUBE_OBJ_WAIT_POLL
                self._delete_monitor_method = KUBE_OBJ_WAIT_POLL
            elif ko.kube_kind != KubeKind.PVC:
                # PVC is the only object we CAN use polling during deletion
                self._delete_monitor_method = KUBE_OBJ_WAIT_POLL

    def generate_tmp_file(self):
        """
        Generate actual tmp file
        :param config: config file template string
        :return:
        """
        config = macro_replace(self._raw, self._replacing)
        self._tmp_file = tempfile.NamedTemporaryFile(dir="/tmp", prefix="kube-obj-", delete=False)
        self._tmp_file.write(config)
        # print("Ready to be installed: \n{}".format(config))
        self._tmp_file.close()

    def delete_tmp_file(self):
        if self._tmp_file:
            os.unlink(self._tmp_file.name)

    def get_swagger_objects(self):
        converter = ApiClient()

        def int_or_string(obj):
            try:
                return int(obj)
            except ValueError:
                return obj

        def handle_service(obj):
            rule = "spec.ports.target_port"
            parse_swagger_object(obj, rule, int_or_string)

        def handle_deployment(obj):
            rule = "spec.template.spec.containers.liveness_probe.http_get.port"
            parse_swagger_object(obj, rule, int_or_string)
            rule = "spec.template.spec.containers.readiness_probe.http_get.port"
            parse_swagger_object(obj, rule, int_or_string)

        supported_kinds = {
            "Service": ("V1Service", handle_service),
            "Deployment": ("V1beta1Deployment", handle_deployment),
            "Secret": ("V1Secret", None)
        }

        ret = []
        config = macro_replace(self._raw, self._replacing)
        for obj in yaml.load_all(config) or []:
            if obj["kind"] in supported_kinds:
                (obj_type, obj_func) = supported_kinds[obj["kind"]]
                converted_obj = converter._ApiClient__deserialize(obj, obj_type)
                if obj_func is not None:
                    obj_func(converted_obj)
                ret.append(converted_obj)
            else:
                logger.warn("Ignoring converting object {} as it is not supported".format(obj))

        return ret

    @property
    def tmp_file_name(self):
        """ Get name of temp file """
        return self._tmp_file.name if self._tmp_file else None

    @property
    def usage(self):
        return self._axsys_cpu, self._axsys_mem, self._daemon_cpu, self._daemon_mem

    @property
    def ping_info(self):
        return self._ping_info

    @property
    def status_info(self):
        return self._status_info

    @property
    def kube_objects(self):
        return self._kube_objects

    @property
    def create_monitor_method(self):
        return self._create_monitor_method

    @property
    def delete_monitor_method(self):
        return self._delete_monitor_method

    @property
    def replacing(self):
        return self._replacing

    @replacing.setter
    def replacing(self, r):
        self._replacing = r


class KubeObject(object):
    """
    Create or delete kubernetes object resource using kubectl create/delete command
    """
    kubectl_bin = "kubectl"

    def __init__(self, config_file, kubepoll=None, replacing=None,
                 kube_config=None, kube_namespace=None):
        """
        Initialize with kubernetes object config file.
        It can be json or yaml format.

        :param kubepoll: kubepoll object
        :param config_file: pathname to yaml or json config file.
        :param replacing: dict for macro replacement.
        :param kube_config: optional saved kube_config for cluster config.
        """
        self._config_file = config_file
        self._replacing = replacing if replacing else {}
        self._kube_config = kube_config
        self._attribute_map = {}
        self._namespace = kube_namespace
        self._kubectl = KubernetesApiClient(config_file=self._kube_config)
        self._kube_poll = kubepoll if kubepoll else KubeObjPoll(kubectl=self._kubectl)
        self._kube_conf_file = KubeObjectConfigFile(self._config_file, self._replacing) if self._config_file else None

        # This is a hack for Daemon Set or multiple replicas, we want to use monitor to make sure there is
        # at least one pod coming up as monitor would give us lots of useful information in case of error,
        # such as container command exe error / image pull error, etc.
        # After at least one pod starts, the other pods are very likely to start as normal. For Daemon Set
        # especially, because it is hard for us to know how many replicas, so we use this flag to do extra
        # poll: the caller shall poll KubeObject.healthy flag until the object is healthy
        self._extra_poll = False

    @property
    def resource_usage(self):
        return self._kube_conf_file.usage

    @property
    def config_file(self):
        return self._config_file

    @config_file.setter
    def config_file(self, cf):
        self._config_file = cf

    @property
    def replacing(self):
        return self._replacing

    @replacing.setter
    def replacing(self, r):
        self._replacing = r
        self._kube_conf_file.replacing = r

    @property
    def kube_config(self):
        return self._kube_config

    @kube_config.setter
    def kube_config(self, kc):
        self.kube_config = kc

    @property
    def namespace(self):
        return self._namespace

    @namespace.setter
    def namespace(self, ns):
        self._namespace = ns

    @property
    def extra_poll(self):
        return self._extra_poll

    def create(self):
        """
        Create a new object using kubectl create -f <config_file>
        """
        if not self._kube_conf_file:
            raise ValueError("Cannot create object without config file")
        self._kube_conf_file.generate_tmp_file()
        stdout, stderr = self._call_kubectl("create")

        # Some of our yaml files have multiple kubernetes object, i.e.
        # Service, Deployments, etc, so the result of the command will
        # contain multiple lines.
        if stdout:
            for l in stdout.splitlines():
                logger.info(l)
        if stderr:
            logger.warning("Failed to create object %s due to %s", self._config_file, stderr)
            if "already exists" in stderr:
                # As a temp work around for AA-3209
                logger.warning("Object %s already exist, which is not expected. Deleting the object and retry create",
                               self._config_file)
                self._call_kubectl("delete")
                retry_stdout, retry_stderr = self._call_kubectl("create")
                if retry_stdout:
                    for l in retry_stdout.splitlines():
                        logger.info(l)
                if retry_stderr:
                    logger.error("Object %s cannot be created after retry: %s", self._config_file, retry_stderr)
                    raise AXPlatformException("Object {} cannot be created after retry".format(self._config_file))
            else:
                raise AXPlatformException("Un-recognized error during create: {}".format(stderr))
        self._kube_conf_file.delete_tmp_file()

    def delete(self):
        """
        Delete an existing object using kubectl delete -f <config_file>
        """
        if not self._kube_conf_file:
            raise ValueError("Cannot delete object without config file")
        self._kube_conf_file.generate_tmp_file()
        self._call_kubectl("delete")
        self._kube_conf_file.delete_tmp_file()
        for ko in self._kube_conf_file.kube_objects:
            assert isinstance(ko, KubeObjectInfo)
            if ko.kube_kind == KubeKind.STATEFULSET:
                # For StatefulSet, we delete pods as well. StatefulSet pvc will be always deleted explicitly
                label = ko.monitor_label
                delete_pod = ["{} delete pods -l {}".format(self.kubectl_bin, label),
                              "--namespace={}".format(self._namespace)]
                if self._kube_config:
                    delete_pod += ["--kubeconfig={}".format(self._kube_config)]
                subprocess.call(shlex.split(' '.join(delete_pod)))

    def delete_all(self, type):
        """
        Delete all objects with specified type in the given namespace
        :param type: kubernetes type
        :return:
        """
        cmd = "{kubectl} delete {type} --all --namespace {namespace}".format(kubectl=self.kubectl_bin,
                                                                             type=type,
                                                                             namespace=self._namespace)
        if self._kube_config:
            cmd += " --kubeconfig={}".format(self._kube_config)
        exe = shlex.split(cmd)
        # logger.debug("Calling %s", exe)
        subprocess.call(exe)

    def get_create_monitor_info(self):
        if self._kube_conf_file.create_monitor_method == KUBE_OBJ_WAIT_POLL:
            # This means caller should poll obj_status
            return None
        if self._namespace != AXNameSpaces.AXSYS:
            return None
        # This kube obj requires monitor
        monitor_info = []
        for ko in self._kube_conf_file.kube_objects:
            if ko.extra_poll:
                self._extra_poll = True
            if ko.kube_kind in POD_LIST or ko.kube_kind == KubeKind.PVC or ko.kube_kind == KubeKind.SERVICE:
                monitor_info += self._get_monitor_info_from_obj_info(ko)
        return monitor_info

    def get_delete_monitor_info(self):
        if self._kube_conf_file.delete_monitor_method == KUBE_OBJ_WAIT_POLL:
            # This means caller should poll obj_status
            return None
        if self._namespace != AXNameSpaces.AXSYS:
            return None
        # This kube obj requires monitor
        monitor_info = []
        for ko in self._kube_conf_file.kube_objects:
            # This try...catch is reasonable as for example, a PVC raises a "ProvisioningFailed" error,
            # there will be exception when we try to get it's PV's name when we generate deletion monitoring
            # info. In this case, we want to have upper layer poll until object does not exist
            try:
                if ko.kube_kind in POD_LIST or ko.kube_kind == KubeKind.PVC or ko.kube_kind == KubeKind.SERVICE:
                    monitor_info += self._get_monitor_info_from_obj_info(ko, "delete")
            except Exception as e:
                logger.exception("Failed to get monitor info for object %s. Error: %s. Using polling for deletion",
                                 ko.name, e)
                return None
        return monitor_info

    def _get_monitor_info_from_obj_info(self, ko, action=""):
        """
        :param ko:
        :param action:
        :return: list of KubeObjMonitorInfo
        """
        assert isinstance(ko, KubeObjectInfo)
        if ko.kube_kind in POD_LIST:
            if ko.kube_kind == KubeKind.STATEFULSET:
                # only consider replicas for StatefulSet, as its name is predictable
                rst = []
                for i in range(ko.replica):
                    rst.append(
                        KubeObjMonitorInfo(monitor_method=KUBE_OBJ_WAIT_MONITOR,
                                           validator_key="wait-for-pod",
                                           kube_kind=KubeKind.POD,
                                           name="{}-{}".format(ko.name, i)
                                           )
                    )
                return rst
            else:
                return [
                   KubeObjMonitorInfo(monitor_method=KUBE_OBJ_WAIT_MONITOR,
                                      validator_key="wait-for-pod",
                                      kube_kind=KubeKind.POD,
                                      name=ko.name
                                      )
                ]
        elif ko.kube_kind == KubeKind.PVC:
            if action == "delete":
                # need to get pv
                pvcs = self._kube_poll.poll_kubernetes_sync(KubeKind.PVC, self._namespace, ko.monitor_label)
                if len(pvcs.items) == 0:
                    return []
                pvc = pvcs.items[0]
                return [
                    KubeObjMonitorInfo(monitor_method=KUBE_OBJ_WAIT_MONITOR,
                                       validator_key="pv-release",
                                       kube_kind=KubeKind.PV,
                                       name=pvc.spec.volume_name
                                       )
                ]
            else:
                return [
                    KubeObjMonitorInfo(monitor_method=KUBE_OBJ_WAIT_MONITOR,
                                       validator_key="wait-for-pvc",
                                       kube_kind=KubeKind.PVC,
                                       name=ko.name
                                       )
                ]
        elif ko.kube_kind == KubeKind.SERVICE:
            if ko.svc_elb:
                return [
                    KubeObjMonitorInfo(monitor_method=KUBE_OBJ_WAIT_MONITOR,
                                       validator_key="wait-for-elb",
                                       kube_kind=KubeKind.SERVICE,
                                       name=ko.name
                                       )
                ]
            else:
                return []
        else:
            raise ValueError("Invalid KubeKind {} for getting monitor info".format(ko.kube_kind))

    def _call_kubectl(self, action):
        """
        Call kubectl with action based on config.
        """
        cmd = ["{kubectl} {action} -f {config_file}".format(kubectl=self.kubectl_bin,
                                                            action=action,
                                                            config_file=self._kube_conf_file.tmp_file_name)]
        cmd += ["--namespace {}".format(self._namespace)]
        if self._kube_config is not None:
            cmd += ["--kubeconfig={}".format(self._kube_config)]
        # logger.debug("Calling [%s]", cmd)

        p = subprocess.Popen(shlex.split(' '.join(cmd)), stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        stdout, stderr = p.communicate()
        return stdout, stderr

    def build_attributes(self):
        """
        This function relies on derived classes having on dict called
        _attribute_map that has the name: jsonpath of attributes to be
        picked up from the getter for the object
        """
        status = self.get_status()
        for attr_name, attr_path in self._attribute_map.iteritems():
            expr = parse(attr_path)
            try:
                self.__setattr__(attr_name, expr.find(status)[0].value)
            except Exception as e:
                logger.debug("Exception %s in parsing status %s for attribute %s", e, status, attr_name)
                self.__setattr__(attr_name, None)

    def get_status(self):
        return {}

    def healthy(self):
        """
        For objects in POD_LIST: healthy = pods running
        For pvc: healthy = pvc bound
        For svc with elb: healthy = svc has elb info
        For objects not in POD_LIST: healthy = existence

        Returns True only when ALL objects in the Kubernetes config file are healthy

        :return: True / False
        """
        status_info = self._kube_conf_file.status_info
        for s in status_info:
            assert isinstance(s, KubeObjMonitorInfo), "Invalid ping info type"
            assert s.monitor_method == KUBE_OBJ_WAIT_POLL, "Only polling is supported for get status"
            obj = self._kube_poll.poll_kubernetes_sync(kind=s.kube_kind,
                                                       namespace=self._namespace,
                                                       label_selector=s.label)
            if obj:
                try:
                    if not s.validator(obj.items):
                        return False
                except Exception as e:
                    # this is not expected, as polling has something returned, obj.itmes should exist
                    logger.exception("Invalid obj returned from polling. Error: %s, Obj: %s.", e, obj)
            else:
                return False
        return True

    def exists(self):
        """
        Ping and see if object exists. This does not care about status of object
        Return True if ANY of the objects in config file exists (no matter what status it has)
        Return False if ALL of the objects in config file does not exist

        Note object exists != object healthy

        :return: True / False
        """
        ping_info = self._kube_conf_file.ping_info
        for ping in ping_info:
            assert isinstance(ping, KubeObjMonitorInfo), "Invalid ping info type"
            assert ping.monitor_method == KUBE_OBJ_WAIT_POLL, "Only polling is supported for get status"
            obj = self._kube_poll.poll_kubernetes_sync(kind=ping.kube_kind,
                                                       namespace=self._namespace,
                                                       label_selector=ping.label)
            if obj:
                try:
                    if ping.validator(obj.items):
                        return True
                except Exception as e:
                    # this is not expected, as polling has something returned, obj.itmes should exist
                    logger.exception("Invalid obj returned from polling. Error: %s, Obj: %s.", e, obj)
        return False

    @staticmethod
    def convert_obj_to_dict(obj):
        if isinstance(obj, list):
            return [KubeObject.convert_obj_to_dict(x) for x in obj]
        elif isinstance(obj, object) and getattr(obj, "to_dict", None):
            return obj.to_dict()
        else:
            return obj


    @staticmethod
    def swagger_obj_extract(obj, obj_map, serializable=False):
        """
        This function takes an object and returns a json dict of fields that
        are passed in the obj_map. The object map is a map of
        "name" : "field.subfield.subsubfield". If the field is missing, then the name
        will contain None, else it will contain the value at that field. If the value is
        an object then it will be another dict.

        :param obj:
        :param obj_map:
        :return:
        """
        ret = {}
        for field in obj_map:
            rule = obj_map[field]
            tmp = return_swagger_subobject(obj, rule)
            ret[field] = KubeObject.convert_obj_to_dict(tmp) if serializable else tmp

        return ret
