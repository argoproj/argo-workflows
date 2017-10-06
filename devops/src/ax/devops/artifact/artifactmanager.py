# Copyright 2015-2016 Applatix, Inc. All rights reserved.

import copy
import json
import logging
import threading
import time
from retrying import retry

from ax.cloud import Cloud
from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.client.amm_client import AxAMMClient
from ax.devops.artifact.constants import DEFAULT_PROCESS_INTERVAL, MILLISECONDS_PER_MIN, MILLISECONDS_PER_HOUR, MILLISECONDS_PER_DAY, MILLISECONDS_PER_WEEK, \
    MILLISECONDS_PER_MONTH, FLAG_ALIAS, FLAG_NOT_DELETED, FLAG_DELETED, FLAG_TO_BE_DELETED, FlAG_EXPIRED, RETENTION_TAG_DEFAULT, RETENTION_TAG_AX_LOG, RETENTION_TAG_AX_LOG_EXTERNAL, \
    RETENTION_TAG_LONG_RETENTION, FLAG_IS_NOT_ALIAS, OPERATION_CREATE, OPERATION_DELETE, OPERATION_RESTORE, ARTIFACT_NUMS, ARTIFACT_SIZE, \
    MILLISECONDS_PER_SECOND, DEFAULT_PAGE_SIZE
from ax.devops.utility.utilities import aggregate_numeric_dictionaries, ax_profiler, get_epoch_time_in_ms, get_error_code, retry_on_errors
from ax.exceptions import AXApiInvalidParam, AXApiInternalError

from ax.cloud.aws import AXS3Bucket

logger = logging.getLogger(__name__)
logging.getLogger('ax.devops.axrequests').setLevel(logging.CRITICAL)
logging.getLogger('ax.devops.utility.utilities').setLevel(logging.ERROR)

RETRY_INTERVAL = 1000
RETRY_MAX_NUMBER = 100


def conditional_update_retry_on_exception(e):
    """Retry on conditional update failure

    :param e:
    :returns:
    """
    errors = ['ERR_AXDB_CONDITIONAL_UPDATE_FAILURE']
    return retry_on_errors(errors=errors, retry=True, caller=__name__)(e)


class ArtifactManager(object):
    """Artifact manager"""

    json_columns = {
        'artifact': {
            'checksum',
            'meta',
            'storage_path',
            'structure_path',
            'tags'
        }
    }

    def __init__(self):
        self.process_interval = DEFAULT_PROCESS_INTERVAL
        self._events = 0
        self._process_cv = threading.Condition()
        self._progress_cv = threading.Condition()
        self._background_thread_stop = False
        self._background_thread = None
        self._background_thread_lock = threading.Lock()
        self._space_aggregation_thread = None
        self._space_aggregation_dict = dict()
        self._space_aggregation_lock = threading.Lock()
        self.axdb_client = None

    def init(self):
        self.axdb_client = AxdbClient()

    def update_artifact_retention_tag(self, artifact_id, new_retention_tag):
        """Update an artifact's retention tag

        :param artifact_id:
        :param new_retention_tag:
        :return:
        """
        logger.info("Update artifact %s's retention tag to %s", artifact_id, new_retention_tag)
        artifact = self.get_artifact(artifact_id=artifact_id, resolve_alias=False)
        retention_policies = self.get_retention_policies(tag_name=new_retention_tag)
        if not retention_policies:
            message = detail = 'Policy with name ({}) does not exist'.format(new_retention_tag)
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)

        payload = {
            'artifact_id': artifact_id,
            'ax_uuid': artifact['ax_uuid'],
            'timestamp': get_epoch_time_in_ms(),
            'retention_tags': new_retention_tag
        }
        logger.info("Update artifact retention policy, %s", str(payload))
        self.axdb_client.update_artifact(payload=payload)

    def get_retention_policies(self, tag_name=None):
        """Get retention policies

        :param tag_name: string
        :returns: list of dictionaries
        """
        logger.info('Searching for retention policies (name: %s) ...', tag_name)
        results = self.axdb_client.get_retention_policies(tag_name=tag_name)
        for result in results:
            retention_tag = result['name']
            with self._space_aggregation_lock:
                if retention_tag in self._space_aggregation_dict:
                    result['total_number'] += self._space_aggregation_dict[retention_tag]['total_number']
                    result['total_size'] += self._space_aggregation_dict[retention_tag]['total_size']
                    result['total_real_size'] += self._space_aggregation_dict[retention_tag]['total_real_size']
        return results

    def add_retention_policy(self, tag_name, policy, description=None):
        """Add retention policy

        :param tag_name: string
        :param policy: integer
        :param description: string
        :returns:
        """
        retention_policies = self.get_retention_policies(tag_name=tag_name)
        if retention_policies:
            message = detail = 'Policy with name ({}) already exists'.format(tag_name)
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)
        if type(policy) != int:
            message = detail = 'Field "policy" expects integer'
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)
        logger.info('Adding retention policy (name: %s, policy: %s) ...', tag_name, policy)
        return self.axdb_client.create_retention_policy(tag_name=tag_name, policy=policy, description=description)

    def update_retention_policy(self, tag_name, policy=None, description=None):
        """Update retention policy

        :param tag_name: string
        :param policy: integer
        :param description:
        :returns:
        """
        retention_policies = self.get_retention_policies(tag_name=tag_name)
        if not retention_policies:
            message = detail = 'Policy with name ({}) does not exist'.format(tag_name)
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)
        if type(policy) != int:
            message = detail = 'Field "policy" expects integer'
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)
        logger.info('Updating retention policy (name: %s, policy: %s) ...', tag_name, policy)
        return self.axdb_client.update_retention_policy(tag_name=tag_name, policy=policy, description=description)

    def delete_retention_policy(self, tag_name):
        """Delete a retention policy

        Log and default retention policy cannot be deleted.

        :param tag_name:
        :returns:
        """
        retention_policies = self.get_retention_policies(tag_name=tag_name)
        if not retention_policies:
            message = detail = 'Policy with name ({}) does not exist'.format(tag_name)
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)
        if tag_name in {RETENTION_TAG_DEFAULT, RETENTION_TAG_AX_LOG}:
            message = detail = 'Not allowed to delete log/default retention policies'
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)
        logger.info('Deleting retention policy (name: %s) ...', tag_name)
        return self.axdb_client.delete_retention_policy(tag_name=tag_name)

    @retry(wait_fixed=5, stop_max_attempt_number=10)
    def _delete_file_from_s3(self, artifact_id, bucket, key):
        """Delete an artifact from S3

        :param artifact_id:
        :param bucket:
        :param key:
        :param max_retry:
        :returns:
        """
        logger.info('Deleting artifact (id: %s) from s3 (bucket: %s, key: %s) ...', artifact_id, bucket, key)
        Cloud().get_bucket(bucket).delete_object(key)
        return True

    def get_live_workflows(self):
        """
        Return a set of workflow ids of workflows that are running or workflows that are in deployment.
        :return:
        """
        result = set()
        live_workflows = self.axdb_client.get_live_workflow()
        for workflow in live_workflows:
            workflow_id = workflow.get('task_id', None)
            if workflow_id not in result:
                result.add(workflow_id)

        try:
            response = AxAMMClient.query_artifacts(conditions={'fields': 'task_id, status'})
            deployments = response.json().get("data", [])
            for deployment in deployments:
                if deployment['status'] != 'Terminated':
                    if deployment['task_id'] not in result:
                        result.add(deployment['task_id'])
        except Exception:
            logger.exception("Failed to retrieve running deployments from amm.")

        return result

    def check_artifact_retention_policy(self, artifact, live_workflows, retention_policies, counter, dry_run=False):
        """
        Check retention policy to decide whether delete from s3

        :param artifact:
        :param live_workflows:
        :param retention_policies:
        :param counter:
        :param dry_run:

        :returns:
            Deleted: True for deleted from s3. False for no deletion
            Number: 1 or 0 represents if the artifact is deleted
            stored_byte: number of bytes stored in s3
            num_byte: number of bytes before compression
            retention_tag: the retention policy the artifact associated with
        """

        try:
            logger.debug("No. %s Check artifact retention, artifact name: %s, uuid: %s, retention_tag: %s, dry_run: %s",
                         counter, artifact['name'], artifact['artifact_id'], artifact['retention_tags'], dry_run)

            # Check if the artifact is already deleted:
            if artifact['deleted'] != 0 and artifact['deleted'] != 2:
                return False, 0, 0, 0, None

            # Check if the workflow is still running, do not delete
            if artifact['workflow_id'] in live_workflows:
                return False, 1, artifact['stored_byte'], artifact['num_byte'], None

            retention_tag = artifact['retention_tags']

            # AA-2381 take care of old retention policies
            if retention_tag == 'default':
                retention_tag = RETENTION_TAG_DEFAULT
            elif retention_tag == 'long-retention':
                retention_tag = RETENTION_TAG_LONG_RETENTION
            elif retention_tag == RETENTION_TAG_AX_LOG_EXTERNAL:
                return False, 0, 0, 0, None

            # Check if retention_tag exists in the retention policies
            if retention_tag not in retention_policies:
                return False, 1, artifact['stored_byte'], artifact['num_byte'], None

            # Check if the artifact has artifact_tags, do not delete
            if artifact['tags'] and json.loads(artifact['tags']):
                return False, 1, artifact['stored_byte'], artifact['num_byte'], retention_tag

            retention_policy = retention_policies[retention_tag]
            current_time = get_epoch_time_in_ms()

            delete_status = 0
            # Check whether the artifact is expired
            if artifact['deleted'] == FLAG_NOT_DELETED and (int(artifact['timestamp']) + int(retention_policy) < current_time):
                delete_status = FLAG_DELETED

            # Check whether the artifact is manually delted by an user
            if artifact['deleted'] == FLAG_TO_BE_DELETED and (int(artifact['timestamp']) + MILLISECONDS_PER_DAY < current_time):
                delete_status = FlAG_EXPIRED

            if delete_status != 0:
                logger.info("Artifact %s, status %s, latest timestamp %s, retention policy %s, current time %s, met the retention deletion requirement",
                            artifact['name'], artifact['deleted'], int(artifact['timestamp']), retention_policy, current_time)
                if dry_run:
                    return True, 0, 0, 0, retention_tag
                if artifact['storage_path']:
                    s3_path = json.loads(artifact['storage_path'])
                    res = self._delete_file_from_s3(artifact_id=artifact['artifact_id'], bucket=s3_path['bucket'], key=s3_path['key'])
                    if res:
                        payload = dict()
                        payload['ax_uuid'] = artifact['ax_uuid']
                        payload['artifact_id'] = artifact['artifact_id']
                        payload['deleted'] = delete_status
                        payload['deleted_date'] = get_epoch_time_in_ms()

                        logger.info("Finishing delete, update artifact table, %s", str(payload))
                        self.axdb_client.update_artifact(payload=payload)
                        return True, 0, 0, 0, retention_tag

            return False, 1, artifact['stored_byte'], artifact['num_byte'], retention_tag
        except Exception as exc:
            logger.exception("Failed to check whether to delete artifact %s based on retention, %s", str(artifact), str(exc))
            return False, 0, 0, 0, None

    def do_check_retention(self):
        """Retention policy checking mechanism"""
        ax_min_time = None
        logger.info("Start to check retention of artifacts.")
        page_counter = 0
        counter = 0
        final_res = dict()

        # Check the last time of a successful retention scan
        try:
            last_finish_time = int(self.axdb_client.get_artifact_meta('retention_last_finish_time')['value'])
            current_time = get_epoch_time_in_ms()
            delta_time = MILLISECONDS_PER_DAY - (current_time - last_finish_time)  # Calculate based on 1 day interval
            if delta_time > 0:
                logger.info("Last finish time is %s. Current time is %s. Going to sleep %s before retention scan starts.",
                            last_finish_time, current_time, delta_time)
                with self._progress_cv:
                    if self._progress_cv.wait(delta_time/1000):
                        logger.info("User manually triggered retention thread.")
                    else:
                        logger.info("%s seconds elapsed. Forcing start retention thread.", delta_time)
        except Exception:
            logger.exception("Failed to retrieve last successful scan time. Start to scan now.")

        while True:
            try:
                page_counter += 1
                logger.info("Start to scan the %s page for retention", page_counter)

                if ax_min_time is None:
                    # Check database to see if any saved retention progress exists
                    try:
                        ax_min_time = int(self.axdb_client.get_artifact_meta('retention_progress')['value'])
                        logger.info("Get retention progress starting point, %s", ax_min_time)
                    except Exception:
                        logger.exception("Failed to retention progress")
                        ax_min_time = 0

                payload = {
                    'ax_min_time': ax_min_time,
                    'storage_method': 's3',
                    'is_alias': FLAG_IS_NOT_ALIAS,
                    'ax_max_entries': DEFAULT_PAGE_SIZE,
                    'ax_orderby_asc': "[\"ax_uuid\"]",  # ensure an ascending order (from oldest to newest)
                }

                # Get list of live workflows, list of artifacts, list of retention policies
                live_workflows = self.get_live_workflows()
                artifacts = self.axdb_client.get_artifacts(payload)
                retention_policies = dict()

                # put all retention policies into a dict
                for item in self.axdb_client.get_retention_policies():
                    retention_policies[item['name']] = item['policy']

                if not artifacts:
                    logger.info("No more entries from artifact table. Stop retention process for now.")
                    break

                ax_min_time = artifacts[-1]['ax_time'] + 1  # update the ax_min_time

                for artifact in artifacts:

                    res_del, res_num, res_compressed, res_size, res_retention = \
                        self.check_artifact_retention_policy(artifact=artifact,
                                                             live_workflows=live_workflows,
                                                             retention_policies=retention_policies,
                                                             counter=counter)
                    logger.info("ax_time: %s, artifact uuid: %s, res_del: %s, res_num: %s, res_compressed: %s, res_size: %s, res_retention: %s",
                                artifact['ax_time'], artifact['artifact_id'], res_del, res_num, res_compressed, res_size, res_retention)
                    counter += 1

                    if res_retention:
                        if res_retention not in final_res:
                            final_res[res_retention] = dict()
                            final_res[res_retention]['total_number'] = 0
                            final_res[res_retention]['total_size'] = 0
                            final_res[res_retention]['total_real_size'] = 0

                        final_res[res_retention]['total_number'] += res_num
                        final_res[res_retention]['total_size'] += res_compressed
                        final_res[res_retention]['total_real_size'] += res_size

                logger.info("Space aggregation, %s", final_res)

                try:
                    self.axdb_client.update_artifact_meta(attribute='retention_progress', value=ax_min_time)
                except Exception:
                    logger.info("Failed to save retention progress to axdb")

            except Exception:
                logger.exception("Error: Failed to scan %s page for retention. This normally should not happen!", page_counter)

        # Reset retention progress since finish a complete sweep
        try:
            self.axdb_client.update_artifact_meta(attribute='retention_progress', value=0)
            self.axdb_client.update_artifact_meta(attribute='retention_last_finish_time', value=get_epoch_time_in_ms())
        except Exception:
            logger.info("Failed to reset retention progress to axdb")

        # Refresh in-memory space accounting before updating axdb
        with self._space_aggregation_lock:
            self._space_aggregation_dict = dict()

        for key, value in final_res.items():
            try:
                # A possible update failure will lose 30 (sleep_interval_second) minute
                # intermediate space accounting for the retention
                self.axdb_client.update_retention_policy_metadata(tag_name=key,
                                                                  total_number=value['total_number'],
                                                                  total_size=value['total_size'],
                                                                  total_real_size=value['total_real_size'])
            except Exception:
                logger.exception("Failed to update metadata of the retention policy in db %s", key)

    def _start_retention_thread(self):
        """Background thread to check retention policy"""
        while True:
            try:
                if self._events == 0:
                    with self._process_cv:
                        # Wait until next process interval, or we are notified of a change, whichever comes first
                        logger.debug("Waiting for event or process interval")
                        if self._process_cv.wait(timeout=self.process_interval):
                            logger.debug("Notified of change event")
                        else:
                            logger.debug("%s seconds elapsed. Forcing processing", self.process_interval)
                if self._background_thread_stop:
                    logger.debug("Stop requested. Exiting request processor")
                    return
                with self._process_cv:
                    logger.debug("%s events occurred since last processing time", self._events)
                    self._events = 0

                # Check to delete artifacts based on retention policy
                self.do_check_retention()

            except Exception as exp:
                logger.exception("Request processor failed, %s", str(exp))

    def _start_space_aggregation_thread(self):
        # Update every hour
        sleep_interval_second = 60 * 30
        while True:
            try:
                time.sleep(sleep_interval_second)
                logger.info("Space aggregation thread wake up. %s", self._space_aggregation_dict)
                retention_policies = self.axdb_client.get_retention_policies()
                retention_policies_copy = copy.deepcopy(retention_policies)
                update_list = list()
                with self._space_aggregation_lock:
                    for idx, retention_policy in enumerate(retention_policies_copy):
                        retention_tag = retention_policy['name']
                        if retention_tag in self._space_aggregation_dict:
                            logger.info("Previous retention accounting: %s", retention_policy)
                            retention_policy['total_number'] += self._space_aggregation_dict[retention_tag]['total_number']
                            retention_policy['total_size'] += self._space_aggregation_dict[retention_tag]['total_size']
                            retention_policy['total_real_size'] += self._space_aggregation_dict[retention_tag]['total_real_size']
                            update_list.append(idx)
                            logger.info("Updated retention accounting: %s", retention_policy)
                    # Reset in-memory space aggregation
                    self._space_aggregation_dict = dict()
                for idx in update_list:
                    # A possible update failure will cause one or more retention lose
                    # less than 30 (sleep_interval_second) minute space accounting for
                    # at most 1 day (self.process_interval)
                    self.axdb_client.update_retention_policy_meta_conditionally(retention_policies[idx], retention_policies_copy[idx])
            except Exception:
                logger.exception("space aggregation thread exception")

    def start_background_process(self):
        """Start background thread for checking retention policy"""
        with self._background_thread_lock:
            if self._background_thread is None:
                logger.info("Background retention thread starting")
                self._background_thread = threading.Thread(target=self._start_retention_thread,
                                                           name="axam_retention_thread",
                                                           daemon=True)
                self._background_thread.start()
                self._space_aggregation_thread = threading.Thread(target=self._start_space_aggregation_thread,
                                                                  name="axam_space_aggregation",
                                                                  daemon=True)
                self._space_aggregation_thread.start()
            else:
                logger.info("Background retention thread already started")

    def stop_background_process(self):
        """Stop background thread for checking retention policy"""
        with self._background_thread_lock:
            if self._background_thread:
                logger.info("Background retention thread stopping")
                self._background_thread_stop = True
                self._trigger_processor()
                self._background_thread.join()
                self._background_thread = None
                self._background_thread_stop = False
                logger.info("Background retention thread stopped")
            else:
                logger.info("Background retention thread already stopped")

    def _trigger_processor(self):
        """Internal trigger to notify background thread to check retention"""
        with self._process_cv:
            self._events += 1
            self._process_cv.notify()
            with self._progress_cv:
                self._progress_cv.notify()

    def show(self):
        """Show status of artifact manager"""
        ret = dict()

        policies = self.get_retention_policies()
        for policy in policies:
            policy['readable'] = self.convert_ms_to_str(policy['policy'])

        ret['retention_policy'] = policies

        to_be_deleted = list()
        ax_max_time = get_epoch_time_in_ms() * 1000
        counter = 0
        while True:
            try:
                payload = {
                    'ax_max_time': ax_max_time,
                    'is_alias': FLAG_IS_NOT_ALIAS,
                    'storage_method': 's3',
                    'deleted': 0,
                }
                live_workflows = self.get_live_workflows()
                artifacts = self.axdb_client.get_artifacts(payload)
                retention_policies = dict()

                # put all retention policies into a dict
                for item in self.axdb_client.get_retention_policies():
                    retention_policies[item['name']] = item['policy']

                if not artifacts:
                    break
                ax_max_time = artifacts[-1]['ax_time']  # last entry's timestamp used for next query

                for artifact in artifacts:
                    res_del, _, _, _, _ = self.check_artifact_retention_policy(artifact=artifact,
                                                                               live_workflows=live_workflows,
                                                                               retention_policies=retention_policies,
                                                                               counter=counter)
                    counter += 1
                    if res_del:
                        to_be_deleted.append(artifact)
            except Exception as exc:
                logger.exception("Failed to retrieve information from axdb, %s", str(exc))
                return False
        ret['next_sweep_will_delete'] = to_be_deleted

        new_payload = {
            'storage_method': 's3',
            'deleted': 1,
            'is_alias': FLAG_IS_NOT_ALIAS,
        }
        ret['deleted'] = self.axdb_client.get_artifacts(new_payload)

        new_payload = {
            'storage_method': 's3',
            'deleted': 2,
            'is_alias': FLAG_IS_NOT_ALIAS,
        }
        ret['temporary_deleted'] = self.axdb_client.get_artifacts(new_payload)

        return ret

    def get_artifacts(self, **kwargs):
        """Search artifacts

        :param kwargs:
        :returns:
        """
        logger.info('Searching for artifacts (%s) ...', kwargs)
        # Construct query
        params = {}
        for k in {'artifact_id', 'workflow_id', 'service_instance_id', 'name', 'is_alias', 'pod_name', 'container_name', 'artifact_type'}:
            if k in kwargs:
                params[k] = kwargs[k]
        artifacts = self.axdb_client.get_artifacts(params)

        # Construct filter
        resolve_alias = kwargs.pop('resolve_alias', True)
        deserialize = kwargs.pop('deserialize', True)
        filtered_artifacts = []
        theoretical_count = 0
        for i in range(len(artifacts)):
            if 'deleted' in kwargs and artifacts[i]['deleted'] not in kwargs['deleted']:
                continue
            if 'retention_tags' in kwargs and artifacts[i]['retention_tags'] not in kwargs['retention_tags']:
                continue
            if 'tags' in kwargs:
                if artifacts[i]['tags']:
                    tags = set(json.loads(artifacts[i]['tags']))
                else:
                    tags = set()
                if not tags.intersection(set(kwargs['tags'])):
                    continue
            artifact = artifacts[i]
            theoretical_count += 1
            if resolve_alias:
                try:
                    artifact = self.resolve_artifact_alias(artifact)
                except AXApiInvalidParam:
                    logger.warning('Unable to resolve artifact alias (%s), skip', artifact['artifact_id'])
                    continue
            if deserialize:
                artifact = self.deserialize_json_columns('artifact', artifact)
            filtered_artifacts.append(artifact)
        logger.info('Found %s artifacts among which %s are valid', theoretical_count, len(filtered_artifacts))
        return sorted(filtered_artifacts, key=lambda artifact: artifact['ax_time'])

    def get_artifact(self, artifact_id, **kwargs):
        """Get an artifact by artifact ID

        :param artifact_id:
        :param kwargs:
        :returns: artifact payload
        """
        artifacts = self.get_artifacts(artifact_id=artifact_id, **kwargs)
        if len(artifacts) > 1:
            message = 'Unexpectedly found more than 1 artifacts'
            detail = message + ' with given ID ({})'.format(artifact_id)
            logger.error(detail)
            raise AXApiInternalError(message, detail)
        if len(artifacts) < 1:
            message = 'Unable to find artifact'
            detail = message + ' with given ID ({})'.format(artifact_id)
            raise AXApiInvalidParam(message, detail)
        artifact = artifacts[0]
        return artifact

    def resolve_artifact_alias(self, artifact):
        """Resolve alias artifact to its source artifact

        :param artifact:
        :returns:
        """
        is_alias = artifact.get('is_alias', FLAG_IS_NOT_ALIAS)
        if is_alias:
            try:
                real_artifact = self.get_artifact(artifact_id=artifact['source_artifact_id'], resolve_alias=False)
                for k in ['artifact_id', 'artifact_type', 'description', 'full_path', 'is_alias', 'name',
                          'retention_tags', 'service_instance_id', 'source_artifact_id']:
                    real_artifact[k] = artifact[k]
                return real_artifact
            except AXApiInvalidParam:
                message = 'Orphaned alias artifact'
                detail = 'Artifact ({}) is an alias but has no valid source artifact'.format(artifact['artifact_id'])
                logger.error(detail)
                raise AXApiInvalidParam(message, detail)
        else:
            return artifact

    def deserialize_json_columns(self, object_type, object):
        """Deserialize json columns when returning results

        :param object_type:
        :param object:
        :returns:
        """
        json_columns = self.json_columns.get(object_type)
        if json_columns:
            for k in json_columns:
                try:
                    v = object[k]
                    if isinstance(v, dict):
                        continue
                    if isinstance(v, list):
                        continue
                    object[k] = json.loads(object[k])
                except ValueError:
                    continue
                except TypeError:
                    logger.error("k=%s o[k]=%s o=%s", k, object[k], object)
                    continue
        return object

    @staticmethod
    def _compare_artifact(a0, a1):
        a0 = copy.deepcopy(a0)
        a1 = copy.deepcopy(a1)
        for a in [a0, a1]:
            for k in ['ax_week', 'ax_time', 'ax_uuid']:
                a.pop(k, None)

        for s in [(a0, a1), (a1, a0)]:
            for k in s[0]:
                v0 = s[0].get(k)
                v1 = s[1].get(k, None)
                if isinstance(v0, list):
                    v0 = str(v0)
                if isinstance(v1, list):
                    v1 = str(v1)
                if (v0 != v1) and v0:
                    logger.debug("artifact not the same: key=%s v0=%s v1=%s", k, v0, v1)
                    return False
        return True

    def _update_in_memory_space_accounting(self, payload, is_addition):
        if 'storage_method' not in payload or payload['storage_method'] != 's3':
            return
        if 'is_alias' not in payload or payload['is_alias'] != FLAG_IS_NOT_ALIAS:
            return

        multiplier = 1 if is_addition else -1

        with self._space_aggregation_lock:
            retention_tag = payload['retention_tags']
            if retention_tag not in self._space_aggregation_dict:
                self._space_aggregation_dict[retention_tag] = dict()
                self._space_aggregation_dict[retention_tag]['total_number'] = 0
                self._space_aggregation_dict[retention_tag]['total_size'] = 0
                self._space_aggregation_dict[retention_tag]['total_real_size'] = 0
            self._space_aggregation_dict[retention_tag]['total_number'] += int(multiplier * 1)
            self._space_aggregation_dict[retention_tag]['total_size'] += int(multiplier * payload['stored_byte'])
            self._space_aggregation_dict[retention_tag]['total_real_size'] += int(multiplier * payload['num_byte'])

    @ax_profiler([('axdb_client', 'create_artifact'),
                  ('axdb_client', 'get_artifacts'),
                  ('artifactmanager', '_update_artifact_nums_and_size')])
    def create_artifact(self, payload):
        """Create an artifact

        :param payload:
        :returns:
        """
        artifact_id = payload.get('artifact_id')
        if not artifact_id:
            message = detail = 'Missing artifact ID'
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)

        artifacts = self.get_artifacts(artifact_id=artifact_id, resolve_alias=False)
        if artifacts:
            if len(artifacts) == 1:
                old = artifacts[0]
                del old['ax_time']
                if self._compare_artifact(old, payload):
                    logger.info("Same artifact %s already exists.", artifact_id)
                    return
            message = 'Artifact already exists'
            detail = 'Different artifact with same id ({}) already exists. {} vs {}'.format(artifact_id, payload, artifacts[0])
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)

        # XXX todo: note that there is still window in which an artifact with the same id but different
        # content can be generated, because it is a time-serials table, the create will success anyway
        logger.info('Creating artifact (id: %s) ...', payload['artifact_id'])
        self.axdb_client.create_artifact(payload)
        logger.info('Successfully created artifact (id: %s)', payload['artifact_id'])

        # Update in memory space accounting
        self._update_in_memory_space_accounting(payload=payload, is_addition=True)

    def delete_artifact(self, artifact_id, deleted_by=None):
        """Temporarily delete an artifact

        :param artifact_id:
        :param deleted_by:
        :returns: none or raise exception
        """
        artifact = self.get_artifact(artifact_id=artifact_id, resolve_alias=False)
        if artifact['deleted'] in {FLAG_TO_BE_DELETED, FLAG_DELETED}:
            logger.warning('Artifact (id: %s) already in DELETED state, skip')
        # Cannot delete tagged artifacts
        elif artifact['tags']:
            message = 'Cannot delete tagged artifacts'
            logger.error(message)
            raise AXApiInvalidParam(message, message)
        else:
            logger.info('Deleting artifact (id: %s) ...', artifact_id)
            payload = {
                'artifact_id': artifact['artifact_id'],
                'ax_uuid': artifact['ax_uuid'],
                'deleted': FLAG_TO_BE_DELETED,
                'deleted_date': int(time.time()),
                'timestamp': get_epoch_time_in_ms()
            }
            if deleted_by:
                payload['deleted_by'] = deleted_by
            artifact.update(payload)
            self.axdb_client.update_artifact(payload)
            logger.info('Successfully deleted artifact (id: %s)', artifact_id)

    def restore_artifact(self, artifact_id):
        """Remove the temporary deletion flag from the artifact

        :param artifact_id:
        :returns: none or raise exception
        """
        artifact = self.get_artifact(artifact_id=artifact_id, resolve_alias=False)
        logger.info('Restoring artifact (id: %s) ...', artifact_id)
        if artifact['deleted'] in {FLAG_NOT_DELETED, FLAG_ALIAS}:
            logger.warning('Artifact (id: %s) not in DELETED state, skip')
        elif artifact['deleted'] == FLAG_DELETED:
            message = 'Unable to restore artifact'
            detail = 'Artifact (id: {}) has been permanently deleted, unable to restore'.format(artifact_id)
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)
        else:
            logger.info('Restoring artifact (id: %s) ...', artifact_id)
            payload = {
                'artifact_id': artifact['artifact_id'],
                'ax_uuid': artifact['ax_uuid'],
                'deleted': FLAG_NOT_DELETED,
                'timestamp': get_epoch_time_in_ms()
            }
            artifact.update(payload)
            self.axdb_client.update_artifact(payload)
            logger.info('Successfully restored artifact (id: %s)', artifact_id)

    def delete_artifacts(self, workflow_ids, retention_tag=None, deleted_by=None):
        """Delete artifacts by retention tag

        :param workflow_ids:
        :param retention_tag:
        :param deleted_by:
        :returns:
        """
        params = {'deleted': [FLAG_NOT_DELETED]}
        if retention_tag:
            params['retention_tags'] = [retention_tag]
        artifacts = self.get_artifacts(**params, resolve_alias=False)
        artifacts = [artifact for artifact in artifacts if artifact['workflow_id'] in workflow_ids]

        for i in range(len(artifacts)):
            if artifacts[i]['tags']:
                continue

            logger.info('Deleting artifact (id: %s) ...', artifacts[i]['artifact_id'])
            payload = {
                'artifact_id': artifacts[i]['artifact_id'],
                'ax_uuid': artifacts[i]['ax_uuid'],
                'deleted': FLAG_TO_BE_DELETED,
                'deleted_date': int(time.time()),
                'timestamp': get_epoch_time_in_ms()
            }
            if deleted_by:
                payload['deleted_by'] = deleted_by
            self.axdb_client.update_artifact(payload)
            logger.info('Successfully deleted artifact (id: %s)', artifacts[i]['artifact_id'])

    def restore_artifacts(self, workflow_ids, retention_tag=None):
        """Restore artifacts by retention tag

        :param workflow_ids:
        :param retention_tag:
        :returns:
        """
        params = {'deleted': [FLAG_TO_BE_DELETED]}
        if retention_tag:
            params['retention_tags'] = [retention_tag]
        artifacts = self.get_artifacts(**params, resolve_alias=False)
        artifacts = [artifact for artifact in artifacts if artifact['workflow_id'] in workflow_ids]

        for i in range(len(artifacts)):
            logger.info('Restoring artifact (id: %s) ...', artifacts[i]['artifact_id'])
            payload = {
                'artifact_id': artifacts[i]['artifact_id'],
                'ax_uuid': artifacts[i]['ax_uuid'],
                'deleted': FLAG_NOT_DELETED,
                'timestamp': get_epoch_time_in_ms()
            }
            self.axdb_client.update_artifact(payload)
            logger.info('Successfully restored artifact (id: %s)', artifacts[i]['artifact_id'])


    def get_service(self, workflow_id):
        """Get a workflow by workflow ID

        :param workflow_id:
        :returns:
        """
        # Find the service instance
        service = self.axdb_client.get_service(workflow_id)
        if not service:
            message = 'Service instance does not exist'
            detail = 'Service instance ({}) does not exist'.format(workflow_id)
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)
        else:
            return service

    @retry(wait_fixed=RETRY_INTERVAL, stop_max_attempt_number=RETRY_MAX_NUMBER, retry_on_exception=conditional_update_retry_on_exception)
    def _update_global_artifact_tags(self, tag_list):
        """Add tag to tag list in artifact meta table

        :param tag_list:
        :returns:
        """
        logger.info('Updating global metadata (attribute: artifact_tags) ...')
        metadata = self.axdb_client.get_artifact_meta('artifact_tags')
        prev_artifact_tags_str = metadata['value']
        artifact_tags = json.loads(prev_artifact_tags_str)
        new_tags = []
        for tag in tag_list:
            if tag not in artifact_tags:
                new_tags.append(tag)
        if not new_tags:
            return
        artifact_tags.extend(new_tags)
        artifact_tags = sorted(artifact_tags)
        next_artifact_tags_str = json.dumps(artifact_tags)
        self.axdb_client.update_artifact_meta_conditionally(
            'artifact_tags', next_artifact_tags_str, prev_artifact_tags_str)
        logger.info('Successfully updated global metadata (attribute: artifact_tags)')

    @staticmethod
    def _add_tag(prev_tags, tag_list):
        """Prepare payload for conditional addition of tags

        :param prev_tags:
        :param tag_list:
        :returns:
        """
        params = {}
        logger.info('Adding tag (%s) ...', tag_list)
        tags = json.loads(prev_tags or '[]')
        for tag in tag_list:
            if tag not in tags:
                tags.append(tag)
        tags = sorted(tags)
        next_tags = json.dumps(tags)
        if next_tags == prev_tags:
            return None
        params['tags'] = next_tags
        params['tags_update_if'] = prev_tags
        return params

    @retry(wait_fixed=RETRY_INTERVAL, stop_max_attempt_number=RETRY_MAX_NUMBER, retry_on_exception=conditional_update_retry_on_exception)
    def _tag_workflow(self, workflow_id, tag_list):
        """Add tag to a workflow

        :param workflow_id:
        :param tag_list:
        :returns:
        """
        logger.info('Tagging workflow (id: %s, tag: %s) ...', workflow_id, str(tag_list))
        service = self.get_service(workflow_id)
        prev_tags = service['tags']
        payload = self._add_tag(prev_tags=prev_tags, tag_list=tag_list)
        if payload is None:
            return
        payload['service_id'] = workflow_id
        payload['template_name'] = service['template_name']
        logger.info('Adding tag to workflow (id: %s) ...', workflow_id)
        self.axdb_client.update_service_conditionally(**payload)
        logger.info('Successfully tagged workflow (id: %s, tag: %s)', workflow_id, str(tag_list))

    @retry(wait_fixed=RETRY_INTERVAL, stop_max_attempt_number=RETRY_MAX_NUMBER, retry_on_exception=conditional_update_retry_on_exception)
    def _tag_artifact(self, artifact_id, tag_list):
        """Add a tag to an artifact

        :param artifact_id:
        :param tag_list:
        :returns:
        """
        logger.info('Tagging artifact (id: %s, tag: %s) ...', artifact_id, str(tag_list))
        artifact = self.get_artifact(artifact_id=artifact_id, resolve_alias=False, deserialize=False)
        prev_tags = artifact['tags']
        payload = self._add_tag(prev_tags=prev_tags, tag_list=tag_list)
        if payload is None:
            return
        payload['artifact_id'] = artifact['artifact_id']
        payload['ax_uuid'] = artifact['ax_uuid']
        payload['timestamp'] = get_epoch_time_in_ms()
        logger.info('Adding tag to artifact (id: %s) ...', artifact_id)
        self.axdb_client.update_artifact_conditionally(**payload)
        logger.info('Successfully tagged artifact (id: %s, tag: %s)', artifact_id, str(tag_list))

    def tag_workflow(self, workflow_id, tag_list):
        """Add tag to workflow and all artifacts associated with the workflow

        :param workflow_id:
        :param tag_list: list of tags
        :returns:
        """
        if not tag_list or not isinstance(tag_list, list):
            message = 'Missing requirement parameter'
            detail = 'Missing requirement parameter (tag)'
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)

        # Update artifact table first to avoid incidental delete for pinned-down
        artifacts = self.get_artifacts(workflow_id=workflow_id, resolve_alias=False)
        for i in range(len(artifacts)):
            self._tag_artifact(artifacts[i]['artifact_id'], tag_list)

        # Update service table second to show properly from the UI
        self._tag_workflow(workflow_id, tag_list)

        # Update the global tag list the last
        self._update_global_artifact_tags(tag_list)

    @staticmethod
    def _remove_tag(prev_tags, tag_list):
        """Prepare payload for conditional removal of tags

        :param prev_tags:
        :param tag_list:
        :returns:
        """
        params = {}
        logger.info('Removing tag (%s) ...', str(tag_list))
        tags = json.loads(prev_tags or '[]')
        for tag in tag_list:
            if tag in tags:
                tags.remove(tag)
        tags = sorted(tags)
        next_tags = json.dumps(tags)
        if next_tags == prev_tags:
            return None
        params['tags'] = next_tags
        params['tags_update_if'] = prev_tags
        return params

    @retry(wait_fixed=RETRY_INTERVAL, stop_max_attempt_number=RETRY_MAX_NUMBER, retry_on_exception=conditional_update_retry_on_exception)
    def _untag_workflow(self, workflow_id, tag_list):
        """Remove tag from a workflow

        :param workflow_id:
        :param tag_list:
        :returns:
        """
        logger.info('Untagging workflow (id: %s, tag: %s) ...', workflow_id, str(tag_list))
        service = self.get_service(workflow_id)
        prev_tags = service['tags']
        payload = self._remove_tag(prev_tags=prev_tags, tag_list=tag_list)
        if payload is None:
            return
        payload['service_id'] = workflow_id
        payload['template_name'] = service['template_name']
        logger.info('Removing tag from workflow (id: %s) ...', workflow_id)
        self.axdb_client.update_service_conditionally(**payload)
        logger.info('Successfully untagged workflow (id: %s, tag: %s)', workflow_id, str(tag_list))

    @retry(wait_fixed=RETRY_INTERVAL, stop_max_attempt_number=RETRY_MAX_NUMBER, retry_on_exception=conditional_update_retry_on_exception)
    def _untag_artifact(self, artifact_id, tag_list):
        """Remove a tag from an artifact

        :param artifact_id:
        :param tag_list:
        :returns:
        """
        logger.info('Untagging artifact (id: %s, tag: %s) ...', artifact_id, str(tag_list))
        artifact = self.get_artifact(artifact_id=artifact_id, resolve_alias=False, deserialize=False)
        prev_tags = artifact['tags']
        payload = self._remove_tag(prev_tags=prev_tags, tag_list=tag_list)
        if payload is None:
            return
        payload['artifact_id'] = artifact['artifact_id']
        payload['ax_uuid'] = artifact['ax_uuid']
        payload['timestamp'] = get_epoch_time_in_ms()
        logger.info('Removing tag from artifact (id: %s) ...', artifact_id)
        self.axdb_client.update_artifact_conditionally(**payload)
        logger.info('Successfully untagged artifact (id: %s, tag: %s)', artifact_id, str(tag_list))

    def untag_workflow(self, workflow_id, tag_list):
        """Remove a tag from all artifacts associated with a workflow

        Cannot untag a workflow with a retention tag.

        :param workflow_id:
        :param tag_list:
        :returns:
        """
        if not tag_list or not isinstance(tag_list, list):
            message = 'Missing requirement parameter'
            detail = 'Missing requirement parameter (tag)'
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)

        # Update artifact table first to avoid incidental delete for pinned-down
        artifacts = self.get_artifacts(workflow_id=workflow_id, resolve_alias=False)
        for i in range(len(artifacts)):
            self._untag_artifact(artifacts[i]['artifact_id'], tag_list)

        # Update service table second to show properly from the UI
        self._untag_workflow(workflow_id, tag_list)

    def browse_artifact(self, artifact_id):
        """Browse the content of an artifact

        :param artifact_id:
        :returns:
        """
        artifact = self.get_artifact(artifact_id=artifact_id, resolve_alias=True, deserialize=True)

        if not artifact['structure_path']:
            message = 'Artifact cannot be browsed'
            detail = 'Artifact (id: {}) has no internal structure and hence, cannot be browsed'.format(artifact_id)
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)

        try:
            structure_path = artifact['structure_path']
            bucket, key = structure_path['bucket'], structure_path['key']
            s3_object = Cloud().get_bucket(bucket).get_object(key)
            structure = s3_object['Body'].read().decode(errors='replace')
            return json.loads(structure)
        except Exception as e:
            detail = 'Failed to download artifact structure (id: {}): {}'.format(artifact_id, str(e))
            logger.exception(detail)
            raise AXApiInternalError('Internal server error', detail)

    def download_artifact(self, artifact):
        """Download an artifact

        :param artifact:
        :returns: generator
        """
        # When an artifact is flagged to be deleted, the artifact theoretically can be deleted
        # at any time. It could be deleted during the download of the artifact. To avoid handling
        # such an exception, we simply block the download of deleted (temporarily / permanently)
        # artifacts. From business side, this assumption also makes sense, as why does a user
        # need to download a deleted artifact? If he/she needs to download a deleted artifact,
        # why not first restore the artifact?
        if artifact['deleted'] in {FLAG_DELETED, FLAG_TO_BE_DELETED}:
            message = 'Cannot download deleted artifact'
            detail = 'Artifact ({}) is deleted' if artifact['deleted'] == FLAG_DELETED else 'Artifact ({}) is flagged to delete'
            raise AXApiInvalidParam(message, detail)

        storage_method = artifact.get('storage_method', None)
        if storage_method == 's3':
            try:
                s3_path = artifact['storage_path']
                bucket, key = s3_path['bucket'], s3_path['key']
                s3_bucket = Cloud().get_bucket(bucket)
                location = s3_bucket.generate_signed_url(key) if AXS3Bucket.supports_signed_url() else "/v1/s3object" \
                                                                                                      "?bucket=" + \
                                                                                                      bucket + "&key=" \
                                                                                                      + key
                logger.info("redirect to %s", location)
                return location, None
            except Exception as e:
                detail = 'Failed to generate download URL for artifact (id: {}): {}'.format(artifact['artifact_id'], str(e))
                logger.exception(detail)
                raise AXApiInternalError('Internal server error', detail)
        elif storage_method == 'inline':
            logger.info("Use inline storage")
            return None, artifact.get('inline_storage', '')
        else:
            raise AXApiInvalidParam('Invalid storage_method {}'.format(storage_method))

    def download_artifact_by_query(self, **kwargs):
        """Download an artifact by query

        :param kwargs:
        :returns: generator
        """
        artifacts = self.get_artifacts(**kwargs, resolve_alias=True)
        if len(artifacts) == 0:
            message = 'Cannot find artifact'
            detail = 'Cannot find artifact ({})'.format(kwargs)
            logger.error(detail)
            raise AXApiInvalidParam(message, detail)

        # find the latest one
        idx = 0
        timestamp = 0
        for i, artifact in enumerate(artifacts):
            try:
                if artifact['timestamp'] > timestamp:
                    idx = i
                    timestamp = artifact['timestamp']
            except Exception:
                logger.exception("i=%s artifact: %s", i, artifact)

        if len(artifacts) > 1:
            logger.warning('Unexpectedly found multiple artifacts, only the latest one (idx=%s id=%s) will be downloaded',
                           idx, artifacts[idx]['artifact_id'])
        logger.debug("found artifact %s", artifacts[idx])
        return self.download_artifact(artifacts[idx])

    def get_artifact_nums(self):
        """Get artifact nums"""
        artifact_nums = self.axdb_client.get_artifact_meta('artifact_nums')['value']
        return json.loads(artifact_nums)

    def get_artifact_size(self):
        """Get artifact size"""
        artifact_size = self.axdb_client.get_artifact_meta('artifact_size')['value']
        return json.loads(artifact_size)

    def get_tags(self, params):
        """Get all tags"""
        logger.info('debugging %s', json.dumps(params))
        tags = json.loads(self.axdb_client.get_artifact_meta('artifact_tags')['value'])
        if not isinstance(tags, list):
            return tags

        logger.info(tags)
        if 'search' in params:
            temp_tags = list()
            for tag in tags:
                if params['search'] in tag:
                    temp_tags.append(tag)
            tags = temp_tags

        if 'limit' in params:
            try:
                tags = tags[0: int(params['limit'])]
            except Exception:
                logger.info('Failed to use limit parameter')

        return tags

    @staticmethod
    def convert_ms_to_str(ms_time):
        """Convert milliseconds to human readable strings"""
        unit = [MILLISECONDS_PER_MONTH, MILLISECONDS_PER_WEEK, MILLISECONDS_PER_DAY,
                MILLISECONDS_PER_HOUR, MILLISECONDS_PER_MIN, MILLISECONDS_PER_SECOND]
        unit_str = ['month', 'week', 'day', ' hour', 'min', 'sec']

        res = []
        for i in range(len(unit)):
            temp_str = int(ms_time / unit[i])
            ms_time = int(ms_time % unit[i])
            if temp_str:
                res.append('{} {}'.format(temp_str, unit_str[i]))
        return ' '.join(res)
