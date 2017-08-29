#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module for artifact.
"""
import collections
import datetime
import logging
import os
import re

from six import string_types
from ax.devops.artifact.constants import FLAG_IS_ALIAS
from ax.devops.client.artifact_client import AxArtifactManagerClient

logger = logging.getLogger(__name__)
artifact_client = AxArtifactManagerClient()


class AXArtifacts:
    def __init__(self):
        pass

    test_service_instance_uuid_prefix = "artifact_test_service_instance_UUID"

    @staticmethod
    def load_from_db(artifact_id=None,
                     service_instance_id=None,
                     artifact_name=None,
                     max_retry=10):
        record = AXArtifacts.load_from_db_helper(artifact_id=artifact_id,
                                                 service_instance_id=service_instance_id,
                                                 artifact_name=artifact_name,
                                                 max_retry=max_retry)
        return record

    @staticmethod
    def load_from_db_helper(artifact_id=None,
                     service_instance_id=None,
                     artifact_name=None,
                     max_retry=10):
        count = 0
        while count < max_retry:
            count += 1
            try:
                if artifact_id:
                    conditions = {"artifact_id": artifact_id}
                else:
                    conditions = {}
                    if service_instance_id:
                        conditions["service_instance_id"] = service_instance_id
                    if artifact_name:
                        conditions["name"] = artifact_name

                if len(conditions):
                    conditions['action'] = 'search'
                    response = artifact_client.query_artifacts(conditions=conditions)
                    del conditions['action']
                    records = response.json().get("data", None)
                    logger.debug("condition: %s", conditions)
                    logger.debug("result: len=%s %s", len(records), records)
                else:
                    logger.debug("no condition, return None")
                    return None

                if not isinstance(records, list):
                    logger.error("get from artifact manager failed condition=%s. response=%s",
                                 conditions, response)
                    return None
                else:
                    # filter
                    max_time = 0
                    newest_record = None
                    for record in records:
                        not_match = False
                        for cond in conditions:
                            if cond in record and record[cond] == conditions[cond]:
                                continue
                            else:
                                not_match = True
                                break
                        if not not_match:
                            t = record.get("ax_time", 0)
                            if t >= max_time:
                                newest_record = record
                                max_time = t
                        else:
                            logger.debug("filter record %s, conditions %s", record, conditions)
                    if newest_record is not None:
                        return newest_record
                    else:
                        logger.error("all records filtered condition=%s", conditions)
                        return None
            except Exception:
                logger.exception("retry %s", count)
                continue

        logger.exception("too many retries")
        return None

    @staticmethod
    def get_artifact_input_destination_file_path(artifact_from_template):
        path = artifact_from_template.get("path", None)
        if not path:
            raise ValueError("path is required in input artifact template {}".format(artifact_from_template))
        return path

    @staticmethod
    def is_test_service_instance(service_instance_id):
        if isinstance(service_instance_id, string_types):
            return service_instance_id.startswith(AXArtifacts.test_service_instance_uuid_prefix)
        else:
            return False

    @staticmethod
    def get_extra_artifact_in_volume_mapping(container, host_scratch_root, in_label, test_mode=False, self_sid=None):
        if "inputs" not in container or container["inputs"] is None or "artifacts" not in container["inputs"]:
            return []

        ret = []
        artifacts = container["inputs"]["artifacts"]
        artifacts = collections.OrderedDict(sorted(artifacts.items(), key=lambda t: t[0]))
        for artifact_idx, artifact_name in enumerate(artifacts):
            artifact = artifacts[artifact_name]
            artifact_id = artifact.get("artifact_id", None)
            artifact_from_spec = artifact.get("from", None)

            logger.debug("Input artifact id {} name {} from {} {}".format(artifact_id, artifact_name, artifact_from_spec, artifact))

            if artifact_from_spec is not None:
                artifact_from_spec = re.sub("%%", "", artifact_from_spec)
                (_, service_instance_id, _, _, artifact_name) = artifact_from_spec.split(".")
                assert artifact_name is not None and service_instance_id is not None, "Cannot find instance id and name from {}".format(
                    artifact_from_spec)
            else:
                if artifact_id is None:
                    service_instance_id = artifact.get("service_instance_id", None) or artifact.get("service_id", None)
                    if artifact_name is not None and service_instance_id is None:
                        # use self service_instance_id in test mode
                        if test_mode:
                            service_instance_id = self_sid
                    if (artifact_name is not None and service_instance_id is None) or \
                            (artifact_name is None and service_instance_id is not None):
                        logger.error("invalid artifact input %s %s", artifact_idx, artifact)
                        return None
                else:
                    service_instance_id = None
                    artifact_name = None

            art = AXArtifacts.load_from_db(artifact_id=artifact_id,
                                           service_instance_id=service_instance_id,
                                           artifact_name=artifact_name)
            if art is None:
                logger.debug("no artifact in DB for %s %s %s %s", artifact_id, service_instance_id,
                             artifact_name, artifact)
            else:
                if artifact_id is not None:
                    if artifact_id != art["artifact_id"]:
                        logger.error("artifact id mismatch!!! %s vs %s, %s %s",
                                     artifact_id, art["artifact_id"],
                                     service_instance_id, artifact)
                        return None

            file_path = AXArtifacts.get_artifact_input_destination_file_path(artifact_from_template=artifact)
            host_file_path = os.path.join(host_scratch_root, in_label, str(artifact_idx))
            art_mapping = (host_file_path, file_path)
            logger.debug("add art_mapping %s", art_mapping)
            ret.append(art_mapping)

        return ret

    @staticmethod
    def gen_artifact_path(prefix, root_id, service_id, add_date, name):
        today = datetime.date.today()
        if add_date:
            prefix = "{prefix}/{date}".format(prefix=prefix,
                                              date=today.strftime("%Y/%m/%d"),)
        path = "{prefix}/{root}/{service}".format(prefix=prefix,
                                                  root=root_id,
                                                  service=service_id)
        if name is not None:
            path += '/' + name
        return path
