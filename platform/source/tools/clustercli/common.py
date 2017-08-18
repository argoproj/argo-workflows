# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import re

from anytree import Node
from .cli import AbstractPrompt
from ax.cluster_management.app.options import ClusterOperationDefaults
from ax.cluster_management.app.options import ClusterInstallDefaults


class CommonPrompts(AbstractPrompt):

    INSTALLCMD = "install"
    UNINSTALLCMD = "uninstall"

    def __init__(self):
        self.aws_profiles = self._get_profiles()
        self._profile_names = [x for x in self.aws_profiles]
        def_profile = ClusterOperationDefaults.CLOUD_PROFILE if ClusterOperationDefaults.CLOUD_PROFILE in self._profile_names else self._profile_names[0]


        self.root = Node("name",
                    prompt=u'Enter a cluster name',
                    validator=u'^[a-zA-Z][a-zA-Z0-9-_]{1,20}$',
                    help=u'This is a friendly name for your cluster. Minimum 2 characters, Maximum 20. Only a-z, A-Z, 0-9, - and _ allowed. First character needs to be a letter',
                    default=ClusterInstallDefaults.CLUSTER_NAME
                    )

        profileNode = Node("profile",
                           prompt=u'Enter the AWS profile to use',
                           values=self._profile_names,
                           default=unicode(def_profile),
                           help=u'Select an AWS profile to install the cluster',
                           parent=self.root
                           )

    def get_root(self):
        return self.root

    @staticmethod
    def _get_profiles():
        from boto3.session import Session
        s = Session()
        return s._session.full_config["profiles"]
