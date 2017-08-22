#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

""" Keeps track of the pricing info for each running instance in the ASGs. """

from collections import deque
from datetime import datetime
import logging
import sys
from threading import Thread, Lock
import time

from ax.util.const import SECONDS_PER_MINUTE
from bunch import bunchify
from flask import Flask, jsonify
from retrying import retry


logger = logging.getLogger("aws.minion-manager.price-reporter")
logging.getLogger('boto3').setLevel(logging.WARNING)
logging.getLogger('botocore').setLevel(logging.WARNING)
logging.getLogger('requests').setLevel(logging.WARNING)


class AWSPriceReporter(object):
    """
    This class keeps track of the pricing info. of each running AWS instance
    in the ASG.
    """
    def __init__(self, ec2_client, bid_advisor, minion_manager):
        # ec2_client is the boto ec2 client.
        self.ec2_client = ec2_client

        # bid_advisor is the AWSBidAdvisor object used for getting some price
        # info.
        self.bid_advisor = bid_advisor

        # The main minion-manager object. The asgs to operate on is obtained from
        # this object.
        self.minion_manager = minion_manager

        # The collector_thread is a Thread that periodically queries AWS and
        # updates the pricing info in memory.
        self.collector_thread = Thread(target=self.price_reporter_main,
                                       name="PriceReporter")
        self.collector_thread.setDaemon(True)

        # The api_thread is the Thread that responds to REST endpoints.
        self.api_thread = Thread(target=self.price_reporter_api,
                                 name="PriceReporterAPI")
        self.api_thread.setDaemon(True)

        # For protecting access to price_info.
        self.price_reporter_lock = Lock()

        # price_info is the dictionary about each instance and it's pricing
        # info.
        self.price_info = {}

    def get_price_info(self):
        """ Returns the price_info dict. """
        with self.price_reporter_lock:
            return self.price_info

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_instance_price(self, instance):
        """
        Given an Instance object, gets the price of that instance based on the
        InstanceType, AZ and StartTime.
        """
        current_time = datetime.now()
        if 'InstanceLifecycle' not in instance:
            on_demand_price = self.bid_advisor.get_on_demand_price(
                instance.InstanceType)
            return {str(current_time): str(on_demand_price)}

        query_time = current_time
        query_time = query_time.replace(minute=instance.LaunchTime.minute)
        query_time = query_time.replace(second=instance.LaunchTime.second)
        query_time = query_time.replace(
            microsecond=instance.LaunchTime.microsecond)
        if current_time.minute >= instance.LaunchTime.minute:
            query_time = query_time.replace(hour=current_time.hour)
        else:
            query_time = query_time.replace(hour=current_time.hour - 1)

        response = self.ec2_client.describe_spot_price_history(
            EndTime=query_time,
            InstanceTypes=[instance.InstanceType],
            ProductDescriptions=['Linux/UNIX (Amazon VPC)'],
            AvailabilityZone=instance.Placement.AvailabilityZone,
            StartTime=query_time
        )
        assert response is not None, "Failed to get spot-instance prices"
        resp = bunchify(response)
        if len(resp.SpotPriceHistory) > 0:
            return {str(query_time): resp.SpotPriceHistory[0].SpotPrice}
        else:
            return {str(query_time): "-1"}

    def price_reporter_work(self):
        """
        Performs one price check and updates the price_info.
        """
        for asg_meta in self.minion_manager.get_asg_metas():
            asg_instance_info = asg_meta.get_instance_info()
            for instance_id, instance in asg_instance_info.iteritems():
                price_data = self.get_instance_price(instance)
                with self.price_reporter_lock:
                    if instance_id in self.price_info:
                        self.price_info[instance.InstanceId].append(price_data)
                    else:
                        price_value_queue = deque(maxlen=24)
                        self.price_info[instance.InstanceId] = price_value_queue
                        price_value_queue.append(price_data)

    def price_reporter_main(self):
        """ Periodically updates the pricing info. """

        # Wait till at least one ASG has populated instances info.
        asg_info_populated = False
        while not asg_info_populated:
            try:
                for asg_meta in self.minion_manager.get_asg_metas():
                    asg_instance_info = asg_meta.get_instance_info()
                    if not asg_instance_info:
                        logger.info("Instance info not populated for %s!",
                                    asg_meta.get_name())
                    else:
                        asg_info_populated = True
                        # Break out as soon as any ASG has instance info populated.
                        break
            finally:
                # Wait before retrying ...
                time.sleep(30)

        logger.info("Collecting price information...")
        while True:
            try:
                self.price_reporter_work()
            except Exception as exc:
                # Log an error and swallow the exception.
                logger.error("Failed while getting instance pricing " +
                             "information: " + str(exc))
            finally:
                # Price check is done every hour.
                time.sleep(60 * SECONDS_PER_MINUTE)

    def price_reporter_api(self):
        """ Thread that responds to the Flask api endpoints. """
        app = Flask("AWSPriceReporterAPI")

        @app.route('/')
        def _return_price_info():
            """ Returns a json comprising the price-information. """
            try:
                output = {}
                with self.price_reporter_lock:
                    for instance, values in self.price_info.iteritems():
                        output[instance] = list(values)
                return jsonify(output)
            except Exception as exc:
                logger.info("Failed while reporting price info: " + str(exc))

        app.run(host='0.0.0.0')

    def run(self):
        """ Main method of the price-updater. """
        assert self.collector_thread is not None, \
            "PriceReporter thread not found"
        assert self.api_thread is not None, \
            "PriceReporterAPI thread not found"
        self.collector_thread.start()
        logger.info("PriceReporter thread started!")

        self.api_thread.start()
        logger.info("PriceReporterAPI thread started!")
        return
