#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import argparse
import logging
import shlex
from subprocess import Popen
import sys

logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S")
logger = logging.getLogger("ax.volume_tests")
logger.setLevel(logging.DEBUG)

def get_cmd(component):
    if component == "axmon":
        suffix = "PlatformVolumeAPITests"
    elif component == "fixturemanager":
        suffix = "FixtureManagerVolumeAPITests"
    else:
        sys.exit("Unknown component " + component)

    cmd = 'python -m pytest -s -vv /src/platform/tests/volume_e2e/volume_tests.py::' + suffix
    logger.info("Running with cmd: %s", cmd)
    return cmd

def run():
    parser = argparse.ArgumentParser(description="Run volume API tests")
    parser.add_argument("--component", help="Name of the component (axmon/fixturemanager)")
    parser.add_argument("--total-runs", help="Total number of times to run the test")
    usr_args = parser.parse_args()

    cmd = get_cmd(usr_args.component)
    assert cmd is not None, "No command found!"
    num_processes = int(usr_args.total_runs)
    assert num_processes is not None and num_processes > 0

    logger.info("Running %d tests in parallel", num_processes)

    processes = []
    for i in range(num_processes):
        processes.append(Popen(shlex.split(cmd)))

    logger.info("Waiting for tests to complete...")
    for process in processes:
        output, error = process.communicate()
        logger.info("Output: %s", output)
        logger.info("Error: %s", error)

        if process.returncode != 0:
            raise Exception("Failed processs: " + str(process.returncode))

    logger.info("Volume tests have left the building!")

if __name__ == "__main__":
    run()

