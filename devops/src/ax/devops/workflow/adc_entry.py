#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2015 Applatix, Inc. All rights reserved.
#

"""
Main entry point for adc (AdmissionController).
"""

import argparse
import logging

from .adc_rest import adc_rest_start
from .adc_main import ADC, __version__

logger = logging.getLogger(__name__)


def main():
    """
    Main entry point for ADC (AdmissionController).
    """
    parser = argparse.ArgumentParser(description='ADC (AdmissionController)')
    parser.add_argument('--port', type=int, help="Run server on the specified port")
    parser.add_argument('--image-registry', help="ax_workflow_executor image registry")
    parser.add_argument('--image-namespace', help="ax_workflow_executor image namespace")
    parser.add_argument('--image-version', help="ax_workflow_executor image version")
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    args = parser.parse_args()

    # Basic logging.
    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s")
    logging.getLogger("ax").setLevel(logging.DEBUG)
    logging.getLogger("transitions").setLevel(logging.INFO)

    ADC.startup_prerequisite()

    ADC().set_param(wfe_registry=args.image_registry, wfe_namespace=args.image_namespace, wfe_version=args.image_version)
    adc_rest_start(port=args.port)
    ADC().run()
