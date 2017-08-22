#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
import boto3
import json
import logging
import re
import shutil
import subprocess
import tempfile

from threading import Lock
from future.utils import with_metaclass

from ax.util.singleton import Singleton
from ax.exceptions import AXNotFoundException, AXConflictException, AXIllegalArgumentException
from ax.meta import AXClusterId, AXClusterConfigPath
from ax.platform.routes import NodeRoute, ExternalRouteVisibility
from ax.platform.operations import Operation
from ax.platform.cluster_config import AXClusterConfig
from ax.devops.axdb.axdb_client import BaseAxdbClient
from ax.cloud.aws.aws_s3 import AXS3Bucket
from ax.platform.cloudprovider.aws.route53 import retry_exponential
from ax.util.hash import generate_hash

logger = logging.getLogger(__name__)

# ELB names can only by in 0-9a-zA-Z and hyphen but not start or end with hyphen
NAME_REGEX = re.compile("^[a-zA-Z0-9][-a-zA-Z0-9]{,30}[a-zA-Z0-9]$")


class NodePortManager(with_metaclass(Singleton, object)):

    def __init__(self, node_port_range=xrange(30000, 32767)):
        self._lock = Lock()
        self._node_port_range = node_port_range
        self._db = BaseAxdbClient()
        self._table = "/axsys/nodeport_table" # name defined in schema_platform/schema.go

        logger.debug("Waiting for nodeport management table to be created")
        self._db.retry_request('get', self._table, retry_on_exception=self._db.wait_for_table_exception)
        logger.debug("Table for nodeport management now exists")

    def get(self, elb_name, listener_port):
        """
        Given an application, deployment and listener_port get a nodeport
        if not allocated or return one that has already been allocated it.
        Returns: the allocated nodeport
        """
        logger.debug("Getting nodeport for {}/{}".format(elb_name, listener_port))
        params = {
            "elb_name": elb_name,
            "listener_port": listener_port
        }
        with self._lock:
            entries = self._db.retry_request('get', self._table, params=params, max_retry=5, retry_on_exception=self._db.get_retry_on_exception, value_only=True)
            if len(entries) == 0:
                new_node_port = self._get_next_available()
                params["node_port"] = new_node_port
                self._db.retry_request('post', self._table, data=params, max_retry=5, retry_on_exception=self._db.create_retry_on_exception, value_only=True)
                return new_node_port
            else:
                assert len(entries) == 1, "Found more than one entry for {}/{} - {}".format(elb_name, listener_port, entries)
                return entries[0]["node_port"]

    def add_elb_addr(self, elb_name, elb_addr):
        logger.debug("Adding address {} to elb {}".format(elb_addr, elb_name))
        with self._lock:
            params = {
                "elb_name": elb_name
            }
            elb_entries = self._db.retry_request('get', self._table, params=params, max_retry=5, retry_on_exception=self._db.get_retry_on_exception, value_only=True)
            if len(elb_entries) < 1:
                raise AXNotFoundException("Did not find an entry for {} in node port table".format(elb_name))
            for elb in elb_entries:
                elb["elb_addr"] = elb_addr
                self._db.retry_request('put', self._table, data=elb, max_retry=5, retry_on_exception=self._db.create_retry_on_exception, value_only=True)

    def get_elb_addr(self, elb_name):
        logger.debug("Getting address {}".format(elb_name))
        with self._lock:
            params = {
                "elb_name": elb_name
            }
            elb_entries = self._db.retry_request('get', self._table, params=params, max_retry=5, retry_on_exception=self._db.get_retry_on_exception, value_only=True)
            for elb in elb_entries or []:
                return elb["elb_addr"]

        raise AXNotFoundException("Did not find an elb address for elb {}".format(elb_name))

    def release(self, elb_name, listener_port):
        """
        Release the nodeport associated with this combination
        """
        logger.debug("Releasing nodeport for {}/{}".format(elb_name, listener_port))
        with self._lock:
            params = {
                "elb_name": elb_name,
                "listener_port": listener_port
            }
            self._db.retry_request('delete', self._table, data=[params], max_retry=5, retry_on_exception=self._db.delete_retry_on_exception, value_only=True)

    def release_all(self, elb_name):
        logger.debug("Releasing nodeports for {}".format(elb_name))
        with self._lock:
            params = {
                "elb_name": elb_name,
            }
            self._db.retry_request('delete', self._table, data=[params], max_retry=5,
                                   retry_on_exception=self._db.delete_retry_on_exception, value_only=True)
    def _get_next_available(self):
        """
        """
        table = self._db.retry_request('get', self._table, max_retry=5, retry_on_exception=self._db.get_retry_on_exception, value_only=True)
        node_ports = sorted([x["node_port"] for x in table])

        for x in self._node_port_range:
            if x not in node_ports:
                return x
        return None


AWS_ELB_TEMPLATE = """
variable "region" {}

# Use the aws provider. All credentials are picked up from the instance role but the region needs to be specified
provider "aws" {
      region = "${var.region}"
}

# Use the S3 backend for storing state. The configuration for state of each service will be managed by the ManagedElb Class
terraform {
    backend "s3" {}
}

variable "application" {}

variable "deployment" {}

variable "cluster_id" {
    description = "The cluster name_id"
}

variable "elb_prefix" {
    description = "Some unique string prefix for the elb"
}

variable "elb_internal" {
    description = "Internal ELB (true/false)"
    default = "true"
}

variable "ports" {
    type = "list"
    description = "List of ports"
}

variable "to_ports" {
    type = "list"
    description = "List of ports that are listened to on minions (one for each to port)"
    default = []
}

variable "protocols" {
    type = "list"
    description = "List of protocols"
}

variable "cidrs" {
    type = "list"
    description = "List of cidr blocks that is applied to each port"
}

variable "asg_name" {
    description = "Name of ASG to add this ELB to"
}

# STEP 1: Get the subnet for the cluster
data "aws_subnet" "cluster_subnet" {
    tags = {
        KubernetesCluster = "${var.cluster_id}"
    }
}

# STEP 2: Get the kubernetes minion security group
data "aws_security_group" "minion" {
    name = "kubernetes-minion-${var.cluster_id}"
}

# STEP 3: Create security group for elb
resource "aws_security_group" "elb_sg" {
    name        = "${var.elb_prefix}"
    description = "Security settings for the ELB"

    vpc_id      = "${data.aws_subnet.cluster_subnet.vpc_id}"

    # outbound internet access
    egress {
        from_port   = 0
        to_port     = 0
        protocol    = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }

    tags {
        KubernetesCluster   = "${var.cluster_id}"
        ManagedResource     = "elb.${var.elb_prefix}"
    }
}


# STEP 4: Loop through and create security rules for all CIDR ranges
resource "aws_security_group_rule" "add_rule_to_sg" {
    count                           = "${length(var.ports)}"
    type                            = "ingress"
    from_port                       = "${element(var.ports, count.index)}"
    to_port                         = "${element(var.ports, count.index)}"
    protocol                        = "tcp"
    cidr_blocks                     = "${var.cidrs}"
    security_group_id               = "${aws_security_group.elb_sg.id}"
}


# STEP 5: Add security group to kubernetes minion
resource "aws_security_group_rule" "add_elb_to_minion" {
    type                            = "ingress"
    from_port                       = 0
    to_port                         = 0
    protocol                        = "-1"
    source_security_group_id        = "${aws_security_group.elb_sg.id}"
    security_group_id               = "${data.aws_security_group.minion.id}"
}

# STEP 6: Create the ELB...finally
resource "aws_elb" "elb" {

    name = "${var.elb_prefix}"
    subnets = ["${data.aws_subnet.cluster_subnet.id}"]
    internal = "${var.elb_internal}"
    security_groups = ["${aws_security_group.elb_sg.id}"]

    # AXMODIFY-LISTENERS
    
    # AXMODIFY-HEALTHCHECK
    
    cross_zone_load_balancing   = true
    idle_timeout                = 400
    connection_draining         = true
    connection_draining_timeout = 400

    tags {
        KubernetesCluster   = "${var.cluster_id}"
        ManagedResource     = "elb.${var.elb_prefix}"
        Application         = "${var.application}"
        Deployment          = "${var.deployment}"
        Version             = "1.0"
        InternalElb         = "${var.elb_internal}"
    }
}

# Add the ASG to ELB so that instances automatically created are added to ELB
resource "aws_autoscaling_attachment" "elb_asg" {
    autoscaling_group_name = "${var.asg_name}"
    elb                    = "${aws_elb.elb.id}"
    depends_on             = ["aws_elb.elb"]
}
"""

LISTENER_TEMPLATE = """
    listener {{
        instance_port = {}
        instance_protocol = "{}"
        lb_port = {}
        lb_protocol = "{}"
        {}
    }}

"""

HEALTHCHECK_TEMPLATE = """
    health_check {{
        healthy_threshold = 2
        unhealthy_threshold = 2
        timeout = 3
        interval = 30
        target = "TCP:{}"
    }}
"""


class ManagedElbNodeOperation(Operation):

    def __init__(self, route):
        token = "{}/{}".format(route.namespace, route.name)
        super(ManagedElbNodeOperation, self).__init__(token=token)

    @staticmethod
    def prettyname():
        return "ManagedElbNodeOperation"


class ManagedElbVars(with_metaclass(Singleton, object)):

    def __init__(self):
        self.name_id = AXClusterId().get_cluster_name_id()
        paths = AXClusterConfigPath(name_id=self.name_id)
        self.bucket = paths.bucket()
        self.terraform_dir = paths.terraform_dir()
        self.region = AXClusterConfig().get_region()
        self.placement = AXClusterConfig().get_zone()
        self.trusted_cidrs = AXClusterConfig().get_trusted_cidr()
        self.s3 = AXS3Bucket(bucket_name=self.bucket)

    def get_vars(self):
        return self.name_id, self.bucket, self.terraform_dir, self.region, self.placement, self.trusted_cidrs, self.s3


class ManagedElb(object):
    """
    TODO: This needs to be an AXResource that gets added to the deployment for non axsys deployments
    """
    protocol_map = {
        "http": "http",
        "https": "http",
        "tcp": "tcp",
        "ssl": "tcp"
    }

    def __init__(self, name, boto_client=None):
        if not NAME_REGEX.match(name):
            raise AXIllegalArgumentException("Name can only contain a-zA-Z0-9 and hyphen and not start or end with hyphen. Max length is 32 characters")

        self.name = name
        self._npm = NodePortManager()
        self._vars = ManagedElbVars()
        (self.name_id, self.bucket, self.terraform_dir, self.region, self.placement, self.trusted_cidrs, self.s3) = self._vars.get_vars()
        if boto_client is None:
            self._boto = boto3.client("elb", region_name=self.region)
        else:
            self._boto = boto_client

    def create(self, application, deployment, ports_spec, internal=True, labels=None):

        self._creation_prechecks(application, deployment)

        # Create the port spec for the NodeRoute and get node ports for each of the listener ports and generate
        # the listener spec and the health check spec
        listener_spec = ""
        health_check_spec = None
        i = 0
        srv_pspec = []
        listen_ports = []
        protocols = []
        for port in ports_spec:
            # Build the port spec for NodeRoute
            curr_port = {}
            curr_port["name"] = "port-{}".format(i)
            curr_port["port"] = port["listen_port"]
            curr_port["target_port"] = port["container_port"]
            curr_port["node_port"] = self._npm.get(self.name, port["listen_port"])
            srv_pspec.append(curr_port)

            # Build array of listen ports and protocol for security rules
            listen_ports.append(port["listen_port"])
            protocols.append(port["protocol"])

            # Build listener spec for ELB
            ssl_cert = ""
            if "https" in port["protocol"] or "ssl" in port["protocol"]:
                ssl_cert = "ssl_certificate_id = \"{}\"".format(port["certificate"])
            listener_spec += LISTENER_TEMPLATE.format(curr_port["node_port"], self.protocol_map[port["protocol"]], port["listen_port"], port["protocol"], ssl_cert)

            # Build health check spec for ELB
            if health_check_spec is None:
                health_check_spec = HEALTHCHECK_TEMPLATE.format(curr_port["node_port"])

            i += 1

        # create the nodeport k8s service
        # Note: We generate a random string [a-z] rather than using the elb name (even though elb name needs to be unique
        #       w.r.t other elb names, because we need to generate a kubernetes service object. It is possible that the
        #       user has generated an internal route with that name and we do not want to clobber the internal route
        nr_name = self.name
        noderoute = NodeRoute(nr_name, application)

        # Get a lock for operations of exist() delete() and create() as the InternalRouteOperation will not be
        # atomic across all these operations.
        with ManagedElbNodeOperation(noderoute):
            if noderoute.exists():
                noderoute.delete()
            noderoute.create(srv_pspec, selector=labels)

        # modify the elb template using the pass port specification
        my_elb_template = AWS_ELB_TEMPLATE.replace("# AXMODIFY-LISTENERS", listener_spec)
        my_elb_template = my_elb_template.replace("# AXMODIFY-HEALTHCHECK", health_check_spec)

        # Create cloud specific resources using terraform
        init_command = 'terraform init -backend=true -backend-config="bucket={}" -backend-config="key={}managed_elbs/{}.tfstate" -backend-config="region={}"  -reconfigure'.format(
            self.bucket, self.terraform_dir, self.name, self.region
        )

        terraform_command = "terraform apply -var 'asg_name={}' -var 'cluster_id={}' -var 'cidrs={}' -var 'elb_prefix={}' -var 'ports={}' -var 'protocols={}' -var 'region={}' -var 'application={}' -var 'deployment={}' -var 'elb_internal={}'".format(
            self._get_asg_name(application), self.name_id, json.dumps(self.trusted_cidrs), self.name, json.dumps(listen_ports), json.dumps(protocols), self.region, application, deployment, (str(internal)).lower()
        )

        logger.debug("Terraform spec\n{}".format(my_elb_template))
        logger.debug("Terraform init command is: {}".format(init_command))
        logger.debug("Terraform command is: {}".format(terraform_command))

        tempdir = tempfile.mkdtemp()
        with open("{}/elb.tf".format(tempdir), "w") as f:
            f.write(my_elb_template)

        # also write the file to S3
        self.s3.put_object("{}managed_elbs/{}.tf".format(self.terraform_dir, self.name), my_elb_template)

        logger.debug("Terraform init\n{}".format(subprocess.check_output(init_command, shell=True, cwd=tempdir)))
        logger.debug("Terraform apply\n{}".format(subprocess.check_output(terraform_command, shell=True, cwd=tempdir)))
        shutil.rmtree(tempdir, ignore_errors=True)

        elb = self._get_load_balancer_info()
        if elb is None:
            raise AXNotFoundException(
                "Created ELB {} was not found. That is odd. Please try creating again".format(self.name))
        dnsname = elb["DNSName"]
        self._npm.add_elb_addr(self.name, dnsname)

        return dnsname

    def delete(self, aws_resources_only=False):

        (application, deployment) = self._get_app_and_dep()
        if application is None or deployment is None:
            raise AXNotFoundException("Could not find application and deployment for deleting managed elb {}".format(self.name))


        if not aws_resources_only:
            # Start by deleting NodeRoute Objects first
            logger.debug("Deleting node routes for the managed elb {}".format(self.name))
            nr_name = self.name
            noderoute = NodeRoute(nr_name, application)
            with ManagedElbNodeOperation(noderoute):
                if noderoute.exists():
                    noderoute.delete()

            # Release all the nodeport after the NodeRoutes are gone
            logger.debug("Releasing nodeports for the managed elb {}".format(self.name))
            self._npm.release_all(self.name)

        # Now unterraform to get rid of the ELB
        init_command = 'terraform init -backend=true -backend-config="bucket={}" -backend-config="key={}managed_elbs/{}.tfstate" -backend-config="region={}"  -reconfigure'.format(
            self.bucket, self.terraform_dir, self.name, self.region
        )

        terraform_command = "terraform destroy --force -var 'asg_name={}' -var 'cluster_id={}' -var 'cidrs=[]' -var 'elb_prefix={}' -var 'ports=[]' -var 'protocols=[]' -var 'region={}' -var 'application={}' -var 'deployment={}'".format(
            self._get_asg_name(application), self.name_id, self.name, self.region, application, deployment
        )

        tempdir = tempfile.mkdtemp()
        self.s3.download_file("{}managed_elbs/{}.tf".format(self.terraform_dir, self.name), "{}/elb.tf".format(tempdir))
        logger.debug("Terraform init\n{}".format(subprocess.check_output(init_command, shell=True, cwd=tempdir)))
        logger.debug("Terraform destroy\n{}".format(subprocess.check_output(terraform_command, shell=True, cwd=tempdir)))
        shutil.rmtree(tempdir, ignore_errors=True)

        # delete the terraform state and config file
        self.s3.delete_object("{}managed_elbs/{}.tf".format(self.terraform_dir, self.name))
        self.s3.delete_object("{}managed_elbs/{}.tfstate".format(self.terraform_dir, self.name))

    def exists(self):
        return True if self._get_load_balancer_info() is not None else False

    def _creation_prechecks(self, application, deployment):
        (app, dep) = self._get_app_and_dep()
        if app is None and dep is None:
            return
        if app != application or dep != deployment:
            raise AXConflictException("Managed ELB {} already exists. Cannot recreate it".format(self.name))

    def _get_app_and_dep(self):
        app = None
        dep = None
        elb = self._get_load_balancer_info()

        if elb:
            for tag in elb["Tags"] or []:
                if tag["Key"] == "Application":
                    app = tag["Value"]
                if tag["Key"] == "Deployment":
                    dep = tag["Value"]

        return (app, dep)

    @retry_exponential(swallow_boto=['LoadBalancerNotFound'])
    def _get_load_balancer_info(self):
        elbtags = self._boto.describe_tags(LoadBalancerNames=[self.name])['TagDescriptions'][0]
        elbinfo = self._boto.describe_load_balancers(LoadBalancerNames=[self.name])['LoadBalancerDescriptions'][0]
        elbinfo.update(elbtags)
        return elbinfo

    def _get_asg_name(self, application):
        if application == "axsys":
            # For system elbs point to axsys nodes
            # format of this is manually generated in the following form
            # NAMEID-minion-ax-PLACEMENT
            return "{}-minion-ax-{}".format(self.name_id, self.placement)
        else:
            # For application elbs point to user variable nodes
            # TODO: What if user-variable is 0 but on-demand is not. Currently that is not possible but in future it might be
            # format is NAMEID-minion-user-variable
            return "{}-minion-user-variable".format(self.name_id)


    def _generate_node_route_name(self, application, deployment):
        return generate_hash("NRV1-{}-{}-{}".format(application, deployment, self.name))

    @staticmethod
    def list_all(boto_client=None):
        import os
        ret = []
        cluster_vars = ManagedElbVars()
        for objs in cluster_vars.s3.list_objects_by_prefix("{}managed_elbs/".format(cluster_vars.terraform_dir)) or []:
            if objs.key.endswith(".tf"):
                elb_name = os.path.splitext(os.path.basename(objs.key))[0]
                m = ManagedElb(elb_name, boto_client=boto_client)
                if m.exists():
                    ret.append(elb_name)

        return ret

    @staticmethod
    def get_elb_name(name_id, prefix):
        return generate_hash("{}-{}".format(name_id, prefix))[:32]


# Global function to delete managed elbs - This function is called during cluster destroy
def delete_managed_elbs(aws_profile, region):
    boto_client = boto3.Session(profile_name=aws_profile).client("elb", region_name=region)
    for elb_name in ManagedElb.list_all(boto_client=boto_client):
            elb = ManagedElb(elb_name)
            try:
                elb.delete(aws_resources_only=True)
            except Exception as e:
                logger.warn("Could not delete Managed ELB {} due to exception {}".format(elb_name, e))


# global function to map visibility to elb addr
def visibility_to_elb_addr(visibility):
    elb_name = visibility_to_elb_name(visibility)
    npm = NodePortManager()
    return npm.get_elb_addr(elb_name)


# global function to map visibility to elb name
def visibility_to_elb_name(visibility):
    name_id = AXClusterId().get_cluster_name_id()
    if visibility == ExternalRouteVisibility.VISIBILITY_ORGANIZATION:
        elb_name = ManagedElb.get_elb_name(name_id, "ing-pri")
    else:
        elb_name = ManagedElb.get_elb_name(name_id, "ing-pub")
    return elb_name

