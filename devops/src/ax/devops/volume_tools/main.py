#!/usr/bin/env python

import argparse
import json
import logging
import sys
import time
import uuid

from ax.version import __version__
import boto3
import requests
from retrying import retry


logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s", stream=sys.stdout)
logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)
AXOPS_HOST = "http://axops-internal.axsys:8085"

def get_boto_ec2_client(region):
    return boto3.Session(region_name=region).client('ec2')

def get_volume_metadata(vol_name):
    """ Get's metadata of the volume with the given name. """
    resp = requests.get(AXOPS_HOST + '/v1/storage/volumes')
    assert resp.status_code == 200, "Failed to get current volumes: {}".format(resp)
    data = resp.json()['data']
    for volume in data:
        if volume['name'] == vol_name:
            return volume

    return None

def get_volume_id(vol_name):
    """ Checks if the given volume exists. Only checks by name. """
    volume_metadata = get_volume_metadata(vol_name)
    if volume_metadata:
        logger.info("%s exists", vol_name)
        return volume_metadata['id']

    return None


def create_if_needed_helper(vol_name, vol_size, storage_class, resource_id=None):
    logger.info("Creating volume with name: %s", vol_name)

    # Actually create the volume!
    body = {
        'name': vol_name,
        'owner': 'system',
        'creator': 'system',
        'resource_id': resource_id,
        'storage_class': storage_class,
        'attributes': {
            'size_gb': vol_size
        }
    }
    body = json.dumps(body)

    headers = {
        'Content-Type': 'application/json',
        'Content-Length': str(len(body))
    }

    resp = requests.post(AXOPS_HOST + '/v1/storage/volumes', data=body, headers=headers)
    try:
        assert resp.status_code == 200, "Failed to create volume: {}".format(resp)
        logger.info("Successfully created volume %s of size %s", vol_name, vol_size)
    except Exception as e:
        if resp.status_code == 400 and get_volume_id(vol_name) is not None:
            logger.info("Volume %s already exists", vol_name)
            return
        logger.error(e)
        raise e

    # Wait for the volume to be created.
    @retry(wait_exponential_multiplier=3000, stop_max_attempt_number=5)
    def wait_for_active():
        volume_active = False
        metadata = get_volume_metadata(vol_name)
        logger.info("Waiting for volume %s to be \"active\". Current status: %s", vol_name, metadata["status"])
        volume_active = metadata["status"] == "active"
        if not volume_active:
            raise Exception("Not active yet!")
    wait_for_active()
    logger.info("Volume %s is ready to use", vol_name)

def create_if_needed(args):
    """ Creates the volume if it doesn't exist. """
    create_if_needed_helper(args.volume_name, args.volume_size_gb, args.storage_class)

def delete_volume(args):
    """ Deletes the volume. """
    volume_id = get_volume_id(args.volume_name)
    if volume_id is None:
        logger.info("Volume %s not found. Already deleted?", args.volume_name)
        return

    resp = requests.delete(AXOPS_HOST + '/v1/storage/volumes/{}'.format(volume_id))
    assert resp.status_code < 300, "Failed to delete volume: {}".format(resp)
    logger.info("Successfully deleted volume %s", args.volume_name)

@retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
def add_tags_to_resource(ec2, resource_id, tags):
    """ Wrapper around EC2 create_tags. With retries. """
    return ec2.create_tags(Resources=[resource_id], Tags=tags)

@retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
def get_volume_info(ec2, volume_resource_id):
    """ Wrapper around EC2 describe_volumes. With retries. """
    response = ec2.describe_volumes(VolumeIds=[volume_resource_id])
    return response

def create_volume_snapshot(args):
    """ Creates snapshot of the given volume. """
    if args.volume_name is None and args.resource_id is None:
        logger.error("Either volume name or AWS resource id is needed")
        return

    volume_resource_id = None
    if args.volume_name:
        volume = get_volume_metadata(args.volume_name)
        volume_resource_id = volume['resource_id']
    else:
        assert args.resource_id is not None
        volume_resource_id = args.resource_id

    assert volume_resource_id is not None, "No resource_id found! Did you specify the correct volume-name or resource-id?"
    logger.info("Creating snapshot of volume %s", volume_resource_id)
    ec2 = get_boto_ec2_client(args.region)

    response = get_volume_info(ec2, volume_resource_id)
    assert 'Volumes' in response and len(response['Volumes']) > 0, "Volume not found: " + str(response)
    volume_tags = response['Volumes'][0]['Tags']
    logger.info("Found tags of source volume: %s", volume_tags)

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
    def create_ebs_volume_snapshot():
        response = ec2.create_snapshot(VolumeId=volume_resource_id,
                Description=volume_resource_id)
        return response

    response = create_ebs_volume_snapshot()
    assert response['SnapshotId'] is not None, "Failed to create snapshot. SnapshotId is none!"

    snapshot_id = response['SnapshotId']
    logger.info("Created snapshot in AWS with id: %s", snapshot_id)

    response = add_tags_to_resource(ec2, snapshot_id, volume_tags)
    assert response is not None, "Failed to create tags for snapshot: " + snapshot_id
    logger.info("Created tags for snapshot: %s", snapshot_id)
    return snapshot_id

def create_volume_from_snapshot(snapshot_id, region):
    """Creates an EBS volume from the given snapshot_id."""
    ec2 = get_boto_ec2_client(region)

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
    def create_ebs_volume_from_snapshot():
        response = ec2.create_volume(SnapshotId=snapshot_id,
            AvailabilityZone=get_availability_zone(), VolumeType='gp2')
        return response

    def get_availability_zone():
        default_zone = "us-west-2a"
        try:
            data = requests.get("http://169.254.169.254/latest/meta-data/placement/availability-zone", timeout=5)
            return data.text
        except requests.exceptions.ConnectTimeout as ce:
            logger.info("Failed to get metadata. Using AZ %s", default_zone)
            return default_zone
        except Exception as e:
            logger.error("Failed to get availability zone: %s", e)
            sys.exit(1)

    response = create_ebs_volume_from_snapshot()
    assert response is not None, "Failed to create volume using snapshot id: " + snapshot_id
    logger.info("Created volume with id %s", response['VolumeId'])

    return response['VolumeId']

def restore_snapshot_to_volume(args):
    """ Creates a new EBS volume based on a snapshot. """
    logger.info("Creating volume named %s from snapshot: %s", args.volume_name, args.snapshot_id)
    volume_id = get_volume_id(args.volume_name)
    if volume_id is not None:
        logger.error("Volume %s already exists. Skipping ...", args.volume_name)
        return

    resource_id = create_volume_from_snapshot(args.snapshot_id, args.region)

    ec2 = get_boto_ec2_client(args.region)
    response = get_volume_info(ec2, resource_id)
    assert 'Volumes' in response and len(response['Volumes']) > 0, "Volume not found: " + str(response)
    volume_size = response['Volumes'][0]['Size']

    create_if_needed_helper(args.volume_name, volume_size, "ssd", resource_id)

def clone_volume(args):
    """ Creates a clone of source volume to a destination volume. Both these are AX named volumes. """
    logger.info("Cloning volume: Source %s, Destination %s", args.source_volume_name, args.destination_volume_name)
    snapshot_args = argparse.Namespace()
    snapshot_args.region = args.region
    snapshot_args.volume_name = args.source_volume_name
    snapshot_id = create_volume_snapshot(snapshot_args)

    # Wait for the snapshot to complete.
    logger.info("Waiting for snapshot to be completed...")
    snapshot = boto3.resource('ec2', region_name=args.region, api_version='2016-04-01').Snapshot(snapshot_id)
    while snapshot.state != 'completed':
        logger.info("Current snapshot state: %s", snapshot.state)
        time.sleep(10)
        snapshot.reload()
    logger.info("Snapshot %s complete", snapshot_id)

    # Create new volume from the above snapshot.
    restore_args = argparse.Namespace()
    restore_args.volume_name = args.destination_volume_name
    restore_args.snapshot_id = snapshot_id
    restore_args.region = args.region
    restore_snapshot_to_volume(restore_args)
    logger.info("Clone of %s to %s complete", args.source_volume_name, args.destination_volume_name)

    # Best effort delete the snapshot. If the snapshot doesn't get deleted here, it will
    # get deleted with the cluster.
    ec2 = get_boto_ec2_client(args.region)
    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
    def delete_volume_snapshot():
        response = ec2.delete_snapshot(SnapshotId=snapshot_id)

    try:
        delete_volume_snapshot()
    except Exception as e:
        logger.info("Failed to create volume snapshot: %s", snapshot_id)

def import_volume(args):
    """ Imports an existing EBS volume as an AX named volume. """
    logger.info("Importing EBS volume %s as AX named volume %s", args.resource_id, args.volume_name)
    ec2 = get_boto_ec2_client(args.region)
    response = get_volume_info(ec2, args.resource_id)
    assert 'Volumes' in response and len(response['Volumes']) > 0, "Volume not found: " + str(response)
    volume_size = response['Volumes'][0]['Size']
    create_if_needed_helper(args.volume_name, volume_size, "ssd", args.resource_id)

def run():
    parser = argparse.ArgumentParser()
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--cloud', action='store', default="aws", help='Cloud provider to use')
    subparsers = parser.add_subparsers()

    parser_create = subparsers.add_parser('create', help="Create a new named volume")
    parser_create.add_argument('--volume-name', action='store', required=True, type=str, help='specify volume name')
    parser_create.add_argument('--volume-size-gb', action='store', required=True, type=int, help='specify volume size in GB')
    parser_create.add_argument('--storage-class', action='store', required=True, type=str, help='Storage class to use for the volume')
    parser_create.set_defaults(function=create_if_needed)

    parser_delete = subparsers.add_parser('delete', help="Delete a named volume")
    parser_delete.add_argument('--volume-name', action='store', required=True, type=str, help='specify volume name')
    parser_delete.set_defaults(function=delete_volume)

    parser_snapshot = subparsers.add_parser('snapshot', help="Snapshot a named volume")
    parser_snapshot.add_argument('--volume-name', action='store', required=False, type=str, help='specify volume name')
    parser_snapshot.add_argument('--resource-id', action='store', required=False, type=str, help='AWS volume resource id')
    parser_snapshot.add_argument('--region', action='store', required=True, type=str, help="Region in which to create the volume")
    parser_snapshot.set_defaults(function=create_volume_snapshot)

    parser_restore = subparsers.add_parser('restore', help="Restore a snapshot into a named volume")
    parser_restore.add_argument('--snapshot-id', action='store', required=True, type=str, help='AWS snapshot id')
    parser_restore.add_argument('--volume-name', action='store', required=True, type=str, help='specify volume name')
    parser_restore.add_argument('--region', action='store', required=True, type=str, help="Region in which to create the volume")
    parser_restore.set_defaults(function=restore_snapshot_to_volume)

    parser_clone = subparsers.add_parser('clone', help="Clone one named volume into another")
    parser_clone.add_argument('--source-volume-name', action='store', required=True, type=str, help='specify volume name')
    parser_clone.add_argument('--destination-volume-name', action='store', required=True, type=str, help='specify volume name')
    parser_clone.add_argument('--region', action='store', required=True, type=str, help="Region in which to create the volume")
    parser_clone.set_defaults(function=clone_volume)

    parser_import = subparsers.add_parser('import', help="Import an EBS volume as a named volume")
    parser_import.add_argument('--resource-id', action='store', required=True, type=str, help='Resource ID of the EBS volume')
    parser_import.add_argument('--volume-name', action='store', required=True, type=str, help='specify volume name')
    parser_import.add_argument('--region', action='store', required=True, type=str, help="Region in which to create the volume")
    parser_import.set_defaults(function=import_volume)

    args = parser.parse_args()

    # Only AWS is currently supported.
    assert args.cloud.lower() == "aws", "Volume tools is currently AWS only!"
    args.function(args)
