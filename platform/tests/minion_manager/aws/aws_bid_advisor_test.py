"""The file has unit tests for the AWSBidAdvisor."""

import unittest

from ax.platform.minion_manager.cloud_provider.aws.aws_bid_advisor import AWSBidAdvisor

REFRESH_INTERVAL = 10
REGION = 'us-west-2'


class AWSBidAdvisorTest(unittest.TestCase):
    """
    Tests for AWSBidAdvisor.
    """
    def test_ba_lifecycle(self):
        """
        Tests that the AWSBidVisor starts threads and stops them correctly.
        """
        bidadv = AWSBidAdvisor(REFRESH_INTERVAL, REFRESH_INTERVAL, REGION)
        assert len(bidadv.all_bid_advisor_threads) == 0
        bidadv.run()
        assert len(bidadv.all_bid_advisor_threads) == 2
        bidadv.shutdown()
        assert len(bidadv.all_bid_advisor_threads) == 0

    def test_ba_on_demand_pricing(self):
        """
        Tests that the AWSBidVisor correctly gets the on-demand pricing.
        """
        bidadv = AWSBidAdvisor(REFRESH_INTERVAL, REFRESH_INTERVAL, REGION)
        assert len(bidadv.on_demand_price_dict) == 0
        updater = bidadv.OnDemandUpdater(bidadv)
        updater.get_on_demand_pricing()
        assert len(bidadv.on_demand_price_dict) > 0

    def test_ba_spot_pricing(self):
        """
        Tests that the AWSBidVisor correctly gets the spot instance pricing.
        """
        bidadv = AWSBidAdvisor(REFRESH_INTERVAL, REFRESH_INTERVAL, REGION)
        assert len(bidadv.spot_price_list) == 0
        updater = bidadv.SpotInstancePriceUpdater(bidadv)
        updater.get_spot_price_info()
        assert len(bidadv.spot_price_list) > 0

    def test_ba_price_update(self):
        """
        Tests that the AXBidVisor actually updates the pricing info.
        """
        bidadv = AWSBidAdvisor(REFRESH_INTERVAL, REFRESH_INTERVAL, REGION)
        od_updater = bidadv.OnDemandUpdater(bidadv)
        od_updater.get_on_demand_pricing()

        sp_updater = bidadv.SpotInstancePriceUpdater(bidadv)
        sp_updater.get_spot_price_info()

        # Verify that the pricing info was populated.
        assert len(bidadv.on_demand_price_dict) > 0
        assert len(bidadv.spot_price_list) > 0

        # Make the price dicts empty to check if they get updated.
        bidadv.on_demand_price_dict = {}
        bidadv.spot_price_list = {}

        od_updater.get_on_demand_pricing()
        sp_updater.get_spot_price_info()

        # Verify that the pricing info is populated again.
        assert len(bidadv.on_demand_price_dict) > 0
        assert len(bidadv.spot_price_list) > 0

    def test_ba_get_bid(self):
        """
        Tests that the bid_advisor's get_new_bid() method returns correct
        bid information.
        """
        bidadv = AWSBidAdvisor(REFRESH_INTERVAL, REFRESH_INTERVAL, REGION)

        instance_type = "m3.large"
        zone = "us-west-2b"
        # Manually populate the prices so that spot-instance prices are chosen.
        bidadv.on_demand_price_dict["m3.large"] = "100"
        bidadv.spot_price_list = [{'InstanceType': instance_type,
                                   'SpotPrice': '80',
                                   'AvailabilityZone': zone}]
        bid_info = bidadv.get_new_bid(zone, instance_type)
        assert bid_info is not None, "BidAdvisor didn't return any " + \
            "new bid information."
        assert bid_info["type"] == "spot"
        assert isinstance(bid_info["price"], str)

        # Manually populate the prices so that on-demand instances are chosen.
        bidadv.spot_price_list = [{'InstanceType': instance_type,
                                   'SpotPrice': '85',
                                   'AvailabilityZone': zone}]
        bid_info = bidadv.get_new_bid(zone, instance_type)
        assert bid_info is not None, "BidAdvisor didn't return any now " + \
            "bid information."
        assert bid_info["type"] == "on-demand"

    def test_ba_get_bid_no_data(self):
        """
        Tests that the BidAdvisor returns the default if the pricing
        information hasn't be obtained yet.
        """
        bidadv = AWSBidAdvisor(REFRESH_INTERVAL, REFRESH_INTERVAL, REGION)
        bid_info = bidadv.get_new_bid('us-west-2a', 'm3.large')
        assert bid_info["type"] == "on-demand"

    def test_ba_get_current_price(self):
        """
        Tests that the BidAdvisor returns the most recent price information.
        """
        bidadv = AWSBidAdvisor(REFRESH_INTERVAL, REFRESH_INTERVAL, REGION)

        od_updater = bidadv.OnDemandUpdater(bidadv)
        od_updater.get_on_demand_pricing()

        sp_updater = bidadv.SpotInstancePriceUpdater(bidadv)
        sp_updater.get_spot_price_info()

        # Verify that the pricing info was populated.
        assert len(bidadv.on_demand_price_dict) > 0
        assert len(bidadv.spot_price_list) > 0

        price_info_map = bidadv.get_current_price()
        assert price_info_map["spot"] is not None
        assert price_info_map["on-demand"] is not None
