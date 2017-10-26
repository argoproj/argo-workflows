#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Module to start/stop AX platform by managing kubernetes objects
"""

import base64
import logging
import os
import random
import requests
import subprocess
import time
import urllib3
import yaml
from multiprocessing.pool import ThreadPool
from retrying import retry

from ax.aws.profiles import AWSAccountInfo
from ax.cloud import Cloud
from ax.kubernetes.ax_kube_dict import KubeKind, AXNameSpaces, KubeKindToKubeApiObjKind
from ax.kubernetes.client import KubernetesApiClient
from ax.kubernetes.kube_object import KubeObject
from ax.meta import AXClusterId, AXClusterConfigPath, AXCustomerId
from ax.platform.application import Applications, Application
from ax.platform.ax_asg import AXUserASGManager
from ax.platform.ax_cluster_info import AXClusterInfo
from ax.platform.ax_monitor import AXKubeMonitor
from ax.platform.ax_monitor_helper import KubeObjStatusCode, KubeObjWaiter
from ax.platform.cluster_config import AXClusterConfig, SpotInstanceOption
from ax.platform.cluster_version import AXVersion
from ax.platform.component_config import SoftwareInfo
from ax.platform.exceptions import AXPlatformException
from ax.platform.kube_env_config import default_kube_up_env
from ax.platform.resource.consts import EC2_PARAMS
from ax.platform.component_config import AXPlatformConfig, AXPlatformObjectGroup, AXPlatformObject, \
    ObjectGroupPolicy, ObjectGroupPolicyPredicate, AXPlatformConfigDefaults
from ax.util.const import COLOR_NORM, COLOR_GREEN
from ax.util.kube_poll import KubeObjPoll


logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)
logging.getLogger("ax.platform.ax_monitor").setLevel(logging.INFO)
logging.getLogger("ax.platform.ax_monitor_helper").setLevel(logging.INFO)

# Hack to disable SSL warnings
urllib3.disable_warnings()
requests.packages.urllib3.disable_warnings()


class AXPlatform(object):

    def __new__(cls, *args, **kwargs):
        if Cloud().target_cloud_gcp():
            from .gke_platform import AXGKEPlatform
            return super(AXPlatform, cls).__new__(AXGKEPlatform)
        else:
            return super(AXPlatform, cls).__new__(cls)

    def __init__(self, cluster_name_id=None, aws_profile=None, debug=True,
                 manifest_root=AXPlatformConfigDefaults.DefaultManifestRoot,
                 config_file=AXPlatformConfigDefaults.DefaultPlatformConfigFile,
                 software_info=None):
        """
        AX Platform bootstrap

        :param cluster_name_id: cluster name id
        :param aws_profile: aws profile to authenticate all aws clients
        :param debug: debug mode
        :param manifest_root: root directory to all ax service objects
        """
        self._software_info = software_info if software_info else SoftwareInfo()
        assert isinstance(self._software_info, SoftwareInfo), "Wrong type ({}) of software info passed in.".format(self._software_info)
        self._aws_profile = aws_profile
        self._manifest_root = manifest_root
        self._config = AXPlatformConfig(config_file)

        logger.info("Using Kubernetes manifest from %s", self._manifest_root)
        logger.info("Using platform configuration \"%s\" from %s", self._config.name, config_file)

        self._cluster_name_id = AXClusterId(cluster_name_id).get_cluster_name_id()
        self._cluster_config = AXClusterConfig(cluster_name_id=self._cluster_name_id, aws_profile=self._aws_profile)
        self._cluster_config_path = AXClusterConfigPath(cluster_name_id)
        self._cluster_info = AXClusterInfo(self._cluster_name_id, aws_profile=self._aws_profile)

        self._region = self._cluster_config.get_region()
        if Cloud().target_cloud_aws() and self._cluster_config.get_provider() not in ["minikube", "gke"]:
            self._account = AWSAccountInfo(aws_profile=self._aws_profile).get_account_id()
        else:
            self._account = ""
        self._bucket_name = self._cluster_config_path.bucket()
        self._bucket = Cloud().get_bucket(self._bucket_name, aws_profile=self._aws_profile, region=self._region)

        # In debug mode, when we failed to create an object, we don't delete it but just
        # leave it for debug.
        self._debug = debug

        # DNS
        self.cluster_dns_name = None

        # Get kube cluster config. Automatic if in pod already.
        self._kube_config = self._cluster_info.get_kube_config_file_path() if self._cluster_name_id else None
        if self._cluster_name_id:
            if not os.path.isfile(self._kube_config):
                logger.info("Can't find config file at %s; downloading from s3", self._kube_config)
                self._kube_config = self._cluster_info.download_kube_config()
            assert os.path.isfile(self._kube_config), "No kube_config file available"

        # Kubernetes related objects and macros
        self.kube_namespaces = [AXNameSpaces.AXSYS, AXNameSpaces.AXUSER]
        self.kube_axsys_namespace = AXNameSpaces.AXSYS
        self.kube_user_namespace = AXNameSpaces.AXUSER
        self.kubectl = KubernetesApiClient(config_file=self._kube_config)
        self.kube_poll = KubeObjPoll(kubectl=self.kubectl)

        self._monitor = AXKubeMonitor(kubectl=self.kubectl)
        self._monitor.reload_monitors(namespace=self.kube_axsys_namespace)
        self._monitor.start()

        # Kube Objects
        self._kube_objects = {}
        self._replacing = {}

    def _load_kube_objects_from_steps(self, steps):
        """
        Extract kube objects from steps in config, and load them into memory
        :param steps: list
        :return:
        """
        for object_group in steps:
            assert isinstance(object_group, AXPlatformObjectGroup)
            for obj in object_group.object_set:
                assert isinstance(obj, AXPlatformObject)
                name = obj.name
                filename = obj.manifest
                namespace = obj.namespace
                if name in self._kube_objects:
                    raise ValueError("Duplicated object name {}".format(name))
                kubeobj_conf_path = os.path.join(self._manifest_root, filename)
                self._kube_objects[name] = KubeObject(
                    config_file=kubeobj_conf_path,
                    kubepoll=self.kube_poll,
                    replacing=None,
                    kube_config=self._kube_config,
                    kube_namespace=namespace
                )

    def _get_trusted_cidr_str(self):
        trusted_cidr = self._cluster_config.get_trusted_cidr()
        if isinstance(trusted_cidr, list):
            trusted_cidr_str = "["
            for cidr in trusted_cidr:
                trusted_cidr_str += "\"{}\",".format(str(cidr))
            trusted_cidr_str = trusted_cidr_str[:-1]
            trusted_cidr_str += "]"
        else:
            trusted_cidr_str = "[{}]".format(trusted_cidr)
        return trusted_cidr_str


    def _generate_replacing_for_user_provisioned_cluster(self):
        trusted_cidr_str = self._get_trusted_cidr_str()
        self._persist_node_resource_rsvp(0, 0)

        with open("/kubernetes/cluster/version.txt", "r") as f:
            cluster_install_version = f.read().strip()

        return {
            "REGISTRY": self._software_info.registry,
            "REGISTRY_SECRETS": self._software_info.registry_secrets,
            "NAMESPACE": self._software_info.image_namespace,
            "VERSION": self._software_info.image_version,
            "AX_CLUSTER_NAME_ID": self._cluster_name_id,
            "AX_AWS_REGION": self._region,
            "AX_AWS_ACCOUNT": self._account,
            "AX_CUSTOMER_ID": AXCustomerId().get_customer_id(),
            "TRUSTED_CIDR": trusted_cidr_str,
            "NEW_KUBE_SALT_SHA1": os.getenv("NEW_KUBE_SALT_SHA1") or " ",
            "NEW_KUBE_SERVER_SHA1": os.getenv("NEW_KUBE_SERVER_SHA1") or " ",
            "AX_KUBE_VERSION": os.getenv("AX_KUBE_VERSION"),
            "AX_CLUSTER_INSTALL_VERSION": cluster_install_version,
            "SANDBOX_ENABLED": str(self._cluster_config.get_sandbox_flag()),
            "ARGO_LOG_BUCKET_NAME": self._cluster_config.get_support_object_store_name(),
            "AX_CLUSTER_META_URL_V1": self._bucket.get_object_url_from_key(key=self._cluster_config_path.cluster_metadata()),
            "DNS_SERVER_IP": os.getenv("DNS_SERVER_IP", default_kube_up_env["DNS_SERVER_IP"]),
            "ARGO_DATA_BUCKET_NAME": AXClusterConfigPath(self._cluster_name_id).bucket(),
            "LOAD_BALANCER_TYPE": "LoadBalancer",
            "ARGO_S3_ACCESS_KEY_ID": base64.b64encode(os.getenv("ARGO_S3_ACCESS_KEY_ID", "")),
            "ARGO_S3_ACCESS_KEY_SECRET": base64.b64encode(os.getenv("ARGO_S3_ACCESS_KEY_SECRET", "")),
        }


    def _generate_replacing(self):
        # Platform code are running in python 2.7, and therefore for trusted cidr list, the str() method
        # will return something like [u'54.149.149.230/32', u'73.70.250.25/32', u'104.10.248.90/32'], and
        # this 'u' prefix cannot be surpressed. With this prefix, our macro replacing would create invalid
        # yaml files, and therefore we construct string manually here
        trusted_cidr_str = self._get_trusted_cidr_str()
        axsys_cpu = 0
        axsys_mem = 0
        daemon_cpu = 0
        daemon_mem = 0
        for name in self._kube_objects.keys():
            cpu, mem, dcpu, dmem = self._kube_objects[name].resource_usage
            axsys_cpu += cpu
            axsys_mem += mem
            daemon_cpu += dcpu
            daemon_mem += dmem

        # kube-proxy (100m CPU and 100Mi memory. Note kube-proxy does not
        # have a memory request, but this is an approximation)
        daemon_cpu += 100
        daemon_mem += 100

        logger.info(
            "Resource Usages: axsys_cpu: %s milicores, axsys_mem: %s Mi, node_daemon_cpu: %s milicores, node_daemon_mem: %s Mi",
            axsys_cpu, axsys_mem, daemon_cpu, daemon_mem)

        axsys_node_count = int(self._cluster_config.get_asxys_node_count())
        axuser_min_count = str(int(self._cluster_config.get_min_node_count()) - axsys_node_count)
        axuser_max_count = str(int(self._cluster_config.get_max_node_count()) - axsys_node_count)
        autoscaler_scan_interval = str(self._cluster_config.get_autoscaler_scan_interval())

        usr_node_cpu_rsvp = float(daemon_cpu) / EC2_PARAMS[self._cluster_config.get_axuser_node_type()]["cpu"]
        usr_node_mem_rsvp = float(daemon_mem) / EC2_PARAMS[self._cluster_config.get_axuser_node_type()]["memory"]
        scale_down_util_thresh = round(max(usr_node_cpu_rsvp, usr_node_mem_rsvp), 3) + 0.001
        logger.info("Setting node scale down utilization threshold to %s", scale_down_util_thresh)

        self._persist_node_resource_rsvp(daemon_cpu, daemon_mem)

        with open("/kubernetes/cluster/version.txt", "r") as f:
            cluster_install_version = f.read().strip()

        # Prepare autoscaler
        asg_manager = AXUserASGManager(self._cluster_name_id, self._region, self._aws_profile)
        asg = asg_manager.get_variable_asg() or asg_manager.get_spot_asg() or asg_manager.get_on_demand_asg()
        if not asg:
            raise AXPlatformException(
                "Failed to get autoscaling group for cluster {}".format(self._cluster_name_id))
        asg_name = asg["AutoScalingGroupName"]

        if not asg_name:
            logger.error("Autoscaling group name not found for %s", self._cluster_name_id)
            raise AXPlatformException("Cannot find cluster autoscaling group")

        # Prepare minion-manager.
        spot_instances_option = self._cluster_config.get_spot_instances_option()
        minion_manager_asgs = ""
        if spot_instances_option == SpotInstanceOption.ALL_SPOT:
            for asg in asg_manager.get_all_asgs():
                minion_manager_asgs = minion_manager_asgs + asg["AutoScalingGroupName"] + " "
            minion_manager_asgs = minion_manager_asgs[:-1]
        elif spot_instances_option == SpotInstanceOption.PARTIAL_SPOT:
            minion_manager_asgs = asg_manager.get_variable_asg()["AutoScalingGroupName"]

        return {
            "REGISTRY": self._software_info.registry,
            "REGISTRY_SECRETS": self._software_info.registry_secrets,
            "NAMESPACE": self._software_info.image_namespace,
            "VERSION": self._software_info.image_version,
            "AX_CLUSTER_NAME_ID": self._cluster_name_id,
            "AX_AWS_REGION": self._region,
            "AX_AWS_ACCOUNT": self._account,
            "AX_CUSTOMER_ID": AXCustomerId().get_customer_id(),
            "TRUSTED_CIDR": trusted_cidr_str,
            "NEW_KUBE_SALT_SHA1": os.getenv("NEW_KUBE_SALT_SHA1") or " ",
            "NEW_KUBE_SERVER_SHA1": os.getenv("NEW_KUBE_SERVER_SHA1") or " ",
            "AX_KUBE_VERSION": os.getenv("AX_KUBE_VERSION"),
            "AX_CLUSTER_INSTALL_VERSION": cluster_install_version,
            "SANDBOX_ENABLED": str(self._cluster_config.get_sandbox_flag()),
            "ARGO_LOG_BUCKET_NAME": self._cluster_config.get_support_object_store_name(),
            "ASG_MIN": axuser_min_count,
            "ASG_MAX": axuser_max_count,
            "AUTOSCALER_SCAN_INTERVAL": autoscaler_scan_interval,
            "SCALE_DOWN_UTIL_THRESH": str(scale_down_util_thresh),
            "AX_CLUSTER_META_URL_V1": self._bucket.get_object_url_from_key(key=self._cluster_config_path.cluster_metadata()),
            "ASG_NAME": asg_name,
            "DNS_SERVER_IP": os.getenv("DNS_SERVER_IP", default_kube_up_env["DNS_SERVER_IP"]),
            "AX_ENABLE_SPOT_INSTANCES": str(spot_instances_option != SpotInstanceOption.NO_SPOT),
            "AX_SPOT_INSTANCE_ASGS": minion_manager_asgs,
        }

    def _persist_node_resource_rsvp(self, user_node_daemon_cpu, user_node_daemon_mem):
        self._cluster_config.set_user_node_resource_rsvp(cpu=user_node_daemon_cpu,
                                                         mem=user_node_daemon_mem)
        self._cluster_config.save_config()

    def start(self):
        """
        Bring up platform using "platform-start.cfg" configuration from manifest directory
        :return:
        """
        # Generate kube-objects
        steps = self._config.steps
        self._load_kube_objects_from_steps(steps)

        if not self._cluster_config.get_cluster_provider().is_user_cluster():
            self._replacing = self._generate_replacing()
        else:
            self._replacing = self._generate_replacing_for_user_provisioned_cluster()

        logger.debug("Replacing ENVs: %s", self._replacing)

        # TODO: remove component's dependencies to AXOPS_EXT_DNS env (#32)
        # At this moment, we MUST separate first step due to the above dependency
        assert len(steps) >= 2, "Should have at least 1 step to create axops"
        self.create_objects(steps[0])
        self.create_objects(steps[1])
        self.create_objects(steps[2])

        # Prepare axops_eip
        if self._cluster_config.get_provider() not in ["minikube", "gke"]:
            self._set_ext_dns()

        info_bound = "=======================================================\n"
        img_namespace = "Image Namespace: {}\n".format(self._software_info.image_namespace)
        img_version = "Image Version: {}\n".format(self._software_info.image_version)
        start_info = "\n\n{}{}{}{}{}".format(info_bound, "Platform Up: Bringing up Argo services...\n",
                                             img_namespace, img_version, info_bound)
        logger.info(start_info)

        # Start rest of the objects
        for i in range(3, len(steps)):
            self.create_objects(steps[i])

        # update application namespace
        logger.info("Updating application managers")
        for app in Applications(client=self.kubectl).list():
            logger.info("--- updating {}".format(app))
            a = Application(app, client=self.kubectl)
            a.create(force_recreate=True)
        logger.info("Done updating application managers")

        # Upload version information to target cluster
        self._update_version()
        logger.info("\n\n%sCluster %s is up. Cluster is available at %s%s\n", COLOR_GREEN, self._cluster_name_id,
                    self.cluster_dns_name, COLOR_NORM)

    def stop(self):
        """
        Bring down platform using "platform-stop.cfg" configuration from manifest directory
        :return:
        """
        # Generate kube-objects (Does not need to generate replacing during platform down)
        # Stop order should be the reverse of start
        steps = self._config.steps
        steps.reverse()
        self._load_kube_objects_from_steps(steps)

        info_bound = "=======================================================\n"
        stop_info = "\n\n{}{}{}".format(info_bound, "Platform Down: Shutting down Argo services...\n",
                                        info_bound)
        logger.info(stop_info)

        # Bring down objects according to steps
        for i in range(len(steps)):
            object_group = steps[i]
            self.delete_objects(object_group)

    def stop_monitor(self):
        self._monitor.stop()

    def create_objects(self, objects):
        """
        Start kubernetes objects based on records.
        Wait for all of them.

        :param objects: AXPlatformObjectGroup
        """
        if objects is None or len(objects.object_set) == 0:
            return

        assert isinstance(objects, AXPlatformObjectGroup)
        if not self._should_create_group(
            policy=objects.policy,
            policy_predicate=objects.policy_predicate,
            consistency=objects.consistency
        ):
            logger.debug("Skipping object group (%s) creation based on policy (%s), policy predicate (%s), consistency (%s)",
                         objects.name, objects.policy, objects.policy_predicate, objects.consistency)
            return
        logger.info("Create step: %s", objects.name)
        logger.info("Creating platform objects\n\n%s", self._generate_object_summary(objects.object_set))
        pool = ThreadPool(len(objects.object_set))
        async_results = {}
        for obj in objects.object_set:
            assert isinstance(obj, AXPlatformObject)
            name = obj.name
            namespace = obj.namespace
            async_results[name] = pool.apply_async(self.start_one, args=(name,), kwds={"namespace": namespace})
        pool.close()
        pool.join()

        report, failed = self._generate_report(async_results, "Create")
        logger.info(report)

        if failed:
            raise AXPlatformException("Failed to create platform objects.")

    def _should_create_group(self, policy, policy_predicate, consistency):
        """
        Take AXPlatformObjectGroup policy, predicate and consistency and determine
        if this group should be created or not
        :param policy:
        :param policy_predicate:
        :param consistency:
        :return:
        """
        # Since we are not using consistency, we should always create if not
        # explicitly told not to, i.e. if there is a PrivateRegistryOnly
        # We are just leaving the interface here that should create or not
        # need to be decided by policy, policy_predicate and consistency

        if policy_predicate == ObjectGroupPolicyPredicate.PrivateRegistryOnly and \
                not self._software_info.registry_is_private():
            return False
        return True

    def delete_objects(self, objects):
        """
        Stop kubernetes objects based on records.
        Wait for all of them.

        :param objects: AXPlatformObjectGroup
        """
        assert isinstance(objects, AXPlatformObjectGroup)
        if not self._should_delete_group(
            policy=objects.policy,
            policy_predicate=objects.policy_predicate
        ):
            logger.debug(
                "Skipping object group (%s) deletion based on policy (%s), policy predicate (%s)",
                objects.name, objects.policy, objects.policy_predicate
            )
            return
        logger.info("Delete step: %s", objects.name)
        logger.info("Deleting platform objects\n\n%s.", self._generate_object_summary(objects.object_set))
        pool = ThreadPool(len(objects.object_set))
        async_results = {}
        for obj in objects.object_set:
            assert isinstance(obj, AXPlatformObject)
            name = obj.name
            namespace = obj.namespace
            async_results[name] = pool.apply_async(self.stop_one, args=(name,), kwds={"namespace": namespace})
        pool.close()
        pool.join()

        report, failed = self._generate_report(async_results, "Delete")
        logger.info(report)
        if failed:
            raise AXPlatformException("Failed to create platform objects.")

    def _should_delete_group(self, policy, policy_predicate):
        """
        Take AXPlatformObjectGroup policy and determine if this group should be deleted or not.
        Consistency is not needed
        for deletion

        :param policy:
        :param policy_predicate:
        :return:
        """
        if policy == ObjectGroupPolicy.CreateMany:
            return True
        return False

    def start_one(self, name, namespace=AXNameSpaces.AXSYS):
        time.sleep(random.randint(0, AXPlatformConfigDefaults.ObjectOperationJitter))
        logger.info("Creating %s in namespace %s ...", name, namespace)
        start = time.time()
        kube_obj = self._kube_objects[name]

        # Update them as there are new updates in replacing in platform start
        kube_obj.namespace = namespace
        kube_obj.replacing = self._replacing

        assert isinstance(kube_obj, KubeObject)
        result = {
            "name": name,
            "code": [],
            "events": [],
            "failed": False,
            "duration": ""
        }
        if kube_obj.healthy():
            result["code"] += ["{:.25s}:{}".format(name, KubeObjStatusCode.OBJ_EXISTS)]
            result["duration"] = str(round(time.time() - start, 2))
            return result

        # Previous platform start might fail, and might result in some componenets created
        # but not healthy (i.e. in CrashLoopBackoff). In this case, we delete the existing
        # object and try to create a new one
        if kube_obj.exists():
            logger.warning("Object %s exists but not healthy. Deleting object for idempotency ...", name)
            self.stop_one(name, namespace)

        assert not kube_obj.exists(), "Kubeobject {} already created but is not healthy. Not Expected".format(name)

        monitor_info = kube_obj.get_create_monitor_info()
        if monitor_info:
            # use monitor
            waiters = []

            # Create and register waiters for all objects that can be monitored
            for m in monitor_info:
                wait_info = {
                    "kind": KubeKindToKubeApiObjKind[m.kube_kind],
                    "name": m.name,
                    "validator": m.validator
                }
                waiter = KubeObjWaiter()
                waiters.append((waiter, wait_info))
                AXKubeMonitor().wait_for_kube_object(
                    wait_info,
                    AXPlatformConfigDefaults.ObjCreateWaitTimeout,
                    waiter
                )

            # Call kubectl create
            kube_obj.create()

            # Wait on all waiters to retrieve status and events
            for waiter, wait_info in waiters:
                waiter.wait()
                result["events"] += waiter.details
                result["code"].append("{:.25s}:{}".format(wait_info["name"], waiter.result))
                if waiter.result == KubeObjStatusCode.OK or waiter.result == KubeObjStatusCode.WARN:
                    logger.info("Successfully created %s with code %s.", wait_info["name"], waiter.result)
                else:
                    result["failed"] = True
                    logger.error("Failed to create %s in %s with code %s. Events: %s",
                                 wait_info["name"], namespace, waiter.result, str(waiter.details))
                    if not self._debug:
                        logger.info("Deleting %s due to creation failure", name)
                        del_rst = self.stop_one(name, namespace)
                        result["code"] += del_rst["code"]
                        result["events"] += del_rst["events"]
                        result["duration"] = str(round(time.time() - start, 2))
                        return result

            # Poll extra if required (for Petset and Deployments with multiple replicas)
            if kube_obj.extra_poll:
                logger.info("Polling till healthy to make sure rest of components of %s are up and running ...", name)
                create_rst = self._poll_till_healthy(
                    name=name,
                    kube_obj=kube_obj,
                    start_time=start,
                    poll_interval=AXPlatformConfigDefaults.ObjCreateExtraPollInterval,
                    poll_max_retry=AXPlatformConfigDefaults.ObjCreateExtraPollMaxRetry,
                    rst=result
                )
                if create_rst["failed"] and not self._debug:
                    logger.info("Deleting %s due to creation failure", name)
                    del_rst = self.stop_one(name, namespace)
                    create_rst["code"] += del_rst["code"]
                    create_rst["events"] += del_rst["events"]
                    create_rst["duration"] = str(round(time.time() - start, 2))
                return create_rst

            # Poll once to confirm all components from this Kubernetes config file exist,
            # In case there are objects in this config file cannot be monitored, i.e. svc
            # without elb. This is really not expected so we don't delete it
            if not kube_obj.healthy():
                logger.error("Object %s created but is not healthy. This is NOT EXPECTED, please check manually.", name)
                result["code"].append("{:.25s}:{}".format(name, KubeObjStatusCode.UNHEALTHY))
                result["failed"] = True
                result["events"].append("Object {} created byt is not healthy".format(name))
            result["duration"] = str(round(time.time() - start, 2))

            if not result["failed"]:
                logger.info("Successfully created object %s.", name)
            return result
        else:
            # use polling
            kube_obj.create()
            create_rst = self._poll_till_healthy(
                name=name,
                kube_obj=kube_obj,
                start_time=start,
                poll_interval=AXPlatformConfigDefaults.ObjCreatePollInterval,
                poll_max_retry=AXPlatformConfigDefaults.ObjCreatePollMaxRetry,
                rst=result
            )
            if create_rst["failed"] and not self._debug:
                logger.info("Deleting %s due to creation failure", name)
                del_rst = self.stop_one(name, namespace)
                create_rst["code"] += del_rst["code"]
                create_rst["events"] += del_rst["events"]
                create_rst["duration"] = str(round(time.time() - start, 2))
            return create_rst

    @staticmethod
    def _poll_till_healthy(name, kube_obj, start_time, poll_interval, poll_max_retry, rst):
        trail = 0
        assert isinstance(kube_obj, KubeObject)
        while True:
            if not kube_obj.healthy():
                trail += 1
                if trail > poll_max_retry:

                    logger.error("Failed to create KubeObject %s", name)
                    rst["code"] += ["{:.25s}:{}".format(name, KubeObjStatusCode.UNHEALTHY)]
                    rst["events"] += ["Object {} creation timeout. Not healthy".format(name)]
                    rst["failed"] = True
                    rst["duration"] = str(round(time.time() - start_time, 2))
                    return rst
            else:
                logger.info("Successfully created %s.", name)
                rst["code"] += ["{:.25s}:{}".format(name, KubeObjStatusCode.OK)]
                rst["failed"] = False
                rst["duration"] = str(round(time.time() - start_time, 2))
                return rst
            time.sleep(poll_interval)

    def stop_one(self, name, namespace=AXNameSpaces.AXSYS):
        time.sleep(random.randint(0, AXPlatformConfigDefaults.ObjectOperationJitter))
        logger.info("Deleting %s in namespace %s ...", name, namespace)
        start = time.time()
        kube_obj = self._kube_objects[name]
        kube_obj.namespace = namespace
        kube_obj.replacing = self._replacing
        assert isinstance(kube_obj, KubeObject)

        result = {
            "name": name,
            "code": [],
            "events": [],
            "failed": False,
            "duration": ""
        }

        # Don't delete if object does not exist
        if not kube_obj.exists():
            result["code"] += ["{:.25s}:{}".format(name, KubeObjStatusCode.DELETED)]
            result["duration"] = str(round(time.time() - start, 2))
            return result

        monitor_info = kube_obj.get_delete_monitor_info()
        if monitor_info:
            # use monitor
            waiters = []

            # Create and register waiters for all objects that can be monitored
            for m in monitor_info:
                wait_info = {
                    "kind": KubeKindToKubeApiObjKind[m.kube_kind],
                    "name": m.name,
                    "validator": m.validator
                }
                waiter = KubeObjWaiter()
                waiters.append((waiter, wait_info))
                AXKubeMonitor().wait_for_kube_object(
                    wait_info,
                    AXPlatformConfigDefaults.ObjDeleteWaitTimeout,
                    waiter
                )

            # Call kubectl delete
            kube_obj.delete()

            # Wait on all waiters to retrieve status and events
            for waiter, wait_info in waiters:
                waiter.wait()
                result["events"] += waiter.details
                if waiter.result == KubeObjStatusCode.OK or waiter.result == KubeObjStatusCode.WARN:
                    result["code"].append("{:.25s}:{}".format(wait_info["name"], KubeObjStatusCode.DELETED))
                    logger.info("Successfully deleted %s in %s with code %s.", wait_info["name"], name, result["code"])
                else:
                    result["failed"] = True
                    result["code"].append("{:.25s}:{}".format(wait_info["name"], KubeObjStatusCode.UNKNOWN))
                    logger.error("Failed to delete %s in %s with code %s. Events: %s",
                                 wait_info["name"], name, result["code"], str(waiter.details))

            # Poll once to confirm all components from this Kubenetes config file exist
            # In case there are objects in this config file cannot be monitored, i.e. svc without elb
            if kube_obj.exists():
                logger.error("Object %s deleted but still exists", name)
                result["failed"] = True
                result["code"].append("{:.25s}:{}".format(name, KubeObjStatusCode.UNKNOWN))
                result["events"].append("Object {} deleted but still exists.".format(name))
            result["duration"] = str(round(time.time() - start, 2))
            logger.info("Successfully deleted %s.", name)
            return result
        else:
            # use polling
            kube_obj.delete()
            return self._poll_till_not_exists(
                name=name,
                kube_obj=kube_obj,
                start_time=start,
                poll_interval=AXPlatformConfigDefaults.ObjDeletePollInterval,
                poll_max_retry=AXPlatformConfigDefaults.ObjDeletePollMaxRetry,
                rst=result
            )

    @staticmethod
    def _poll_till_not_exists(name, kube_obj, start_time, poll_interval, poll_max_retry, rst):
        trail = 0
        assert isinstance(kube_obj, KubeObject)
        while True:
            if kube_obj.exists():
                trail += 1
                if trail > poll_max_retry:
                    logger.error("Failed to delete KubeObject %s", name)
                    rst["code"] += ["{:.25s}:{}".format(name, KubeObjStatusCode.UNKNOWN)]
                    rst["events"] += ["Object {} deletion timeout. Please manually check remaining pods".format(name)]
                    rst["failed"] = True
                    rst["duration"] = str(round(time.time() - start_time, 2))
                    return rst
            else:
                logger.info("Successfully deleted %s.", name)
                rst["code"] += ["{:.25s}:{}".format(name, KubeObjStatusCode.DELETED)]
                rst["failed"] = False
                rst["duration"] = str(round(time.time() - start_time, 2))
                return rst
            time.sleep(poll_interval)

    def _generate_object_summary(self, objects):
        """
        :param objects: list of AXPlatformObject
        :return:
        """
        report_title = "\n{:25s} |  {:110s} |  {:20s}\n".format("NAME", "MANIFEST", "NAMESPACE")
        report_bar = "{}\n".format("-" * 174)
        content = ""
        for obj in objects:
            assert isinstance(obj, AXPlatformObject)
            name = obj.name
            filename = os.path.join(self._manifest_root, obj.manifest)
            namespace = obj.namespace
            content += "{:25s} |  {:110s} |  {:20s}\n".format(name, filename, namespace)

        return report_title + report_bar + content

    @staticmethod
    def _generate_report(results, operation):
        failed = False
        report_body = ""
        warnings = "\n======= WARNING EVENTS =======\n"
        for name in results.keys():
            individual_report = "{:25s} |  {:110s} |  {:20s}\n"
            individual_warning = "{name}: {events}\n\n"
            try:
                result = results[name].get()
                if result["failed"]:
                    failed = True
                code = result["code"][0]
                for c in result["code"][1:]:
                    code += " / {}".format(c)
                individual_report = individual_report.format(name, code, result["duration"], 2)
                if len(result["events"]) > 0:
                    warnings += individual_warning.format(name=name, events=str(result["events"]))
            except Exception as e:
                failed = True
                logger.exception(str(e))
                individual_report = individual_report.format(name, "EXCEPTION", "UNKNOWN")
                warnings += individual_warning.format(name=name, events=str(e))
            report_body += individual_report

        report_head = "\n\nPlatform {} {}. Report:\n".format(operation, "FAILED" if failed else "SUCCESSFULLY")
        report_title = "\n{:25s} |  {:110s} |  {:20s}\n".format("NAME", "STATUS", "TIME (sec)")
        report_bar = "{}\n".format("-"*174)
        return "{}{}{}{}{}{}".format(report_head, report_title, report_bar, report_body,
                                     warnings, "==============================\n"), failed

    def _get_eip_from_config_map(self):
        try:
            cmd = ["kubectl", "get", "configmap", "cluster-dns-name", "-o", "yaml",
                   "--namespace", self.kube_axsys_namespace,
                   "--kubeconfig", self._kube_config]
            out = subprocess.check_output(cmd)
            return [yaml.load(out)["data"]["cluster-external-dns-name"]]
        except Exception:
            logger.error("Failed to get cluster dns name from config map.")
            return None

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
    def _get_svc_eip(self, svclabel, namespace):
        svc = self.kube_poll.poll_kubernetes_sync(KubeKind.SERVICE, namespace, svclabel)
        assert len(svc.items) == 1, "Currently services should only have one ingress"
        rst = []
        for ig in svc.items[0].status.load_balancer.ingress:
            if ig.hostname:
                rst.append(ig.hostname)
            if ig.ip:
                rst.append(ig.ip)
        return rst

    def _set_ext_dns(self):
        axops_eip = self._get_eip_from_config_map() or self._get_svc_eip(svclabel="app=axops",
                                                                         namespace=AXNameSpaces.AXSYS)

        if not axops_eip:
            logger.error("Platform Start Failed: cannot find External IP for AXOPS")
            raise AXPlatformException("AXOPS elastic IP does not exist")

        self.cluster_dns_name = axops_eip[0]
        # Don't change format of this message. Portal parses this line to get cluster IP/DNS.
        logger.info("\n\n%s>>>>> Starting Argo platform... cluster DNS: %s%s\n", COLOR_GREEN, self.cluster_dns_name,
                    COLOR_NORM)
        self._replacing["AXOPS_EXT_DNS"] = self.cluster_dns_name

    def get_cluster_external_dns(self):
        if not self.cluster_dns_name:
            self._set_ext_dns()
        return self.cluster_dns_name

    def _set_autoscaling(self):
        # Prepare autoscaler
        asg_manager = AXUserASGManager(self._cluster_name_id, self._region, self._aws_profile)
        asg = asg_manager.get_variable_asg() or asg_manager.get_spot_asg() or asg_manager.get_on_demand_asg()
        if not asg:
            raise AXPlatformException("Failed to get autoscaling group for cluster {}".format(self._cluster_name_id))
        asg_name = asg["AutoScalingGroupName"]

        if asg_name is not None:
            self._replacing["ASG_NAME"] = asg_name
        else:
            logger.error("Autoscaling group name not found for %s", self._cluster_name_id)
            raise AXPlatformException("Cannot find cluster autoscaling group")

    # TODO (#157) Version should only be uploaded during install and upgrade time
    def _update_version(self):
        # Software info we get during install / upgrade does not contain ami id
        # need to persist it as well
        self._software_info.ami_id = self._cluster_config.get_ami_id()

        AXVersion(
            AXCustomerId().get_customer_id(),
            self._cluster_name_id,
            self._aws_profile
        ).update(
            self._software_info.to_dict()
        )
