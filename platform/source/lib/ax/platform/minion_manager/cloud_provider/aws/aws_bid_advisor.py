#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""BidAdvisor implementation for AWS."""

import csv
from datetime import datetime, timedelta
import logging
import os
import threading
import time

from ax.util.const import SECONDS_PER_MINUTE
import boto3
import requests
from retrying import retry


logger = logging.getLogger("aws.minion-manager.bid-advisor")
logging.getLogger('requests').setLevel(logging.WARNING)

# Info about AWS Pricing API:
# https://aws.amazon.com/blogs/aws/new-aws-price-list-api/
# Info about reading the csv:
# http://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/reading-an-offer.html
AWS_PRICING_URL = 'https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/index.csv'
HOURLY_TERM_CODE = 'JRTCKXETXF'
RATE_CODE = '6YS6EN2CT7'
AWS_LINUX_VPC = 'Linux/UNIX (Amazon VPC)'

# The pricing API provided above only uses long region names and not the short
# one (like us-west-2). And, there
# doesn't seem to be any API that maps the short names to the long names.
# Therefore, we have to maintain this mapping ourselves.

AWS_REGIONS = {
    'ap-northeast-1': "Asia Pacific (Tokyo)",
    'ap-northeast-2': "Asia Pacific (Seoul)",
    'ap-southeast-1': "Asia Pacific (Singapore)",
    'ap-southeast-2': "Asia Pacific (Sydney)",
    'ca-central-1': "Canada (Central)",
    'ap-south-1': "Asia Pacific (Mumbai)",
    'eu-central-1': "EU (Frankfurt)",
    'eu-west-1': "EU (Ireland)",
    'eu-west-2': "EU (London)",
    'sa-east-1': "South America (Sao Paulo)",
    'us-east-1': "US East (N. Virginia)",
    'us-east-2': "US East (Ohio)",
    'us-west-1': "US West (N. California)",
    'us-west-2': "US West (Oregon)"
}

# This is returned in case the BidAdvisor is unable to compare the
# spot-instance price and on-demand price.
DEFAULT_BID = {"type": "on-demand"}


class AWSBidAdvisor(object):
    """
    The AWSBidAdvisor object is responsible for keeping track of instance
    prices for on-demand as well as spot instances. It exposes a method
    called get_bid() that returns the bid information. The AWSMinionManager
    object can then chose to use the bid information or not.
    """

    def __init__(self, on_demand_refresh_interval, spot_refresh_interval,
                 region):
        # This dictionary stores pricing information about on-demand instances
        # for all instance types.
        # E.g. {'d2.2xlarge': '1.3800000000', 'g2.8xlarge': '2.6000000000',
        # 'm3.large': '0.1330000000',...}
        self.on_demand_price_dict = {}

        # This list stores pricing information obtained from AWS. This
        # includes AZ, instance-type, price. Also, this list is sorted by time
        # and has guaranteed 1000 elements.
        # [{'Timestamp': datetime.datetime(2017, 1, 10, 21, 55, 29,
        # tzinfo=tzutc()), 'ProductDescription': 'Linux/UNIX', 'InstanceType':
        # 'c3.2xlarge', 'SpotPrice': '0.089100', 'AvailabilityZone':
        # 'us-west-2a'}, ...]
        self.spot_price_list = []

        self.ec2 = boto3.Session().client('ec2', region_name=region)

        # The interval at which the on-demand pricing information should be
        # refreshed. The on-demand pricing doesn't change often. It should be
        # fine to have this in the order of few hours.
        self.on_demand_refresh_interval = on_demand_refresh_interval

        # The interval at which the spot-pricing information should be
        # refreshed. This information can change frequently. This refresh
        # interval therefore should be in the order of few minutes.
        self.spot_refresh_interval = spot_refresh_interval

        self.region = region
        self.terminate_thread = False
        self.all_bid_advisor_threads = []

        self.lock = threading.Lock()

    class OnDemandUpdater(threading.Thread):
        """
        This thread periodically updates the on-demand instance pricing.
        """
        def __init__(self, bid_advisor):
            threading.Thread.__init__(self)
            assert bid_advisor, "BidAdvisor can't be None"
            self.bid_advisor = bid_advisor

        @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
        def get_on_demand_pricing(self):
            """ Issues the AWS api for getting on-demand pricing info. """
            region_full_name = AWS_REGIONS[self.bid_advisor.region]
            resp = requests.get(url=AWS_PRICING_URL, stream=True)
            line_iterator = resp.iter_lines()
            line = None
            for line in line_iterator:
                # Ignore lines till the PriceDescription is reached.
                if "PriceDescription" in line:
                    line = line.replace('"', '')
                    break
            assert line, "Failed while iteration over on-demand price info"
            reader = csv.DictReader(line_iterator, fieldnames=line.split(','))
            for row in reader:
                if HOURLY_TERM_CODE + "." + RATE_CODE in row["RateCode"] and \
                        "OnDemand" in row["TermType"] and \
                        region_full_name in row["Location"] and \
                        row["Operating System"] == "Linux" and \
                        row["Tenancy"] == "Shared":
                    with self.bid_advisor.lock:
                        self.bid_advisor.on_demand_price_dict[
                            row["Instance Type"]] = row["PricePerUnit"]
            logger.info("On-demand pricing info updated")

        def run(self):
            """ Main method of the OnDemandUpdater thread. """
            while self.bid_advisor.terminate_thread is False:
                sleep_required = True
                try:
                    self.get_on_demand_pricing()
                except Exception as ex:
                    sleep_required = False
                    raise Exception("Error while getting on-demand price " +
                                    "info: " + str(ex))
                finally:
                    if sleep_required:
                        time.sleep(self.bid_advisor.on_demand_refresh_interval)

    class SpotInstancePriceUpdater(threading.Thread):
        """
        This thread periodically updates the spot instance pricing.
        """
        def __init__(self, bid_advisor):
            threading.Thread.__init__(self)
            assert bid_advisor, "BidAdvisor can't be None"
            self.bid_advisor = bid_advisor

        @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
        def get_spot_price_info(self):
            """ Issues AWS apis to get spot instance prices. """
            ec2 = self.bid_advisor.ec2
            hour_ago = datetime.now() - timedelta(hours=1)
            spot_price_info = []
            next_token = ''
            while True:
                response = ec2.describe_spot_price_history(
                    ProductDescriptions=[AWS_LINUX_VPC],
                    StartTime=hour_ago, NextToken=next_token)
                if response is None:
                    raise Exception("Failed to get spot-instance prices")
                spot_price_info += response['SpotPriceHistory']
                if response['NextToken']:
                    next_token = response['NextToken']
                else:
                    break
            with self.bid_advisor.lock:
                self.bid_advisor.spot_price_list = spot_price_info
            logger.info("Spot instance pricing info updated")

        def run(self):
            """ Main method of the SpotInstancePriceUpdater thread. """
            while self.bid_advisor.terminate_thread is False:
                try:
                    self.get_spot_price_info()
                except Exception as ex:
                    raise Exception("Error while getting spot-instance " +
                                    "price info: " + str(ex))
                finally:
                    time.sleep(self.bid_advisor.spot_refresh_interval)

    def run(self):
        """ Main method of the AWSBidAdvisor. """
        if len(self.all_bid_advisor_threads) > 0:
            logger.debug("BidAdvisor already running!")
            return

        logger.info("Starting the BidAdvisor")

        # The on_demand_thread and spot_instance_thread are run in Daemon mode.
        # These threads will be run forever but shouldn't cause problems when
        # the minion-manager process is terminated.
        on_demand_thread = self.OnDemandUpdater(self)
        on_demand_thread.setDaemon(True)
        self.all_bid_advisor_threads.append(on_demand_thread)

        spot_instance_thread = self.SpotInstancePriceUpdater(self)
        spot_instance_thread.setDaemon(True)
        self.all_bid_advisor_threads.append(spot_instance_thread)

        on_demand_thread.start()
        spot_instance_thread.start()

        # Wait for the threads to get pricing information.
        while True:
            logger.info("Waiting for initial pricing information...")
            try:
                with self.lock:
                    if len(self.on_demand_price_dict) != 0 and \
                            len(self.spot_price_list) != 0:
                        return
            finally:
                time.sleep(SECONDS_PER_MINUTE)

    def basic_bid_strategy(self, spot_price, on_demand_price, bid_options):
        """
        Implements a very basic bid strategy. Checks if the spot instance price
        less than or equal to 80% of on-demand price. If so, selects the spot
        price. Otherwise chooses the on-demand price.

        If the spot instance price is closer to the on-demand price, on-demand
        instances are chosen for reliability reasons (on-demand instances won't
        be terminated with price hikes).

        TODO: BidStrategy should be it's own class. And basic_bid_strategy
        should be one implementation of that class. There could be other
        more interesting strategies too.

        :param spot_price: The price for spot-instances.
        :param on_demand_price: The price for on-demand instances.
        :param bid_options: Any options that the bidding strategy should need.
        :return bid_info: A dictionary with necessary bidding information.
        """
        bid_info = {}
        threshold = bid_options["spot_to_on_demand_threshold"]

        if spot_price <= threshold * on_demand_price:
            bid_info["price"] = str(on_demand_price)
            bid_info["type"] = "spot"
        else:
            # On-demand nodes do not require price information.
            bid_info["price"] = ""
            bid_info["type"] = "on-demand"
        return bid_info

    def get_current_price(self):
        """
        Returns the current price for on-demand and spot-instances.
        """
        price_map = {}
        with self.lock:
            price_map["on-demand"] = self.on_demand_price_dict
            price_map["spot"] = self.spot_price_list
        return price_map

    def get_on_demand_price(self, instance_type):
        """
        Returns the price for on-demand instances of the given type.
        """
        if instance_type in self.on_demand_price_dict.keys():
            return float(self.on_demand_price_dict[instance_type])

        return None

    def get_spot_instance_price(self, instance_type, zone):
        """
        Returns the spot-instance price for the given instance_type and zone.
        """
        # The spot price list is sorted by time. Find the latest instance
        # for the zone and instance_type and use that as the spot price.
        with self.lock:
            for price_info in self.spot_price_list:
                if price_info["InstanceType"] == instance_type and \
                        price_info["AvailabilityZone"] == zone:
                    return float(price_info["SpotPrice"])
        return None

    def get_new_bid(self, zone, instance_type):
        """
        Compare the last known spot-instance and on-demand instance prices and
        return a dict with the best possible bid options. If the pricing info.
        hasn't been collected yet, the default is to use on-demand instances.

        :param zone: The availability zone in which to check pricing.
        :param instance_type: The type of the EC2 instance.
        :return bid_info: A dictionary with necessary bidding information.
        """
        spot_instances_enabled = os.environ.get(
            "MM_SPOT_INSTANCE_ENABLED", "True").lower() == "true"
        if not spot_instances_enabled:
            logger.info("Spot instances disabled! Using DEFAULT_BID")
            return DEFAULT_BID

        with self.lock:
            if len(self.on_demand_price_dict) == 0 or \
                    len(self.spot_price_list) == 0:
                logger.info("Pricing data not available! Using DEFAULT_BID")
                return DEFAULT_BID

            spot_price = self.get_spot_instance_price(instance_type, zone)

            on_demand_price = self.get_on_demand_price(instance_type)

            if spot_price is None:
                logger.error("Spot price info not found. Using DEFAULT_BID")
                return DEFAULT_BID
            if on_demand_price is None:
                logger.error("On demand price info not found. " +
                             "Using DEFAULT_BID")
                return DEFAULT_BID

            logger.info("Using spot_instance price %f, on-demand price %f " +
                        "for instance type: %s, zone: %s",
                        spot_price, on_demand_price, instance_type, zone)

            bid_options = {"spot_to_on_demand_threshold": 0.8}
            return self.basic_bid_strategy(spot_price, on_demand_price,
                                           bid_options)

    def shutdown(self):
        """ Sets the flag to terminate all threads. """
        self.terminate_thread = True
        for thread in self.all_bid_advisor_threads:
            thread.join()

        del self.all_bid_advisor_threads[:]
        logger.info("BidAdvisor has left the building!")

