#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
import botocore
import boto3
import json
import logging
import re
import uuid

from botocore.exceptions import ClientError

from ax.exceptions import AXConflictException, AXNotFoundException, AXIllegalArgumentException
from ax.platform.exceptions import AXPlatformException
from ax.platform.cluster_config import AXClusterConfig
from ax.util.validators import hostname_validator

from retrying import retry

logger = logging.getLogger(__name__)

subdomain_re = re.compile("[a-z0-9](?:[a-z0-9\-]{0,61}[a-z0-9])?")


def subdomain_name_check(subdomain):
    mobj = subdomain_re.match(subdomain)
    if not mobj:
        raise AXIllegalArgumentException("subdomain {} must match regex {}".format(subdomain, subdomain_re.pattern))

    if len(mobj.group(0)) != len(subdomain):
        raise AXIllegalArgumentException("subdomain {} must match regex {}".format(subdomain, subdomain_re.pattern))


def parse_boto_exception(func):
    """
    This decorator parses botocore ClientError exceptions
    and generates AX exceptions
    """
    def exception_handler(*args, **kwargs):
        try:
            return func(*args, **kwargs)
        except botocore.exceptions.ClientError as e:
            code = e.response["ResponseMetadata"]["HTTPStatusCode"]
            msg = e.message
            if code == 409:
                raise AXConflictException(msg)
            elif code == 404 or code == 400:
                # 400 also seems to be used for not found by AWS
                raise AXNotFoundException(msg)
            else:
                raise AXPlatformException(msg)

    return exception_handler


def retry_exponential(swallow=[], noretry=[], swallow_boto=[], noretry_boto=[]):
    """
    This decorator does exponential retry for boto exceptions using
    common sense retry values
    """
    def call_boto_with_retry(func):

        def raise_exception_unless(e):
            if isinstance(e, botocore.exceptions.ClientError):
                code = e.response["ResponseMetadata"]["HTTPStatusCode"]
                if code in noretry:
                    return False
                boto_code = e.response["Error"]["Code"]
                if boto_code in noretry_boto:
                    return False
            return True

        @parse_boto_exception
        @retry(retry_on_exception=raise_exception_unless,
               wait_exponential_multiplier=2000,
               stop_max_attempt_number=5)
        def wrapped_func(*args, **kwargs):
            try:
                return func(*args, **kwargs)
            except botocore.exceptions.ClientError as e:
                code = e.response["ResponseMetadata"]["HTTPStatusCode"]
                if code in swallow:
                    return None
                boto_code = e.response["Error"]["Code"]
                if boto_code in swallow_boto:
                    return None
                raise e

        return wrapped_func

    return call_boto_with_retry


class Route53(object):
    """
    This class is a wrapper for route53 calls
    """
    def __init__(self, client):
        """
        Args:
            client: Route53 boto3 client
        """
        self.client = client

    def list_hosted_zones(self):
        """
        Returns: an iterator of Route53HostedZone
        """
        @retry_exponential()
        def get_hosted_zones(marker=None):
            # we keep max items to 10 to have a limit on response from aws
            if marker:
                return self.client.list_hosted_zones(Marker=marker, MaxItems="10")
            else:
                return self.client.list_hosted_zones(MaxItems="10")

        marker = None
        while True:
            response = get_hosted_zones(marker=marker)
            zones = response["HostedZones"]

            # iter through zones and yield
            for zone in zones or []:
                r53_zone = Route53HostedZone(self, zone["Name"], zone)
                yield r53_zone

            if response["IsTruncated"]:
                marker = response["NextMarker"]
            else:
                break

        return

    def list_records(self, zone_id):
        """
        Returns an iterator for records
        Args:
            zone_id: The hosted zone id for which records are listed
        """

        @retry_exponential()
        def get_records(start_name=None, start_type=None):
            args_dict = {
                "MaxItems": "100",
                "HostedZoneId": zone_id,
            }
            if start_name:
                args_dict["StartRecordName"] = start_name
                args_dict["StartRecordType"] = start_type

            return self.client.list_resource_record_sets(**args_dict)

        sname = None
        stype = None
        while True:
            response = get_records(start_name=sname, start_type=stype)
            records = response["ResourceRecordSets"]
            for record in records:
                yield record

            if response["IsTruncated"]:
                sname = response["NextRecordName"]
                stype = response["NextRecordType"]
            else:
                break

    def get_hosted_zone(self, name):
        for zone in self.list_hosted_zones():
            if zone.name == name:
                return zone
        return None

    @retry_exponential(noretry=[409])
    def create_hosted_zone(self, zone_def_dict):
        return self.client.create_hosted_zone(**zone_def_dict)

    @retry_exponential()
    def delete_hosted_zone(self, zone_id):
        return self.client.delete_hosted_zone(Id=zone_id)

    @retry_exponential(swallow=[404, 400])
    def change_record(self, zone_id, action, record):
        return self.client.change_resource_record_sets(
            HostedZoneId=zone_id,
            ChangeBatch={
                "Changes": [
                    {
                        "Action": action,
                        "ResourceRecordSet": record
                    }
                ]
            }
        )


class Route53HostedZone(object):
    """
    This class creates and manages a route53 hosted zone
    """

    def __init__(self, route53, name, aws_zone=None):
        """
        Args:
            route53: Route53 object
            name: The name of the hosted zone
        """
        assert isinstance(route53, Route53), "Instance of client needs to be of type {}".format(type(Route53))
        self.client = route53

        if name.endswith("."):
            check_name = name[:-1]
            self.name = name
        else:
            check_name = name
            self.name = name + "."

        if not hostname_validator(check_name):
            raise AXIllegalArgumentException("Hosted Zone {} needs to be a valid domain name".format(name))

        self._id = None
        if aws_zone:
            self._id = aws_zone["Id"]

    def exists(self):
        """
        Returns: Boolean if the hosted zone exists
        """
        return True if self.client.get_hosted_zone(self.name) else False

    def create(self, **kwargs):
        """
        Create a hosted zone in route53. Only support public hosted zone for now
        Supports a bunch of optional kwargs:
         * annotations: A json dict
        """
        d = {
            "Name": self.name,
            "CallerReference": str(uuid.uuid4()),
            "HostedZoneConfig": {
                "PrivateZone": False
            }
        }
        annotations = kwargs.get("annotations", None)
        if annotations:
            d["HostedZoneConfig"]["Comment"] = json.dumps(annotations)

        try:
            resp = self.client.create_hosted_zone(d)
            self._id = resp["HostedZone"]["Id"]
        except AXConflictException:
            self._id = self.get_hosted_zone_id()

    def delete(self):
        """
        Deletes a zone if present.
        """
        try:
            self.get_hosted_zone_id()
            self.client.delete_hosted_zone(self._id)
        except AXNotFoundException:
            pass

    def create_subdomain(self, subdomain):
        """
        Creates a subdomain inside this hosted zone
        Returns: A Route53HostedZone for the subdomain
        """
        subdomain_name_check(subdomain)
        subzone = Route53HostedZone(self.client, "{}.{}".format(subdomain, self.name))
        subzone.create()
        self.create_ns_record(subzone.name, subzone.get_nameservers())

    def delete_subdomain(self, subdomain):
        subdomain_name_check(subdomain)
        subzone = Route53HostedZone(self.client, "{}.{}".format(subdomain, self.name))
        subzone.delete()
        self.delete_record(subzone.name, "NS")

    def list_records(self):
        self.get_hosted_zone_id()
        return self.client.list_records(self._id)

    def get_record(self, name, type):
        for record in self.list_records():
            if record['Name'] == name and record['Type'] == type:
                return record
        raise AXNotFoundException("Could not find record {} of type {} in hosted zone {}".format(name, type, self.name))

    def get_nameservers(self):
        for record in self.list_records():
            if record["Type"] != "NS":
                continue
            if record["Name"] != self.name:
                continue
            return [x["Value"] for x in record["ResourceRecords"]]

    def create_ns_record(self, name, nameservers):
        """"
        Create an NS record in this hosted zone
        Args:
            name: the name of the NS record
            nameservers: a list of nameservers
        """
        self.get_hosted_zone_id()
        record = {
            "Name": name,
            "Type": "NS",
            "TTL": 172800,
            "ResourceRecords": [{"Value": x} for x in nameservers or []]
        }
        self.client.change_record(self._id, "UPSERT", record)

    def create_alias_record(self, name, elb_addr, elb_name=None):
        """
        Create an alias record. This is idempotent as it uses UPSERT (create or update if exists)
        Args:
            name: The name of the alias. E.g. For zone test.acme.com if you want to create a alias alias1.test.acme.com
                  then name is alias1
            elb_addr: The ELB address to point to
            elb_hostedzoneid: the hosted zone ID of the ELB

        Returns:

        """
        name = name.lower()
        subdomain_name_check(name)

        @retry_exponential(noretry=[404])
        def get_load_balancer_info(elb_client, lb_name, elb_addr):
            for elb in elb_client.describe_load_balancers(LoadBalancerNames=[lb_name])['LoadBalancerDescriptions']:
                if elb['DNSName'] == elb_addr:
                    return elb['CanonicalHostedZoneNameID']
            return None

        region = AXClusterConfig().get_region()
        elb_client = boto3.Session().client("elb", region_name=region)
        if elb_name is None:
            # try to infer from elb_addr
            # http://docs.aws.amazon.com/elasticloadbalancing/2012-06-01/APIReference/API_CreateLoadBalancer.html
            # From the link above "This name must be unique within your set of load balancers for the region, must have
            # a maximum of 32 characters, must contain only alphanumeric characters or hyphens, and cannot begin or end with a hyphen."
            load_balancer_name = elb_addr[:32]
        else:
            load_balancer_name = elb_name

        elb_hostedzoneid = get_load_balancer_info(elb_client, load_balancer_name, elb_addr)
        if elb_hostedzoneid is None:
            raise AXNotFoundException("Could not find the ELB for the elb_addr {} in AWS".format(elb_addr))

        self.get_hosted_zone_id()
        record = {
            "Name": "{}.{}".format(name, self.name),
            "Type": "A",
            "AliasTarget": {
                'DNSName': elb_addr,
                'HostedZoneId': elb_hostedzoneid,
                'EvaluateTargetHealth': False
            }
        }
        self.client.change_record(self._id, "UPSERT", record)

    def delete_record(self, name, type):
        """
        Delete a record if it exists. Does not throw an exception if it does not exist
        Args:
            name: The name of the record
            type: The type of record (see record types in route53)

        """
        if not name.endswith("."):
            name = "{}.{}".format(name, self.name)
        self.get_hosted_zone_id()
        try:
            record = self.get_record(name, type)
            self.client.change_record(self._id, "DELETE", record)
        except AXNotFoundException as e:
            logger.debug("Could not delete due to exception {}".format(e))
            pass

    def get_hosted_zone_id(self):
        if not self._id:
            zone = self.client.get_hosted_zone(self.name)
            if not zone:
                raise AXNotFoundException("Hosted zone {} not found".format(self.name))
            self._id = zone.get_hosted_zone_id()

        return self._id

    def __str__(self):
        return "Name {}, Id {}".format(self.name, self._id)
