#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
import json
import logging
import os
import subprocess
import sys
from threading import Thread
import time
import uuid

from ax.devops.artifact.constants import RETENTION_TAG_DEFAULT, RETENTION_TAG_AX_LOG, RETENTION_TAG_AX_LOG_EXTERNAL, \
    RETENTION_TAG_USER_LOG, RETENTION_TAG_LONG_RETENTION, ARTIFACT_TYPE_AX_LOG, ARTIFACT_TYPE_AX_LOG_EXTERNAL, ARTIFACT_TYPE_USER_LOG
from ax.devops.client.artifact_client import AxArtifactManagerClient
from ax.devops.utility.utilities import retry_on_errors
from ax.kubernetes.client import KubernetesApiClient
from ax.meta import AXClusterId, AXLogPath, AXClusterDataPath
from ax.platform.container_specs import is_ax_aux_container
from ax.platform.exceptions import AXPlatformException
from ax.util.ax_artifact import AXArtifacts
from ax.cloud import Cloud

from inotify.calls import InotifyError
from retrying import retry


if sys.platform == "darwin":
    # Mac does not have inotify support,
    # this only makes tests easier
    import inotify as Inotify
else:
    from inotify.adapters import Inotify


logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)

artifact_client = AxArtifactManagerClient()


class LogEvents(object):
    IN_CREATE = "IN_CREATE"
    IN_OPEN = "IN_OPEN"
    IN_ATTRIB = "IN_ATTRIB"
    IN_CLOSE_WRITE = "IN_CLOSE_WRITE"
    IN_MOVED_FROM = "IN_MOVED_FROM"
    IN_MOVED_TO = "IN_MOVED_TO"


def _log_exception_during_retry(exception):
    """ For debugging exception during retry """
    logger.warning("ContainerLogCollector retries with exception %s", exception)
    return True


class PodLogManager(object):
    """
    This manager spins up threads that run as daemon along with `wait_for_container()`. It uses
    inotify to monitor changes inside log directory and uploads rotated logs
    to S3 bucket.

    It does NOT handle logs that are not rotated - it's container_outer_executor's job
    This thread manages logs for 1 container

    Kubernetes has docker-container configuration for logrotate as follows in their salt

        /var/lib/docker/containers/*/*-json.log {
            rotate 5
            copytruncate
            missingok
            notifempty
            compress
            maxsize 10M
            daily
            dateext
            dateformat -%Y%m%d-%s
            create 0644 root root
        }

    """
    def __init__(self, pod_name, service_id, root_id, leaf_full_path, namespace="axuser", app_mode=False):
        """
        Initialize information.
        :param pod_name: We collect log for this pod
        :param service_id: ServiceID (job) / DeploymentID (application)
        :param root_id: WorkflowID (job) / ApplicationID (application)
        :param leaf_full_path: WorkflowPath (job) / DeploymentName (application)
        :param app_mode: upload xxx-json.log upon termination
        :param apprecord ApplicationRecord singleton
        """
        self._pod_name = pod_name
        self._namespace = namespace
        self._kubectl = KubernetesApiClient()

        self._service_id = service_id
        self._root_id = root_id
        self._leaf_full_path = leaf_full_path
        self._log_root = os.getenv("LOGMOUNT_PATH")
        # key:val = cid:cname
        self._container_info = {}
        self._local_log_dirs = {}
        self._bucket = None
        self._log_s3_prefix = None
        self._bucket_ax = None
        self._log_s3_prefix_ax = None

        self._collectors = {}
        self._app_mode = app_mode

        self._set_s3()

    def _set_s3(self):
        """
        Set bucket, log_s3_prefix, s3_processor
        """
        logger.info("Setting up s3 ...")

        cluster_name_id = AXClusterId().get_cluster_name_id()

        self._bucket_name = AXClusterDataPath(cluster_name_id).bucket()
        self._bucket = Cloud().get_bucket(self._bucket_name)
        artifact_prefix = AXClusterDataPath(cluster_name_id).artifact()
        self._log_s3_prefix = artifact_prefix

        self._bucket_ax_is_external = AXLogPath(cluster_name_id).is_external()
        self._bucket_name_ax = AXLogPath(cluster_name_id).bucket()
        self._bucket_ax = Cloud().get_bucket(self._bucket_name_ax)
        artifact_prefix_ax = AXLogPath(cluster_name_id).artifact()

        self._log_s3_prefix_ax = artifact_prefix_ax

        assert self._bucket.exists(), "S3 bucket {} DOES NOT exist".format(self._bucket_name)
        assert self._bucket_ax.exists(), "S3 bucket {} DOES NOT exist".format(self._bucket_name_ax)
        logger.info("Using S3 bucket %s, with log prefix %s", self._bucket.get_bucket_name(), self._log_s3_prefix)
        logger.info("Using S3 bucket %s, with log prefix %s for AX", self._bucket_ax.get_bucket_name(), self._log_s3_prefix_ax)

    def start_log_watcher(self, cname, cid):
        logger.info("Starting log collector for container %s (%s)", cname, cid)
        path = os.path.join(self._log_root, cid)
        if cid in self._collectors:
            logger.info("Log collector for container %s (%s) has already started", cname, cid)
            return
        assert os.path.isdir(path), "Log path {} is not a valid directory".format(path)
        self._container_info[cid] = cname
        try:
            collector = ContainerLogCollector(
                pod_name=self._pod_name,
                namespace=self._namespace,
                watch_dir=path,
                cid=cid,
                cname=self._container_info[cid],
                service_id=self._service_id,
                root_id=self._root_id,
                full_path=self._leaf_full_path,
                bucket=self._bucket,
                bucket_name=self._bucket_name,
                s3_prefix=self._log_s3_prefix,
                bucket_ax_is_external=self._bucket_ax_is_external,
                bucket_ax=self._bucket_ax,
                bucket_name_ax=self._bucket_name_ax,
                s3_prefix_ax=self._log_s3_prefix_ax,
                app_mode=self._app_mode
            )
            self._collectors[cid] = collector
            collector.start()
            self._local_log_dirs[cid] = path
            logger.info("Watching logs on %s", path)
        except Exception as e:
            logger.exception("%s", e)

    def stop_log_watcher(self, cid):
        """
        Stop a single log watcher
        :param cid:
        :return:
        """
        if not self._collectors.get(cid, None):
            return
        self._collectors[cid].terminate()
        log_dir = self._local_log_dirs[cid]
        # Touch a file so the collectors can check its "terminate" flag
        sig_file_name = os.path.join(log_dir, ".ax_go_ipo")
        try:
            subprocess.check_call(["touch", sig_file_name])
            subprocess.check_call(["rm", sig_file_name])
        except subprocess.CalledProcessError as cpe:
            logger.error("Cannot create sigfile with error %s", cpe)
        self._collectors[cid].join()
        self._collectors.pop(cid, None)

    def terminate(self):
        for cid in list(self._collectors.keys()):
            self.stop_log_watcher(cid)
        logger.info("All log collectors terminated")

    def is_active(self):
        return len(self._collectors) > 0

    def get_containers(self):
        return self._collectors.keys()


class ContainerLogCollector(Thread):
    """
    Collects and send request to upload logs
    Two assumptions:
       1. inotify returns timely, i.e. once there is some change, it reports event (Lets trust kernel for now)
       2. It takes s3 less time to upload a rotated log than new log gets bigger than 10M
    NOTE: systematic change might be needed if these 2 assumptions are no longer valid
    """
    def __init__(self, pod_name, namespace, watch_dir, cid, cname, service_id, root_id, full_path,
                 bucket, bucket_name, s3_prefix, bucket_ax_is_external,
                 bucket_ax, bucket_name_ax, s3_prefix_ax, app_mode):
        super(ContainerLogCollector, self).__init__()
        self.name = "log-collector-{}.{}".format(pod_name, cname)

        self._pod_name = pod_name
        self._namespace = namespace

        self._terminate = False
        self._app_mode = app_mode
        self._db = None

        self._log_watcher = None
        self._watch_dir = watch_dir

        self._target_container_id = cid
        self._target_container_name = cname

        self._service_id = service_id
        self._root_id = root_id
        self._full_path = full_path

        self._bucket = bucket
        self._bucket_name = bucket_name
        self._s3_archive_prefix = s3_prefix
        self._bucket_ax_is_external = bucket_ax_is_external
        self._bucket_ax = bucket_ax
        self._bucket_name_ax = bucket_name_ax
        self._s3_archive_prefix_ax = s3_prefix_ax

        # A dictionary records events of ".gz" file for compressing current ".log" file to ".gz" file.
        # key : val = file_name : file_last_state
        self._file_records = {}
        self._set_log_watcher()

    @property
    def watch_dir(self):
        return self._watch_dir

    def terminate(self):
        self._terminate = True

    def is_busy(self):
        logger.debug("File records for %s during busy check: %s", self.name, self._file_records)
        return len(self._file_records) != 0

    def _check_and_upload_rotated_logs(self):
        """
        Check if there is any already rotated log. Because docker json-file log driver
        is doing file renaming, we always upload all rotated logs
        :return:
        """
        dir_content = os.listdir(self._watch_dir)
        cur_files = []
        for f in dir_content:
            full_path = os.path.join(self._watch_dir, f)
            if "-json.log." in f and os.path.isfile(full_path) and self._target_container_id in f:
                cur_files.append(full_path)
        # .log.x files will be arranged from earliest to most recent to ensure timestamp
        cur_files.sort(reverse=True)

        logger.debug("Current log directory: %s, content: %s",
                     self._watch_dir, cur_files)
        for f in cur_files:
            self._persist_log_artifact(f)

    def _set_log_watcher(self):
        """
        set log_watcher
        """
        logger.info("Setting log watcher ...")
        self._log_watcher = Inotify()
        self._log_watcher.add_watch(self._watch_dir.encode('utf-8'))

    def _persist_log_artifact(self, fname):
        local_file = os.path.join(self._watch_dir, fname)
        if os.path.isfile(local_file):
            ts = str(int(time.time() * 1000))
            artifact_name, full_name = self._generate_artifact_name(ts, fname.endswith("gz"))
            logger.info("Uploading log file %s, %s", local_file, full_name)
            self._do_save_artifacts(path1=local_file,
                                    timestamp=ts,
                                    artifact_name=artifact_name,
                                    full_name=full_name)
        else:
            logger.warning("Log %s is not a file, not uploading", local_file)

    def _do_save_artifacts(self, path1, timestamp, artifact_name, full_name):
        """
        Upload artifacts to s3 and save information to DB.
        :param path1: local file path if action is "UPLOAD"
        :param timestamp: Time we start to process this log artifact
        :param artifact_name: name of artifacts we persist to DB
        :param full_name: full name of the log
        :return:
        """

        @retry(retry_on_exception=_log_exception_during_retry,
               wait_exponential_multiplier=1000,
               stop_max_attempt_number=3)
        def _upload(s, d, meta_data):
            # To be consistent with file uploaded from container_outer_executor
            #   - StorageClass is by default "STANDARD", and
            #   - ContentLength is used only when file size cannot be automatically
            #     determined, but as we upload .gz files after IN_CLOSE_WRITE, we
            #     don't need to set it
            extra_args = {
                "ACL": "bucket-owner-full-control",
                "ServerSideEncryption": "AES256",
                "Metadata": meta_data,
                "ContentDisposition": "attachment; filename={}".format(full_name)
            }
            logger.info("about to upload log %s to %s (%s) to s3", s, d, artifact_uuid)
            if not bucket.put_file(local_file_name=s, s3_key=d, ExtraArgs=extra_args):
                raise AXPlatformException("Failed to put object {} to s3 {}".format(s, d))
            logger.debug("upload %s done", artifact_uuid)

        def _add_artifact(artifact_uuid, meta_data):
            try:
                stored_byte = os.stat(path1).st_size
                db_data = {
                    "artifact_id": artifact_uuid,
                    "service_instance_id": self._service_id,
                    "full_path": self._full_path,
                    "name": artifact_name,
                    "description": "ax container log",
                    "storage_method": "s3",
                    "storage_path": json.dumps({"bucket": bucket_name, "key": s3_full_path}),
                    "num_byte": stored_byte,
                    "num_dir": 0,
                    "num_file": 1,
                    "num_other": 0,
                    "num_skip_byte": 0,
                    "num_skip": 0,
                    "pod_name": self._pod_name,
                    "container_name": self._target_container_name,
                    "compression_mode": "gz" if path1.endswith(".gz") else "",
                    "archive_mode": "",
                    "stored_byte": stored_byte,
                    "meta": json.dumps(meta_data),
                    "timestamp": timestamp,
                    "workflow_id": self._root_id,
                    "checksum": "",  # xxx todo
                    "tags": json.dumps([]),
                    "retention_tags": retention,
                    "artifact_type": artifact_type,
                    "deleted": 0,
                }

                logger.info("about to create_artifact artifact %s (%s)", artifact_uuid, db_data)
                artifact_client.create_artifact(artifact=db_data, max_retry=150,
                                                retry_on_exception=retry_on_errors(errors=['ERR_API_INVALID_PARAM'],
                                                                                   retry=False))
                logger.debug("artifact %s created", artifact_uuid)
            except Exception:
                logger.exception("insert log %s to db failure through artifact manager.", full_name)

        artifact_uuid = str(uuid.uuid4())
        meta_data = {"ax_artifact_id": artifact_uuid,
                     "ax_container_log": "True",
                     "ax_timestamp": timestamp
                     }

        if is_ax_aux_container(self._target_container_name):
            retention = RETENTION_TAG_AX_LOG_EXTERNAL if self._bucket_ax_is_external else RETENTION_TAG_AX_LOG
            artifact_type = ARTIFACT_TYPE_AX_LOG_EXTERNAL if self._bucket_ax_is_external else ARTIFACT_TYPE_AX_LOG
            bucket_name = self._bucket_name_ax
            bucket = self._bucket_ax
            s3_full_path = AXArtifacts.gen_artifact_path(prefix=self._s3_archive_prefix_ax,
                                                         root_id=self._root_id,
                                                         service_id=self._service_id,
                                                         add_date=True,
                                                         name=artifact_name)
        else:
            retention = RETENTION_TAG_USER_LOG
            artifact_type = ARTIFACT_TYPE_USER_LOG
            bucket_name = self._bucket_name
            bucket = self._bucket
            s3_full_path = AXArtifacts.gen_artifact_path(prefix=self._s3_archive_prefix,
                                                         root_id=self._root_id,
                                                         service_id=self._service_id,
                                                         add_date=False,
                                                         name=artifact_name)

        _upload(path1, s3_full_path, meta_data)
        _add_artifact(artifact_uuid, meta_data)

        record_key = os.path.basename(path1)
        logger.info("Uploaded %s to %s. Removing %s from record %s", path1, s3_full_path, record_key, self._file_records)
        try:
            del self._file_records[record_key]
        except KeyError:
            pass

    def _generate_artifact_name(self, timestamp, log_compressed):
        artifact_name = "{cname}.{cid}.log.{ts}".format(cname=self._target_container_name,
                                                        cid=self._target_container_id,
                                                        ts=timestamp)
        if log_compressed:
            artifact_name += ".gz"
        if self._full_path:
            full_name = self._full_path + "." + artifact_name
        else:
            full_name = artifact_name
        return artifact_name, full_name

    def _process_log_gz_create(self, event, fname):
        if self._file_records.get(fname, None):
            raise AXPlatformException("Log {} rotated while previous log is not uploaded.".format(fname))
        self._file_records[fname] = event
        self._persist_log_artifact(fname)

    def _ok_to_terminate(self):
        """
        Shutdown routine. If return true, main thread is safe to return cleanly
        :return:
        """
        if self._terminate and not self.is_busy():
            logger.info("ContainerLogCollector for %s terminating", self._target_container_id)
            try:
                self._log_watcher.remove_watch(self._watch_dir.encode('utf-8'))
                logger.info("ContainerLogCollector for %s successfully removed inotify watch",
                            self._target_container_id)
            except InotifyError:
                logger.warning("Inotify error during remove_watch")
            return True
        return False

    def _do_house_keeping(self):
        if self._app_mode:
            try:
                # For application, upload unrotated log
                self._persist_log_artifact(fname=os.path.join(self._watch_dir,
                                                              "{}-json.log".format(self._target_container_id)))
            except OSError as oe:
                if "No such file or directory" in str(oe):
                    logger.warning("Log directory has been removed. Log collector quitting. Detail: %s", oe)
                else:
                    logger.exception("Exception caught during house keeping. Detail: %s", oe)

    def run(self):
        try:
            self._check_and_upload_rotated_logs()
        except Exception as e:
            logger.exception("ContainerLogCollector for %s caught exception %s", self._target_container_id, e)

        while True:
            logger.info("ContainerLogCollector for %s running ...", self._target_container_id)
            try:
                for event in self._log_watcher.event_gen():
                    if self._ok_to_terminate():
                        self._do_house_keeping()
                        return

                    if not event:
                        continue
                    (header, type_names, watch_path, filename) = event
                    if not type_names or not isinstance(type_names, list):
                        continue

                    watch_path = watch_path.decode('utf-8')
                    if watch_path != self._watch_dir:
                        logger.error("Received watch path \"%s\" is not expected. Expected \"%s\"", watch_path,
                                     self._watch_dir)
                        continue

                    filename = filename.decode('utf-8')
                    if not filename.endswith("-json.log.1"):
                        # We only upload log when xxx-json.log is rotated to xxx-json.log.1
                        continue

                    if self._target_container_id not in filename:
                        logger.error("Zipped file \"%s\" does not container correct container id. Expected \"%s\"",
                                     filename, self._target_container_id)
                        continue

                    for t in type_names:
                        if t == LogEvents.IN_MOVED_TO:
                            logger.info("Processing %s for file %s", t, filename)
                            self._process_log_gz_create(event=t, fname=filename)

                            if self._ok_to_terminate():
                                self._do_house_keeping()
                                return
            except Exception as e:
                logger.exception("ContainerLogCollector for %s caught exception %s", self._target_container_id, e)

