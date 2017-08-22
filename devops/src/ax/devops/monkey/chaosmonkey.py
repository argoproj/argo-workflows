import datetime
import logging
import json
import random
import threading
import time
import os
import sys
import smtplib
import queue
from threading import Thread
from email.mime.application import MIMEApplication
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.header import Header
from email.utils import formataddr

import yaml
from pytz import timezone
from apscheduler.schedulers.background import BackgroundScheduler
from ax.kubernetes.client import KubernetesApiClient
from voluptuous import Schema, Required, Optional
import urllib3
import requests
from requests.packages.urllib3.exceptions import InsecureRequestWarning
import boto3


from . import LOG_FILE_NAME

urllib3.disable_warnings()
requests.packages.urllib3.disable_warnings(InsecureRequestWarning)


logger = logging.getLogger(__name__)

job_schema = Schema({
    Required('total_kills'): int,
    Required('simultaneously_kill'): bool,
    Required('max_simultaneously_kill'): int,
    Required('wait_for_recovery_second'): int,
    Required('retry_separation_second'): int,
    Required('terminate_if_not_recovery'): bool,
    Required('health_check_before_kill'): bool,
    Optional('target_pods'): [str]
}, extra=True)


config_schema = Schema({
    Required('duration_in_hour'): float,
    Required('min_separation_second'): int,
    Required('notification_email'): str,
    Optional('enable_notification', default=True): bool,
    Optional('report_successful', default=True): bool,
    Required('pod'): job_schema,
    Required('instance'): job_schema,
}, extra=True)


AX_NAMESPACE = 'axsys'
AX_PODS = ['axconsole', 'axdb-0', 'axdb-1', 'axdb-2', 'axmon', 'axnotification', 'axops-deployment', 'axscheduler', 'axstats', 'axworkflowadc', 'commitdata', 'cron',
           'fixturemanager', 'fluentd', 'gateway', 'kafka-zk-1', 'kafka-zk-2', 'kafka-zk-3', 'redis']


class ChaosMonkeyJob(object):
    WAITING_RESULT = "WAITING"
    SKIPPING_RESULT = "SKIPPING"
    FAILURE_RESULT = "FAILURE"
    SUCCESS_RESULT = "SUCCESS"

    FAIL_TO_KILL = "FAIL_TO_KILL_POD"
    FAIL_HEALTH_CHECK = "FAILED_TO_PASS_POD_HEALTH_CHECK"
    FAIL_POD_NAME_NOT_EXIST = "TARGET_POD_NAME_NOT_EXIST"

    def __init__(self, job_id, run_date, target, result=WAITING_RESULT, detail=None):
        self.id = job_id
        self.run_date = run_date
        self.target = target
        self.result = result
        self.detail = detail

    def jsonify(self):
        return {self.id: [self.run_date, self.target, self.result]}


class PodChaosMonkeyJob(ChaosMonkeyJob):

    def __repr__(self):
        date_time = self.run_date.strftime("%Y-%m-%d %H:%M")
        detail = ""
        if self.detail:
            detail = ", details: {}".format(self.detail)
        return "Kill Pod Job, id: {}, run at: {}, target: {}, result: {}{}".\
            format(self.id, date_time, self.target, self.result, detail)


class InstanceChaosMonkeyJob(ChaosMonkeyJob):
    def __init__(self, job_id, run_date, target, result="WAITING", detail=None, total_kill=None, action="reboot"):
        self.total_kill = total_kill
        self.action = action
        super().__init__(job_id, run_date, target, result, detail=None)

    def __repr__(self):
        date_time = self.run_date.strftime("%Y-%m-%d %H:%M")
        detail = ""
        if self.detail:
            detail = ", details: {}".format(self.detail)
        return "Kill Instance Job, id: {}, run at: {}, {} {} {} instances, result: {}{}".\
            format(self.id, date_time, self.action, self.total_kill, self.target, self.result, detail)


class ChaosMonkey(object):
    def __init__(self, config_file=None, cluster_name=None):
        self.config_file = config_file
        self.cluster_name = cluster_name
        self.kube_client = None
        self.monkey_config = None
        self.scheduler = BackgroundScheduler()
        self._safe_lock = threading.Lock()
        self.job_store = dict()
        self.service_url = "{}/api/v1/proxy/namespaces/{}/services/{}"
        self.email_client = None
        self.dns_name = None

    def init(self):
        if self.cluster_name:
            self.kube_client = KubernetesApiClient(config_file="/tmp/ax_kube/cluster_{}.conf".format(self.cluster_name))
        else:
            self.kube_client = KubernetesApiClient()

        if self.config_file:
            config_location = self.config_file
        else:
            config_location = '/ax/etc/config.yaml' if getattr(sys, 'frozen', False) else 'config.yaml'
        with open(config_location) as f:
            yaml_result = yaml.load(f)
        self.monkey_config = config_schema(yaml_result)

        try:
            url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'axops-internal:8085/v1/tools?category=notification')
            resp = self.kube_client.session.get(url)
            smtp_config = resp.json()['data']
            if not smtp_config:
                self.email_client = ChaosMonkeyNotify(None, self.monkey_config['notification_email'], False)
            else:
                self.email_client = ChaosMonkeyNotify(smtp_config[0], self.monkey_config['notification_email'], self.monkey_config['enable_notification'])
        except Exception as exc:
            logger.exception("Failed to retrieve smtp configuration from AxOps. Will not notify. %s", str(exc))
            self.email_client = ChaosMonkeyNotify(None, self.monkey_config['notification_email'], False)

        try:
            url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'axops-internal:8085/v1/system/settings/dnsname')
            resp = self.kube_client.session.get(url)
            self.dns_name = "({})".format(resp.json()['dnsname'])
        except Exception as exc:
            logger.exception("Failed to retrieve dnsname from AxOps. %s", str(exc))
            self.dns_name = ""

        # init ec2
        boto3.setup_default_session(profile_name=self.monkey_config['instance']['aws_profile'])
        self.ec2 = boto3.resource('ec2')

    def run(self):
        self.init()
        self.add_jobs()
        self.email_client.send_email(self.get_schedules(), "Chaos Monkey is going to start {}".format(self.dns_name))
        logger.info("Configuration: \n%s", json.dumps(self.monkey_config, indent=2))
        while True:
            if len(self.get_active_jobs()) == 0:
                logger.info("Finish all jobs. Exit...")
                self.email_client.send_email(self.get_schedules(), "Chaos Monkey successfully finished {}".format(self.dns_name))
                sys.exit()
            time.sleep(20)

    @staticmethod
    def shuffle_list(current_list, number):
        """Shuffle list."""
        random.shuffle(current_list)
        return current_list[0: min(number, len(current_list))]

    def generate_random_schedule(self, namespace_list=list(), buffer_seconds=5):
        """Generate job schedule."""
        duration_in_hour = self.monkey_config['duration_in_hour']
        min_separation_second = self.monkey_config['min_separation_second']

        total_kills = sum([self.monkey_config[namespace]['total_kills'] for namespace in namespace_list])

        time_list = list(range(buffer_seconds, int(duration_in_hour*60*60)-buffer_seconds, min_separation_second))
        time_list = ChaosMonkey.shuffle_list(time_list, min(total_kills, len(time_list)))

        start_index = 0
        result = dict()
        for namespace in namespace_list:
            number = int(len(time_list) * (float(self.monkey_config[namespace]['total_kills'])/total_kills))
            for index in range(start_index, start_index + number):
                result[time_list[index]] = namespace
            start_index += number

        return result

    def add_jobs(self):
        """Add jobs into scheduler"""
        if self.scheduler.running:
            self.scheduler.shutdown()

        rand_schedule = self.generate_random_schedule(['pod', 'instance'])
        current_time = datetime.datetime.now(timezone('UTC')) + datetime.timedelta(seconds=10)
        current_time = current_time.astimezone(timezone('US/Pacific'))
        counter = 0

        for key in sorted(rand_schedule):
            job_id = counter
            run_time = current_time + datetime.timedelta(seconds=key)
            if rand_schedule[key] == 'pod':
                simultaneously_kill = self.monkey_config['pod']['simultaneously_kill']
                max_simultaneously_kill = self.monkey_config['pod']['max_simultaneously_kill']
                target_pods = self.monkey_config['pod']['target_pods']
                targets = ChaosMonkey.shuffle_list(target_pods, random.randint(1, max_simultaneously_kill) if simultaneously_kill else 1)
                job = PodChaosMonkeyJob(job_id, run_time, targets)
                self.scheduler.add_job(self.kill_pod_job, 'date', run_date=run_time, timezone='US/Pacific', args=[job])
                self.job_store[job_id] = job
            elif rand_schedule[key] == 'instance':
                if self.monkey_config['instance']['simultaneously_kill'] == True:
                    total_kill = random.randint(1, self.monkey_config['instance']['max_simultaneously_kill'])
                else:
                    total_kill = 1
                job = InstanceChaosMonkeyJob(job_id, run_time, "minion", total_kill=total_kill)
                self.scheduler.add_job(self.kill_instance_job, 'date', run_date=run_time, timezone='US/Pacific', args=[job])
                self.job_store[job_id] = job
            else:
                logger.error("Cannot recognize job type, %s", rand_schedule[key])
            counter += 1

        logger.info(self.get_schedules())

        for i in range(5):
            logger.info("Starting chaos monkey in {} seconds".format(5 - i))
            time.sleep(1)

        self.scheduler.start()

    def check_all_instance_healthy(self, job_id):
        """Check all pods healthy."""
        logger.info("[job {}] Perform health check for all instances".format(job_id))
        counter = 0
        time.sleep(3)
        while counter * 5 < self.monkey_config['instance']['wait_for_recovery_second']:
            if self.check_all_instance_healthy_helper(job_id):
                return True
            else:
                time.sleep(self.monkey_config['instance']['retry_separation_second'])
                counter += 1

        logger.error("Failed health check due to timeout ({}s)".format(job_id, self.monkey_config['instance'][
            'wait_for_recovery_second']))
        return False

    def check_all_instance_healthy_helper(self, job_id):
        """Helper function for health check."""
        # check to make sure at lease one master instances is running
        instance_name = "{}-master".format(self.cluster_name)
        running_master = 0
        for i in self.ec2.instances.filter(Filters=[{"Name": "tag-value", "Values": [instance_name]}]):
            logger.info("[job {}] Found {} instance ID {}".format(job_id, i.state['Name'], i.instance_id))
            if i.state['Name'] == "running":
                running_master = running_master + 1

        # check to make sure minion instances are running according to auto-scaling policy
        instance_name = "{}-minion".format(self.cluster_name)
        running_minion = 0
        for i in self.ec2.instances.filter(Filters=[{"Name": "tag-value", "Values": [instance_name]}]):
            logger.info("[job {}] Found {} instance ID {}".format(job_id, i.state['Name'], i.instance_id))
            if i.state['Name'] == "running":
                running_minion = running_minion + 1

        # TODO: get desire num from autoscaling group.
        if running_master < 1:
            logger.error('[job {}] only {} master instances running. Health check failed.'.format(running_master))
            return False
        elif running_minion < 2:
            logger.error('[job {}] only {} minion instances running. Health check failed.'.format(running_minion))
            return False
        logger.info(
            '[job {}] {} master and {} minion instances running. Health check passed.'.format(job_id, running_master,
                                                                                              running_minion))
        return True

    def kill_instance(self, job_id, target="minion", num=1, action="reboot", DryRun=True):
        instance_ids = []
        instance_name = "{}-{}".format(self.cluster_name, target)
        for i in self.ec2.instances.filter(Filters=[{"Name": "tag-value", "Values": [instance_name]}]):
            #logger.info("[job {}] Found {} instance ID {}".format(job_id, i.state['Name'], i.instance_id))
            if i.state['Name'] == "running":
                instance_ids.append(i.instance_id)

        for i in random.sample(instance_ids, num):
            logger.info("[job {}] {} instance ID {}".format(job_id, action, i))
            try:
                if action == "terminate":
                    self.ec2.instances.filter(InstanceIds=[i]).terminate(DryRun=DryRun)
                elif action == "reboot":
                    self.ec2.instances.filter(InstanceIds=[i]).reboot(DryRun=DryRun)
                time.sleep(5)
            except Exception as exp:
                logger.exception("[job {}] failed to {} instance {}, %s".format(job_id, action, i), str(exp))
                return False
        return True

    def kill_instance_job(self, job):
        """Process the job that kills a list of instances"""
        if self._safe_lock.acquire(timeout=5):
            try:
                if self.monkey_config['instance']['health_check_before_kill']:
                    if not self.check_all_instance_healthy(job.id):
                        logger.error(
                            'Failed to check all instances healthy before killing. Please verify system. Exit...')
                        self.program_exit()

                # action can be one of following: reboot, terminate
                job.action = self.monkey_config['instance']['action']
                if not self.kill_instance(job.id, num=job.total_kill, DryRun=False, action=job.action,
                                          target=job.target):
                    job.result = ChaosMonkeyJob.FAIL_TO_KILL

                if not self.check_all_instance_healthy(job.id):
                    job.result = ChaosMonkeyJob.FAILURE_RESULT
                    logger.error('Failed to check all instances healthy after killing. Please verify system. Exit...')
                    self.program_exit()
                logger.info("Successfully run health check for all instances.\n")
                job.result = ChaosMonkeyJob.SUCCESS_RESULT
                logger.info(self.get_schedules())
                self.email_client.send_email(self.get_schedules(),
                                             "Chaos Monkey Job {} Result {} {}".format(job.id, job.result,
                                                                                       self.dns_name))
            except Exception as exc:
                logger.exception("Error during kill instance job, %s", str(exc))
                self.program_exit()
            finally:
                self._safe_lock.release()
        else:
            logger.info("[job {}] there is another job running, skip this scheduled kill.".format(job.id))
            job.result = ChaosMonkeyJob.SKIPPING_RESULT

    def kill_pod_job(self, job):
        """Process the job that kills a list of pods"""
        if self._safe_lock.acquire(timeout=5):
            try:
                if self.monkey_config['pod']['health_check_before_kill']:
                    if not self.check_all_pod_healthy(job.id):
                        logger.error('Failed to check all services healthy before killing. Please verify system. Exit...')
                        self.program_exit()
                    logger.info("Successfully run health check for all services.\n")

                threads = []
                q = queue.Queue()

                for target in job.target:
                    t = Thread(name="job-{}-kill-{}-thread".format(job.id, target),
                               target=self.kill_pod, args=(target, job.id, q))
                    t.daemon = True
                    threads.append(t)

                # Start all threads
                for x in threads:
                    x.start()

                # Wait for all of them to finish
                for x in threads:
                    x.join()

                details = list()
                job_fail = False
                while not q.empty():
                    res = q.get()
                    for value in res.values():
                        if value != ChaosMonkeyJob.SUCCESS_RESULT:
                            job_fail = True
                        details.append(value)

                job.details = details
                job.result = ChaosMonkeyJob.FAILURE_RESULT if job_fail else ChaosMonkeyJob.SUCCESS_RESULT

                logger.info(self.get_schedules())
                if job.result != ChaosMonkeyJob.SUCCESS_RESULT or self.monkey_config['report_successful']:
                    self.email_client.send_email(self.get_schedules(), "Chaos Monkey Job {} Result {} {}".format(job.id, job.result, self.dns_name))

                if job_fail:
                    self.program_exit()
            except Exception as exc:
                logger.exception("Error during kill pod job, %s", str(exc))
                self.program_exit()
            finally:
                self._safe_lock.release()
        else:
            logger.info("[job {}] there is another job running, skip this scheduled kill.".format(job.id))
            job.result = ChaosMonkeyJob.SKIPPING_RESULT

    def kill_pod(self, target, job_id, result_q):
        """Kill one target"""
        pod = self.get_pod_name(target, job_id)
        if not pod:
            logger.error("[job {}] Cannot find pod with name, {}".format(job_id, target))
            result_q.put({target: ChaosMonkeyJob.FAIL_POD_NAME_NOT_EXIST})
        else:
            logger.info("[job {}] Killing pod {}".format(job_id, pod))
            pod_obj = self.kube_client.api.read_namespaced_pod(AX_NAMESPACE, pod)
            try:
                from ax.kubernetes.swagger_client import V1DeleteOptions
                self.kube_client.api.delete_namespaced_pod(V1DeleteOptions(), pod_obj.metadata.namespace, pod_obj.metadata.name)
            except Exception as exc:
                logger.exception("[job {}] failed to kill pod {}, %s".format(job_id, pod), str(exc))
                result_q.put({target: ChaosMonkeyJob.FAIL_TO_KILL})
                return
            if not self.pod_health_check(target, job_id):
                result_q.put({target: ChaosMonkeyJob.FAIL_HEALTH_CHECK})
                return
            result_q.put({target: ChaosMonkeyJob.SUCCESS_RESULT})

    def get_pod_name(self, short_name, job_id):
        """Get pod name with the service name. If multiple returned, randomly select one."""
        result = list()
        for pod in self.kube_client.api.list_namespaced_pod(AX_NAMESPACE).items:
            if str(pod.metadata.name).startswith(short_name):
                result.append(str(pod.metadata.name))
        logger.info("[job {}] Get pod name of {} is {}".format(job_id, short_name, result))
        if result:
            return result[random.randint(0, len(result)-1)]
        return None

    def get_pod(self, short_name, job_id):
        """Get pod with the service name."""
        result = list()
        for pod in self.kube_client.api.list_namespaced_pod(AX_NAMESPACE).items:
            if str(pod.metadata.name).startswith(short_name):
                result.append(pod)
        return result

    def pod_health_check(self, short_name, job_id):
        """Check pod health after killing the pod."""
        logger.info("Perform health check after killing {}".format(short_name))
        counter = 0
        time.sleep(3)
        while counter*5 < self.monkey_config['pod']['wait_for_recovery_second']:
            for pod in self.kube_client.api.list_namespaced_pod(AX_NAMESPACE).items:
                if str(pod.metadata.name).startswith(short_name):
                    if self.check_pod_is_running(pod, job_id):
                        if self.check_pod_recover(str(pod.metadata.name), job_id):
                            logger.info("[job {}] pod {} comes back healthy".format(job_id, pod.metadata.name))
                            return True
                        else:
                            logger.info("[job {}] pod {} is running but NOT healthy".format(job_id, pod.metadata.name))
                            time.sleep(self.monkey_config['pod']['retry_separation_second'])
                            counter += 1
                    else:
                        logger.info('[job {}] pod {} still not running'.format(job_id, pod.metadata.name))
                        time.sleep(self.monkey_config['pod']['retry_separation_second'])
                        counter += 1
        logger.error("[job {}] pod {} failed health check due to timeout ({}s)".format(job_id, short_name, self.monkey_config['pod']['wait_for_recovery_second']))
        return False

    @classmethod
    def check_pod_is_running(cls, pod, job_id):
        """Check if a pod is in running state."""
        if pod.status.phase == "Running":
            for container in pod.status.container_statuses:
                if container.ready is False:
                    logger.warn("[job {}] pod {} is running but container {} is not ready.".format(job_id, pod.metadata.name, container.name))
                    return False
            return True
        return False

    def check_pod_recover(self, pod_name, job_id=None):
        """Check individual pod health based on health links."""
        job_id = "[job {}] ".format(job_id) if job_id is None else ""
        try:
            logger.info("{}Check pod {} health".format(job_id, pod_name))
            if pod_name.startswith('axdb-'):  # axdb
                result = self.kube_client.exec_cmd('axsys', pod_name, ['sh', '-c', '/ax/axdb/health.sh > /dev/null 2>&1 && echo $?'])
                try:
                    if int(result) == 0:
                        return True
                except:
                    return False
            elif pod_name.startswith('kafka-zk-'):  # kafka
                index = pod_name.split('kafka-zk-')[1][0]
                result = self.kube_client.exec_cmd('axsys', pod_name, ['sh', '-c', 'timeout 5 nc -z kafka-zk-{} 9092 > /dev/null 2>&1 && echo $?'.format(index)], container='kafka')
                try:
                    if int(result) == 0:
                        return True
                except:
                    return False
            elif pod_name.startswith('axconsole-'):  # axconsole
                url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'axconsole/ping')
                if self.kube_client.session.get(url).status_code == 200:
                    return True
            elif pod_name.startswith('axmon-'):  # axmon
                url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'axmon/v1/axmon/ping')
                if self.kube_client.session.get(url).status_code == 200:
                    return True
            elif pod_name.startswith('axnotification-'):  # axnotification
                url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'axnotification/ping')
                if self.kube_client.session.get(url).status_code == 200:
                    return True
            elif pod_name.startswith('axscheduler-'):  # axscheduler
                url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'axscheduler/v1/scheduler/ping')
                if self.kube_client.session.get(url).status_code == 200:
                    return True
            elif pod_name.startswith('axworkflowadc-'):  # axworkflowadc
                url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'axworkflowadc/v1/adc/ping')
                if self.kube_client.session.get(url).status_code == 200:
                    return True
            elif pod_name.startswith('fixturemanager-'):  # fixturemanager
                url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'fixturemanager/v1/fixture/ping')
                if self.kube_client.session.get(url).status_code == 200:
                    return True
            elif pod_name.startswith('gateway-'):  # gateway
                url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'gateway/')
                if self.kube_client.session.get(url).status_code == 200:
                    return True
            elif pod_name.startswith('redis-'):  # redis
                result = self.kube_client.exec_cmd('axsys', pod_name, ['sh', '-c', 'redis-cli ping'])
                if str(result).strip() == 'PONG':
                    return True
            elif pod_name.startswith('axops-deployment-'):  # axops
                url = self.service_url.format(self.kube_client.url, AX_NAMESPACE, 'axops-internal:8085/v1/ping')
                if self.kube_client.session.get(url).status_code == 200:
                    return True
            else:
                return True
        except Exception as exc:
            logger.exception("{} failed to check {} health, %s".format(job_id, pod_name), str(exc))

        return False

    def check_all_pod_healthy(self, job_id):
        """Check all pods healthy."""
        logger.info("[job {}] Perform health check for all pods before kill job".format(job_id))
        counter = 0
        time.sleep(3)
        while counter*5 < self.monkey_config['pod']['wait_for_recovery_second']:
            if self.check_all_pod_healthy_helper(job_id):
                return True
            else:
                time.sleep(self.monkey_config['pod']['retry_separation_second'])
                counter += 1

        logger.error("Failed health check due to timeout ({}s before killing)".format(job_id, self.monkey_config['pod']['wait_for_recovery_second']))
        return False

    def check_all_pod_healthy_helper(self, job_id):
        """Helper function for health check."""
        for name in AX_PODS:
            pods = self.get_pod(name, job_id)
            if not pods:
                logger.error('[job {}] Cannot find pod with service name {}. Health check failed.'.format(job_id, name))
                return False
            else:
                for pod in pods:
                    if not (self.check_pod_is_running(pod, job_id) and self.check_pod_recover(str(pod.metadata.name), job_id)):
                        logger.info('[job {}] Pod {} is not running healthy'.format(job_id, pod.metadata.name))
                        return False
        return True

    def get_schedules(self):
        """Get the scheduled jobs in the current scheduler."""
        result = "\nJob schedules and results:"
        for key in sorted(self.job_store):
            result += "\n" + repr(self.job_store[key])

        result += "\nRemaining job count: {}".format(len(self.get_active_jobs()))
        return result

    def get_active_jobs(self):
        """Get active jobs from scheduler."""
        result = list()
        for key, job in self.job_store.items():
            if job.result == ChaosMonkeyJob.WAITING_RESULT:
                result.append(job)
        return result

    def program_exit(self, error_code=1):
        self.email_client.send_email(self.get_schedules(), "Chaos Monkey Exit Error Log {}".format(self.dns_name))
        os._exit(error_code)


class ChaosMonkeyNotify(object):
    def __init__(self, smtp_config, email_address, active=True):
        self.email_address = email_address.strip().split(',')
        self.active = active
        self.smtp_config = smtp_config

    def send_email(self, content, subject=None):
        """Send email notification."""
        if not self.active:
            return
        if not self.email_address:
            return
        try:
            mail_server = smtplib.SMTP(self.smtp_config['url'],
                                       self.smtp_config['port'],
                                       timeout=self.smtp_config['timeout'])
            if self.smtp_config['use_tls']:
                mail_server.ehlo()
                mail_server.starttls()
                mail_server.ehlo()
            if self.smtp_config['username'] or self.smtp_config['password']:
                mail_server.login(self.smtp_config['username'], self.smtp_config['password'])

            msg = MIMEMultipart()
            msg['From'] = formataddr((str(Header('Argo Chaos Monkey', 'utf-8')), self.smtp_config['admin_address']))
            msg['To'] = ", ".join(self.email_address)
            msg['Subject'] = subject if subject else "Chaos Monkey in action"
            body = MIMEText(content, 'plain')
            msg.attach(body)

            with open(LOG_FILE_NAME, "rb") as fil:
                part = MIMEApplication(fil.read(), Name=LOG_FILE_NAME)
                part['Content-Disposition'] = 'attachment; filename="%s"' % LOG_FILE_NAME
                msg.attach(part)

            mail_server.sendmail(msg['From'], self.email_address, msg.as_string())
            logger.info("Successful sending email notification to %s", self.email_address)
            mail_server.quit()
        except Exception as exc:
            logger.exception("Failed to send email notification to {}, %s".format(self.email_address), str(exc))
