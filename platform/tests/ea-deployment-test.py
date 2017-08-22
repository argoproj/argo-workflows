#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
import argparse
import requests
import os
import time
import sys
from retrying import retry


class DeploymentTest(object):

    def __init__(self):
        self.timeout = os.environ.get("TEST_TIMEOUT", 600)
        self.service_instance_id = None
        self.root_id = os.environ.get("AX_ROOT_SERVICE_INSTANCE_ID", None)
        print "Timeout is {} secounds".format(self.timeout)

        assert self.root_id is not None, "Root instance id not found"

        data = self.get_service_info()
        for child in data["children"] or []:
            if "labels" in child and "ax_ea_deployment" in child["labels"]:
                self.service_instance_id = child["id"]

        assert self.service_instance_id is not None, "Could not find the deployment step"

    @retry(wait_fixed=1000, stop_max_delay=60000)
    def get_service_info(self):
        re = requests.get("http://axops-internal.axsys:8085/v1/services/{}".format(self.root_id))
        if not re.ok:
            print "Error in requests {}".format(re.reason)
        re.raise_for_status()
        return re.json()

    def get_elb_of_sibling(self):

        data = self.get_service_info()
        try:
            for child in data["children"] or []:
                if "endpoint" in child:
                    return child["endpoint"]

            return None
        except KeyError as ke:
            print "KeyError in {}".format(ke)
            return None

    def test_elb(self, path=None):
        elb = None
        time_spent = 0
        while elb is None and time_spent < self.timeout:
            elb = self.get_elb_of_sibling()
            print "ELB is {} and service instance id is {}".format(elb, self.service_instance_id)
            if elb is None:
                time.sleep(5)
                time_spent += 5

        while time_spent < self.timeout:
            try:
                url = "http://" + elb
                if path is not None:
                    url += path
                re = requests.get(url)
                print "Got {} from {}".format(re.reason, url)
                if re.ok:
                    return True
            except Exception as e:
                print "Exception {}".format(e)
            time.sleep(5)
            time_spent += 5

        return False

    def test_route53(self, host):

        @retry(wait_fixed=5000, stop_max_delay=self.timeout)
        def ping_host():
            print "About to check host {}".format(host)
            re = requests.get("http://{}".format(host))
            print "Got {} from {}".format(re.reason, host)
            re.raise_for_status()

        try:
            ping_host()
            return True
        except Exception as e:
            print "Got exception {}".format(e)

        return False

    @retry(wait_exponential_multiplier=100,
           stop_max_attempt_number=10)
    def kill_deployment(self):

        url = "http://axworkflowadc.axsys:8911/v1/adc/workflows/{}".format(self.root_id)

        re = requests.delete(url)
        print "Delete for {} is {}".format(url, re.reason)
        if re.ok or re.status_code == 404:
            return
        else:
            re.raise_for_status()

if __name__ == "__main__":
    test = DeploymentTest()

    parser = argparse.ArgumentParser()
    parser.add_argument("--use-route53", help="Test nginx with route53 for provided hostname")

    parsed_args, unknown_args = parser.parse_known_args(sys.argv)

    if parsed_args.use_route53:
        ok = test.test_route53(parsed_args.use_route53)
    elif len(unknown_args) > 1:
        ok = test.test_elb(path=unknown_args[1])
    else:
        ok = test.test_elb()
    try:
        test.kill_deployment()
    except Exception as e:
        print "Got exception when trying to delete deployment {}".format(e)

    if not ok:
        print ("Test failed in {} seconds".format(test.timeout))
        sys.exit(1)
    sys.exit(0)
