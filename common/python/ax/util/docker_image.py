#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module to parse and check docker container image names.
"""

class DockerImage(object):
    def __init__(self, fullname=None, registry=None, namespace=None, name=None, version=None):
        """
        Initialize and check image name syntax.
        Can only specify either fullname or others.
        Always break it down to four components and then assemble back to correct format.

        :param fullname: Fullname
        :param registry: FQDN registry name
        :param namespace: First component after registry and separated by "/", if any
        :param name: short name for image, could possibly container more "/"
        :param version: version after ":". Default to "latest" if not set.
        """
        if fullname:
            assert not any([registry, namespace, name, version]), "Can't specify others with full name"
            self._registry, self._namespace, self._name, self._version = self._parse_fullname(fullname)

        else:
            if registry:
                assert "." in registry, "Registry must have . {}".format(registry)
            if namespace:
                assert "/" not in namespace, "Namespace can't have / {}".format(namespace)
            assert name, "Name can't be empty {}".format(namespace)
            version = version if version else "latest"

            self._registry = registry
            self._namespace = namespace
            self._name = name
            self._version = version
        self._fullname = self._make_fullname(self._registry, self._namespace, self._name, self._version)

    def full_name(self):
        """
        Return correct full name for image.
        """
        return self._fullname

    def docker_names(self):
        """
        Return three tuple name components in docker format.

        Docker API requires different format: registry, repo and tag.
        This is registry, namespace/name, version.
        """
        new_name = "/".join([self._namespace, self._name]) if self._namespace else self._name
        return self._registry, new_name, self._version

    def docker_repo(self):
        """
        Return full repo name in docker format.

        Docker full repo is registry/namespace/name without version.
        """
        return self._fullname.partition(":")[0]

    def ax_names(self):
        """
        Return tuple of all four components in AX format.
        """
        return self._registry, self._namespace, self._name, self._version

    @staticmethod
    def _parse_fullname(fullname):
        has_registry = "/" in fullname and "." in fullname.split("/")[0]
        if has_registry:
            registry, _, part2 = fullname.partition("/")
        else:
            registry = "docker.io"
            part2 = fullname
        name, _, version = part2.partition(":")
        if "/" in name:
            namespace, _, name = name.partition("/")
        else:
            namespace = ""
        version = version if version else "latest"
        return registry, namespace, name, version

    @staticmethod
    def _make_fullname(registry, namespace, name, version):
        full = ""
        if registry:
            full += registry
            full += "/"
        if namespace:
            full += namespace
            full += "/"
        full += name
        full += ":"
        full += version
        return full
