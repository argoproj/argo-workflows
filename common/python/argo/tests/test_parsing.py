#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import copy
import json
import pytest

from argo.template.v1.container import ContainerTemplate
from argo.template.v1.deployment import RollingUpdateStrategy

CONTAINER_IMAGE = "ubuntu:latest"
CPU_CORES = 0.5
MEM_MIB = 512

base_container_dict = \
{
    "image": CONTAINER_IMAGE,
    "resources": {
        "cpu_cores": CPU_CORES,
        "mem_mib": MEM_MIB
    }
}


def check_val(obj, field, expected):
    actual = getattr(obj, field, None)
    assert expected == actual, "Incorrect {}.{} expected {} got {}".format(obj.__class__, field, expected, actual)


def test_container_no_error():
    """
    This test is for a simple container with no errors
    """
    c = ContainerTemplate()
    c.parse(base_container_dict)
    check_val(c, "image", CONTAINER_IMAGE)
    check_val(c.resources, "cpu_cores", CPU_CORES)
    check_val(c.resources, "mem_mib", MEM_MIB)


def test_container_no_image():
    """
    This test is expected to raise ValueError as image and resources are not specified
    """
    c = ContainerTemplate()
    with pytest.raises(ValueError):
        c.parse({})


cont_dict_with_inputs = copy.deepcopy(base_container_dict)
cont_dict_with_inputs.update(
{
    "inputs": {
        "parameters": {
            "param1": {},
            "param2": {
                "default": 10
            }
        }
    }
}
)


def test_container_with_inputs():
    """
    This test checks input params and their default values
    """
    c = ContainerTemplate()
    check_val(c.inputs, "parameters", {})
    c.parse(cont_dict_with_inputs)
    check_val(c.inputs.parameters["param1"], "default", None)
    check_val(c.inputs.parameters["param2"], "default", 10)

    d = c.inputs.parameters["param2"].to_dict()
    assert d["default"] == 10, "Conversion to dict has some error"

    d = c.to_dict()
    assert d["inputs"]["parameters"]["param2"]["default"] == 10, "Conversion to dict has some error {}".format(json.dumps(d))


cont_dict_with_env  = copy.deepcopy(cont_dict_with_inputs)
cont_dict_with_env.update({"env": [{"name": "FOO", "value": "BAR"}, {"name": "BAR"}]})


def test_container_with_env():
    """
    This test checks env variables
    """
    c = ContainerTemplate()
    c.parse(cont_dict_with_env)
    check_val(c.env[0], "name", "FOO")
    check_val(c.env[0], "value", "BAR")
    check_val(c.env[1], "name", "BAR")
    check_val(c.env[1], "value", "")


cont_dict_with_art = copy.deepcopy(cont_dict_with_inputs)
cont_dict_with_art.update(
{
    "inputs": {
        "artifacts": {
            "art0": {
                "from": "fake",
                "path": "/fake"
            }
        }
    }
}
)


def test_container_with_input_artifact():
    c = ContainerTemplate()
    c.parse(cont_dict_with_art)
    check_val(c.inputs.artifacts["art0"], "from_loc", "fake")
    check_val(c.inputs.artifacts["art0"], "path", "/fake")
    d = c.to_dict()
    assert d["inputs"]["artifacts"]["art0"] == cont_dict_with_art["inputs"]["artifacts"]["art0"], "Error in dict conversion"


def test_rolling_update_int_conversion():
    r = RollingUpdateStrategy()
    data = {
        "max_surge": "2",
        "max_unavailable": "2"
    }
    r.parse(data)
    assert r.max_surge == 2, "Conversion error"
    assert r.max_unavailable == 2, "Conversion error"

    data_back = r.to_dict()
    assert data_back["max_surge"] == "2", "dict conversion error"
    assert data_back["max_unavailable"] == "2", "dict conversion error"


def test_annotations_docker_spec():

    cont_dict_with_art.update(
        {
            "annotations": {
                "ax_ea_docker_enable": '{"graph-storage-size": "20Gi", "cpu_cores": 1, "mem_mib": 512}',
                "ax_ea_graph_storage_volume": '{"graph-storage-size": "20Gi", "mount-path": "/var/lib/docker"}',
                "ax_ea_privileged": 'true',
                "ax_ea_executor": '{"disable": true}',
                "ax_ea_hostname": "somelabel"
            }
        }
    )

    c = ContainerTemplate()
    c.parse(cont_dict_with_art)
    check_val(c.docker_spec, "graph_storage_size_mib", 20*1024)
    check_val(c.docker_spec, "cpu_cores", 1)
    check_val(c.docker_spec, "mem_mib", 512)
    check_val(c.graph_storage, "graph_storage_size_mib", 20*1024)
    check_val(c.graph_storage, "mount_path", "/var/lib/docker")
    check_val(c, "privileged", True)
    check_val(c.executor_spec, "disable", True)
    check_val(c, "hostname", "somelabel")
