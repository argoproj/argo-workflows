#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

'''
Module to update statistics for host and container
'''

import time
import logging
import calendar
import datetime
import random
import re
from threading import Thread

from ax.util.const import KiB, MiB, NS_PER_SECOND
from ax.util.axsleep import AXSleep
from ax.cadvisor import AXCadvisorClient
from ax.platform.service_config import AXKubeServiceConfig
from ax.platform.exceptions import AXPlatformException
from ax.kubernetes.client import KubernetesApiClient
from ax.axevent import AXEventClient

logger = logging.getLogger(__name__)

HOST_STATUS_OK = 0
HOST_STATUS_WARNING = -1
HOST_STATUS_CRITICAL = -2

COLLECT_STOP = "COLLECT_STOP"

CHECK_LIVELINESS_INTERVAL = 5
NODE_REFRESH_INTERVAL = 120
STATS_COLLECT_INTERVAL = 60
STATS_ENDPOINT_REGEX = re.compile(r'^/kubepods/.*pod[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}/([a-fA-F0-9]{64})')


def post_start_container_event(uuid, name, iid, ip, cost_id, service_id, cpu, mem, url_run, url_done, endpoint=None, max_retry=1):
    count = 0
    while True:
        count += 1
        try:
            config = {
                "id": uuid,
                "name": name,
                "host_id": iid,
                "host_name": ip,
                "cost_id": cost_id,
                "service_id": service_id,
                "cpu": cpu,
                "mem": mem,
                "url_run": url_run,
                "url_done": url_done
            }

            if endpoint:
                config["endpoint"] = endpoint

            AXEventClient.post_from_axmon(topic='containers', key=uuid, op= "update", data=config)
            return
        except Exception as e:
            logger.warn("Exception in stats reporting %s %s/%s", e, count, max_retry)
            if count < max_retry:
                time.sleep(5)
                continue
            else:
                raise e


def container_oom_cb(iid, ip, name, service_id, event):
    """
    Call back for handling container OOM event.

    Currently we print out log for debug purpose, but this event needs to be further
    handled by upper layer.
    """
    if event['stage'] == 'oom':
        logger.debug("Container %s from %s (%s) has been OOM killed. Service ID: %s. Event detail: %s", name, iid, ip, service_id, event)
    return


class AXStats(object):
    def __init__(self):
        self._collectors = {}
        self._hosts = set()

    def watch(self):
        '''
        Watch stats from local system and update AXDB.
        '''
        while True:
            with AXSleep(NODE_REFRESH_INTERVAL):
                try:
                    old_hosts = self._hosts
                    new_hosts = set()
                    nodes = KubernetesApiClient(use_proxy=True).api.list_node().items
                    for n in nodes:
                        try:
                            host_detail, instance_id = self.get_node_detail(n)
                            new_hosts.add(instance_id)
                            if instance_id not in old_hosts:
                                self._collectors[instance_id] = AXStatsCollector(host_detail["private_ip"][0], host_detail)
                                self._collectors[instance_id].start()
                        except Exception as e:
                            logger.exception("Cannot collect stats for node %s, error: %s", str(n.to_dict()), str(e))
                    self._hosts = new_hosts
                    # Instance IDs (iid) in old_hosts but not in new_hosts are those ready to delete
                    stale_hosts = old_hosts - new_hosts
                    logger.info("Main Loop: Old hosts: %s, New hosts: %s, Stale hosts: %s", old_hosts, new_hosts, stale_hosts)
                    for iid in stale_hosts:
                        self._collectors[iid].stop()
                        del self._collectors[iid]
                except Exception as e:
                    logger.exception("Main Loop: Died due to exception {}".format(e))

    @staticmethod
    def get_node_detail(kube_node_info):
        metadata = kube_node_info.metadata
        spec = kube_node_info.spec
        status = kube_node_info.status
        private_ip = ""
        for ip in status.addresses:
            if ip.type == "InternalIP":
                private_ip = ip.address
                break
        if not private_ip:
            raise AXPlatformException("AXStats: Cannot find node private IP")

        mem_in_Ki = float(status.capacity["memory"][:len(status.capacity["memory"])-2])
        return {
            "id": spec.external_id,
            "model": metadata.labels["beta.kubernetes.io/instance-type"],
            "name": metadata.name,
            "private_ip": [private_ip],
            "cpu": float(status.capacity["cpu"]),
            "mem": mem_in_Ki * KiB / MiB,
            "status": HOST_STATUS_OK
        }, spec.external_id


class AXStatsCollector(Thread):
    def __init__(self, ip, host_info):
        super(AXStatsCollector, self).__init__()
        self.daemon = True
        self._cli = AXCadvisorClient(ip=ip)
        self._stop = False
        self._service_config = AXKubeServiceConfig()

        self._containers = {}
        self._event_start = ""

        self._ip = ip
        self._host_record = host_info
        self._host_info = {
            "id": host_info["id"],
            "model": host_info["model"],
            "name": host_info["name"],
            "private_ip": host_info["private_ip"],
            "cpu": host_info["cpu"],
            "mem": host_info["mem"],
            "status": HOST_STATUS_OK
        }

    def stop(self):
        self._stop = True

    def run(self):
        logger.info("Start collecting stats for node %s. Node info: %s", self._ip, self._host_info)
        self.name = "collector-{}".format(self._ip)
        try:
            start_up_jitter = random.randint(1, 10)
            logging.info("Adding jitter %s seconds during startup. Sleeping ...", start_up_jitter)
            time.sleep(start_up_jitter)
            with AXSleep(STATS_COLLECT_INTERVAL):
                self.first_run()
            while True:
                if self._stop:
                    logger.info("Stop collecting stats for node %s.", self._ip)
                    return
                AXEventClient.post(topic='hosts', key=self._host_info['id'], op= "update", data=self._host_info)
                deleted_docker = []
                created_docker = []
                with AXSleep(STATS_COLLECT_INTERVAL):
                    try:
                        self.store_spec_info()
                        event_info = self._cli.get_events(self._event_start)
                        self._event_start = datetime.datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ')
                        self.process_cadvisor_events(event_info, deleted_docker, created_docker)

                        deleted_docker = list(set(deleted_docker) - set(created_docker))
                        if created_docker:
                            # Should only update created containers but safe to update all
                            self.update_container_info()

                        docker_stats = self._cli.get_docker_stats()
                        self.process_cadvisor_stats(docker_stats)
                        for d in deleted_docker:
                            if d in self._containers:
                                # We do not post to axevent because saas is not
                                # handling container usage delete event. Ask Ying
                                del self._containers[d]
                                self._service_config.unregister_container(d)
                        logger.debug("IP: %s. Loop finish\n\n\n", self._ip)
                    except Exception as e:
                        logger.exception("Stats collector on node %s has exception %s", self._ip, str(e))
        except Exception as e:
            logger.exception("AXStatsCollector thread for IP {} got exception {}".format(self._ip, e))

    def first_run(self):
        logger.info("IP: %s, FIRST RUN =============", self._ip)
        try:
            AXEventClient.post(topic='hosts', op="update", key=self._host_info['id'], data=self._host_info)
            self._event_start = datetime.datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ')
            self.update_container_info()
        except Exception as e:
            logger.exception("IP: %s, First Run exception: %s", self._ip, e)

    def update_container_info(self):
        self.store_spec_info()
        for cid in self._containers.keys():
            container_detail = self.get_container_detail(cid)
            AXEventClient.post(topic='containers', op="update", key=cid, data=container_detail)

    def store_spec_info(self):
        spec_info = self._cli.get_spec_info()
        for k, v in spec_info.items():
            if self.is_docker_key(k) and self.get_docker_id(k) not in self._containers:
                # CPU and Mem information can also be stored here. Because saas is not
                # handling these two values, we don't store.
                try:
                    ns = v["labels"].get("io.kubernetes.pod.namespace", None)
                    pname = v["labels"].get("io.kubernetes.pod.name", None)
                    cid = v['aliases'][1]
                    cname = v['aliases'][0]
                    if not all([ns, pname, cid, cname]):
                        logger.error("Cannot process container information: %s", v)
                        continue
                    self.set_container(containers=self._containers,
                                       id=cid,
                                       name=cname,
                                       kube_namespace=ns,
                                       kube_pod_name=pname)
                    self._service_config.register_container(cid=cid, container=self._containers[cid])
                except KeyError as ke:
                    logger.exception("Cannot store container information %s, exception: %s", v, ke)

    def get_container_detail(self, cid):
        container = self._containers.get(cid, None)
        if not container:
            raise AXPlatformException("No container record for container id {}".format(cid))
        container_config = dict(id=cid,
                                name=container["name"],
                                host_id=self._host_info['id'],
                                host_name=self._host_info['name'],
                                cost_id=self._service_config.get_cost_id(cid),
                                service_id=self._service_config.get_service_id(cid))

        # TODO: add reserved cpu/mem when upper layer is handling them
        # Discussed with Ying that we should not post these two fields now
        #
        # resources = self._service_config.get_resources(cname)
        # container_config['cpu'] = getattr(resources, 'cpu_cores', 0.0) if resources else 0.0
        # container_config['mem'] = getattr(resources, 'mem_mib', 0.0) if resources else 0.0
        return container_config

    def process_cadvisor_events(self, event_info, deleted_docker, created_docker):
        logger.debug("Processing events for node %s", self._ip)
        for event in event_info:
            if self.is_docker_key(event['container_name']):
                if event['event_type'] == 'containerDeletion':
                    if deleted_docker.count(self.get_docker_id(event['container_name'])) == 0:
                        deleted_docker.append(self.get_docker_id(event['container_name']))
                elif event['event_type'] == 'containerCreation':
                    if created_docker.count(self.get_docker_id(event['container_name'])) == 0:
                        created_docker.append(self.get_docker_id(event['container_name']))
                elif event['event_type'] == 'oom':
                    logger.error("OOM event occurred for container %s at time %s", event['container_name'],
                                 event['timestamp'])
                elif event['event_type'] == 'oomKill':
                    logger.error(
                        "OOM killed event occurred for container %s at time %s", event['container_name'],
                        event['timestamp'])

    def process_cadvisor_stats(self, docker_stats):
        # Ported from branch prod/0.9.1, Chengjie's commit:
        #
        # Cadvisor gets memory usage from Linxu cgroup sysfs. Memory usage is from memory.usage_in_bytes.
        # It's RSS + cache, where cache is likely what is attributed to each container for per container number.
        # But for global usage in /sys/fs/cgroup/memory.usage_in_bytes,
        # it might have included all buffers and page cache.
        # Global usage != sum(all container usage). The global number is not correct.
        #
        # We add up usage from all containers and use that for host usage.
        # This doesn't include global page cache, or additional processes not covered by container.
        # It tends to under estimate actual usage. But it's closer to real number than reported global number,
        # which is grossly over estimated.
        # See https://lwn.net/Articles/432224/

        if docker_stats is None:
            logger.info("Docker stats is None")
            return
        logger.debug("Processing stats for node %s", self._ip)
        host_mem_usage = 0.0
        host_cpu_request = 0.0
        host_cpu_request_used = 0.0
        host_mem_request = 0.0
        host_usage = None
        for k, v in docker_stats.items():
            if self.is_docker_key(k):
                container_usage = self.process_container_usage(self.get_docker_id(k), v)
                if bool(container_usage):
                    AXEventClient.post(topic='container_usages', op="create", key=self.get_docker_id(k),
                                       data=container_usage)
                    host_mem_usage += container_usage['mem']
                    host_cpu_request += container_usage['cpu_request']
                    host_cpu_request_used += container_usage['cpu_request_used']
                    host_mem_request += container_usage['mem_request']
            elif k == '/':
                if host_usage is not None:
                    logger.error("More than one sample for {}".format(self._ip))
                host_usage = self.process_host_usage(v)

        if bool(host_usage):
            # Replace memory usage with revised number from sum of all containers.
            host_usage['mem'] = host_mem_usage
            host_usage['mem_percent'] = float(host_usage['mem'] * 100) / self._host_record['mem']
            host_usage['cpu_request'] = host_cpu_request
            host_usage['cpu_request_used'] = host_cpu_request_used
            host_usage['mem_request'] = host_mem_request
            if host_usage['mem_percent'] > 100:
                logger.warning("Calculated host mem percentage %s greater than 100. Set it to 100",
                               host_usage['mem_percent'])
                host_usage['mem_percent'] = 100
            AXEventClient.post(topic='host_usages', op="create", key=self._host_info['id'], data=host_usage)

    def process_host_usage(self, stat):
        logger.debug("Processing host usage for node %s", self._ip)
        last_update = self._host_record.get("update_ts", None)
        current_time, current_cpu, current_mem = self.calc_current_usage(stat, last_update)

        if not (current_time and current_cpu and current_mem):
            return None

        usage = self.generate_host_usage(current_time, current_cpu, current_mem)
        self._host_record["update_ts"] = current_time
        self._host_record["last_cpu"] = current_cpu
        self._host_record["last_mem"] = current_mem
        return usage

    def process_container_usage(self, cid, stat):
        logger.debug("Processing container usage for node %s. cid: %s", self._ip, cid)
        if cid not in self._containers:
            logger.info(
                'Ignoring the stats since container %s needs to be bootstrapped or no newer stats received', cid)
            return None

        last_update = self.get_container_val(self._containers, cid, "update_ts")
        current_time, current_cpu, current_mem = self.calc_current_usage(stat, last_update)

        if not (current_time and current_cpu and current_mem):
            return None

        usage = self.generate_container_usage(cid, current_time, current_cpu, current_mem)

        self.set_container_val(self._containers, cid, 'update_ts', current_time)
        self.set_container_val(self._containers, cid, 'cpu', current_cpu)
        self.set_container_val(self._containers, cid, 'mem', current_mem)
        return usage

    def calc_current_usage(self, stat, last_update):
        tmp_time = []
        tmp_cpu = []
        tmp_mem = []
        for val in stat:
            # This is due to a known Python threading issue described in
            # http://stackoverflow.com/questions/2427240/thread-safe-equivalent-to-pythons-time-strptime and
            # http://bugs.python.org/issue7980
            # Followed suggestion to import it before calling
            from time import strptime
            tmp_ts = calendar.timegm(strptime(val['timestamp'].split('.')[0], '%Y-%m-%dT%H:%M:%S'))
            if not last_update or tmp_ts > last_update:
                tmp_time.append(tmp_ts)
                tmp_cpu.append(val['cpu']['usage']['total'])
                tmp_mem.append(val['memory']['usage'])
        if not self.is_update_ts_req(tmp_time, last_update):
            return None, None, None
        current_cpu = self.get_metric_val(tmp_cpu)
        current_mem = self.get_metric_val(tmp_mem)
        current_time = max(tmp_time)
        return current_time, current_cpu, current_mem

    def generate_host_usage(self, current_time_usage, current_cpu_usage, current_memory_usage):
        host_usage = {'host_id': self._host_record['id'], 'host_name': self._host_record['name'],
                      'cpu_total': float(current_cpu_usage) / NS_PER_SECOND, 'mem': float(current_memory_usage) / MiB}
        host_usage['mem_percent'] = float(host_usage['mem'] * 100) / self._host_record['mem']

        last_time = self._host_record.get("update_ts", None)
        last_cpu_usage = self._host_record.get("last_cpu", None)

        if last_cpu_usage is None:
            host_usage['cpu_used'] = 0
            host_usage['cpu'] = 0
            host_usage['cpu_percent'] = 0
        else:
            delta_cpu = current_cpu_usage - last_cpu_usage
            delta_time = current_time_usage - last_time
            host_usage['cpu_used'] = float(delta_cpu) / NS_PER_SECOND
            host_usage['cpu'] = float(host_usage['cpu_used']) / delta_time
            host_usage['cpu_percent'] = float(host_usage['cpu'] * 100) / self._host_record['cpu']
            if host_usage['cpu_percent'] > 100:
                logger.warning("Calculated host cpu percentage %s greater than 100. Set it to 100",
                               host_usage['cpu_percent'])
                host_usage['cpu_percent'] = 100
        return host_usage

    def generate_container_usage(self, cid, current_time_usage, current_cpu_usage, current_memory_usage):
        # For cpu related numbers.
        #                   Description                                    Accumulative (always growing)  Unit
        # cpu_total:        Total CPU usage since kernel starts counting.  Yes                            core * second
        # cpu_used:         CPU used in last sample period.                No                             core * second
        # cpu:              Average CPU used per second in last period.    No                             core
        # cpu_percent:      Average CPU usage percent in last period.      No                             0-100 %
        # cpu_request:      Average CPU request for container.             No                             core
        # cpu_request_used: Total CPU request in last sample period.       No                             core * second
        container_usage = {'container_id': cid}
        cname = self.get_container_val(self._containers, cid, 'name')
        container_usage['container_name'] = cname
        container_usage['host_id'] = self._host_record['id']
        container_usage['cost_id'] = self._service_config.get_cost_id(cid)
        container_usage['service_id'] = self._service_config.get_service_id(cid)
        container_usage['cpu_request'] = self._service_config.get_cpu_request(cid)
        container_usage['mem_request'] = self._service_config.get_memory_request(cid)
        container_usage['cpu_total'] = float(current_cpu_usage) / NS_PER_SECOND
        container_usage['mem'] = float(current_memory_usage) / MiB
        container_usage['mem_percent'] = float(container_usage['mem'] * 100) / self._host_record['mem']
        if container_usage['mem_percent'] > 100:
            logger.warning("Calculated container mem percentage %s greater than 100. Set it to 100",
                           container_usage['mem_percent'])

        last_time = self.get_container_val(self._containers, cid, 'update_ts')
        last_cpu_usage = self.get_container_val(self._containers, cid, 'cpu')
        if last_cpu_usage is None:
            container_usage['cpu_used'] = 0
            container_usage['cpu'] = 0
            container_usage['cpu_percent'] = 0
            container_usage['cpu_request_used'] = 0
        else:
            delta_cpu = current_cpu_usage - last_cpu_usage
            delta_time = current_time_usage - last_time
            container_usage['cpu_used'] = float(delta_cpu) / NS_PER_SECOND
            container_usage['cpu'] = float(container_usage['cpu_used']) / delta_time
            container_usage['cpu_request_used'] = container_usage['cpu_request'] * delta_time
            container_usage['cpu_percent'] = float(container_usage['cpu'] * 100) / self._host_record['cpu']
            if container_usage['cpu_percent'] > 100:
                logger.warning("Calculated container cpu percentage %s greater than 100. Set it to 100",
                               container_usage['cpu_percent'])
                container_usage['cpu_percent'] = 100
        return container_usage

    @staticmethod
    def is_docker_key(docker_key):
        match = STATS_ENDPOINT_REGEX.match(docker_key)
        if match is not None:
            return True
        else:
            # for v1.4.3 clusters
            return docker_key.startswith('/docker/') and len(docker_key.split('/')) == 3

    @staticmethod
    def get_docker_id(docker_url):
        match = STATS_ENDPOINT_REGEX.match(docker_url)
        if match is not None:
            return match.groups()[0]
        else:
            # for v1.4.3 clusters
            return docker_url.split('/')[-1]

    @staticmethod
    def set_container(containers, id, name, kube_pod_name, kube_namespace):
        containers[id] = {}
        # The following 3 fields is guaranteed
        containers[id]["name"] = name
        containers[id]["kube_pod_name"] = kube_pod_name
        containers[id]["kube_namespace"] = kube_namespace

    @staticmethod
    def set_container_val(containers, id, key, val):
        containers[id][key] = val

    @staticmethod
    def get_container_val(containers, cid, key):
        return containers[cid][key] if cid in containers and key in containers[cid] else None

    @staticmethod
    def is_update_ts_req(time_arr, last_update_ts):
        # this is a python3 protection. In the first run, last_update_ts = None
        # in python3 last_update_ts < max(time_arr) would throw TypeError, but
        # in python2 last_update_ts < max(time_arr) would return True
        if not last_update_ts:
            return True
        return True if time_arr and last_update_ts < max(time_arr) else False

    @staticmethod
    def get_metric_val(arr, metric_calc='max'):
        return {
            'avg': sum(arr) / float(len(arr)),
            'min': min(arr),
            'max': max(arr)
        }.get(metric_calc)
