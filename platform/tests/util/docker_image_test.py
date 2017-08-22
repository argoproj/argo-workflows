#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import unittest
from nose.tools import assert_raises

from ax.util.docker_image import DockerImage


class DockerImageTest(unittest.TestCase):
    @staticmethod
    @unittest.skip("Stale test not needed")
    def test_docker_image():
        image = DockerImage(fullname="ubuntu")
        assert "ubuntu:latest" == image.full_name()
        assert ("", "", "ubuntu", "latest") == image.ax_names()
        assert ("", "ubuntu", "latest") == image.docker_names()
        assert "ubuntu" == image.docker_repo()

        image = DockerImage(fullname="docker.example.com/ubuntu")
        assert "docker.example.com/ubuntu:latest" == image.full_name()
        assert ("docker.example.com", "", "ubuntu", "latest") == image.ax_names()
        assert ("docker.example.com", "ubuntu", "latest") == image.docker_names()
        assert "docker.example.com/ubuntu" == image.docker_repo()

        image = DockerImage(fullname="docker.example.com/ax/foo")
        assert "docker.example.com/ax/foo:latest" == image.full_name()
        assert ("docker.example.com", "ax", "foo", "latest") == image.ax_names()
        assert ("docker.example.com", "ax/foo", "latest") == image.docker_names()
        assert "docker.example.com/ax/foo" == image.docker_repo()

        image = DockerImage(fullname="docker.example.com/ax/foo:v1.2.3")
        assert "docker.example.com/ax/foo:v1.2.3" == image.full_name()
        assert ("docker.example.com", "ax", "foo", "v1.2.3") == image.ax_names()
        assert ("docker.example.com", "ax/foo", "v1.2.3") == image.docker_names()
        assert "docker.example.com/ax/foo" == image.docker_repo()

        image = DockerImage(registry="docker.local", namespace="$(AX_NAMESPACE)", name="foo",
                            version="$(AX_IMAGE_VERSION)")
        assert "docker.local/$(AX_NAMESPACE)/foo:$(AX_IMAGE_VERSION)" == image.full_name()
        assert ("docker.local", "$(AX_NAMESPACE)", "foo", "$(AX_IMAGE_VERSION)") == image.ax_names()
        assert ("docker.local", "$(AX_NAMESPACE)/foo", "$(AX_IMAGE_VERSION)") == image.docker_names()
        assert "docker.local/$(AX_NAMESPACE)/foo" == image.docker_repo()

        # Can't specify both fullname and others
        with assert_raises(Exception):
            DockerImage(fullname="abc", registry="xyz")

        # Registry must be FQDN
        with assert_raises(Exception):
            DockerImage(registry="abc")

        # Name can't be empty
        with assert_raises(Exception):
            DockerImage(registry="abc.xyz", version="1")
