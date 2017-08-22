# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import logging
import requests

from retrying import retry
from uuid import UUID
from ax.kubernetes.client import KubernetesApiClient, parse_kubernetes_exception
from ax.kubernetes import swagger_client

logger = logging.getLogger(__name__)


def job_complete(ref):
    """
    Helper function that checks if the job with this ref/name has been completed
    Args:
        ref: Either a service instance id or a jobname

    Returns: true if job is complete, false for all other conditions
    """

    @retry(wait_exponential_multiplier=100, stop_max_attempt_number=10)
    def query_from_adc(workflow_id):
        url = "http://axworkflowadc.axsys:8911/v1/adc/workflows/{}?state_only=true".format(workflow_id)
        re = requests.get(url)
        # don't raise error and retry if not found. ADC returns 400 instead of 404
        if re.status_code == requests.codes.bad_request:
            return re
        re.raise_for_status()
        return re

    @parse_kubernetes_exception
    @retry(wait_exponential_multiplier=100, stop_max_attempt_number=10)
    def query_from_kubernetes(task_name):
        client = KubernetesApiClient(use_proxy=True)
        try:
            job = client.batchv.read_namespaced_job_status("axuser", task_name)
            assert isinstance(job, swagger_client.V1Job), "Expect to see an object of type V1Job"
            return job
        except swagger_client.rest.ApiException as e:
            if e.status == 404:
                return None
        return None

    ref_is_service_instance_id = False
    try:
        UUID(ref)
        ref_is_service_instance_id = True
    except ValueError:
        pass

    if ref_is_service_instance_id:
        response = query_from_adc(ref)
        if response.status_code == requests.codes.ok:
            data = response.json()
            logger.debug("Ref {} has the following status in ADC {}".format(ref, data))
            s = frozenset(["INTERRUPTED_STATE", "SUCCEED_STATE", "FAILED_STATE"])
            if data['state'] in s:
                return True

        return False
    else:
        try:
            if query_from_kubernetes(ref) is None:
                return True
        except Exception as e:
            logger.debug("Task {} check resulted in exception {}".format(ref, e))

    return False
