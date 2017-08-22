#!/usr/bin/python2.7

import base64
import argparse
import boto3
from bunch import bunchify
import time

parser = argparse.ArgumentParser()
parser.add_argument("--profile", help="Name of the AWS profile")
parser.add_argument("--asg", help="Name of the ASG")
parser.add_argument("--bid", help="Bid price (only for spot), Use 'on-demand'")
parser.add_argument("--lc", help="Name of the new launch config")
usr_args = parser.parse_args()

bid_info = {}
if usr_args.bid.lower() == "on-demand":
    bid_info["type"] = "on-demand"
else:
    bid_info["type"] = "spot"
    bid_info["price"] = usr_args.bid

if usr_args.profile:
    ac = boto3.Session(profile_name=usr_args.profile).client('autoscaling')
else:
    ac = boto3.Session(region_name="us-west-2").client('autoscaling')

def create_lc_with_spot(lc_name, lc, spot_price):
    response = ac.create_launch_configuration(
                LaunchConfigurationName=lc_name,
                ImageId=lc.ImageId,
                KeyName=lc.KeyName,
                SecurityGroups=lc.SecurityGroups,
                ClassicLinkVPCSecurityGroups=lc.ClassicLinkVPCSecurityGroups,
                UserData=base64.b64decode(lc.UserData),
                InstanceType=lc.InstanceType,
                BlockDeviceMappings=lc.BlockDeviceMappings,
                InstanceMonitoring=lc.InstanceMonitoring,
                SpotPrice=spot_price,
                IamInstanceProfile=lc.IamInstanceProfile,
                EbsOptimized=lc.EbsOptimized,
                AssociatePublicIpAddress=lc.AssociatePublicIpAddress)
    print response

def create_lc_on_demand(lc_name, lc):
    response = ac.create_launch_configuration(
                LaunchConfigurationName=lc_name,
                ImageId=lc.ImageId,
                KeyName=lc.KeyName,
                SecurityGroups=lc.SecurityGroups,
                ClassicLinkVPCSecurityGroups=lc.ClassicLinkVPCSecurityGroups,
                UserData=base64.b64decode(lc.UserData),
                InstanceType=lc.InstanceType,
                BlockDeviceMappings=lc.BlockDeviceMappings,
                InstanceMonitoring=lc.InstanceMonitoring,
                IamInstanceProfile=lc.IamInstanceProfile,
                EbsOptimized=lc.EbsOptimized,
                AssociatePublicIpAddress=lc.AssociatePublicIpAddress)
    print response

def update_lc(bid_info):
        asg_name = usr_args.asg
        resp = ac.describe_auto_scaling_groups(AutoScalingGroupNames=[asg_name])
        response = bunchify(resp)
        orig_lc_name = response.AutoScalingGroups[0].LaunchConfigurationName

        instance_ids = []
        for instance in response.AutoScalingGroups[0].Instances:
            instance_ids.append(instance.InstanceId)
        print "Current instances: " + str(instance_ids)

        response = ac.describe_launch_configurations(LaunchConfigurationNames=[orig_lc_name])
        assert len(response["LaunchConfigurations"]) == 1
        lc = bunchify(response).LaunchConfigurations[0]
        print lc["LaunchConfigurationName"]

        spot_price = bid_info["price"] if bid_info["type"] == "spot" else None
        lc_name = usr_args.lc
        if lc_name is None:
            if lc.LaunchConfigurationName[-2:] == "-0":
                lc_name = lc.LaunchConfigurationName[:-2]
            else:
                lc_name = lc.LaunchConfigurationName + "-0"
        if spot_price is None:
            create_lc_on_demand(lc_name, lc)
        else:
            create_lc_with_spot(lc_name, lc, spot_price)

        print "Created LC " + lc_name

        print "Updating ASG ..."
        response = ac.update_auto_scaling_group(AutoScalingGroupName=asg_name, LaunchConfigurationName=lc_name)
        print response


        print "Deleting old lc ..."
        response = ac.delete_launch_configuration(LaunchConfigurationName=orig_lc_name)
        print response

        print "Terminating current instances ..."
        for instance_id in instance_ids:
            ac.terminate_instance_in_auto_scaling_group(InstanceId=instance_id, ShouldDecrementDesiredCapacity=False)

        print "Done!!"
update_lc(bid_info)
