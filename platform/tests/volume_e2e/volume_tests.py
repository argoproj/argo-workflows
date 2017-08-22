#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
import json
import logging
import time
import unittest
import uuid

import requests


AXMON_HOST = "axmon.axsys"
FIXTUREMANAGER_HOST = "fixturemanager.axsys"

logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S")
logger = logging.getLogger("axmon.platform_volume_api_test")
logger.setLevel(logging.DEBUG)

class PlatformVolumeAPITests(unittest.TestCase):
    """
    Test for volume apis in axmon.
    """
    def test_platform_crud(self):
        """
        Test volume CRUD operations succeed.
        """

        volume_id = str(uuid.uuid4())
        vol_opts = {}
        vol_opts['storage_provider_name'] = 'ebs'
        vol_opts['size_gb'] = '4'
        vol_opts['volume_type'] = 'gp2'
        vol_opts['zone'] = 'us-west-2a'
        vol_opts['axrn'] = 'vol:/abcd@applatix.com/my_db_volume'

        headers = {'content-type': 'application/json'}
        response = requests.post('http://' + AXMON_HOST + ':8901/v1/axmon/volume/' + volume_id,
                                 data=json.dumps(vol_opts), headers=headers)
        assert response is not None

        volume_response = response.json()
        logger.info("Created volume with id: %s", volume_response)
        aws_volume_id = volume_response['result']

        # Verify GET request succeeds.
        response = requests.get('http://' + AXMON_HOST + ':8901/v1/axmon/volume/' + volume_id)
        volume_response = response.json()
        assert volume_response is not None
        for tag in volume_response['result']['Tags']:
            if tag['Key'] == 'axrn':
                assert tag['Value'] == vol_opts['axrn']
            elif tag['Key'] == 'AXVolumeID':
                assert tag['Value'] == volume_id
            elif tag['Key'] == 'VolumeId':
                assert['Value'] == aws_volume_id

        # Verify update (PUT) succeeds.
        new_vol_opts = {'axrn':'vol:/abcd@applatix.com/my_db_volume-2' }
        response = requests.put('http://' + AXMON_HOST + ':8901/v1/axmon/volume/' + volume_id,
                                data=json.dumps(new_vol_opts), headers=headers)
        volume_response = response.json()
        assert volume_response is not None
        logger.info("Response on UPDATE (PUT): %s", volume_response)
        assert volume_response['result'] == 'ok'

        # Verify that DELETE succeeds.
        response = requests.delete('http://' + AXMON_HOST + ':8901/v1/axmon/volume/' + volume_id)
        volume_response = response.json()
        assert volume_response is not None
        logger.info("Response on DELETE: %s", volume_response)
        assert volume_response['result'] == 'ok'

class FixtureManagerVolumeAPITests(unittest.TestCase):
    """
    Test for volume apis in fixturemanager.
    """
    def test_fixturemanager_crud(self):
        """
        Test volume CRUD operations succeed via fixturemanager.
        """
        random_volume_name = str(uuid.uuid4())
        vol_opts = {
            "name" : random_volume_name,
            "owner" : "testuser@applatix.com",
            "creator" : "testuser@applatix.com",
            "attributes" : {
                "size_gb" : 1,
            }
        }

        # Create a new volume
        headers = {'content-type': 'application/json'}
        response = requests.post('http://' + FIXTUREMANAGER_HOST + ':8912/v1/storage/volumes',
                                 data=json.dumps(vol_opts), headers=headers)
        assert response is not None
        vol_metadata = response.json()
        assert vol_metadata['id'] is not None
        assert vol_metadata['status'] == 'init'

        volume_id = vol_metadata['id']
        volume_status = 'init'
        logger.info('Volume created with ID %s, status %s', volume_id, volume_status)

        # Get all volumes
        while volume_status != 'active':
            time.sleep(30)
            response = requests.get('http://' + FIXTUREMANAGER_HOST + ':8912/v1/storage/volumes')
            assert response is not None
            all_vols = response.json()['data']
            volume_found = False
            for vol in all_vols:
                if vol['id'] == volume_id:
                    volume_found = True
                    volume_status = vol['status']
                    break
            assert volume_found is True

        logger.info('Volume status %s', volume_status)

        # Delete newly created volume
        response = requests.delete('http://' + FIXTUREMANAGER_HOST + ':8912/v1/storage/volumes/' + volume_id)
        assert response is not None
        logger.info("DELETE response %s", response)
