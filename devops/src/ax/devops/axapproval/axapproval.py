"""
This is a script that will reach out a list of recipients asking for approves for whether to
proceed to the next step.

Usage:
axapproval.py --required_list <required_list>
              --optional_list <optional_list>
              --number_optional <number_optional>
              --timeout <timeout>
"""
import argparse
import logging
import os
import sys
import time
import json
import jwt

from ax.version import __version__
from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.axsys.axsys_client import AxsysClient
from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.redis.redis_client import RedisClient, DB_RESULT


logger = logging.getLogger(__name__)

axdb_client = AxdbClient()
axsys_client = AxsysClient()
axops_client = AxopsClient()
redis_client = RedisClient(host='redis.axsys', db=DB_RESULT, retry_max_attempt=10, retry_wait_fixed=5000)


class AXApprovalException(RuntimeError):
    pass


class AXApproval(object):
    FAILURE_STATE = "FAILURE"
    WAITING_STATE = "WAITING"
    APPROVE_STRING = "APPROVE"
    DECLINE_STRING = "DECLINE"

    def __init__(self, required_list, optional_list, number_optional, timeout):
        self.task_id = None
        self.root_id = None
        self.leaf_id = None
        self.redis_key = None
        self.required_list = None
        self.optional_list = None
        self.number_optional = None
        self.timeout = None
        self.approved_required_list = list()
        self.approved_optional_list = list()
        self.declined_optional_list = list()
        self.precheck(required_list, optional_list, number_optional, timeout)

    def precheck(self, required_list, optional_list, number_optional, timeout):
        """Precheck for the environment variable and passed-in arguments"""

        self.task_id = os.getenv('AX_CONTAINER_NAME')
        if not self.task_id:
            logger.error("AX_CONTAINER_NAME cannot be found in the container ENV.")
            sys.exit(1)

        self.root_id = os.getenv('AX_ROOT_SERVICE_INSTANCE_ID')
        if not self.root_id:
            logger.error("AX_ROOT_SERVICE_INSTANCE_ID cannot be found in the container ENV.")
            sys.exit(1)

        self.leaf_id = os.getenv('AX_SERVICE_INSTANCE_ID')
        if not self.leaf_id:
            logger.error("AX_SERVICE_INSTANCE_ID cannot be found in the container ENV.")
            sys.exit(1)

        self.redis_key = self.leaf_id + '-axapproval'

        required_list = required_list.strip()
        optional_list = optional_list.strip()

        if not required_list:
            required_list = []
        else:
            required_list = [x.strip() for x in required_list.split(',')]

        if not optional_list:
            optional_list = []
        else:
            optional_list = [x.strip() for x in optional_list.split(',')]

        if not required_list and not optional_list:
            logger.error('required_list and optional_list cannot both be empty.')
            sys.exit(1)

        try:
            number_optional = int(number_optional)
            timeout = int(timeout)
        except Exception:
            logger.exception('number_optional, timeout must be integer')
            sys.exit(1)

        if not isinstance(number_optional, int) or not isinstance(timeout, int):
            logger.error('number_optional, timeout must be integer')
            sys.exit(1)

        if number_optional < 0 or timeout < 0:
            logger.error('number_optional or timeout cannot be negative.')
            sys.exit(1)

        if number_optional > len(optional_list):
            logger.error('number_optional cannot be greater than optional_list.')
            sys.exit(1)

        required_set = set(required_list)
        optional_set = set(optional_list)

        intersection_set = required_set.intersection(optional_set)
        if intersection_set:
            logger.error('%s cannot be in both required_list and optional_list.' % str(intersection_set))
            sys.exit(1)

        self.required_list = required_list
        self.optional_list = optional_list
        self.number_optional = int(number_optional)
        self.timeout = int(timeout)

        # Backward compatible for axops hostname
        global axops_client
        if not axops_client.ping():
            # Using the old hostname for axops
            axops_client = AxopsClient(host='axops.axsys')

        if not axops_client.get_tools(type='smtp'):
            logger.error("Email notification is not configured. Please configure the smtp email notification first.")
            sys.exit(1)

    def notification(self, approver_list):
        """Send notifications to required and optional reviewers"""
        dns_name = axops_client.get_dns()
        job_id = self.root_id
        url_to_ui = 'https://{}/app/jobs/job-details/{}'.format(dns_name, job_id)
        service = axops_client.get_service(job_id)

        html_payload = """
<html>
<body>
  <table class="email-container" style="font-size: 14px;color: #333;font-family: arial;">
    <tr>
      <td class="msg-content" style="padding: 20px 0px;">
        The {} job is waiting for your approval. The job was triggered by {}.
      </td>
    </tr>
    <tr>
      <td class="commit-details" style="padding: 20px 0px;">
        <table cellspacing="0" style="border-left: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;border-top: 1px solid #e3e3e3;">
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Author</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Repo</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Branch</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Description</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Revision</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{}</td>
          </tr>
        </table>
      </td>
    </tr>
    <tr>
      <td class="view-job">
        <div>
          <!--[if mso]>
  <v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="{}" style="height:40px;v-text-anchor:middle;width:150px;" arcsize="125%" strokecolor="#00BDCE" fillcolor="#7fdee6">
    <w:anchorlock/>
    <center style="color:#333;font-family:arial;font-size:14px;font-weight:bold;">VIEW JOB</center>
  </v:roundrect>
<![endif]--><a href="{}" style="background-color:#7fdee6;border:1px solid #00BDCE;border-radius:50px;color:#333;display:inline-block;font-family:arial;font-size:14px;font-weight:bold;line-height:40px;text-align:center;text-decoration:none;width:150px;-webkit-text-size-adjust:none;mso-hide:all;">VIEW JOB</a></div>
      </td>
    </tr>
  <tr>
      <td class="view-job">
        <div>
          <!--[if mso]>
  <v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="{}" style="height:40px;v-text-anchor:middle;width:150px;" arcsize="125%" strokecolor="#00BDCE" fillcolor="#7fdee6">
    <w:anchorlock/>
    <center style="color:#333;font-family:arial;font-size:14px;font-weight:bold;">APPROVE</center>
  </v:roundrect>
<![endif]--><a href="{}" style="background-color:#7fdee6;border:1px solid #00BDCE;border-radius:50px;color:#333;display:inline-block;font-family:arial;font-size:14px;font-weight:bold;line-height:40px;text-align:center;text-decoration:none;width:150px;-webkit-text-size-adjust:none;mso-hide:all;">APPROVE</a></div>
      </td>
    </tr>
  <tr>
      <td class="view-job">
        <div>
          <!--[if mso]>
  <v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="{}" style="height:40px;v-text-anchor:middle;width:150px;" arcsize="125%" strokecolor="#00BDCE" fillcolor="#7fdee6">
    <w:anchorlock/>
    <center style="color:#333;font-family:arial;font-size:14px;font-weight:bold;">DECLINE</center>
  </v:roundrect>
<![endif]--><a href="{}" style="background-color:#7fdee6;border:1px solid #00BDCE;border-radius:50px;color:#333;display:inline-block;font-family:arial;font-size:14px;font-weight:bold;line-height:40px;text-align:center;text-decoration:none;width:150px;-webkit-text-size-adjust:none;mso-hide:all;">DECLINE</a></div>
      </td>
    </tr>
    <tr>
      <td class="thank-you" style="padding-top: 20px;line-height: 22px;">
          Thanks,<br>
        Argo Project
      </td>
    </tr>
  </table>
</body>
</html>
"""

        for user in approver_list:

            approve_token, decline_token = self.generate_token(user=user, dns_name=dns_name)

            approve_link = "https://{}/v1/results/id/approval?token={}".format(dns_name, approve_token)
            decline_link = "https://{}/v1/results/id/approval?token={}".format(dns_name, decline_token)

            msg = {
                'to': [user],
                'subject': 'The {} job requires your approval to proceed'.format(service['name']),
                'body': html_payload.format(service['name'], service['user'],
                                            service['commit']['author'], service['commit']['repo'],
                                            service['commit']['branch'], service['commit']['description'], service['commit']['revision'],
                                            url_to_ui, url_to_ui, approve_link, approve_link, decline_link, decline_link),
                'html': True
            }

            if service['user'] != 'system':
                try:
                    user_result = axops_client.get_user(service['user'])
                    msg['display_name'] = "{} {}".format(user_result['first_name'], user_result['last_name'])
                except Exception as exc:
                    logger.error("Fail to get user %s", str(exc))

            logger.info('Sending approval requests to %s', str(user))
            result = axsys_client.send_notification(msg)

            # TODO: Tianhe adding retry mechanism
            if result.status_code != 200:
                logger.error('Cannot send approval request, %s', result.content)
                sys.exit(1)
        logger.info('Successfully sent approval requests to reviewers.')

    def generate_token(self, user, dns_name):
        token_dict = {
            'root_id': self.root_id,
            'leaf_id': self.leaf_id,
            'dns': dns_name,
            'user': user,
            'result': 'approved'
        }

        approve_token = jwt.encode(token_dict, 'ax', algorithm='HS256')

        token_dict['result'] = 'declined'
        decline_token = jwt.encode(token_dict, 'ax', algorithm='HS256')

        return approve_token.decode('utf-8'), decline_token.decode('utf-8')

    def get_notification_list(self):
        email_list = list()
        for user in self.required_list:
            if user not in self.approved_required_list:
                email_list.append(user)

        for user in self.optional_list:
            if user not in self.approved_optional_list and user not in self.declined_optional_list:
                email_list.append(user)
        return email_list

    def check_result(self):
        """Check Redis and axdb for results of the approval request"""
        logger.info("Wait for results")
        self.check_items(self.get_user_results_from_db())
        tuples = redis_client.brpop(self.redis_key, timeout=300)
        if tuples:
            logger.info("Received results")
            self.check_items([json.loads(tuples[1])])

    def check_items(self, items):
        """Check the incoming result items and update internal accounting"""
        for item in items:
            if item['user'] in self.required_list and item['user'] not in self.approved_required_list:
                if item['result'] is True or item['result'] == 'approved':
                    logger.info("%s approved", item['user'])
                    self.approved_required_list.append(item['user'])
                else:
                    logger.info("%s declined", item['user'])
                    error_msg = "Since {} is a required member for approval, this approval step fails. ".format(item['user'])
                    logger.error(error_msg)
                    self.exit(rc=2, detail=error_msg)
            if item['user'] in self.optional_list and item['user'] not in self.approved_optional_list:
                if item['result'] is True or item['result'] == 'approved':
                    logger.info("%s approved", item['user'])
                    if item['user'] in self.declined_optional_list:
                        self.declined_optional_list.remove(item['user'])
                    self.approved_optional_list.append(item['user'])
                else:
                    logger.info("%s declined", item['user'])
                    self.declined_optional_list.append(item['user'])
                    if len(self.declined_optional_list) >= (len(self.optional_list) - self.number_optional):
                        error_msg = "Not be able to fulfill requirement that {} optional approvals, since {} declined request.".format(
                            self.number_optional, self.declined_optional_list)
                        logger.error(error_msg)
                        self.exit(rc=2, detail=error_msg)

        if len(self.approved_required_list) >= len(self.required_list) \
                and len(self.approved_optional_list) >= self.number_optional:
            logger.info("Approval requirements are fully met. Exit gracefully.")
            self.exit(0)

    def recover(self):
        """Recover by reading previous results if any"""
        if self.get_info_from_db():
            logger.info("Recover by reading previous results")
            self.check_items(self.get_user_results_from_db())
        else:
            self.create_info_in_db()  # create record in axdb

    def run(self):
        """Approval procedure"""
        self.recover()
        self.notification(self.get_notification_list())

        logger.info("Start to wait for the reviewer responses")
        start_time = time.time()  # this does not handle the case for recovery
        last_reminder_time = start_time

        while True:
            self.check_result()
            current_time = time.time()
            if self.timeout and current_time - start_time > (self.timeout * 60.0):
                logger.error("Timeout for getting approvals. Exit.")
                self.exit(1)
            if current_time - last_reminder_time > 24*60*60:  # last reminder more than a day old
                last_reminder_time = current_time
                try:
                    self.notification(self.get_notification_list())
                except Exception as exc:
                    logger.exception("Sending reminder failed, %s", str(exc))

    def exit(self, rc, detail=None):
        """Approval exit"""
        if rc == 0:
            self.update_info_in_db(result=AXApproval.APPROVE_STRING)
        elif rc == 1:
            self.update_info_in_db(result=AXApproval.FAILURE_STATE, detail="Timeout for waiting for approvals.")
        elif rc == 2:
            self.update_info_in_db(result=AXApproval.DECLINE_STRING, detail=detail)
        else:
            self.update_info_in_db(result=AXApproval.FAILURE_STATE, detail=detail)
        sys.exit(rc)

    def create_info_in_db(self):
        """Create an approval info record in axdb"""
        axdb_client.create_approval_info(root_id=self.root_id,
                                         leaf_id=self.leaf_id,
                                         required_list=json.dumps(self.required_list),
                                         optional_list=json.dumps(self.optional_list),
                                         optional_number=self.number_optional,
                                         timeout=self.timeout,
                                         result=AXApproval.WAITING_STATE)

    def get_info_from_db(self):
        """Get an approval info record from axdb"""
        return axdb_client.get_approval_info(root_id=self.root_id, leaf_id=self.leaf_id)

    def update_info_in_db(self, result, detail=None):
        """Update an approval info result in axdb"""
        payload = {'result': result}
        if detail:
            payload['detail'] = detail
        axdb_client.update_approval_info(root_id=self.root_id,
                                         leaf_id=self.leaf_id,
                                         approval_result=payload)

    def get_user_results_from_db(self):
        """Get user approval results from axdb"""
        results = axdb_client.get_approval_results(leaf_id=self.leaf_id)
        return results


def main():
    logging.basicConfig(stream=sys.stdout, level=logging.INFO,
                        format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S")
    parser = argparse.ArgumentParser()
    parser.add_argument('--required_list', default=None, type=str,
                        help='List of reviewers that must approve to proceed')
    parser.add_argument('--optional_list', default=None, type=str,
                        help='List of approvers that can optionally approve to proceed')
    parser.add_argument('--number_optional', default=0, type=int,
                        help='Number of optional reviewers to collect in order to proceed')
    parser.add_argument('--timeout', default=0, type=int,
                        help='Timeout for waiting for approvals. Default is set to infinite.')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))

    args = parser.parse_args()
    axapproval = AXApproval(args.required_list,
                            args.optional_list,
                            args.number_optional,
                            args.timeout)
    axapproval.run()
