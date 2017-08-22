# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import sys
import requests
import time

from retrying import retry


def load_service_template(template_name, branch):

    @retry(wait_fixed=5000)
    def load_templates():
        print("Trying to load templates from axops")
        re = requests.get("http://axops-internal.axsys:8085/v1/templates")
        if re.status_code == 404:
            return None
        re.raise_for_status()
        return re.json()

    templates = load_templates()["data"]
    for template in templates:
        if template["name"] == template_name and template["branch"] == branch:
            return template

    return None


@retry(wait_fixed=5000)
def get_commit_info(commit):
    re = requests.get("http://axops-internal.axsys:8085/v1/commits/{}".format(commit))
    re.raise_for_status()
    return re.json()


def macro_replace(template, replace, replace_with):
    try:
        template["inputs"]["parameters"][replace]["default"] = replace_with
    except KeyError as e:
        print "Could not replace {} with {} due to {}".format(replace, replace_with, e)


class CreateSingleTest(object):
    """
    This class creates a single test using a service template and posting it
    /v1/services.
    """
    def __init__(self, name, template):
        self.name = name
        self.template = template
        self.task_id = None
        self.status = None

    @retry(wait_fixed=5000)
    def _start_test(self):
        re = requests.post("http://axops-internal.axsys:8085/v1/services", json=self.template)
        re.raise_for_status()
        return re.json()

    def start(self):
        response = self._start_test()
        self.task_id = response["task_id"]
        self.status = "ISSUED"
        return self.task_id

    def wait(self):
        while not self.done():
            time.sleep(10)

    def done(self):
        try:
            re = requests.get("http://axops-internal.axsys:8085/v1/services/{}".format(self.task_id))
            if re.status_code == 404:
                print "Task {} not found".format(self.task_id)
            if re.ok:
                status_code = re.json().get("status", -2)
                self.status = re.json().get("status_detail", {}).get("code", "UNKNOWN")
                if status_code == -1 or status_code == 0:
                    return True
        except Exception as e:
            print "Got exception {}".format(e)

        return False

    def get_status(self):
        return self.status


class TestManager(object):

    def __init__(self):
        self.tests = {}
        self.results = {}
        self.total = 0
        self.running = 0

    def add(self, name, test):
        self.tests[name] = test

    def start(self):
        for name, t in self.tests.iteritems():
            tid = t.start()
            self.results[name] = {
                "id": tid,
                "status": t.get_status()
            }
            self.total = len(self.tests)
            self.running = self.total

    def update_status(self):
        for name, t in self.tests.iteritems():
            if not t:
                continue
            done = t.done()
            self.results[name]["status"] = t.get_status()
            if done:
                self.tests[name] = None
                self.running -= 1

    def print_status(self):
        print "NAME, TASKID, STATUS"
        for name in sorted(self.results):
            tid = self.results[name]["id"]
            status = self.results[name]["status"]
            print "{}, {}, {}".format(name, tid, status)
        print "--- END ---"

    def done(self):
        self.update_status()
        if self.running == 0:
            return True

        return False


if __name__ == "__main__":

    if len(sys.argv) < 5:
        print "USAGE: {} template branch commit count".format(sys.argv[0])
        exit(1)

    template_name = sys.argv[1]
    branch = sys.argv[2]
    commit = sys.argv[3]
    count = int(sys.argv[4])

    template = load_service_template(template_name, branch)
    assert template, "Could not find template {} in branch {}".format(template_name, branch)

    commit_info = get_commit_info(commit)
    assert commit_info, "Could not find commit with id {}".format(commit)

    replacements = [
        ("commit", commit),
        ("repo", template["repo"])
    ]

    for (rep, rep_with) in replacements or []:
        macro_replace(template, rep, rep_with)

    final_template = {
        'template': template,
        'commit': commit_info
    }

    print("Final template is {}".format(final_template))

    manager = TestManager()
    for x in xrange(count):
        test = "quick-deployment-test-{}".format(x)
        t = CreateSingleTest(test, final_template)
        manager.add(test, t)

    manager.start()

    while not manager.done():
        manager.print_status()
        time.sleep(5)
    manager.print_status()








