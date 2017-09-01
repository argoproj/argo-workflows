#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import copy
import random
import json
import logging
import os
import re
import requests
import subprocess
import time
from requests.exceptions import HTTPError

from ax.devops.axrequests.axrequests import AxRequests
from ax.devops.settings import AxSettings
from ax.exceptions import AXApiInvalidParam, AXApiInternalError

logger = logging.getLogger(__name__)

CREATE_SERVICE = 'create'
DELETE_SERVICE = 'delete'
SHOW_SERVICE = 'show'
CLUSTER_CONFIG = 'cluster_config'

INGRESS_METHOD = '/application/axsys/deployment/axops-deployment/external_route'


class AxsysClient(object):
    """AXSYS client."""

    CONTAINER_PENDING = "CONTAINER_PENDING"
    CONTAINER_RUNNING = "CONTAINER_RUNNING"
    CONTAINER_IMAGE_PULL_BACKOFF = "CONTAINER_IMAGE_PULL_BACKOFF"
    CONTAINER_STOPPED = "CONTAINER_STOPPED"
    CONTAINER_FAILED = "CONTAINER_FAILED"
    CONTAINER_NOTFOUND = "CONTAINER_NOTFOUND"
    CONTAINER_UNKNOWN = "CONTAINER_UNKNOWN"

    def __init__(self, cluster=None):
        """Initialize AXSYS client.

        :param cluster:
        :return:
        """
        if cluster is None:  # Inside the cluster
            self.cluster = self.get_cluster()
            self.internal = True
        else:  # Outside the cluster
            env_cluster = self.get_cluster()
            self.cluster = cluster
            if env_cluster == self.cluster:
                logger.info("env_cluster is also %s, force internal to True", env_cluster)
                self.internal = True
            else:
                logger.info("env_cluster is %s, cluster is %s.", env_cluster, cluster)
                self.internal = False
        self._axmon_client = AxRequests(host=AxSettings.AXMON_HOSTNAME, port=AxSettings.AXMON_PORT, timeout=5 * 60)
        self.axmon_client = AxRequests(url=self.get_axmon_url(), timeout=900)
        logger.info("axmon_client %s", self.axmon_client)
        self.axconsole_client = None
        self.axnotification_client = AxRequests(host=AxSettings.AXNOTIFICATION_HOSTNAME,
                                                port=AxSettings.AXNOTIFICATION_PORT,
                                                version='v1', protocol='http', timeout=600)

    @staticmethod
    def get_axmon_url():
        """Get axmon url.

        :return:
        """
        return "http://axmon.axsys:8901/v1/axmon"

    @staticmethod
    def get_axmon_hostname():
        """Get axmon hostname."""
        return "axmon.axsys"

    ####################
    # Inside container #
    ####################

    @staticmethod
    def get_cluster():
        """Get cluster name.

        :return:
        """
        return os.environ.get('AX_CLUSTER', None)

    @staticmethod
    def get_container_name():
        """Get container name.

        :return:
        """
        return os.environ.get('AX_CONTAINER_NAME', None)

    @staticmethod
    def get_container_uuid():
        """Get container UUID.

        :return:
        """
        try:
            cmd = 'grep ":cpuset:" /proc/self/cgroup'
            output = subprocess.check_output(cmd, shell=True).decode("utf-8").strip()
            # Example output: b'3:cpuset:/docker/a85a1d6191847a1a578ced9ce70347105f01cc54a3caa455dc4b7947cc2afa60\n'
            # Or '9:cpuset:/3b6119badace8c2249a585e8bb01ed7b3ef502e5b9af9243f481fb512b7a427f'
            return output.split('/')[-1]
        except Exception as exc:
            logger.exception(exc)
            return None

    ####################################
    # Based on AXMON Internal/External #
    ####################################

    @staticmethod
    def _retry_for_autoscaling(resp):
        """Check the scenarios require retrying
        :param resp:
        :return:
        """
        # {'code' : self.code,
        #  'message' : self.args[0],
        #  'detail' : self.detail if self.detail else ""
        #  }
        try:
            if resp.status_code >= 400 and resp.json().get('code', None) == 'ERR_INSUFFICIENT_RESOURCE':
                logger.info('Trigger auto scale ...')
                return True
            else:
                logger.error("status=%s code=%s json=%s", resp.status_code, resp.json().get('code', None), resp.json())
                return False
        except Exception:
            logger.exception("")
            return False

    def _run_service(self, action, metadata, retry_on_500=False):
        """Run service

        :param action:
        :param metadata:
        :return:
        """

        def _run():
            """Run a single axmon API

            :return:
            """
            default_headers = {'Content-Type': 'application/json', 'Accept': 'application/json'}
            resp = None
            logger.debug("AXSYSClient for {} with metadata {}".format(action, metadata))
            count = -1
            stop_max_delay_ms = 30 * 60 * 1000 # 0.5 hour

            def exponential_sleep(cnt, wait_exponential_multiplier_ms=1000, wait_exponential_max_ms=20000):
                sleep_time_ms = (2 ** cnt) * wait_exponential_multiplier_ms + random.randint(0, 1000)
                sleep_time_ms = min(sleep_time_ms, wait_exponential_max_ms)
                time.sleep(sleep_time_ms / 1000.0)

            begin = time.time()
            timeout_second = stop_max_delay_ms / 1000.0
            end = begin + timeout_second
            while True:
                count += 1
                try:
                    if action == CREATE_SERVICE:
                        path = '/task'
                        resp = self.axmon_client.post(path, data=json.dumps(metadata), headers=default_headers, raise_exception=False)
                    elif action == SHOW_SERVICE:
                        path = '/task/{}'.format(metadata["id"])
                        resp = self.axmon_client.get(path, headers=default_headers, raise_exception=False)
                    elif action == DELETE_SERVICE:
                        path = '/task/{}?{}={}&{}={}&{}={}'.format(metadata["id"],
                                                                   "force", metadata["force"],
                                                                   "delete_pod", metadata["delete_pod"],
                                                                   "stop_running_pod_only", metadata["stop_running_pod_only"])
                        resp = self.axmon_client.delete(path, headers=default_headers, raise_exception=False)
                    elif action == CLUSTER_CONFIG:
                        path = '/cluster/nodes'
                        # Raise exception if failed
                        resp = self.axmon_client.get(path, headers=default_headers, raise_exception=True)
                    else:
                        assert False, "Action {} not implemented".format(action)
                    if retry_on_500 and (500 <= resp.status_code < 600):
                        logger.error("Got %s response %s", resp.status_code, resp)
                        if time.time() <= end:
                            logger.error("Retry request")
                            exponential_sleep(count)
                            continue
                        else:
                            logger.error("Timeout after %s seconds. return %s response", timeout_second, resp.status_code)
                    return resp
                except (requests.ConnectionError, requests.Timeout) as e:
                    logger.error("Got exception %s, count=%s", e, count)
                    if time.time() > end:
                        logger.error("timeout after %s seconds", timeout_second)
                        raise
                    exponential_sleep(count)

        if action not in {CREATE_SERVICE, DELETE_SERVICE, SHOW_SERVICE, CLUSTER_CONFIG}:
            raise NotImplementedError('Action (%s) not supported', action)

        response = _run()

        return response

    def create_service(self, service_template, dry_run=False):
        """Create service.

        :param service_template:
        :param dry_run:
        :return:
        """
        if dry_run:
            service_template = copy.copy(service_template)
            service_template["dry_run"] = True
        metadata = {'service': service_template, 'action': 'service/{}'.format(CREATE_SERVICE)}
        response = self._run_service(CREATE_SERVICE, metadata, retry_on_500=True)
        try:
            response_json = response.json()
        except Exception:
            logger.exception("bad response", response)
            response_json = {}
        if 200 <= response.status_code < 300:
            # Return a list of containers
            containers = list(response.json().get('result', {}).keys())
            return True, containers, response.status_code, response_json
        else:
            logger.error('FAILURE: code=%s action=%s response=%s meta=%s',
                         response.status_code, 'create',
                         response_json,
                         metadata)
            return False, None, response.status_code, response_json

    def delete_service(self, service_name, delete_pod=True, stop_running_pod_only=False, force=False):
        """Delete service.

        :param service_name:
        :param delete_pod:
        :param stop_running_pod_only:
        :param force:
        :return:
        """
        if stop_running_pod_only:
            assert delete_pod is False
            assert force is False
        metadata = {
            'id': service_name,
            'action': 'service/{}'.format(DELETE_SERVICE),
            'delete_pod': 'True' if delete_pod else 'False',
            'stop_running_pod_only': 'True' if stop_running_pod_only else 'False',
            'force': 'True' if force else 'False'}
        response = self._run_service(DELETE_SERVICE, metadata, retry_on_500=True)
        if 200 <= response.status_code < 300:
            return True, None
        elif response.status_code == 404:
            logger.debug('NOTFOUND: code=%s action=%s meta=%s',
                         response.status_code, 'delete', metadata)
            return False, None
        else:
            logger.error('FAILURE: code=%s action=%s response=%s meta=%s',
                         response.status_code, 'delete',
                         response.json(),
                         metadata)
            return False, None

    def show_service(self, container_name):
        """Show service.

        :param container_name:
        :return:
        """
        # Example for show_service return
        # { "status_code" : <int>,
        #   "error": '',  # String if error, else EMPTY
        #   "result": {'/axdb':
        #                 { "uuid" :  [a730d1c8400cdb7aa119a2aa328bcf390005fa5307df5154a312319952701815],
        #                  ...
        #                  },
        #              '/axpython':
        #                 { "uuid" :  [...328bcf390005fa5307df5154a312319952701815],
        #                  ...
        #                  },
        #              }
        # }
        metadata = {'id': container_name, 'action': 'service/{}'.format(SHOW_SERVICE)}
        response = self._run_service(SHOW_SERVICE, metadata, retry_on_500=True)

        if 200 <= response.status_code < 300:
            return True, response.json()
        elif response.status_code == 404:
            logger.debug('NOTFOUND: code=%s action=%s meta=%s',
                         response.status_code, 'show', metadata)
            return True, None
        else:
            logger.error('FAILURE: code=%s action=%s response=%s meta=%s',
                         response.status_code, 'create',
                         response.json(),
                         metadata)
            return False, None

    def restart_service(self, service_name):
        """Restart service.

        :param service_name:
        :return:
        """
        raise NotImplementedError('Service restart not implemented yet')

    def get_cluster_config(self):
        response = self._run_service(action=CLUSTER_CONFIG, metadata=None, retry_on_500=True)
        return response.json()

    def get_container_status(self, container_name):
        """Check whether a container is running or not

        :param container_name:
        :return:
        """
        container_name = self.canonical_container_full_name(container_name)
        is_success, result = self.show_service(container_name)
        if is_success:
            if result is None:
                return AxsysClient.CONTAINER_NOTFOUND
            else:
                r = result.get('result', {})

            try:
                failed = r.get('failed', False)
                if failed:
                    logger.debug("failed result=%s", r)
                    return AxsysClient.CONTAINER_FAILED
                if r.get('succeeded', False):
                    return AxsysClient.CONTAINER_STOPPED
                elif r.get('active', False):
                    reason = r.get('reason', None)
                    if reason == 'ImagePullBackOff':
                        return AxsysClient.CONTAINER_IMAGE_PULL_BACKOFF
                    else:
                        if reason:
                            logger.warning("Other wait reason: %s", reason)
                        return AxsysClient.CONTAINER_RUNNING
                else:
                    return AxsysClient.CONTAINER_PENDING
            except Exception:
                logger.exception("bad result %s", result)
                return AxsysClient.CONTAINER_UNKNOWN
        else:
            return AxsysClient.CONTAINER_UNKNOWN

    @staticmethod
    def guess_container_full_name(service_template, expand_override=None):
        try:
            container_name = service_template["template"]["name"]
            container_name = re.sub(r"[^0-9a-z-]+", "-", container_name.lower())
            instance_id = service_template.get("id", None)
            if expand_override is None:
                expand = service_template["template"]["container"].get("expand", True)
            else:
                expand = expand_override
            # this must match the function in platform/tasks.py:_get_container_name()
            if expand and instance_id:
                kube_max_job_name = 55  # xxx todo, change to 63 after fix the monitor issue
                max_len = kube_max_job_name - len(instance_id) - 1
                assert max_len > 0
                ret = "{}-{}".format(container_name[:max_len], instance_id)
            else:
                ret = container_name
            return AxsysClient.canonical_container_full_name(ret)
        except Exception:
            logger.exception("bad %s", service_template)
            return None

    @staticmethod
    def canonical_container_full_name(xid):
        return "{}".format(xid.lstrip("/"))

    #############
    # Axconsole #
    #############

    def create_port_mapping(self, hostname, port):
        """Request axconsole to do a port mapping.

        :param hostname:
        :param port:
        :return:
        """
        data = {'address': '{}:{}'.format(hostname, port)}
        response = self.axconsole_client.post('/api/portforward', data=json.dumps(data), value_only=True)
        return response.get('port', None)

    def delete_port_mapping(self, hostname, port):
        """Request axconsole to delete port mapping.

        :param hostname:
        :param port:
        :return:
        """
        path = '/api/portforward/{}:{}'.format(hostname, port)
        response = self.axconsole_client.delete(path, value_only=True)
        return response.get('result', False)

    ###########
    # General #
    ###########

    def send_notification(self, message):
        default_headers = {'Content-Type': 'application/json',
                           'Accept': 'application/json'}
        path = '/notifications/email'
        resp = self.axnotification_client.post(path, data=json.dumps(message),
                                               headers=default_headers)
        return resp

    def update_security_group(self, ip, port, target, add_or_remove=True):
        """Add/remove security group

        :param ip: in the form of '0.0.0.0/0'.
        :param port:
        :param target:
        :param add_or_remove: true for add, false for remove
        :returns:
        """
        try:
            payload = {
                'target': target,
                'action': 'enable' if add_or_remove else 'disable',
                'port': port,
                'ip_ranges': [ip]
            }
            return self._axmon_client.put('/v1/security_groups', data=json.dumps(payload), value_only=True)
        except HTTPError as e:
            logger.error('Failed to add security group (%s)', str(e))
            status_code = e.response.status_code
            if 400 <= status_code < 500 and status_code != 404:
                error = AXApiInvalidParam
            else:
                error = AXApiInternalError
            error_payload = e.response.json()
            raise error(message=error_payload.get('message', ''), detail=error_payload.get('detail', ''))
        except Exception as e:
            message = 'Failed to add security group'
            detail = 'Failed to add security group ({})'.format(str(e))
            logger.error(detail)
            raise AXApiInternalError(message=message, detail=detail)

    def get_webhook(self):
        """Get webhook"""
        try:
            return self._axmon_client.get('/v1/webhook', value_only=True)
        except HTTPError as e:
            logger.error('Failed to delete webhook: %s', str(e))
            status_code = e.response.status_code
            if 400 <= status_code < 500 and status_code != 404:
                error = AXApiInvalidParam
            else:
                error = AXApiInternalError
            error_payload = e.response.json()
            raise error(message=error_payload.get('message', ''), detail=error_payload.get('detail', ''))
        except Exception as e:
            message = 'Failed to delete webhook'
            detail = 'Failed to delete webhook ({})'.format(str(e))
            logger.error(detail)
            raise AXApiInternalError(message=message, detail=detail)

    def create_webhook(self, ip_range, external_port, internal_port):
        """Create webhook

        :param ip_range:
        :param external_port:
        :param internal_port:
        :returns:
        """
        try:
            payload = {
                'port_spec': [
                    {
                        'name': 'webhook',
                        'port': external_port,
                        'targetPort': internal_port
                    }
                ],
                'ip_ranges': ip_range
            }
            return self._axmon_client.put('/v1/webhook', data=json.dumps(payload), value_only=True)
        except HTTPError as e:
            logger.error('Failed to create webhook: %s', str(e))
            status_code = e.response.status_code
            if 400 <= status_code < 500 and status_code != 404:
                error = AXApiInvalidParam
            else:
                error = AXApiInternalError
            error_payload = e.response.json()
            raise error(message=error_payload.get('message', ''), detail=error_payload.get('detail', ''))
        except Exception as e:
            message = 'Failed to create webhook'
            detail = 'Failed to create webhook ({})'.format(str(e))
            logger.error(detail)
            raise AXApiInternalError(message=message, detail=detail)

    def delete_webhook(self):
        """Delete webhook"""
        try:
            self._axmon_client.delete('/v1/webhook')
        except HTTPError as e:
            logger.error('Failed to delete webhook: %s', str(e))
            status_code = e.response.status_code
            if 400 <= status_code < 500 and status_code != 404:
                error = AXApiInvalidParam
            else:
                error = AXApiInternalError
            error_payload = e.response.json()
            raise error(message=error_payload.get('message', ''), detail=error_payload.get('detail', ''))
        except Exception as e:
            message = 'Failed to delete webhook'
            detail = 'Failed to delete webhook ({})'.format(str(e))
            logger.error(detail)
            raise AXApiInternalError(message=message, detail=detail)

    def create_volume(self, volume):
        """Create a volume. Returns the resource id of the created volume"""
        result = self.axmon_client.post("/volume/{}".format(volume['id']), data=json.dumps(volume), value_only=True)
        return result['result']

    def get_volume(self, volume_id):
        """Get a volume"""
        result = self.axmon_client.get("/volume/{}".format(volume_id), value_only=True)
        return result['result']

    def update_volume(self, volume):
        """Update a volume"""
        return self.axmon_client.put("/volume/{}".format(volume['id']), data=json.dumps(volume), value_only=True)

    def delete_volume(self, volume_id):
        """Delete a volume"""
        return self.axmon_client.delete("/volume/{}".format(volume_id), value_only=True)

    def create_ingress(self, dnsname, port=8088, whitelist=None, visibility='world'):
        """
        :param dnsname:
        :param port:
        :param whitelist:
        :param visibility:
        :return:
        """
        if whitelist is None:
            whitelist = ['0.0.0.0/0']

        payload = {'dns_name':     dnsname,
                   'target_port':  port,
                   'whitelist':    whitelist,
                   'visibility':   visibility
                   }
        logger.info('Create ingress policy for Jira webhook: %s', payload)
        return self.axmon_client.post(INGRESS_METHOD, data=json.dumps(payload), value_only=True)

    def delete_ingress(self, dnsname):
        """
        :param dnsname:
        :return:
        """
        logger.info('Delete ingress policy for endpoint: %s', dnsname)
        return self.axmon_client.delete(INGRESS_METHOD + '/' + dnsname, value_only=True)
