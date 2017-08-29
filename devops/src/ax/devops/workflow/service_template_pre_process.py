#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
AX service API backend implementation.
Support Marathon only.
"""

from ax.util.az_patch import az_patch
az_patch()

import argparse
import copy
import json
import logging
import os

from ax.devops.utility.axjson import substitute

logger = logging.getLogger(__name__)


def _get_service_context(service_template, leaf_id, root_id, parent_id=None, my_name=None):
    if service_template is None:
        return None
    current_id = service_template.get("id", None)
    if current_id == leaf_id:
        return {"parent_service_instance_id": parent_id,
                "leaf_name": my_name,
                "service_instance_id": leaf_id,
                "root_workflow_id": root_id}

    template = service_template.get("template", None)
    if isinstance(template, dict):
        for st in ["steps", "fixtures"]:
            steps = template.get(st, None)
            if isinstance(steps, list):
                for step in steps:
                    if isinstance(step, dict):
                        for key, value in step.items():
                            ret = _get_service_context(service_template=value, leaf_id=leaf_id, root_id=root_id,
                                                       parent_id=current_id, my_name=key)
                            if ret is not None:
                                return ret
    return None


def _construct_full_path(parent_name, name):
    if name:
        if parent_name:
            return "{}.{}".format(parent_name, name)
        else:
            return name
    else:
        return parent_name


def _find_artifact_aliases(service_template, service_instance_id, artifact_name, full_path):
    result = []
    if service_template is None:
        return result
    current_id = service_template.get("id", None)
    template = service_template.get("template", None)
    parameters = service_template.get("parameters", {})
    if isinstance(template, dict):
        if current_id is not None:
            outputs = template.get("outputs", None)
            if isinstance(outputs, dict):
                artifacts = outputs.get("artifacts", None)
                if isinstance(artifacts, dict):
                    for key, value in artifacts.items():
                        service_id, name = get_artifact_sid_and_name(value)
                        # service_id = value.get("service_id", None)
                        # name = value.get("name", None)
                        match = False
                        if isinstance(service_id, str) and \
                                isinstance(name, str) and artifact_name == name:
                            if service_id == service_instance_id:
                                match = True
                            for para_key, para_value in parameters.items():
                                service_id_sub = service_id.replace("%%{}%%".format(para_key), para_value)
                                if service_id_sub == service_instance_id:
                                    match = True
                                    break
                        if match:
                            result.append({"service_instance_id": current_id,
                                           "artifact_name": key,
                                           "full_path": full_path})

        for st in ["steps", "fixtures", "volumes"]:
            steps = template.get(st, None)
            if isinstance(steps, list):
                for step in steps:
                    if isinstance(step, dict):
                        for key, value in step.items():
                            ret = _find_artifact_aliases(service_template=value,
                                                         service_instance_id=service_instance_id,
                                                         artifact_name=artifact_name,
                                                         full_path=_construct_full_path(full_path, key))
                            result += ret

    return result


def _find_artifact_aliases_deep(service_template, service_instance_id, artifact_name, full_path):
    result = _find_artifact_aliases(service_template, service_instance_id, artifact_name, full_path)
    result2 = []
    for a in result:
        result2 += _find_artifact_aliases_deep(service_template,
                                               service_instance_id=a["service_instance_id"],
                                               artifact_name=a["artifact_name"],
                                               full_path=full_path)
    return result + result2


def get_artifact_sid_and_name(art):
    if isinstance(art, dict):
        f = art.get("from", None)
        if f:
            # example: "from": "%%service.d0973bcc-6ba0-11e7-ba1c-0a58c0a8811c.outputs.artifacts.BIN-OUTPUT%%"
            s1 = f.strip("%").split(".")
            if s1[0] == "service" and s1[2] == "outputs" and s1[3] == "artifacts":
                return s1[1], s1[4]
            else:
                logger.warning("bad artifact %s", art)
        else:
            service_instance_id = art.get("service_instance_id", None)
            if service_instance_id is None:
                service_instance_id = art.get("service_id", None)
            name = art.get("name", None)
            return service_instance_id, name

    return None, None


def service_template_pre_process(service_template_root, leaf, parameter, name, full_path):
    leaf = copy.deepcopy(leaf)
    my_id = leaf.get("id", None)
    if "service_context" not in leaf:
        logger.debug("generate service_context")
        service_context = _get_service_context(service_template=service_template_root, leaf_id=my_id, root_id=service_template_root['id'])
        if service_context:
            if service_context['leaf_name'] != name and name != 'root':
                logger.error("name not match; %s vs %s (%s)", service_context['leaf_name'], name, full_path)
            service_context['leaf_full_path'] = full_path
            service_context['artifact_tags'] = service_template_root['template'].get('artifact_tags', [])
        leaf["service_context"] = service_context
    logger.debug("service_context=%s", service_context)

    if "template" in leaf:
        template = leaf["template"]
        if "type" in template and (template["type"] == "workflow" or template["type"] == "container"):
            if "inputs" in template and template["inputs"]:
                # change service_id tag to service_instance_id
                if "artifacts" in template["inputs"]:
                    artifacts = template["inputs"]["artifacts"]
                    if isinstance(artifacts, list): # old format
                        for art in artifacts:
                            if isinstance(art, dict):
                                service_id = art.pop("service_id", None)
                                if service_id:
                                    art["service_instance_id"] = service_id
                                logger.info("artifact translated: %s", art)
                    elif isinstance(artifacts, dict): # yaml-v2
                        for _, art in artifacts.items():
                            if isinstance(art, dict):
                                service_id, name = get_artifact_sid_and_name(art)
                                if service_id:
                                    art["service_instance_id"] = service_id
                                if name:
                                    art["name"] = name
                                logger.info("artifact translated: %s", art)
                    else:
                        logger.warning("bad artifacts %s", artifacts)

            if "outputs" in template and template["outputs"]:
                artifacts = template["outputs"].get("artifacts", None)
                if isinstance(artifacts, dict):
                    for key, values in artifacts.items():
                        aliases = _find_artifact_aliases_deep(service_template_root, my_id, key, '')
                        if aliases and len(aliases):
                            template["outputs"]["artifacts"][key]["aliases"] = aliases

        if template.get('steps') is None and isinstance(parameter, dict) and parameter:
            leaf['template'] = substitute(template, **parameter)

    return leaf


def _test_servcice_template(service_template_leaf, service_template_root):
    service_template_pre_process(service_template_root=service_template_root, leaf=copy.deepcopy(service_template_leaf), parameter={}, name='', full_path='')


def _test_service_template_recursive(service_template, service_template_root):
    template = service_template.get("template", None)
    if isinstance(template, dict):
        for st in ["steps", "fixtures"]:
            steps = template.get(st, None)
            if isinstance(steps, list):
                for step in steps:
                    if isinstance(step, dict):
                        for key, value in step.items():
                            _test_service_template_recursive(value, service_template_root)
            else:
                # no steps, must be leaf
                _test_servcice_template(service_template, service_template_root)


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)

    my_path = os.path.dirname(os.path.abspath(__file__))
    test_json_file = os.path.join(my_path, "service_template_sample.json")

    parser = argparse.ArgumentParser(description='service template processing test',
                                     formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    parser.add_argument('--filename', default=test_json_file, help='file that contains the service template')

    main_args = parser.parse_args()

    if main_args.filename:
        test_json_file = os.path.abspath(main_args.filename)

    with open(test_json_file) as data_file:
        main_service_template = json.load(data_file)

    _test_service_template_recursive(service_template=main_service_template,
                                     service_template_root=main_service_template)
