#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import json
import os
import subprocess
import time

class KubectlWrapper(object):

    def get_service_ip(self, name):
        try:
            output = subprocess.check_output(["kubectl", "get", "services", "-o", "json"]).decode("utf-8")
            services = json.loads(output)
            items = services["items"]
            for item in items:
                if item["metadata"]["name"] == name:
                    return item["spec"]["clusterIP"]
        except subprocess.CalledProcessError:
            return None
        except json.JSONDecodeError:
            return None
        except KeyError:
            return None

    def run_command(self, filename, command):
        try:
            subprocess.check_output(["kubectl", command, "-f", filename]).decode("utf-8")
            return True
        except subprocess.CalledProcessError:
            return False

    def wait_for_service(self, name, sleep_for=10, timeout=120):
        ip = self.get_service_ip(name)
        if ip is not None:
            return True

        if timeout <= 0:
            return False
        time.sleep(sleep_for)
        return self.wait_for_service(name, sleep_for, timeout - sleep_for)

    def get_pod_status(self, name=None, selector=None):

        assert(name is not None or selector is not None and "Name or selector must be set")
        try:
            output = ""
            if name is not None:
                output = subprocess.check_output(["kubectl", "get", "pods", name, "-o", "json"]).decode("utf-8")
                pods = json.loads(output)
                return pods["status"]["phase"]
            else:
                output = subprocess.check_output(["kubectl", "get", "pods", "-l", selector, "-o", "json"]).decode("utf-8")
                pods = json.loads(output)
                return pods["items"][0]["status"]["phase"]
        except subprocess.CalledProcessError:
            return None
        except json.JSONDecodeError:
            return None
        except KeyError:
            return None

    def wait_for_pod_status(self, status, name=None, selector=None, sleep_for=10, timeout=120):
        s = self.get_pod_status(name=name, selector=selector)
        if s == status:
            return True
        if timeout <= 0:
            return False
        time.sleep(sleep_for)
        return self.wait_for_pod_status(status, name, selector, sleep_for, timeout - sleep_for)

    def run_one_time_command(self, image, name, command, *args):
        try:
            with open(os.devnull, "w") as devnull:
                subprocess.call(['kubectl', 'delete', 'pod', name], stdout=devnull, stderr=devnull)
            command = ["kubectl", "run", name, "--image={}".format(image), "--restart=Never", "--command", "--", command]
            command.extend(args)
            subprocess.check_output(command).decode("utf-8")
            if not self.wait_for_pod_status("Succeeded", name=name):
                return None
            output = subprocess.check_output(['kubectl', 'logs', name]).decode("utf-8")

            with open(os.devnull, "w") as devnull:
                subprocess.call(['kubectl', 'delete', 'pod', name], stdout=devnull, stderr=devnull)

            return output

        except subprocess.CalledProcessError:
            return None
