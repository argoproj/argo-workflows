#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

#
# Insert builtin yaml workflows into the database
#

from gevent import monkey
monkey.patch_all()

import argparse
import os
import requests

from retrying import retry

from ax.platform.component_config import SoftwareInfo
from ax.version import __version__
from ax.util.macro import macro_replace


@retry(wait_exponential_multiplier=1000, stop_max_attempt_number=10)
def post_template(path, template):
    """
    This function posts the template to axops
    """
    print("Trying to post template {} to axops".format(path))
    url = "http://axops-internal.axsys:8085/v1/yamls"
    data = {
        "repo": "ax-builtin",
        "branch": "ax-builtin",
        "revision": "v1",
        "files": [
            {
                "path": path,
                "content": template
            }
        ]
    }
    response = requests.post(url=url, json=data, timeout=120)
    print("Got the following response {}".format(response.json()))
    response.raise_for_status()
    print("Posted successfully to axops {}".format(response.json()))
    return


def load_templates_from_dir(dir_name):
    """
    This function loads builtin templates from dir and returns each template as it loads it.
    This is a generator function.
    Args:
        dir: dir to look for template files
    """
    for curr_dir, _, files in os.walk(dir_name):
        for file in files:
            path = "{}/{}".format(curr_dir, file)
            with open(path) as f:
                data = f.read()
                yield path, data


def modify_templates(template, replacements):
    """
    This function returns a modified template based on replacements
    Args:
        template: A stringified template
        replacements: a dict of string: string
    Returns:
        stringified modified template
    """
    return macro_replace(template, replacements)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Managed ELB creator')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--templates', help='Path to templates', default=".")
    args = parser.parse_args()

    dir_name = args.templates

    software_info = SoftwareInfo()
    replacements = {
        "REGISTRY": software_info.registry,
        "NAMESPACE": software_info.image_namespace,
        "VERSION": software_info.image_version
    }
    print("Macro replacements are {}".format(replacements))

    for path, template in load_templates_from_dir(dir_name):
        print("Processing template {}".format(path))
        mod_template = modify_templates(template, replacements)
        post_template(path, mod_template)

