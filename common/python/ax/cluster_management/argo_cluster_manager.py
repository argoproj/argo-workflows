# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import argparse
import logging
import os
import sys
import uuid

from ax.cloud import Cloud
from ax.cloud.aws import SecurityToken
from ax.platform.cluster_infra import get_host_ip
from ax.util.const import COLOR_NORM, COLOR_RED
from .app import ClusterInstaller, ClusterPauser, ClusterResumer, ClusterUninstaller, ClusterUpgrader, \
    CommonClusterOperations, PlatformOnlyInstaller
from .app.options import add_install_flags, add_platform_only_flags, add_platform_only_uninstall_flags, ClusterInstallConfig, \
    add_pause_flags, ClusterPauseConfig, add_restart_flags, PlatformOnlyInstallConfig, ClusterRestartConfig, \
    add_uninstall_flags, ClusterUninstallConfig, add_upgrade_flags, ClusterUpgradeConfig, \
    add_misc_flags, ClusterMiscOperationConfig

import subprocess
import requests
from retrying import retry

from kubernetes import client, config
from ax.kubernetes.client import KubernetesApiClient
from kubernetes.client.rest import ApiException

logger = logging.getLogger(__name__)

S3PROXY_NAMESPACE = "axs3"


class ArgoClusterManager(object):
    def __init__(self):
        self._parser = None

    def add_flags(self):
        self._parser = argparse.ArgumentParser(description="Argo cluster management",
                                               formatter_class=argparse.ArgumentDefaultsHelpFormatter)
        main_subparser = self._parser.add_subparsers(dest="command")

        # Add install cluster flags
        install_parser = main_subparser.add_parser("install", help="Install Argo cluster")
        add_install_flags(install_parser)

        # Add pause cluster flags
        pause_parser = main_subparser.add_parser("pause", help="Pause Argo cluster")
        add_pause_flags(pause_parser)

        # Add restart cluster flags
        restart_parser = main_subparser.add_parser("resume", help="Resume Argo cluster")
        add_restart_flags(restart_parser)

        # Add uninstall cluster flags
        uninstall_parser = main_subparser.add_parser("uninstall", help="Uninstall Argo cluster")
        add_uninstall_flags(uninstall_parser)

        # Add upgrade cluster flags
        upgrade_parser = main_subparser.add_parser("upgrade", help="Upgrade Argo cluster")
        add_upgrade_flags(upgrade_parser)

        # Add download credential flags
        download_cred_parser = main_subparser.add_parser("download-cluster-credentials", help="Download Argo cluster credentials")
        add_misc_flags(download_cred_parser)

        # Install on existing cluster
        platform_only_installer = main_subparser.add_parser("install-argo-only", help="Install Argo only")
        add_platform_only_flags(platform_only_installer)

        # Uninstall on existing cluster
        platform_only_uninstaller = main_subparser.add_parser("uninstall-argo-only", help="Uninstall Argo services")
        add_platform_only_uninstall_flags(platform_only_uninstaller)

    def parse_args_and_run(self):
        assert isinstance(self._parser, argparse.ArgumentParser), "Please call add_flags() to initialize parser"
        args = self._parser.parse_args()
        if not args.command:
            self._parser.print_help()
            return

        try:
            cmd = args.command.replace("-", "_")
            getattr(self, cmd)(args)
        except NotImplementedError as e:
            self._parser.error(e)
        except Exception as e:
            logger.exception(e)
            print("\n{} !!! Operation failed due to runtime error: {} {}\n".format(COLOR_RED, e, COLOR_NORM))

    def install(self, args):
        install_config = ClusterInstallConfig(cfg=args)
        install_config.default_or_wizard()
        err = install_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(install_config.cloud_profile)
        ClusterInstaller(install_config).start()

    def pause(self, args):
        pause_config = ClusterPauseConfig(cfg=args)
        pause_config.default_or_wizard()
        err = pause_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(pause_config.cloud_profile)
        ClusterPauser(pause_config).start()

    def resume(self, args):
        resume_config = ClusterRestartConfig(cfg=args)
        resume_config.default_or_wizard()
        err = resume_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(resume_config.cloud_profile)
        ClusterResumer(resume_config).start()

    def uninstall(self, args):
        uninstall_config = ClusterUninstallConfig(cfg=args)
        uninstall_config.default_or_wizard()
        err = uninstall_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(uninstall_config.cloud_profile)
        ClusterUninstaller(uninstall_config).start()

    def download_cluster_credentials(self, args):
        config = ClusterMiscOperationConfig(cfg=args)
        config.default_or_wizard()
        err = config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(config.cloud_profile)
        if config.dry_run:
            logger.info("DRY RUN: downloading credentials for cluster %s.", config.cluster_name)
            return
        ops = CommonClusterOperations(
            input_name=config.cluster_name,
            cloud_profile=config.cloud_profile
        )
        ops.cluster_info.download_kube_config()
        ops.cluster_info.download_kube_key()

    def upgrade(self, args):
        upgrade_config = ClusterUpgradeConfig(cfg=args)
        upgrade_config.default_or_wizard()
        err = upgrade_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(upgrade_config.cloud_profile)
        ClusterUpgrader(upgrade_config).start()

    def _set_env_if_present(self, args):
        try:
            os.environ["AX_AWS_REGION"] = args.cloud_region
        except Exception:
            pass

        try:
            os.environ["ARGO_S3_ACCESS_KEY_ID"] = args.access_key
        except Exception:
            pass

        try:
            os.environ["ARGO_S3_ACCESS_KEY_SECRET"] = args.secret_key
        except Exception:
            pass

        try:
            os.environ["ARGO_S3_ENDPOINT"] = args.bucket_endpoint
        except Exception:
            pass

    def _get_s3_proxy_port(self, kubeconfig):
        k8s = KubernetesApiClient(config_file=kubeconfig)
        resp = k8s.api.list_namespaced_service(S3PROXY_NAMESPACE)
        for i in resp.items:
            if i.metadata.name == "s3proxy":
                return i.spec.ports[0].node_port
        return None

    def _get_s3_proxy_endpoint(self, kubeconfig):
        host = get_host_ip(kubeconfig)
        port = self._get_s3_proxy_port(kubeconfig)
        return "http://" + host + ":" + str(port)

    def _install_s3_proxy(self, kube_config):
        logger.info("Installing s3 proxy ...")
        self._install_s3_proxy_namespace(kube_config)
        self._install_s3_proxy_pvc(kube_config)
        self._install_s3_proxy_service(kube_config)
        self._install_s3_proxy_pod(kube_config)

    def _install_s3_proxy_namespace(self,kube_config):
        k8s = KubernetesApiClient(config_file=kube_config)
        resp = k8s.api.list_namespace()
        for i in resp.items:
            if i.metadata.name == S3PROXY_NAMESPACE:
                return None
        subprocess.check_call(["kubectl", "--kubeconfig", kube_config, "create", "-f", "/ax/config/service/argo-wfe/axs3-namespace.yml"])


    def _install_s3_proxy_pvc(self,kube_config):
        k8s = KubernetesApiClient(config_file=kube_config)
        resp = k8s.api.list_namespaced_persistent_volume_claim(S3PROXY_NAMESPACE)
        for i in resp.items:
            if i.metadata.name == "s3proxy-pvc":
                return None
        subprocess.check_call(["kubectl", "--kubeconfig", kube_config, "create", "--namespace", S3PROXY_NAMESPACE, "-f", "/ax/config/service/argo-wfe/s3proxy-pvc.yml"])

    def _install_s3_proxy_service(self,kube_config):
        k8s = KubernetesApiClient(config_file=kube_config)
        resp = k8s.api.list_namespaced_service(S3PROXY_NAMESPACE)
        for i in resp.items:
            if i.metadata.name == "s3proxy":
                return None
        subprocess.check_call(["kubectl", "--kubeconfig", kube_config, "create", "--namespace", S3PROXY_NAMESPACE, "-f", "/ax/config/service/argo-wfe/s3proxy-svc.yml"])

    def _install_s3_proxy_pod(self,kube_config):
        k8s = KubernetesApiClient(config_file=kube_config)
        resp = k8s.api.list_namespaced_pod(S3PROXY_NAMESPACE)
        for i in resp.items:
            if str(i.metadata.name).startswith("s3proxy-deployment"):
                return None
        subprocess.check_call(["kubectl", "--kubeconfig", kube_config, "create", "--namespace", S3PROXY_NAMESPACE, "-f",
                                       "/ax/config/service/argo-wfe/s3proxy.yml"])

    @retry(wait_exponential_multiplier=3000, stop_max_attempt_number=5)
    def _create_s3_proxy_bucket(self, endpoint, bucket_name):
        location = endpoint + "/" + bucket_name
        logger.info("Creating s3 bucket using location: %s", location)
        requests.put(location)

    def install_argo_only(self, args):
        logger.info("Installing Argo platform ...")

        try:
            assert args.cluster_name
        except Exception:
            print("--cluster-name needs to be specified")
            sys.exit(1)

        if args.cloud_provider == "minikube" and not args.bucket_endpoint:
            Cloud(target_cloud="aws")
            args.cluster_bucket = "argo"
            # TODO:revisit
            # access key and secret is required by code in aws_s3
            # use dummy access key and secret for s3proxy
            args.access_key = "fake-access-key"
            args.secret_key = "fake-secret-key"
            self._install_s3_proxy(args.kubeconfig)
            args.bucket_endpoint = self._get_s3_proxy_endpoint(args.kubeconfig)
            # Create bucket
            self._create_s3_proxy_bucket(args.bucket_endpoint, args.cluster_bucket)
        elif args.cloud_provider == "aws":
            assert args.cluster_bucket, "--cluster-bucket is required"
            assert args.cloud_region, "--cloud-region is required"

        logger.info("s3 bucket endpoint: %s", args.bucket_endpoint)

        os.environ["AX_CUSTOMER_ID"] = "user-customer-id"
        os.environ["ARGO_LOG_BUCKET_NAME"] = args.cluster_bucket
        os.environ["ARGO_DATA_BUCKET_NAME"] = args.cluster_bucket
        os.environ["ARGO_KUBE_CONFIG_PATH"] = args.kubeconfig
        os.environ["AX_TARGET_CLOUD"] = Cloud.CLOUD_AWS

        self._set_env_if_present(args)
        platform_install_config = PlatformOnlyInstallConfig(cfg=args)
        PlatformOnlyInstaller(platform_install_config).run()
        return

    def uninstall_argo_only(self, args):
        logger.info("Uninstalling Argo platform ...")
        config.load_kube_config(args.kubeconfig)
        api = client.CoreV1Api()
        for namespace in ["axuser", "axsys", "axs3"]:
            try:
                api.delete_namespace(namespace, client.V1DeleteOptions())
            except ApiException as ae:
                if ae.status == 404:
                    pass
                else:
                    raise

        logger.info("Done!")
        return

    @staticmethod
    def _ensure_customer_id(cloud_profile):
        if os.getenv("AX_CUSTOMER_ID", None):
            logger.info("Using customer ID %s", os.getenv("AX_CUSTOMER_ID"))
            return

        # TODO (#111): set customer id to GCP
        if Cloud().target_cloud_aws():
            account_info = SecurityToken(aws_profile=cloud_profile).get_caller_identity()
            customer_id = str(uuid.uuid5(uuid.NAMESPACE_OID, account_info["Account"]))
            logger.info("Using AWS account ID hash (%s) for customer id", customer_id)
            os.environ["AX_CUSTOMER_ID"] = customer_id

    @staticmethod
    def _continue_or_die(err):
        if err:
            print("\n{}====== Errors:\n".format(COLOR_RED))
            for e in err:
                print(e)
            print("\n!!! Operation failed due to invalid inputs{}\n".format(COLOR_NORM))
            sys.exit(1)
