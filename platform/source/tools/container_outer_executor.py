#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

# expand the lib path, very hacky way run this script in a container

from ax.util.az_patch import az_patch
az_patch()

import ast
import argparse
import base64
import binascii
import collections
import hashlib
import json
import logging
import os
import re
import shutil
import signal
import subprocess
import sys
import time
import urllib
import uuid

import boto3
from retrying import retry
from six import string_types

from ax.cloud import Cloud
from ax.cloud.aws import AXS3Bucket

from ax.util.axdb import AXDBClient
from ax.util.ax_tarfile import AXTarfile
from ax.util.ax_artifact import AXArtifacts
from ax.util.ax_signal import traceback_multithread
from ax.axdb.axsys import artifacts_table_path
from ax.devops.redis.redis_client import RedisClient, REDIS_HOST, DB_RESULT
from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.client.artifact_client import AxArtifactManagerClient
from ax.devops.workflow.ax_workflow import AXWorkflow
from ax.devops.artifact.constants import RETENTION_TAG_DEFAULT, RETENTION_TAG_AX_LOG, RETENTION_TAG_AX_LOG_EXTERNAL, RETENTION_TAG_USER_LOG, \
    RETENTION_TAG_LONG_RETENTION, FLAG_IS_ALIAS, FLAG_IS_NOT_ALIAS, ARTIFACT_TYPE_INTERNAL, ARTIFACT_TYPE_AX_LOG, ARTIFACT_TYPE_AX_LOG_EXTERNAL, \
    ARTIFACT_TYPE_USER_LOG, ARTIFACT_TYPE_EXPORTED
from ax.devops.utility.utilities import retry_on_errors
from ax.devops.utility.junit_xml_parser import Parser
from ax.version import __version__
from ax.platform.pod import ARTIFACTS_CONTAINER_SCRATCH_PATH
from ax.platform.container_specs import is_ax_aux_container

from ax.platform.applet.amclient import ApplicationManagerClient
from ax.platform.applet.consts import HeartBeatType

from argo.template.v1.deployment import DeploymentTemplate

logger = logging.getLogger("ax.container_outer_executor")
# logging.getLogger("botocore").setLevel(logging.DEBUG)
axdb_client = AxdbClient()
axops_client = AxopsClient()
artifact_client = AxArtifactManagerClient()


class ContainerOuterExecutor(object):
    def __init__(self,
                 docker_inspect,
                 host_scratch_root,
                 container_scratch_root,
                 executor_sh,
                 input_label,
                 output_label,
                 pod_name,
                 job_name,
                 pod_ip,
                 post_mode):

        self._test_mode = False
        self._max_db_retry = 150
        self._cannot_find_return_code_ret_code = 10001
        self._cannot_load_artifact_ret_code = 10002
        self._cannot_relaunch_run_once = 10003
        self._cannot_launch_workflow_finished = 10004
        self._cannot_save_artifact_ret_code = 10005

        self._host_scratch_root = host_scratch_root
        self._container_scratch_root = container_scratch_root
        self._executor_sh = executor_sh
        self._input_label = input_label
        self._output_label = output_label
        self._pod_name = pod_name
        self._job_name = job_name
        self._pod_ip = pod_ip
        self._post_mode = post_mode
        self._return_code_postfix = "_ax_return"
        self._container_done_flag_postfix = "_ax_container_done_flag"
        self._container_command_list_postfix = "_ax_command_list"
        self._ax_command_path = "/ax-execu-host/art"
        self._tar_command = os.path.join(self._ax_command_path, "ax_tar_ax")
        self._cat_command = os.path.join(self._ax_command_path, "ax_cat_ax")
        self._echo_command = os.path.join(self._ax_command_path, "ax_echo_ax")
        self._kill_command = os.path.join(self._ax_command_path, "ax_kill_ax")
        self._mv_command = os.path.join(self._ax_command_path, "ax_mv_ax")
        self._cp_command = os.path.join(self._ax_command_path, "ax_cp_ax")
        self._gzip_command = os.path.join(self._ax_command_path, "ax_gzip_ax")
        self._bzip2_command = os.path.join(self._ax_command_path, "ax_bzip2_ax")
        self._docker_enable = False
        self._new_deployment = os.getenv("AX_DEPLOYMENT_NEW", None)
        self._volume_mounts = os.getenv("AX_VOL_MOUNT_PATHS", None)

        data = None
        with open("/etc/axspec/annotations") as f:
            for line in f:
                key, _, var = line.partition("=")
                if key == "AX_SERVICE_ENV":
                    data = json.loads(base64.b64decode(var).decode(errors='replace'))
                elif key == "AX_IDENTIFIERS":
                    ax_ids = json.loads(json.loads(var))
                    logger.debug(ax_ids)

        assert data is not None, "Need to pass service template to artifacts container"

        if self._new_deployment:
            self._generate_globals_deployment(data, ax_ids)
        else:
            self._generate_globals_task(data)

        if not post_mode:
            with open(docker_inspect) as data_file:
                self._docker_inspect = json.load(data_file)
        else:
            self._docker_inspect = ""

        self._host_return_code_file = os.path.join(self._host_scratch_root,
                                                   self._output_label,
                                                   self._return_code_postfix)
        self._container_return_code_file = os.path.join(self._container_scratch_root,
                                                        self._output_label,
                                                        self._return_code_postfix)
        self._container_done_flag_file = os.path.join(self._container_scratch_root,
                                                      self._output_label,
                                                      self._container_done_flag_postfix)
        self._container_command_list_file = os.path.join(self._container_scratch_root,
                                                         self._output_label,
                                                         self._container_command_list_postfix)
        self._pre_container_docker_id_file = os.path.join(self._host_scratch_root,
                                                          self._output_label,
                                                          "pre_docker_id.txt")

        logger.info("<env>: container %s", self._container)
        logger.info("<env>: commands %s", self._commands)
        logger.info("<env>: args %s", self._args)
        logger.info("<env>: artifacts %s %s", self._inputs, self._outputs)
        logger.info("<env>: service_context %s", self._service_context)
        logger.info("<env>: docker_inspect %s", self._docker_inspect)

        logger.info("<env>: s3_bucket: %s", self._s3_bucket)
        logger.info("<env>: s3_key_prefix: %s", self._s3_key_prefix)
        logger.info("<env>: s3_bucket_ax: %s", self._s3_bucket_ax)
        logger.info("<env>: s3_key_prefix_ax: %s", self._s3_key_prefix_ax)
        logger.info("<env>: keep_return_code: %s", self._keep_return_code)
        logger.info("<env>: host_scratch_root: %s", self._host_scratch_root)
        logger.info("<env>: container_scratch_root: %s", self._container_scratch_root)
        logger.info("<env>: executor_sh: %s", self._executor_sh)
        logger.info("<env>: input: %s", self._input_label)
        logger.info("<env>: output: %s", self._output_label)
        logger.info("<env>: post_mode: %s", self._post_mode)
        logger.info("<env>: container_name: %s", self._container_name)
        logger.info("<env>: leaf_name: %s", self._leaf_name)
        logger.info("<env>: leaf_full_path: %s", self._leaf_full_path)
        logger.info("<env>: artifact_tags: %s", self._artifacts_tags)
        logger.info("<env>: docker_enable: %s", self._docker_enable)

        if Cloud().in_cloud_aws():
            self._s3 = boto3.Session().resource('s3')
        elif Cloud().in_cloud_gcp():
            # TODO: Don't need it. Clean it.
            self._s3 = None
        self._init_db()

    def _generate_globals_deployment(self, data, ax_ids):
        logger.debug("Service template for deployment is {}".format(json.dumps(data)))
        template = DeploymentTemplate()
        template.parse(data)
        self._docker_enable = False # TODO: template.is_docker_enabled()
        self._container = template.get_main_container()
        self._commands = self._container.command
        self._args = self._container.args
        self._inputs = None
        if self._container.inputs.count() > 0:
            self._inputs = self._container.inputs.to_dict()

        self._outputs = None
        self._uuid = None
        self._cookie = None
        self._run_once = False
        self._is_wfe = False

        self._service_context = { "service_instance_id": "fake"}
        self._s3_bucket = None
        self._s3_key_prefix = None
        self._s3_bucket_ax_is_external = None
        self._s3_bucket_ax = None
        self._s3_key_prefix_ax = None
        self._keep_return_code = False

        self._service_instance_id = None
        self._root_workflow_id = None
        self._container_name = ""
        self._leaf_name = ""
        self._leaf_full_path = ""
        self._artifacts_tags = []

        self._artifacts_map = {}
        self._logs_map = {}

        self._amclient = ApplicationManagerClient()
        self._appname = self._job_name
        self._dep_id = ax_ids["deployment_id"]
        self._repo = ""

    def _generate_globals_task(self, data):
        logger.info("Service template is {}".format(json.dumps(data)))

        self._docker_enable = data.get('docker_enable', False)
        self._container = data['container']
        self._repo = self._container.get("repo", "")

        if "docker" in self._container:
            self._commands = self._container["docker"].get("commands", None)
            self._args = self._container["docker"].get("args", None)
        else:
            self._commands = None
            self._args = None

        self._inputs = self._container.get("inputs", None)
        self._outputs = self._container.get("outputs", None)

        if self._outputs:
            reporting_callback = self._outputs.get("reporting_callback", {})
            self._uuid = reporting_callback.get("uuid")
            self._cookie = reporting_callback.get("cookie")
            self._run_once = reporting_callback.get("run_once", False)
            self._is_wfe = reporting_callback.get("is_wfe", None)
        else:
            self._uuid = None
            self._cookie = None
            self._run_once = False
            self._is_wfe = False

        self._service_context = self._container.get("service_context", None)
        self._s3_bucket = data['s3_bucket']
        self._s3_key_prefix = data['s3_key_prefix']
        self._s3_bucket_ax_is_external = data.get('s3_bucket_ax_is_external', False)
        self._s3_bucket_ax = data.get('s3_bucket_ax', self._s3_bucket)
        self._s3_key_prefix_ax = data.get('s3_key_prefix_ax', self._s3_key_prefix)
        self._keep_return_code = data.get('keep_return_code', False)

        if self._service_context:
            self._service_instance_id = self._service_context.get('service_instance_id', None)
            self._root_workflow_id = self._service_context.get('root_workflow_id', None)
            assert self._service_instance_id
            assert isinstance(self._service_instance_id, string_types)
            if AXArtifacts.is_test_service_instance(self._service_instance_id):
                self._test_mode = True
            self._return_code_postfix += "_" + self._service_instance_id
            self._container_command_list_postfix += "_" + self._service_instance_id
            self._container_name = self._service_context.get('name', "")
            self._leaf_name = self._service_context.get('leaf_name', "")
            self._leaf_full_path = self._service_context.get('leaf_full_path', "")
            self._artifacts_tags = self._service_context.get('artifact_tags', [])
        else:
            self._service_instance_id = None
            self._root_workflow_id = None
            self._container_name = ""
            self._leaf_name = ""
            self._leaf_full_path = ""
            self._artifacts_tags = []

        self._artifacts_map = {}
        self._logs_map = {}

    def _write_executor_sh(self):
        # write executor_sh
        assert self._generated_executor_sh, "no executor_sh"

        self._generated_executor_sh += ["{echo} $last_error_code > {rc_file}".format(echo=self._echo_command,
                                                                                     rc_file=self._container_done_flag_file),
                                        "if [ -f {}/.ax_delete ]; then exit 0; fi".format(ARTIFACTS_CONTAINER_SCRATCH_PATH),
                                        "exit {}".format("$last_error_code" if self._keep_return_code else "0")]
        executor_sh_content = "\n".join(self._generated_executor_sh)
        # logger.debug("<step>: executor_sh_content %s", executor_sh_content)

        with open(self._executor_sh, "w+", encoding="utf8") as text_file:
            text_file.write(executor_sh_content)
        # make it executable and readable by everyone
        st = os.stat(self._executor_sh)
        os.chmod(self._executor_sh, st.st_mode | 0o111 | 0o444)

    def _write_container_docker_id_file(self):
        try:
            container_uuid = AxsysClient.get_container_uuid()
            if container_uuid is not None:
                docker_ids = {"axinit": container_uuid}
                logger.info("write docker_id %s to %s", docker_ids, self._pre_container_docker_id_file)
                with open(self._pre_container_docker_id_file, "w") as f:
                    f.write(json.dumps(docker_ids))
        except Exception:
            logger.exception("cannot write %s", self._pre_container_docker_id_file)

    def _exec_executor_sh_in_test(self):
        assert self._test_mode
        logger.info("<step>: exec_executor_sh_in_test")
        os.system(self._executor_sh)

    def _check_aleady_launch_acked(self):
        if not self._uuid:
            return False
        launch_ack_key = AXWorkflow.WFL_LAUNCH_ACK_KEY.format(self._uuid)
        count = 0
        max_count = 100  # 100 mins
        while True:
            try:
                count += 1
                container_result, _ = axdb_client.get_workflow_kv(launch_ack_key)
                if container_result:
                    logger.info("launch-acked before %s from key %s", container_result, launch_ack_key)
                    return True
                else:
                    logger.info("never launch-acked")
                    return False
            except Exception:
                logger.exception("")
                time.sleep(60)

            if count >= max_count:
                logger.info("assume no launch-acked after %s retry", count)
                return False

    def _check_already_had_result(self):
        if not self._uuid:
            return False
        result_key = AXWorkflow.WFL_RESULT_KEY.format(self._uuid)
        count = 0
        max_count = 100  # 100 mins
        while True:
            try:
                count += 1
                container_result, _ = axdb_client.get_workflow_kv(result_key)
                if container_result:
                    try:
                        jr = json.loads(container_result)
                        event_type = jr.get("event_type", None)
                        if event_type == "HAVE_RESULT":
                            logger.info("find result-callback %s from key %s", container_result, result_key)
                            return True
                    except Exception:
                        logger.exception("bad json")
                    logger.info("non-result-callback %s from key %s, treat as no result-callback", container_result, result_key)
                    return False
                else:
                    logger.info("no result-callback yet")
                    return False
            except Exception:
                logger.exception("")
                time.sleep(60)

            if count >= max_count:
                logger.info("assume no result-callback after %s retry", count)
                return False

    def _report_and_wait_for_workflow_executor(self):

        def wait_for_workflow_executor_permission():
            count = 0
            max_count = 60  # 1 hours max wait time to avoid long time deadlock
            while True:
                count += 1
                try:
                    container_result, _ = axdb_client.get_workflow_kv(launch_ack_key)
                    if container_result:
                        logger.info("got result %s from key", container_result)
                        return True
                    keys = [launch_ack_list_key]
                    logger.info("wait for %s count=%s", keys, count)
                    tuple_val = redis_client.brpop(keys, timeout=60)
                    if tuple_val is not None:
                        logger.info("got result %s from list key", tuple_val)
                        return True
                    if self._root_workflow_id:
                        workflow = AXWorkflow.get_workflow_by_id_from_db(workflow_id=self._root_workflow_id, need_load_template=False)
                        if workflow:
                            if workflow.status not in [AXWorkflow.RUNNING, AXWorkflow.RUNNING_DEL, AXWorkflow.RUNNING_DEL_FORCE]:
                                logger.error("workflow %s already in %s. stop waiting.", self._root_workflow_id, workflow.status)
                                return False
                        else:
                            logger.error("cannot find workflow %s", self._root_workflow_id)
                    else:
                        logger.error("no workflow id")

                    if count >= max_count:
                        logger.info("still no permission after %s retry", count)
                        raise Exception("No permission after retry.")

                except Exception as e:
                    logger.exception("")
                    if count >= max_count:
                        logger.info("no permission after %s retry", count)
                        raise e
                    time.sleep(60)

        if self._is_wfe:
            logger.info("is_wfe, do not ask permission")
            return True

        if self._uuid:
            launch_key = AXWorkflow.WFL_LAUNCH_KEY.format(self._uuid)
            launch_list_key = AXWorkflow.REDIS_LAUNCH_LIST_KEY.format(self._uuid)
            launch_ack_key = AXWorkflow.WFL_LAUNCH_ACK_KEY.format(self._uuid)
            launch_ack_list_key = AXWorkflow.REDIS_LAUNCH_ACK_LIST_KEY.format(self._uuid)

            redis_client = RedisClient(host=REDIS_HOST,
                                       db=DB_RESULT,
                                       retry_max_attempt=360,
                                       retry_wait_fixed=5000)

            p = {"name": self._container_name,
                 "instance_id": self._uuid,
                 "pod_name": self._pod_name,
                 "job_name": self._job_name,
                 "ip": self._pod_ip}
            if self._cookie:
                p["cookie"] = self._cookie

            notification_result = self._send_notification(redis_list_key=launch_list_key,
                                                          axdb_key=launch_key,
                                                          value=p,
                                                          max_retry=180)
            if notification_result:
                logger.info("<step>: ask permission to launch p=%s, %s",
                            p, launch_key)
                if self._is_wfe is None:
                    # for update compatibility reason, we can remove this later
                    logger.info("is_wfe is None, do not wait for permission")
                    return True
                # wait for permission
                try:
                    return wait_for_workflow_executor_permission()
                except Exception:
                    logger.exception("cannot get permission. exit(3)")
                    sys.exit(3)
            else:
                logger.error("cannot ask permission to launch to %s, %s. exit(2)", launch_key, launch_list_key)
                sys.exit(2)
        else:
            logger.info("no uuid, do not ask permission")
            return True

    def _pre_run(self):
        logger.info("<step>: pre_run")
        if self._test_mode:
            logger.info("<step>: test_mode prepare")
            self._gen_user_command_head()
            bad_artifacts1, _ = self._do_save_artifacts(dry_run_mode=True,
                                                        dry_run_use_host_dir_instead_of_container=True)
            if len(bad_artifacts1):
                logger.error("bad_artifacts1: %s", bad_artifacts1)
            self._write_executor_sh()
            self._exec_executor_sh_in_test()
            bad_artifacts2, _ = self._do_save_artifacts(dry_run_mode=False)
            if len(bad_artifacts2):
                logger.error("bad_artifacts2: %s", bad_artifacts2)

        if (not self._is_wfe) and self._check_aleady_launch_acked():
            if self._check_already_had_result():
                logger.info("has result already, don't need to re-run")
                self._gen_user_command_for_already_done()
                self._write_executor_sh()
                return True
            else:
                if self._run_once:
                    logger.info("run_once, cannot re-run")
                    self._gen_user_command_for_run_once_failed()
                    self._write_executor_sh()
                    return False
                else:
                    logger.info("re-run since no result")

        self._send_notification_report_status(event_type="LOADING_ARTIFACTS")

        bad_artifacts3, error_messages = self._do_load_artifacts()

        if len(bad_artifacts3) > 0:
            logger.error("load_artifacts failed: %s", bad_artifacts3)

            if self._new_deployment:
                # for deployment when loading fails, we report to am and sleep
                logger.info("Notifying AM of failure")
                while True:
                    self._send_notification_report_status(event_type="ARTIFACT_LOAD_FAILED")
                    logger.info("Notification sent...Waiting for termination from AM")
                    time.sleep(60)
            if isinstance(error_messages, list):
                error_messages = "\n".join(error_messages)
            self._gen_user_command_with_failed_load_artifact(bad_artifacts3, error_messages)
            self._write_executor_sh()
            return False

        self._gen_user_command()
        bad_artifacts4, _ = self._do_save_artifacts(dry_run_mode=True)
        if len(bad_artifacts4):
            logger.error("bad_artifacts2: %s", bad_artifacts4)

        self._write_executor_sh()

        self._write_container_docker_id_file()

        if not self._report_and_wait_for_workflow_executor():
            self._gen_user_command_for_workflow_already_gone()
            self._write_executor_sh()
            return False
        return True

    def _post_run(self):
        logger.info("<step>: post_run")
        if (not self._is_wfe) and self._check_already_had_result():
            logger.info("already had result-callback, skip post_run")
            return True

        self._send_notification_report_status(event_type="SAVING_ARTIFACTS")
        bad_artifacts, error_messages = self._do_save_artifacts(dry_run_mode=False)

        # AX integrations:
        # 1. Parse Junit XML report, if specified at service template
        # 2. TODO: other integrations
        self._do_parse_junit_report()

        self._upload_container_logs()
        ret_reporting = self._do_reporting(bad_artifacts, error_messages)
        return len(bad_artifacts) == 0 and ret_reporting

    def _prepare_directories(self):
        try:
            omask = os.umask(0)
            for d in [os.path.join(self._host_scratch_root, self._input_label),
                      os.path.join(self._host_scratch_root, self._output_label)]:
                if not os.path.exists(d):
                    try:
                        os.makedirs(d)
                    except Exception:
                        logger.exception("cannot makedirs %s", d)
        finally:
            os.umask(omask)


    def _upload_container_logs(self):
        docker_ids = {}
        for filename in ["/docker_id.txt", self._pre_container_docker_id_file]:
            try:
                with open(filename) as f:
                    ids = json.load(f)
                if not isinstance(ids, dict):
                    logger.warning("Could not get docker_id for container from %s", filename)
                else:
                    docker_ids.update(ids)
            except Exception:
                logger.exception("cannot get docker_ids from %s", filename)

        for container_name, docker_id in docker_ids.items():
            try:
                if is_ax_aux_container(container_name):
                    retention = RETENTION_TAG_AX_LOG_EXTERNAL if self._s3_bucket_ax_is_external else RETENTION_TAG_AX_LOG
                    artifact_type = ARTIFACT_TYPE_AX_LOG_EXTERNAL if self._s3_bucket_ax_is_external else ARTIFACT_TYPE_AX_LOG
                    s3_bucket = self._s3_bucket_ax
                    s3_key_prefix = self._s3_key_prefix_ax
                    add_date = True
                else:
                    retention = RETENTION_TAG_USER_LOG
                    artifact_type = ARTIFACT_TYPE_USER_LOG
                    s3_bucket = self._s3_bucket
                    s3_key_prefix = self._s3_key_prefix
                    add_date = False
                timestamp = get_current_epoch_timestamp_in_ms()
                artifact_uuid = str(uuid.uuid4())
                file_path = os.path.join(os.environ["LOGMOUNT_PATH"], "{}/{}-json.log".format(docker_id, docker_id))
                name = "{}.{}.log".format(container_name, docker_id)
                if self._leaf_full_path:
                    full_name = self._leaf_full_path + '.' + name
                else:
                    full_name = name
                key = AXArtifacts.gen_artifact_path(prefix=s3_key_prefix,
                                                    root_id=self._root_workflow_id,
                                                    service_id=self._service_instance_id,
                                                    add_date=add_date,
                                                    name=name)
                meta_data = {"ax_artifact_id": artifact_uuid,
                             "ax_container_log": "True",
                             "ax_timestamp": str(timestamp)
                             }
                logger.debug("Trying to upload container log {} to {}/{}".format(file_path, s3_bucket, key))
                ret, checksum, structure_res = self._upload_file(artifact_name=full_name,
                                                                 s3=self._s3,
                                                                 bucketname=s3_bucket,
                                                                 key=key,
                                                                 file_path=file_path,
                                                                 content_disposition_name=full_name,
                                                                 meta_data=meta_data)
                if ret:
                    self._logs_map[name] = "{}/{}".format(s3_bucket, key)
                else:
                    logger.error("Cannot save log %s", name)
                    continue
            except Exception as e:
                logger.exception("Caught exception while trying to upload container log %s", full_name)
                continue

            logger.info("<step>: record log pointer %s to db", full_name)

            try:
                stored_byte = os.stat(file_path).st_size
                db_data = {
                    "artifact_id": artifact_uuid,
                    "service_instance_id": self._service_instance_id,
                    "full_path": self._leaf_full_path,
                    "name": name,
                    "is_alias": 0,
                    "description": "ax container log",
                    "storage_method": "s3",
                    "storage_path": json.dumps({"bucket": s3_bucket, "key": key}),
                    "num_byte": stored_byte,
                    "num_dir": 0,
                    "num_file": 1,
                    "num_other": 0,
                    "num_skip_byte": 0,
                    "num_skip": 0,
                    "compression_mode": "",
                    "archive_mode": "",
                    "stored_byte": stored_byte,
                    "meta": json.dumps(meta_data),
                    "timestamp": timestamp,
                    "workflow_id": self._root_workflow_id,
                    "pod_name": self._pod_name,
                    "container_name": container_name,
                    "checksum": checksum,
                    "tags": json.dumps(self._artifacts_tags),
                    "retention_tags": retention,
                    "artifact_type": artifact_type,
                    "deleted": 0,
                }

                logger.info("<step>: create log-artifact entry %s %s", full_name, db_data)
                artifact_client.create_artifact(artifact=db_data, max_retry=self._max_db_retry, retry_on_exception=retry_on_errors(errors=['ERR_API_INVALID_PARAM'], retry=False))
                logger.info("<step>: log-artifact %s created", artifact_uuid)

            except Exception:
                logger.exception("insert log %s to db failure through artifact manager", full_name)
                continue

    def _update_mount_permissions(self):
        if not self._volume_mounts:
            logger.info("No volume mounts to update")
            return

        volume_mounts = ast.literal_eval(self._volume_mounts)
        for path in volume_mounts:
            if os.path.exists(path):
                logger.info("Changed permissions of: %s", path)
                os.chmod(path, 0o777)
            else:
                logger.info("Path %s not found. Ignored...", path)

    def run(self):
        if self._post_mode:
            return self._post_run()
        else:
            self._prepare_directories()
            self._update_mount_permissions()
            return self._pre_run()

    def _init_db(self):
        self._db = AXDBClient.get_local_axdb_client(table=artifacts_table_path)

    @staticmethod
    def _split_cmd(cmdString):
        # see http://stackoverflow.com/questions/366202/regex-for-splitting-a-string-using-space-when-not-surrounded-by-single-or-double
        assert isinstance(cmdString, string_types)
        cmdArray = []
        pattern = "[^\\s\"']+|\"([^\"]*)\"|'([^']*)'"

        regex = re.compile(pattern)

        for match in regex.finditer(cmdString):
            if match.group(1):
                cmdArray.append(match.group(1))
            elif match.group(2):
                cmdArray.append(match.group(2))
            else:
                cmdArray.append(match.group())

        return cmdArray

    @staticmethod
    def _get_real_command(entrypoint_in_image, command_in_image, command, args):
        logger.info("<env>: Entrypoint_in_image:%s Cmd_in_image:%s command:%s args:%s",
                    entrypoint_in_image, command_in_image, command, args)

        # According to https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/
        # 1. If you do not supply command or args for a Container, the defaults defined in the Docker image are used.
        # 2. If you supply a command but no args for a Container, only the supplied command is used. The default EntryPoint and the default Cmd defined in the Docker image are ignored.
        # 3. If you supply only args for a Container, the default Entrypoint defined in the Docker image is run with the args that you supplied.
        # 4. If you supply a command and args, the default Entrypoint and the default Cmd defined in the Docker image are ignored. Your command is run with your args.

        # Condition #1
        if command is None and args is None:
            # process entrypoint_in_image
            if entrypoint_in_image and isinstance(entrypoint_in_image, string_types):
                entrypoint_in_image = ["/bin/sh", "-c", entrypoint_in_image]
            if entrypoint_in_image and not isinstance(entrypoint_in_image, list):
                logger.error("<env>: bad entrypoint_in_image %s", entrypoint_in_image)
                entrypoint_in_image = None
            # process command_in_image
            if command_in_image and isinstance(command_in_image, string_types):
                command_in_image = ["/bin/sh", "-c", command_in_image]
            if command_in_image and not isinstance(command_in_image, list):
                logger.error("<env>: bad command_in_image %s", command_in_image)
                command_in_image = None

            if entrypoint_in_image is None and command_in_image is None:
                return None
            elif entrypoint_in_image is None:
                return command_in_image
            elif command_in_image is None:
                return entrypoint_in_image
            else:
                return entrypoint_in_image + command_in_image

        # Condition #2
        if command is not None and args is None:
            if isinstance(command, list):
                return command
            else:
                logger.error("<env>: bad command %s, expecting a list", command)
                return None

        # Condition #3
        if command is None and args is not None:
            # process entrypoint_in_image
            if entrypoint_in_image and isinstance(entrypoint_in_image, string_types):
                entrypoint_in_image = ["/bin/sh", "-c", entrypoint_in_image]
            if entrypoint_in_image and not isinstance(entrypoint_in_image, list):
                logger.error("<env>: bad entrypoint_in_image %s", entrypoint_in_image)
                entrypoint_in_image = None

            if isinstance(args, list):
                if entrypoint_in_image is None:
                    return args
                else:
                    return entrypoint_in_image + args
            else:
                logger.error("<env>: bad command %s, expecting a list", command)
                if entrypoint_in_image is None:
                    return None
                else:
                    return entrypoint_in_image

        # Condition #4
        if command is not None and args is not None:
            if not isinstance(command, list):
                command = []
            if not isinstance(args, list):
                args = []

            return command + args

    def _get_real_command_for_container(self):
        try:
            entrypoint_in_image = self._docker_inspect[0]['Config'].get("Entrypoint", None)
            command_in_image = self._docker_inspect[0]['Config'].get("Cmd", None)
        except Exception:
            logger.error("cannot inspect docker image %s. exit", self._docker_inspect)
            sys.exit(1)

        real_cmd = self._get_real_command(entrypoint_in_image=entrypoint_in_image,
                                          command_in_image=command_in_image,
                                          command=self._commands,
                                          args=self._args)

        return real_cmd

    @staticmethod
    def _split_cmd(cmd_string):
        # see http://stackoverflow.com/questions/366202/regex-for-splitting-a-string-using-space-when-not-surrounded-by-single-or-double
        assert isinstance(cmd_string, string_types)
        cmd_array = []
        pattern = "[^\\s\"']+|\"([^\"]*)\"|'([^']*)'"

        regex = re.compile(pattern)

        for match in regex.finditer(cmd_string):
            if match.group(1):
                cmd_array.append(match.group(1))
            elif match.group(2):
                cmd_array.append(match.group(2))
            else:
                cmd_array.append(match.group())

        return cmd_array

    def _gen_user_command_head(self, set_x=True):
        self._generated_executor_sh = ["#!/ax-execu-host/art/ax_bash_ax",
                                       "last_error_code=0"]

        if set_x:
            self._generated_executor_sh.append("set -x")

    def _get_secret_from_file(self, namespace, name, key):
        with open("/ax_secrets/{}/{}/{}".format(namespace, name, key)) as f:
            return f.read()

    def _gen_user_command(self):
        real_cmd = self._get_real_command_for_container()
        logger.info("<env>: real_cmd: %s", real_cmd)

        if isinstance(real_cmd, list):
            real_cmd_show = "\n".join(real_cmd)
            real_cmd_show_replaced = real_cmd_show

            # replace secrets in the commands
            matches = re.findall(r"%%config\.(.+?)\.([A-Za-z0-9-]+)\.([A-Za-z0-9-]+)%%", real_cmd_show_replaced)
            logger.info("Found the following secrets {}".format(matches))
            for (cfg_ns, cfg_name, cfg_key) in matches:
                logger.info("Looking for secret {} {} {}".format(cfg_ns, cfg_name, cfg_key))
                secret_val = self._get_secret_from_file(cfg_ns, cfg_name, cfg_key)
                logger.info("Replacing secret {} {} {}".format(cfg_ns, cfg_name, cfg_key))
                real_cmd_show_replaced = re.sub(r"%%config\..+?\.[A-Za-z0-9-]+\.[A-Za-z0-9-]+%%", secret_val, real_cmd_show_replaced)

            with open(self._container_command_list_file, "w+", encoding="utf8") as the_file:
                the_file.write(real_cmd_show_replaced)
            real_cmd = "/ax-execu-host/art/ax_exec_ax" + " " + self._container_command_list_file + " &"
        else:
            real_cmd = "empty command" + " &"
            real_cmd_show = real_cmd

        self._gen_user_command_head(set_x=False)

        # setup traps for signal forwarding
        trap_signal_cmds = []
        for sig in ["SIGHUP", "SIGINT", "SIGQUIT", "SIGABRT",
                    "SIGUSR1", "SIGUSR2", "SIGTSTP", "SIGTERM", "SIGCONT"]:
            trap_signal_cmds += ["_term_{sig}() {{".format(sig=sig),
                                 '  {echo} "Caught {sig} signal!"'.format(echo=self._echo_command, sig=sig),
                                 '  {kill} -{sig} "$ax_real_cmd_pid" 2>/dev/null'.format(kill=self._kill_command, sig=sig),
                                 "}",
                                 "trap _term_{sig} {sig}".format(sig=sig)]
        self._generated_executor_sh += trap_signal_cmds

        # It is time to not do this anymore....
        #if not self._encrypted_strings:  # Disable so that we do not print password
        #    self._generated_executor_sh.append("set -x")

        # wait for docker sidecar container if needed
        if self._docker_enable:
            docker_wait_commands = [
                "docker_enabled=0",
                "docker &> /dev/null",
                "if [ $? -eq 0 ]",
                "then",
                "  for cnt in {1..120}",
                "    do docker ps",
                "    code=$?",
                "    if [ $code -eq 0 ]",
                "    then",
                "      docker_enabled=1",
                "      break",
                "    fi",
                "    {echo} Waiting for docker daemon to start up $cnt ...".format(echo=self._echo_command),
                "    sleep 1",
                "  done",
                "else",
                "  {echo} No docker command.".format(echo=self._echo_command),
                "fi",
                "if [ $docker_enabled -eq 0 ]",
                "then",
                "  {echo} Failed to load docker daemon".format(echo=self._echo_command),
                "  {echo} 1 > {rc_file}".format(echo=self._echo_command, rc_file=self._container_done_flag_file),
                "  exit 0",
                "fi"
            ]
            self._generated_executor_sh += docker_wait_commands

        if self._new_deployment:
            self._generated_executor_sh += self._gen_deployment_handshake()

        # display real command
        self._generated_executor_sh += ["{cat} << 'axexEOF'".format(cat=self._cat_command),
                                        "real_cmd: " + real_cmd_show,
                                        "axexEOF",
                                        real_cmd]

        # launch real command and wait and collect return code
        collect_return_code_cmds = ["ax_real_cmd_pid=$!",
                                    "{echo} $ax_real_cmd_pid > {scratch}/.ax_pid".format(echo=self._echo_command,
                                                                                         scratch=ARTIFACTS_CONTAINER_SCRATCH_PATH),
                                    "if [ -f {scratch}/.ax_delete ]; then {kill} -9 $ax_real_cmd_pid; fi".format(scratch=ARTIFACTS_CONTAINER_SCRATCH_PATH, kill=self._kill_command),
                                    "wait $ax_real_cmd_pid",
                                    "last_error_code=$?",
                                    "{echo} $last_error_code > {rc_file}".format(echo=self._echo_command,
                                                                                 rc_file=self._container_return_code_file)]
        self._generated_executor_sh += collect_return_code_cmds

        # logger.info("<step>: generated executor sh %s", self._generated_executor_sh)

    def _gen_deployment_handshake(self):
        # TODO: Should we fail here?
        return [
            "echo \"Contacting Applatix infrastructure ...\"",
            "msg=\"${AX_HANDSHAKE_VERSION}##RTS##${AX_POD_NAME}##${AX_POD_NAMESPACE}\"",
            "while true; do",
            "    ret=$(/ax-execu-host/art/handshake /tmp/applatix.io/applet.sock ${msg} 3)",
            "    if [ \"${ret}\" == \"OTS\" ]; then",
            "        echo \"Received ok-to-start from Applatix. Main container proceeds\"",
            "        break",
            "    fi",
            "    echo \"Received ${ret} from Applatix. Waiting for ok-to-start ...\"",
            "    sleep 2",
            "done"
        ]

    def _gen_user_command_for_already_done(self):
        self._gen_user_command_head()
        self._generated_executor_sh += ["{cat} << axexEOF".format(cat=self._cat_command),
                                        "container is already done",
                                        "axexEOF",
                                        "last_error_code={}".format(0),
                                        "{echo} $last_error_code > {rc_file}".format(echo=self._echo_command,
                                                                                     rc_file=self._container_return_code_file),
                                        "{echo} container is already done >> {rc_file}".format(echo=self._echo_command,
                                                                                               rc_file=self._container_return_code_file)]

    def _gen_user_command_for_run_once_failed(self):
        self._gen_user_command_head()
        self._generated_executor_sh += ["{cat} << axexEOF".format(cat=self._cat_command),
                                        "FATAL ERROR: cannot relaunch run-once container",
                                        "axexEOF",
                                        "last_error_code={}".format(self._cannot_relaunch_run_once),
                                        "{echo} $last_error_code > {rc_file}".format(echo=self._echo_command,
                                                                                     rc_file=self._container_return_code_file),
                                        "{echo} cannot relaunch run-once container >> {rc_file}".format(echo=self._echo_command,
                                                                                                        rc_file=self._container_return_code_file)]

    def _gen_user_command_for_workflow_already_gone(self):
        self._gen_user_command_head()
        self._generated_executor_sh += ["{cat} << axexEOF".format(cat=self._cat_command),
                                        "FATAL ERROR: cannot launch, root workflow already finished",
                                        "axexEOF",
                                        "last_error_code={}".format(self._cannot_launch_workflow_finished),
                                        "{echo} $last_error_code > {rc_file}".format(echo=self._echo_command,
                                                                                     rc_file=self._container_return_code_file),
                                        "{echo} cannot launch, root workflow already finished >> {rc_file}".format(echo=self._echo_command,
                                                                                                                   rc_file=self._container_return_code_file)]

    def _gen_user_command_with_failed_load_artifact(self, artifacts, error_messages):
        self._gen_user_command_head()
        self._generated_executor_sh += ["{cat} << axexEOF".format(cat=self._cat_command),
                                        "FATAL ERROR: cannot load required artifacts: {art_name}".format(art_name=artifacts),
                                        "axexEOF",
                                        "last_error_code={}".format(self._cannot_load_artifact_ret_code),
                                        "{echo} $last_error_code > {rc_file}".format(echo=self._echo_command,
                                                                                     rc_file=self._container_return_code_file),
                                        "{echo} cannot load required artifacts: {art_name}, detail: {error_msg} >> {rc_file}".format(echo=self._echo_command,
                                                                                                                                     art_name=artifacts,
                                                                                                                                     error_msg=error_messages,
                                                                                                                                     rc_file=self._container_return_code_file)]

    def _gen_user_command_with_failed_decryption(self, token, error_messages):
        self._gen_user_command_head()
        # adding escape character so that $ sign can be printed out
        token = token.replace('$', '\$')
        self._generated_executor_sh += ["{cat} << axexEOF".format(cat=self._cat_command),
                                        "FATAL ERROR: cannot decrypt token: {token}".format(token=token),
                                        "axexEOF",
                                        "last_error_code={}".format(self._cannot_load_artifact_ret_code),
                                        "{echo} $last_error_code > {rc_file}".format(echo=self._echo_command,
                                                                                     rc_file=self._container_return_code_file),
                                        "{echo} cannot decrypt token: {token}, detail: {error_msg} >> {rc_file}".format(echo=self._echo_command,
                                                                                                                        token=token,
                                                                                                                        error_msg=error_messages,
                                                                                                                        rc_file=self._container_return_code_file)]


    @staticmethod
    def _download_file_from_gcp(bucketname, key, file_path):
        bucket = Cloud().get_bucket(bucketname)
        data = bucket.get_object(key)

        try:
            if os.path.exists(file_path):
                os.remove(file_path)
            with open(file_path, 'wb') as f:
                f.write(data)
            return True
        except Exception as e:
            logger.error("Failed to download from GCP: %s", e)
        return False

    @staticmethod
    def _download_file(artifact_name, s3, bucketname, key, file_path, expected_size, max_retry=10, checksum=""):
        logger.info("[%s]: download_file from s3: bucket=%s key=%s local_dst=%s size=%s",
                    artifact_name, bucketname, key, file_path, expected_size)

        if Cloud().in_cloud_gcp():
            return ContainerOuterExecutor._download_file_from_gcp(bucketname, key, file_path)

        # Using the aws_s3 object defined in platform
        s3_bucket = AXS3Bucket(bucket_name=bucketname)
        count = 0
        while count < max_retry:
            count += 1
            try:
                if os.path.exists(file_path):
                    os.remove(file_path)

                s3_bucket.download_file(key=key, file_name=file_path)

                if checksum:
                    logger.info("checksum from art_db: %s", checksum)

                    data = open(file_path, 'rb')

                    # Compared md5 checksum
                    binary_str, calculated_checksum = ContainerOuterExecutor.hashfile(afile=data, hasher=hashlib.md5())

                    logger.info("Compare downloaded checksum: %s, with uploaded checksum:%s", calculated_checksum, checksum)
                    if str(checksum) != str(calculated_checksum):
                        logger.error("checksum not equal: %s and %s", checksum, calculated_checksum)
                        raise Exception("Failed downloaded checksum is not same as uploaded checksum.")

                download_size = os.stat(file_path).st_size
                if download_size != expected_size:
                    logger.error("[%s]: size mismatch for %s, repected=%s vs downloaded=%s",
                                 artifact_name, file_path, expected_size, download_size)
                    return False
                return True
            except Exception:
                logger.exception("[%s]: cannot download %s", artifact_name, file_path)

        logger.error("[%s]: download_file max_retry reached", artifact_name)
        return False

    @staticmethod
    def _upload_file_to_gcp(artifact_name, bucketname, file_path, key, source_size):
        logger.info("Uploading object %s from file %s to GCP (%s)", key, file_path, bucketname)
        bucket = Cloud().get_bucket(bucketname)

        with open(file_path, "rb") as f:
            binary_str, hex_str = ContainerOuterExecutor.hashfile(afile=f, hasher=hashlib.md5())
            content_md5 = ContainerOuterExecutor.get_aws_format_checksum(binary_str)
            logger.info("[%s]: md5 checksum: %s", artifact_name, content_md5)
            checksum_str = hex_str

            f.seek(0)
            data = f.read()
        bucket.put_object(key, data)

        upload_success = True
        upload_structure = True
        logger.info("GCS: Put object %s in bucket %s", key, bucketname)
        return upload_success, checksum_str, upload_structure


    @staticmethod
    def _upload_file(artifact_name, s3, bucketname, key, file_path,
                     meta_data,
                     content_disposition_name,
                     compression_mode=None,
                     max_retry=5,
                     file_structure=None,
                     checksum_enabled=False):
        source_size = os.stat(file_path).st_size
        count = 0
        logger.info("[%s]: upload_file to s3: bucket=%s key=%s local_src=%s meta=%s size=%s checksum_enabled=%s",
                    artifact_name, bucketname, key, file_path, meta_data, source_size, checksum_enabled)

        upload_success = False
        upload_structure = False
        checksum_str = ""
        content_md5 = ""

        if Cloud().in_cloud_gcp():
            try:
                return ContainerOuterExecutor._upload_file_to_gcp(artifact_name, bucketname, file_path, key, source_size)
            except Exception as e:
                logger.info("Failed to upload file: %s", e)
                return False, None, False

        content_disposition = "attachment; filename={}".format(content_disposition_name)
        # Using the aws_s3 object defined in platform
        s3_bucket = AXS3Bucket(bucket_name=bucketname)

        while count < max_retry:
            count += 1
            try:
                if checksum_enabled is True:
                    # Calculate md5 checksum
                    with open(file_path, 'rb') as f:
                        binary_str, hex_str = ContainerOuterExecutor.hashfile(afile=f, hasher=hashlib.md5())
                        content_md5 = ContainerOuterExecutor.get_aws_format_checksum(binary_str)
                        logger.info("[%s]: md5 checksum: %s", artifact_name, content_md5)
                        checksum_str = hex_str

                # Copy the file into a new file. This is to avoid the problem with uploading
                #  a growing file (a log file) which results in s3 issue: https://github.com/aws/aws-cli/issues/602
                if file_path.endswith('.log'):
                    try:
                        import shutil
                        shutil.copy2(file_path, '/tmp')  # Copy to /tmp and use that for uploading to s3
                    except Exception:
                        logger.exception("[%s]: copy to /tmp failed", artifact_name)
                    file_path = os.path.join('/tmp', os.path.basename(file_path))
                    source_size = os.stat(file_path).st_size

                data = open(file_path, 'rb')
                params = {
                    'ACL': 'bucket-owner-full-control',
                    'Metadata': meta_data,
                    'ServerSideEncryption': 'AES256',
                    'StorageClass': 'STANDARD',
                    'ContentDisposition': content_disposition,
                    'ContentLength': source_size
                }
                if content_md5:
                    params['ContentMD5'] = content_md5

                if not s3_bucket.put_object(key=key, data=data, **params):
                    raise Exception("Failed to put object")
                data.close()
                upload_success = True
                break
            except Exception:
                logger.exception("[%s]: upload_file failed", artifact_name)
                data.close()

        if upload_success and file_structure:
            # upload directory structure to s3 with suffix _ax_structure
            structure_key = "{}_ax_structure".format(key)
            logger.info("uploaded file_structure to s3: bucket=%s, key=%s", bucketname, structure_key)
            count = 0
            while count < max_retry:
                count += 1
                try:
                    data = file_structure.encode('utf-8')
                    structure_params = {
                        'ACL': 'bucket-owner-full-control',
                        'ServerSideEncryption': 'AES256',
                        'StorageClass': 'STANDARD'
                    }
                    if not s3_bucket.put_object(key=structure_key, data=data, **structure_params):
                        raise Exception("Failed to put object")
                    upload_structure = True
                    break
                except Exception:
                    logger.exception("[%s]: upload_structure failed", artifact_name)

        if upload_success:
            return upload_success, checksum_str, upload_structure

        logger.error("[%s]: upload_file max_retry reached", artifact_name)
        return False, None, False

    def _load_artifact(self, artifact, artifact_name, artifact_idx):
        artifact_name_idx = "{}_idx".format(artifact_idx)
        logger.info("[%s]: load artifact %s: %s", artifact_name_idx, artifact_name, artifact)

        art_url = artifact.get('url', None)
        tar = artifact.get("url_archive_mode", None)
        excludes = artifact.get("excludes", None)
        artifact_id = artifact.get("artifact_id", None)
        artifact_from_spec = artifact.get("from", None)
        error_msg = None

        if artifact_from_spec is not None:
            artifact_from_spec = re.sub("%%", "", artifact_from_spec)
            (_, service_instance_id, _, _, artifact_name) = artifact_from_spec.split(".")
            assert artifact_name is not None and service_instance_id is not None, "Cannot find instance id and name from {}".format(artifact_from_spec)
        else:
            if artifact_id is None:
                artifact_name = artifact.get("name", None)
                service_instance_id = artifact.get("service_instance_id", None) or artifact.get("service_id", None)
                if artifact_name is not None and service_instance_id is None:
                    if self._test_mode:
                        logger.info("[%s]: test_mode, use itself service instance id %s",
                                    artifact_name_idx, self._service_instance_id)
                        service_instance_id = self._service_instance_id
                    else:
                        error_msg = "[{}]: invalid artifact input {}".format(artifact_name_idx, artifact)
                        logger.error(error_msg)
                        return False, error_msg
                if (artifact_name is not None and service_instance_id is None) or (artifact_name is None and service_instance_id is not None):
                    error_msg = "[{}]: invalid artifact input {}".format(artifact_name_idx, artifact)
                    logger.error(error_msg)
                    return False, error_msg
            else:
                service_instance_id = None
                artifact_name = None

        art_db = AXArtifacts.load_from_db(artifact_id=artifact_id,
                                          service_instance_id=service_instance_id,
                                          artifact_name=artifact_name,
                                          max_retry=10)

        if art_db is None:
            if art_url is None:
                error_msg = "Cannot find artifact from s3 service_instance_id={} artifact_name={}".format(service_instance_id, artifact_name)
                logger.error("[%s]: Cannot find artifact artifact_id=%s service_instance_id=%s artifact_name=%s",
                             artifact_name_idx, artifact_id, service_instance_id, artifact_name)
                return False, error_msg
        else:
            logger.debug("[%s]: db=%s", artifact_name_idx, art_db)
            if artifact_id and art_db["artifact_id"] != artifact_id:
                error_msg = "Artifact id mismatch!! {} vs {}, service_instance_id={} artifact_name={}".format(artifact_id, art_db["artifact_id"], service_instance_id, artifact_name)
                logger.error("[%s]: artifact id mismatch!!! %s vs %s, %s %s",
                             artifact_name_idx, artifact_id, art_db["artifact_id"],
                             service_instance_id, artifact)
                return False, error_msg

            tar = art_db.get("archive_mode", None)
            artifact_name_idx = "{}_{}".format(artifact_idx, art_db["name"])
            storage_method = art_db.get("storage_method", None)
            if storage_method != "s3":
                error_msg = "[{}]: unsupported storage method {}, artifact_name={}".format(artifact_name_idx, storage_method, artifact_name)
                logger.error(error_msg)
                return False, error_msg

        # determine where to store the downloaded file
        target_file_path = os.path.join(self._host_scratch_root, self._input_label, str(artifact_idx))
        if tar == "tar":
            local_file = os.path.join("/tmp", str(uuid.uuid4()) + ".tgz")
        else:
            error_msg = "Expecting artifact {} to be a tar file".format(artifact_name)
            logger.debug(error_msg)
            return False, error_msg

        # download
        strip_top_dir = False
        if art_db is None:
            # download from url
            logger.info("[%s]: load from url %s %s",
                        artifact_name_idx,
                        art_url, local_file)

            download_count = 0
            while download_count < 10:
                download_count += 1
                try:
                    urllib.urlretrieve(art_url, local_file)
                    download_success = True
                    break
                except Exception:
                    logger.exception("cannot download %s", art_url)
                    download_success = False
                    time.sleep(5)
        else:
            # download from s3
            strip_top_dir = (art_db.get("num_dir", 0) != 0)
            storage_path = art_db["storage_path"]
            logger.info("[%s]: load from blob store %s %s %s",
                        artifact_name_idx,
                        storage_path["bucket"], storage_path["key"], local_file)
            # xxx todo handle error
            download_success = self._download_file(artifact_name=artifact_name_idx,
                                                   s3=self._s3,
                                                   bucketname=storage_path["bucket"],
                                                   key=storage_path["key"],
                                                   file_path=local_file,
                                                   expected_size=art_db.get("stored_byte", 0),
                                                   checksum=art_db.get("checksum", ""))
        if not download_success:
            error_msg = "[{}]: download {} {} failed".format(artifact_name_idx, local_file, artifact_name)
            logger.error("[%s]: download %s %s failed", artifact_name_idx, local_file, artifact)
            return False, error_msg

        # untar if necessary
        if tar == "tar":
            # make tar_base_dir
            tar_base_path = target_file_path

            logger.info("[%s]: start extracting artifact %s tar file", artifact_name_idx, artifact_name)

            success_extract = self._extract_tar(local_file=local_file, excludes=excludes,
                                                tar_base_path=tar_base_path, strip_top_dir=strip_top_dir)

            if not success_extract:
                error_msg = "Failed to extracting artifact {} tar file".format(artifact_name)
                return False, error_msg

            try:
                if os.path.exists(local_file):
                    os.remove(local_file)
            except Exception:
                logger.exception("[%s]: cannot cleanup local file %s", artifact_name_idx, local_file)

            return True, None
        else:
            return True, None

    def _extract_tar(self, local_file, excludes, tar_base_path, strip_top_dir, max_retry=10):
        count = 0
        while count < max_retry:
            count += 1
            try:
                logger.info("Untar try No. %s", count)
                # prepare tar cmd
                tar_cmd = [self._tar_command, 'xf', local_file]

                if excludes:
                    for exc in excludes:
                        if exc:
                            tar_cmd += ["--exclude", "exc"]

                tar_cmd += ["-C", tar_base_path]

                # is the artifact a directory?
                if strip_top_dir:
                    logger.debug("Artifact is a tar file for a directory")
                    tar_cmd += ["--strip-components", "1"]

                # run tar cmd
                logger.info("invoke tar cmd %s", tar_cmd)
                p = subprocess.Popen(tar_cmd,
                                     shell=False)
                p.communicate()

                logger.info("tar cmd return code: %s", p.returncode)

                if p.returncode != 0:
                    logger.exception("extraction failed %s to %s with return code: %s", local_file, tar_base_path, p.returncode)
                else:
                    return True
            except Exception:
                logger.exception("cannot extract %s to %s", local_file, tar_base_path)

        return False

    def _save_artifact(self, name, artifact,
                       dry_run_mode,
                       dry_run_use_host_dir_instead_of_container):
        assert name, "artifact must have a name"
        logger.info("[%s]: save artifact %s", name, artifact)

        storage_method = artifact.get('storage_method', 'blob')
        retention = artifact.get('retention', RETENTION_TAG_DEFAULT)
        assert retention in [RETENTION_TAG_LONG_RETENTION, RETENTION_TAG_DEFAULT], "bad retention {}".format(retention)
        assert storage_method == 'blob', "only support s3 blob, not {}".format(storage_method)
        artifact_src_path = artifact.get("path", None)
        error_msg = None

        if (artifact_src_path is None) or (not isinstance(artifact_src_path, string_types)):
            error_msg = "[{}]: invalid path {}".format(name, artifact_src_path)
            logger.error(error_msg)
            return False, error_msg

        if not os.path.isabs(artifact_src_path):
            error_msg = "[{}]: src is not a absolute path {}".format(name, artifact_src_path)
            logger.error(error_msg)
            return False, error_msg

        if artifact_src_path == '/':
            error_msg = "[{}]: cannot use / as artifact src".format(name)
            logger.error(error_msg)
            return False, error_msg

        artifact_src_dir, artifact_src_file = os.path.split(artifact_src_path)
        tar_mode = artifact.get("archive_mode", "tar")
        if tar_mode != "tar":
            error_msg = "[{}]: cannot use {} as archive_mode".format(name, tar_mode)
            logger.error(error_msg)
            return False, error_msg
        symlink_mode = artifact.get("symlink_mode", None)
        compression_mode = artifact.get("compression_mode", "gz")
        excludes = artifact.get("excludes", None)

        if dry_run_mode:
            # this uses busybox tar
            sym_op = ""
            sym_cp_op = ""

            if compression_mode == "bz2":
                compression_op = "| {} -n ".format(self._bzip2_command)
            elif compression_mode == "gz":
                compression_op = "| {} -n ".format(self._gzip_command)
            else:
                compression_op = ""
            if excludes:
                excludes_op = ""
                for exclude in excludes:
                    assert isinstance(exclude, string_types), "exclude {} is not string".format(exclude)
                    excludes_op += " --exclude '{}'".format(exclude)
            else:
                excludes_op = ""

            if dry_run_use_host_dir_instead_of_container:
                scratch_root = self._host_scratch_root
            else:
                scratch_root = self._container_scratch_root
            output_file = os.path.join(scratch_root, self._output_label, name)
            # use _ax_tar to avoid name collision
            output_file_with_tar_extension = output_file + "._ax_tar"
            output_file_indicate_dir = output_file + "._ax_is_dir"

            file_handle_cmd = "{tar} -c {sym_op} -C '{src_dir}' '{src}' {compression_opt} > '{output}'".format(tar=self._tar_command,
                                                                                                               sym_op=sym_op,
                                                                                                               src_dir=artifact_src_dir,
                                                                                                               src=artifact_src_file,
                                                                                                               compression_opt=compression_op,
                                                                                                               output=output_file_with_tar_extension)

            dir_handle_cmd = "{tar} -c {sym_op} {excludes_op} -C '{src_dir}' '{src}' {compression_op} > '{output}' ; {echo} 1 > '{output_ind}'".\
                format(tar=self._tar_command,
                       compression_op=compression_op,
                       sym_op=sym_op,
                       excludes_op=excludes_op,
                       output=output_file_with_tar_extension,
                       src_dir=artifact_src_dir,
                       src=artifact_src_file,
                       echo=self._echo_command,
                       output_ind=output_file_indicate_dir)

            cmd = "if [ -f '{art_src}' ]; then {fhandle} ; elif [ -d '{art_src}' ]; then {dhandle}; else {echo} no artifact file '{art_src}'; fi". \
                format(fhandle=file_handle_cmd,
                       art_src=artifact_src_path,
                       dhandle=dir_handle_cmd,
                       echo=self._echo_command)
            self._generated_executor_sh.append(cmd)
            return True, error_msg

        else:
            output_file = os.path.join(self._host_scratch_root, self._output_label, name)
            output_file_with_tar_extension = output_file + "._ax_tar"
            output_file_indicate_dir = output_file + "._ax_is_dir"
            s3_upload_src_structure = None

            if os.path.exists(output_file):
                upload_tar_file = False
                s3_upload_src = output_file
                to_db_num_dir = 0
                to_db_num_file = 1
                to_db_num_other = 0
                to_db_num_symlink = 0
                compression_mode = ""
            elif os.path.exists(output_file_with_tar_extension):
                upload_tar_file = True
                s3_upload_src = output_file_with_tar_extension

                try:
                    # xxx todo, use tar bin to collect tar info
                    axtarfile = AXTarfile()
                    success = axtarfile.tar_get_info(tar_name=s3_upload_src,
                                                     compression_mode=compression_mode)

                    # Todo: Disabled structure calculation for M7, will add back in M8 (Tianhe)
                    # s3_upload_src_structure = axtarfile.get_tar_structure(tar_name=s3_upload_src,
                    #                                                       compression_mode=compression_mode)
                    if success:
                        to_db_num_dir = axtarfile.num_dir
                        to_db_num_file = axtarfile.num_files
                        to_db_num_other = axtarfile.num_other
                        to_db_num_symlink = axtarfile.num_symlink
                        to_db_num_byte = axtarfile.num_byte
                    else:
                        error_msg = "[{}]: failed to get tarinfo {}, comp={}".format(name, s3_upload_src, compression_mode)
                        logger.error(error_msg)
                        return False, error_msg
                except Exception:
                    error_msg = "[{}]: failed to get tarinfo {}, comp={}".format(name, s3_upload_src, compression_mode)
                    logger.exception(error_msg)
                    return False, error_msg
            else:
                error_msg = "[{}]: no artifact file {}".format(name, output_file)
                logger.error(error_msg)
                return False, error_msg

            if os.path.exists(output_file_indicate_dir):
                to_db_excludes = excludes
            else:
                to_db_excludes = []

            to_db_stored_byte = os.stat(s3_upload_src).st_size
            if not upload_tar_file:
                to_db_num_byte = to_db_stored_byte

            to_db_num_skip_byte = 0
            to_db_num_skip = 0
            to_db_tar = tar_mode
            to_db_compression = compression_mode

        meta_data = artifact.get("meta_data", {})
        # need this workaround before ying's fix
        if not isinstance(meta_data, dict):
            logger.error("[%s]: xxx, ask ying. meta_data has to be dict %s. set to empty for now", name, meta_data)
            meta_data = {}

        if self._test_mode:
            artifact_uuid = artifact.get("test_force_UUID", None)
        else:
            artifact_uuid = None

        if artifact_uuid is None:
            artifact_uuid = str(uuid.uuid4())

        timestamp = get_current_epoch_timestamp_in_ms()
        meta_data["ax_timestamp"] = str(timestamp)
        meta_data["ax_artifact_id"] = artifact_uuid

        # for test only
        if self._test_mode:
            self._testUUID = artifact_uuid

        if self._leaf_full_path:
            full_name = self._leaf_full_path + "." + name
        else:
            full_name = name

        key = AXArtifacts.gen_artifact_path(prefix=self._s3_key_prefix,
                                            root_id=self._root_workflow_id,
                                            service_id=self._service_instance_id,
                                            add_date=False, # don't add load because we want to distribute the load
                                            name=full_name)

        content_disposition_name = full_name
        if to_db_tar == "tar":
            if compression_mode == 'gz':
                content_disposition_name += '.tgz'
            else:
                content_disposition_name += '.tar'

        upload_success, checksum, upload_structure = self._upload_file(artifact_name=name,
                                                                       s3=self._s3,
                                                                       bucketname=self._s3_bucket,
                                                                       key=key,
                                                                       file_path=s3_upload_src,
                                                                       content_disposition_name=content_disposition_name,
                                                                       meta_data=meta_data,
                                                                       compression_mode=to_db_compression,
                                                                       file_structure=s3_upload_src_structure,
                                                                       checksum_enabled=True)

        # remove the temp tar file
        for f in [output_file, output_file_with_tar_extension, output_file_indicate_dir]:
            try:
                if os.path.exists(f):
                    os.remove(f)
            except Exception:
                logger.exception("[%s]: cannot remove %s", full_name, f)

        if not upload_success:
            error_msg = "[{}]: failed to upload to s3 {}".format(full_name, artifact)
            logger.error(error_msg)
            return False, error_msg

        aliases = artifact.get("aliases", [])
        aliases = [{"service_instance_id": self._service_instance_id, "artifact_name": name, "full_path": self._leaf_full_path}] + aliases
        logger.info("[%s]: begin insert %s artifact entries to axdb", name, len(aliases))
        if_exported = False

        for alias in aliases:
            sid = alias.get("service_instance_id", None)
            if sid == self._root_workflow_id:  # Check if service_id is workflow_id, indicates a root level artifact export
                if_exported = True

        for alias in aliases:
            sid = alias.get("service_instance_id", None)
            n = alias.get("artifact_name", None)
            full_path = alias.get("full_path", "")
            assert sid, "sid cannot be empty"
            assert n, "name cannot be empty"
            retention_tag = RETENTION_TAG_LONG_RETENTION if if_exported else retention
            artifact_type = ARTIFACT_TYPE_EXPORTED if sid == self._root_workflow_id else ARTIFACT_TYPE_INTERNAL
            if sid == self._service_instance_id:
                is_alias = FLAG_IS_NOT_ALIAS
                effective_artifact_id = artifact_uuid
                db_data = {
                    "src_path": artifact_src_dir,
                    "src_name": artifact_src_file,
                    "excludes": json.dumps(to_db_excludes),
                    "storage_method": "s3",
                    "storage_path": json.dumps({"bucket": self._s3_bucket, "key": key}),
                    "num_byte": to_db_num_byte,
                    "num_dir": to_db_num_dir,
                    "num_file": to_db_num_file,
                    "num_symlink": to_db_num_symlink,
                    "num_other": to_db_num_other,
                    "num_skip_byte": to_db_num_skip_byte,
                    "num_skip": to_db_num_skip,
                    "compression_mode": to_db_compression,
                    "archive_mode": to_db_tar,
                    "stored_byte": to_db_stored_byte,
                    "symlink_mode": symlink_mode,
                    "meta": json.dumps(meta_data),
                    "timestamp": timestamp,
                    "checksum": checksum,
                    "deleted": 0,
                }
                if upload_structure:
                    db_data['structure_path'] = json.dumps({"bucket": self._s3_bucket, "key": "{}_ax_structure".format(key)})

            else:
                is_alias = FLAG_IS_ALIAS
                effective_artifact_id = str(uuid.uuid4())
                db_data = {
                    "deleted": -1,
                    "source_artifact_id": artifact_uuid,
                }

            commons = {
                "artifact_id": effective_artifact_id,
                "service_instance_id": sid,
                "full_path": full_path,
                "name": n,
                "is_alias": is_alias,
                "description": n + " description",
                "workflow_id": self._root_workflow_id,
                "pod_name": self._pod_name,
                "container_name": self._leaf_name,
                "retention_tags": retention_tag,
                "artifact_type": artifact_type,
                "tags": json.dumps(self._artifacts_tags),
            }

            db_data.update(commons)

            logger.info("[%s]: add artifact entry to axdb %s %s %s %s", name, artifact_uuid, sid, n, is_alias)
            logger.debug("[%s]: insert db_data: %s %s %s", name, n, full_path, db_data)
            try:
                logger.info("<step>: create artifact entry %s %s %s", full_path, n, db_data)
                artifact_client.create_artifact(artifact=db_data, max_retry=self._max_db_retry, retry_on_exception=retry_on_errors(errors=['ERR_API_INVALID_PARAM'], retry=False))
                logger.info("<step>: artifact %s created", effective_artifact_id)

            except Exception:
                error_msg = "[{}]: Failed to save artifact result to db {} through artifact manager".format(name, artifact)
                logger.error(error_msg)
                return False, error_msg

        self._artifacts_map[name] = {"artifact_uuid": artifact_uuid, "s3_path": "{}/{}".format(self._s3_bucket, key), "size": to_db_stored_byte, "archive": to_db_tar}
        return True, error_msg

    def _do_load_artifacts(self):
        logger.info("<step>: load artifacts")
        bad_artifacts = []
        error_messages = []
        if self._service_context and self._inputs and 'artifacts' in self._inputs:
            artifacts_ordered = collections.OrderedDict(sorted(self._inputs["artifacts"].items(), key=lambda t: t[0]))
            for artifact_idx, artifact_name in enumerate(artifacts_ordered):
                artifact = artifacts_ordered[artifact_name]
                result, error_msg = self._load_artifact(artifact=artifact, artifact_name=artifact_name, artifact_idx=artifact_idx)
                if not result and artifact.get("required", True):
                    logger.error("cannot load required artifact %s", artifact)
                    bad_artifacts.append(artifact)
                    if error_msg is not None:
                        error_messages.append(error_msg)
        else:
            logger.info("<step>: no artifacts to load")

        return bad_artifacts, error_messages

    def _do_save_artifacts(self, dry_run_mode=False, dry_run_use_host_dir_instead_of_container=False):
        logger.info("<step>: save artifacts dry_run_mode=%s", dry_run_mode)
        bad_artifacts = []
        error_messages = []
        if self._service_context and self._outputs and 'artifacts' in self._outputs:
            assert isinstance(self._outputs["artifacts"], dict)
            for name in self._outputs["artifacts"]:
                artifact = self._outputs["artifacts"][name]
                result, error_msg = self._save_artifact(name=name,
                                                        artifact=artifact,
                                                        dry_run_mode=dry_run_mode,
                                                        dry_run_use_host_dir_instead_of_container=dry_run_use_host_dir_instead_of_container)
                if not result and artifact.get("required", True):
                    logger.error("cannot save required artifact %s", artifact)
                    bad_artifacts.append(artifact)
                    if error_msg is not None:
                        error_messages.append(error_msg)
        else:
            logger.info("<step>: no artifacts to save")

        return bad_artifacts, error_messages

    def _do_parse_junit_report(self):
        """ Scan artifacts to filter whether it requires 'test_reporting'
        :return:
        """
        TEST_REPORTING_KEY = 'test_reporting'  # Pre-defined key from service template
        TEST_REPORTING_MODE = 'junit'  # Pre-defined parser
        has_artifacts = self._service_context and self._outputs and self._outputs.get('artifacts', None)

        if not has_artifacts:
            return

        for artifact_idx, name in enumerate(self._outputs['artifacts']):
            # Parse each artifact
            artifacts = self._outputs['artifacts'][name]
            artifacts_path = artifacts.get('path', None)

            # Follows ServiceTemplate syntax
            meta_data = artifacts.get('meta_data', None)
            need_parse = artifacts_path and meta_data and '{}:{}'.format(TEST_REPORTING_KEY, TEST_REPORTING_MODE) in meta_data

            # Skip if it is not for Junit report format
            if not need_parse:
                continue

            # Downloading artifact from s3 for parser
            s3_data = self._artifacts_map.get(name, None)
            if s3_data is None:
                logger.info('<step>: No report to parse')
                continue
            # must be user artifact that use _s3_bucket instead of _s3_bucket_ax
            s3_key = s3_data.get('s3_path', '').split('{}/'.format(self._s3_bucket))[-1]
            s3_expected_size = int(s3_data.get('size', 0))
            downloaded_dst = os.path.join(self._host_scratch_root, self._output_label, '{}_{}'.format(name, int(time.time())))

            download_success = self._download_file(artifact_name=name,
                                                   s3=self._s3,
                                                   bucketname=self._s3_bucket,
                                                   key=s3_key,
                                                   file_path=downloaded_dst,
                                                   expected_size=s3_expected_size)
            if not download_success:
                logger.error("[%s]: download %s failed", name, downloaded_dst)
                return False

            base_path = os.path.join(self._host_scratch_root, self._output_label)
            success_extract = self._extract_tar(local_file=downloaded_dst, excludes=None,
                                                tar_base_path=base_path, strip_top_dir=False)
            if not success_extract:
                return False
            logger.debug('<step>: Save %s test result to DB', TEST_REPORTING_MODE)

            untar_path = os.path.join(base_path, os.path.basename(artifacts_path))
            logger.info('Untar path: %s', untar_path)

            xml_list = []
            is_dir = os.path.isdir(untar_path)

            if is_dir:
                for f in os.listdir(untar_path):
                    f = os.path.join(untar_path, f)
                    if os.path.isfile(f) and f.endswith('.xml'):
                        xml_list.append(f)
            else:
                xml_list = [str(untar_path)]
            logger.info('JUnit XML files: "%s"', xml_list)

            for each_xml in xml_list:
                try:
                    self._parse_junit_xml(name, each_xml)
                except Exception as exc:
                    logger.error('Fail to parse JUnit XML file: "%s"', exc)
            else:
                if os.path.exists(downloaded_dst):
                    logger.debug('Clean up %s', downloaded_dst)
                    if os.path.isdir(downloaded_dst):
                        shutil.rmtree(downloaded_dst)
                    else:
                        os.remove(downloaded_dst)

    def _parse_junit_xml(self, name, artifacts_path):
        """ Call API to parse test cases from JUnit XML
        :param name:
        :param artifacts_path:
        :return:
        """
        JUNIT_RESULT_TABLE = 'axdevops/junit_result'
        db_obj = AXDBClient.get_local_axdb_client(table=JUNIT_RESULT_TABLE)

        logger.info("[%s]: Parse test result:  %s", name, artifacts_path)
        parser = Parser(artifacts_path)
        _, _, cases = parser.run()
        # TODO: use concurrent in python3
        for case in cases:
            logger.debug('[%s]: insert test case data into AXDB: %s', name, case.__dict__)
            self._load_junit_result_to_db(db_obj, case)

    def _load_junit_result_to_db(self, db_obj, case):
        """Load each test case context to AXDB
        :param db_obj:
        :param case:
        :return:
        """
        EMPTY_STR = ''
        _data = {
            'result_id':    str(uuid.uuid4()),
            'leaf_id':      self._service_instance_id,
            'name':         case.name,
            'status':       case.status if case.status else EMPTY_STR,
            'message':      case.message if case.message else EMPTY_STR,
            'classname':    case.classname if case.classname else EMPTY_STR,
            'stderr':       case.stderr if case.stderr else EMPTY_STR,
            'stdout':       case.stdout if case.stdout else EMPTY_STR,
            'testsuite':    case.testsuite if case.testsuite else EMPTY_STR,
            'testsuites':   case.testsuites if case.testsuites else EMPTY_STR,
            'duration':     case.duration if case.duration else 0.0
        }
        try:
            _result = db_obj.insert(_data, max_retry=self._max_db_retry)
            if not _result:
                logger.error('DB insert failure "%s"', _data)
                return False
        except Exception:
            logger.error('DB insert failure "%s"', _data)
            return False

        return True

    @staticmethod
    def _read_return_code_from_file(filename):
        try:
            with open(filename, 'r') as f:
                first_line = f.readline().rstrip()
                return_code = int(first_line)
                logger.debug("return code is %s", return_code)
                rest = "\n".join(f.readlines())

            k8s_info = None
            try:
                k8s_info_filename = "/k8s_info.txt"
                with open(k8s_info_filename) as f:
                    k8s_info = json.load(f)
                    if not isinstance(k8s_info, dict):
                        logger.warn("Could not get reasons for container from %s", k8s_info_filename)
            except Exception:
                logger.exception("cannot get reasons from %s", k8s_info_filename)

            return return_code, k8s_info, rest
        except Exception:
            logger.exception("no return code from %s", filename)
            return None, None, None

    def _do_reporting(self, bad_artifacts_save, error_messages):
        if not os.path.exists(self._container_done_flag_file):
            # we sleep here to reduce the chance we report the failed-result to WFE in case
            # the pop is deliberately non-gracefully deleted by axmon.
            # After the sleep, in theory, the race is still there, but in practice, the chance is very very close to zero
            # The side effect is that if the main container is non-gracefully killed because other reasons, e.g. OOM,
            # there will be a x second delay before WFE knows about it.
            NO_RETURN_CODE_SLEEP_SECONDS = 0
            logger.info("no return code, sleep %s seonds in case the pod is deliberately non-gracefully deleted by axmon", NO_RETURN_CODE_SLEEP_SECONDS)
            time.sleep(NO_RETURN_CODE_SLEEP_SECONDS)

        return_code, k8s_info, rest_message = self._read_return_code_from_file(self._host_return_code_file)
        if return_code is None:
            ret0 = False
            return_code = self._cannot_find_return_code_ret_code
        else:
            ret0 = True

        messages = []
        if rest_message:
            messages.append(rest_message)
        if bad_artifacts_save:
            messages.append("cannot save artifacts: {}".format(bad_artifacts_save))
            if return_code == 0:
                # this indicates main container succeeds but artifacts saving failed
                # will override return code
                return_code = self._cannot_save_artifact_ret_code
                if isinstance(error_messages, list):
                    messages.extend(error_messages)  # append error_messages using the error_messages from save artifacts
        message = "\n".join(messages)

        ret1 = self._record_return_code(return_code, k8s_info, message)
        ret2 = self._send_notification_report_return_code(return_code, k8s_info, message)
        return ret0 and ret1 and ret2

    def _record_return_code(self, return_code, k8s_info, message):
        if self._service_instance_id is None:
            logger.error("no service instance id, no insert return code %s to db", return_code)
            return False
        if k8s_info:
            message = "{} {}".format(message, json.dumps(k8s_info))

        logger.info("<step>: record return_code %s to db, message=%s",
                    return_code, message)
        artifact_uuid = str(uuid.uuid4())
        db_data = {
            "artifact_id": artifact_uuid,
            "service_instance_id": self._service_instance_id,
            "full_path": self._leaf_full_path,
            "name": "{}.{}".format(self._container_name, "ax_reserve_return_code"),
            "is_alias": 0,
            "description": message,
            "storage_method": "inline",
            "storage_path": "",
            "inline_storage": str(return_code),
            "num_byte": 0,
            "num_dir": 0,
            "num_file": 0,
            "num_other": 0,
            "num_skip_byte": 0,
            "num_skip": 0,
            "compression_mode": "",
            "archive_mode": "",
            "stored_byte": 0,
            "timestamp": get_current_epoch_timestamp_in_ms(),
            "workflow_id": self._root_workflow_id,
            "pod_name": self._pod_name,
            "container_name": self._leaf_name,
            "checksum": "",
            "tags": json.dumps(self._artifacts_tags),  # TODO: whether alias have artifact_tags?
            "retention_tags": RETENTION_TAG_AX_LOG,
            "artifact_type": ARTIFACT_TYPE_AX_LOG,
            "deleted": 0,
        }

        try:
            logger.info("<step>: create return-code-artifact entry %s %s", self._leaf_full_path, db_data)
            artifact_client.create_artifact(artifact=db_data, max_retry=self._max_db_retry, retry_on_exception=retry_on_errors(errors=['ERR_API_INVALID_PARAM'], retry=False))
            logger.info("<step>: return-code-artifact %s created.", artifact_uuid)

        except Exception:
            logger.exception("insert return code %s to db failure through artifact manager", return_code)
            return False
        return True

    def _send_notification_deployment(self, event_type, value=None):
        event_map = {
            "LOADING_ARTIFACTS": (HeartBeatType.ARTIFACT_LOAD_START, "Init", "Starting to load artifacts"),
            "ARTIFACT_LOAD_FAILED": (HeartBeatType.ARTIFACT_LOAD_FAILED, "ArtifactPullFailed", str(value)),
        }
        try:
            event = event_map[event_type]
            timestamp = int(time.time())

            @retry(wait_fixed=10000)
            def send_hb():
                self._amclient.send_heart_beat(self._appname, self._pod_name, self._dep_id, event[0], timestamp, None, event[1], event[2])

            send_hb()
        except KeyError:
            logger.debug("Deployment notification for event {} not sent to AM".format(event_type))

    def _send_notification_report_status(self, event_type, value=None, max_retry=180):
        if self._new_deployment:
            self._send_notification_deployment(event_type, value=value)
            return

        if self._uuid:
            result_key = AXWorkflow.WFL_RESULT_KEY.format(self._uuid)
            result_list_key = AXWorkflow.REDIS_RESULT_LIST_KEY.format(self._uuid)

            p = {"event_type": event_type,
                 "uuid": self._uuid,
                 "timestamp": get_current_epoch_timestamp_in_ms()}

            if self._cookie:
                p["cookie"] = self._cookie

            if value:
                p.update(value)

            notification_result = self._send_notification(redis_list_key=result_list_key,
                                                          axdb_key=result_key,
                                                          value=p,
                                                          max_retry=max_retry)
            if notification_result:
                logger.info("<step>: reported event %s %s to key %s %s",
                            event_type, p, result_key, result_list_key)
                return True
            else:
                logger.critical("<step>: reported event %s %s to key %s %s failed",
                                event_type, p, result_key, result_list_key)
                return False
        else:
            logger.warning("no uuid, so no reporting")
            return False

    def _send_notification(self, redis_list_key, axdb_key, value, max_retry=180):
        redis_client = RedisClient(host=REDIS_HOST,
                                   db=DB_RESULT,
                                   retry_max_attempt=360,
                                   retry_wait_fixed=5000)

        count = 0
        while count < max_retry:
            count += 1
            try:
                axdb_client.put_workflow_kv(key=axdb_key, value=value)
                redis_client.rpush(key=redis_list_key, value=value,
                                   expire=AXWorkflow.REDIS_LIST_EXPIRE_SECONDS,
                                   encoder=json.dumps)
                return True
            except Exception:
                logger.exception("exception in send notification to %s %s", redis_list_key, axdb_key)
                time.sleep(10)
                pass
        logger.critical("cannot send notification to %s %s", redis_list_key, axdb_key)
        return False

    def _send_notification_report_return_code(self, return_code, k8s_info, message, max_retry=180):
        logger.info("<step>: reporting callback...")

        p = {"return_code": return_code,
             "name": self._container_name,
             "artifacts_outputs": self._artifacts_map,
             "logs": self._logs_map}
        if message:
            p["message"] = message
        if k8s_info:
            p["k8s_info"] = k8s_info

        notification_result = self._send_notification_report_status(value=p, event_type="HAVE_RESULT", max_retry=max_retry)
        if notification_result:
            logger.info("<step>: reported return_code %s p=%s",
                        return_code, p)
            return True
        else:
            logger.error("cannot report return_code %s of %s", return_code, p)
            return False

    @staticmethod
    def get_aws_format_checksum(binary_str):
        checksum = base64.b64encode(binary_str)
        return str(checksum.decode('utf-8'))

    @staticmethod
    def convert_hex_to_aws_format_checksum(hex_str):
        binary_from_hex = binascii.unhexlify(hex_str)
        checksum = base64.b64encode(binary_from_hex)
        return str(checksum.decode('utf-8'))

    @staticmethod
    def hashfile(afile, hasher, blocksize=65536):
        """Compute checksum using memory efficiently."""
        buf = afile.read(blocksize)
        while len(buf) > 0:
            hasher.update(buf)
            buf = afile.read(blocksize)
        return hasher.digest(), str(hasher.hexdigest())


def get_current_epoch_timestamp_in_ms():
    import time
    return int(time.time() * 1000)


def signal_debugger(signal_num, frame):
    logger.info("Artifact outer_executor debugged with signal %s", signal_num)
    result = traceback_multithread(signal_num, frame)
    logger.info(result)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='manage ax cluster',
                                     formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    parser.add_argument('--docker-inspect-result', help='file that contains the "docker inspect" result of this image')
    parser.add_argument('--host-scratch-root', help='the path of root scratch directory')
    parser.add_argument('--container-scratch-root', help='the path of root scratch directory in container')
    parser.add_argument('--executor-sh', help='the path of in container executor')
    parser.add_argument('--input-label', help='the input directory postfix')
    parser.add_argument('--output-label', help='the output directory postfix')
    parser.add_argument('--pod-name', help='the pod name')
    parser.add_argument('--job-name', help='the job name')
    parser.add_argument('--pod-ip', help='the pod ip')
    parser.add_argument('--post-mode', help='do post mode', action="store_true")
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    args = parser.parse_args()

    assert args.docker_inspect_result
    assert args.host_scratch_root
    assert args.container_scratch_root
    assert args.executor_sh
    assert args.input_label
    assert args.output_label

    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d: %(message)s")
    logging.getLogger("ax").setLevel(logging.DEBUG)

    signal.signal(signal.SIGUSR1, signal_debugger)

    executor = ContainerOuterExecutor(args.docker_inspect_result,
                                      args.host_scratch_root,
                                      args.container_scratch_root,
                                      args.executor_sh,
                                      args.input_label,
                                      args.output_label,
                                      args.pod_name,
                                      args.job_name,
                                      args.pod_ip,
                                      post_mode=args.post_mode)

    if executor.run():
        logger.info("Executor ran without error")
        sys.exit(0)
    else:
        logger.error("Executor ran into error")
        sys.exit(0)
