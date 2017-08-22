# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import logging
import os

from .common import ProdBuilder, SRC_PATH, run_cmd, ARGO_BASE_REGISTRY

logger = logging.getLogger('ax.build')

PLATFORM_BUILDER_IMAGE = '{}/argobase/axplatbuilder:v15'.format(ARGO_BASE_REGISTRY)

PRODUCTION_SERVICES = [
    "axmon",
    "axstats",
    "axconsole",
    "test",
    "axnotification",
    "axclustermanager",
    "kube-init",
    "fluentd",
    "artifacts",
    "volume-mounts-fixer",
    "master-manager",
    "minion-manager",
    "applet",
    "prometheus",
    "node-exporter",
    "managedlb"
]

BASE_CONTAINERS = [
    'axplatbuilder',
    'axplatbuilder-debian'
]

class PlatformBuilder(ProdBuilder):

    def __init__(self, **kwargs):
        super(PlatformBuilder, self).__init__(python_version=2, **kwargs)
        self.builder_image = kwargs.get('builder') or PLATFORM_BUILDER_IMAGE
        if kwargs.get('no_builder'):
            logger.info("BUILDING base images first {}".format(BASE_CONTAINERS))
            bak_services = self.services
            self.services = BASE_CONTAINERS
            # call the superclass method to avoid the builder freeze image dependency
            super(PlatformBuilder, self).build()
            logger.info("Base images built successfully")
            self.builder_image = "%s/%s/%s:%s" % (self.registry, self.image_namespace, "axplatbuilder", self.image_version)
            self.services = bak_services

    def get_default_services(self):
        return PRODUCTION_SERVICES

    def get_container_build_path(self, container):
        return os.path.join(SRC_PATH, "platform/containers", container)

    def build(self):
        # Output the frozen pip requirements to the logs so we record this
        logger.info("Frozen requirements:")
        # ensure that we always pull the latest builder image
        run_cmd("docker pull {}".format(self.builder_image))
        run_cmd("docker run --rm {} pip2 freeze".format(self.builder_image))
        return super(PlatformBuilder, self).build()

    def build_one(self, container, **kwargs):
        builder_image = self.builder_image
        if container in ['artifacts']:
            image, version = builder_image.split(':')
            builder_image = '{}-debian:{}'.format(image, version)
        return super(PlatformBuilder, self).build_one(container, builder_image=builder_image, **kwargs)

    @staticmethod
    def argparser():
        parser = super(PlatformBuilder, PlatformBuilder).argparser()
        grp = parser.add_mutually_exclusive_group()
        grp.add_argument('--builder', default=PLATFORM_BUILDER_IMAGE,
                            help='Use a supplied builder image instead of default ({})'.format(PLATFORM_BUILDER_IMAGE))
        grp.add_argument('--no-builder', action='store_true', help='Do not use a prebuilt base image')
        args = parser.parse_args()

        return parser

