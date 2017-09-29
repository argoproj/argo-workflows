"""The file has unit tests for the install_options."""

import argparse
import unittest

from ax.cluster_management.app.options import PlatformOnlyInstallConfig


class InstallOptionsTest(unittest.TestCase):
    """
    Tests for the InstallOptionsTest
    """
    def add_default_options(self, cfg):
        cfg.cluster_bucket = "argo-bucket"
        cfg.cluster_name = "argo-cluster"
        cfg.cluster_id = "argo-cluster-id"
        cfg.kubeconfig = "/tmp/kconfig"
        cfg.cloud_provider = "minikube"
        cfg.cloud_profile = "default"
        cfg.service_manifest_root = "/tmp/service_manifest_root"
        cfg.platform_bootstrap_config = cfg.service_manifest_root + "/platform-bootstrap.cfg"
        cfg.silent = False
        cfg.dry_run = False

        cfg.cloud_region = "us-west-2"
        cfg.cloud_placement = "us-west-2a"

    def test_platform_only_install_basic(self):
        cfg = argparse.ArgumentParser()
        self.add_default_options(cfg)

        # Explicitly set the provider to aws and verify that the manifest root and bootstrap_config are set
        # correctly.
        cfg.cloud_provider = "aws"
        p1 = PlatformOnlyInstallConfig(cfg)
        assert p1.service_manifest_root == "/tmp/service_manifest_root"
        assert p1.platform_bootstrap_config == p1.service_manifest_root + "/platform-bootstrap.cfg"

        # Same as above but with minikube
        cfg.cloud_provider = "minikube"
        p2 = PlatformOnlyInstallConfig(cfg)
        assert p2.service_manifest_root == "/ax/config/service/basic"
        assert p2.platform_bootstrap_config == "/ax/config/service/config/basic-platform-bootstrap.cfg"

