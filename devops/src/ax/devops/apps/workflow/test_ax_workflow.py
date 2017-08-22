#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2016 Applatix, Inc. All rights reserved.
#

"""
Module for test AXWorkflow
"""

import json
import os

from ax.devops.apps.workflow.ax_workflow import AXWorkflow

test_db_dict = {}
test_db_file = "test_db_file"


def load_test_db():
    global test_db_dict
    if os.path.exists(test_db_file):
        fd = os.open(test_db_file, os.O_RDONLY | (hasattr(os, "O_SHLOCK") and os.O_SHLOCK))
        test_db_dict = json.loads(os.read(fd, 1024*1024*1024).decode('utf-8'))
        os.close(fd)
    else:
        test_db_dict = {}


def save_test_db():
    global test_db_dict
    fd = os.open(test_db_file, os.O_WRONLY | os.O_CREAT | (hasattr(os, "O_EXLOCK") and os.O_EXLOCK) |os.O_TRUNC)
    os.write(fd, json.dumps(test_db_dict).encode('utf-8'))
    os.close(fd)


def post_workflow_to_db(workflow):
    global test_db_dict
    load_test_db()
    if workflow._workflow_id in test_db_dict:
        return False
    timestamp = AXWorkflow.get_current_epoch_timestamp_in_ms()
    test_db_dict[workflow._workflow_id] = {
        "service_template": workflow.service_template,
        "status": workflow.status,
        "resource": workflow.resource,
        "sn": workflow.sn,
        "timestamp": timestamp
    }
    save_test_db()
    workflow.set_timestamp(timestamp)
    return True


def update_workflow_status_in_db(workflow, new_status):
    global test_db_dict
    load_test_db()
    if workflow._workflow_id not in test_db_dict:
        return False
    if test_db_dict[workflow._workflow_id]["status"] != workflow.status:
        return False
    timestamp = AXWorkflow.get_current_epoch_timestamp_in_ms()
    test_db_dict[workflow._workflow_id]["status"] = new_status
    test_db_dict[workflow._workflow_id]["timestamp"] = timestamp
    workflow.set_timestamp(timestamp)
    save_test_db()
    return True


def get_workflow_by_id_from_db(workflow_id, need_load_template=False):
    global test_db_dict
    load_test_db()
    if workflow_id not in test_db_dict:
        return None
    value = test_db_dict[workflow_id]
    if need_load_template:
        st = value["service_template"]
    else:
        st = None
    w = AXWorkflow(workflow_id=workflow_id,
                   service_template=st,
                   status=value["status"],
                   resource=value["resource"],
                   timestamp=value.get("timestamp", None),
                   sn=value["sn"])

    return w


def get_workflows_by_status_from_db(status):
        global test_db_dict
        load_test_db()
        ret = []
        for key, value in test_db_dict.items():
            if value["status"] == status:
                ret.append(AXWorkflow(workflow_id=key,
                                      service_template=None,
                                      status=value["status"],
                                      resource=value["resource"],
                                      timestamp=value.get("timestamp", None),
                                      sn=value["sn"]))

        return ret
