#!/usr/bin/env python

"""
When the Python SDK was migrated to the Argo Workflows repository, we kept the `argo-workflows` name for the Python
package as we wanted to publish it to the same PyPi project on the public index. However, the existing `argo-workflows`
package was on version 5.0.0 already, while Argo Workflows was on 3.x.x. We wanted to publish the SDK as version 6.0.0
to indicate backwards incompatibility. So, this script takes the Argo Workflows tag, when a new release is created,
takes the major version, adds 3 to it, and prints to stdout the new version, which is:
- ARGO_MAJOR+3.ARGO_MINOR.ARGO_PATCH
"""

import os
import re

VERSION_PREFIX = 'v'
VERSION_INCREMENT = 3
MAJOR_VERSION_INDEX = 0
UNTAGGED = 'untagged'

FAILED = 'FAILED'  # indicator captured by the makefile to know when something failed
UNTAGGED_VERSION = '0.0.0-pre'
git_tag_cmd = 'git describe --exact-match --tags --abbrev=0 2> /dev/null || echo untagged'
try:
    git_tag = os.popen(git_tag_cmd).read().strip()
    if git_tag == UNTAGGED:
        print(UNTAGGED_VERSION)  # this goes to sys.stdout, so it's captured by the Makefile
        exit(0)

    rc_version_suffix = re.findall("-.*", git_tag)
    if len(rc_version_suffix) > 0:
        git_tag = git_tag.replace(rc_version_suffix[0], '')
    version_digits = [int(i) for i in git_tag.replace(VERSION_PREFIX, '').split('.')]
    version_digits[MAJOR_VERSION_INDEX] += VERSION_INCREMENT

    version = '.'.join([str(i) for i in version_digits])
    if len(rc_version_suffix) > 0:
        version += rc_version_suffix[0]
    print(version)
    exit(0)
except Exception as e:
    raise e
