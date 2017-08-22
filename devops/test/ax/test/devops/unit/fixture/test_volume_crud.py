"""
Fixture manager volume tests
"""

import copy
import logging
import threading
import time
import uuid

import pytest

from ax.exceptions import AXApiResourceNotFound, AXIllegalOperationException, AXApiInvalidParam
from ax.devops.fixture.common import VolumeStatus, HTTP_AX_USERID_HEADER, HTTP_AX_USERNAME_HEADER, FIX_REQUESTER_AXAMM, FIX_REQUESTER_AXWORKFLOWADC
from .testutil import http_get, http_post, http_put, http_delete, wait_for_assignment, fake_volume_atime, wait_for_processor
from .testdata import TEST_VOLUMES, populate_linux_instances
from .mock import MockAxsysClient

logger = logging.getLogger(__name__)

def test_volume_axdb_crud(fixmgr):
    """Verify CRUD operations on volumes"""
    create_payload = copy.deepcopy(TEST_VOLUMES[0])

    # Create the volume
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    assert created_vol['status'] == VolumeStatus.INIT, "Created volume not in '{}' status".format(VolumeStatus.INIT)

    # Read the created volume
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert get_vol == created_vol

    # Mark the volume for deletion
    http_delete('/v1/storage/volumes/{}'.format(created_vol['id']))

    # Verify volume is marked as 'deleting'
    get_vol_after_mark_deletion = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert get_vol_after_mark_deletion['status'] == VolumeStatus.DELETING, "Volume not marked for deletion"
    assert fixmgr.volumemgr.axdb_client.get_volume(created_vol['id'])['status'] == VolumeStatus.DELETING, "Volume not marked for deletion in axdb"

    # Delete the volume database entry manually, verify we get AXApiResourceNotFound during GET
    fixmgr.volumemgr._delete_volume(created_vol['id'])
    with pytest.raises(AXApiResourceNotFound):
        http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert not fixmgr.volumemgr.axdb_client.get_volume(created_vol['id']), "Volume not deleted from axdb"

    # Test idempotency of delete
    http_delete('/v1/storage/volumes/{}'.format(created_vol['id']))

def test_volume_axdb_crud_invalid(fixmgr):
    """Attempt to perform create with fields that are not supposed to be updatable by user (e.g. status, time fields, etc...)"""
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    create_payload['anonymous'] = True
    create_payload['status'] = VolumeStatus.ACTIVE
    create_payload['ctime'] = 12345
    create_payload['foo'] = "bar"
    create_payload['attributes']['filesystem'] = 'xfs'
    create_payload['attributes']['volume_type'] = 'supersonic-io'
    create_payload['attributes']['my_custom_attr'] = 'foo'
    # Create the volume
    now = int(time.time())
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    assert not created_vol['anonymous']
    assert created_vol['status'] == VolumeStatus.INIT, "Created volume not in '{}' status".format(VolumeStatus.INIT)
    for time_field in ['ctime', 'mtime', 'atime']:
        assert created_vol[time_field] >= now
    assert 'foo' not in created_vol
    assert created_vol['attributes']['filesystem'] == 'ext4'
    assert created_vol['attributes']['volume_type'] == 'gp2'
    assert created_vol['attributes']['my_custom_attr'] == 'foo'
    assert 'storage_provider_id' not in created_vol['attributes']

    create_payload2 = copy.deepcopy(TEST_VOLUMES[1])
    # Test supplying invalid storage class
    create_payload2['storage_class'] = 'foo'
    with pytest.raises(AXApiInvalidParam):
        http_post('/v1/storage/volumes', data=create_payload2)
    create_payload2['storage_class'] = 'ssd'

    # Test without supplying size_gb
    create_payload2['attributes'].pop('size_gb', None)
    with pytest.raises(AXApiInvalidParam):
        http_post('/v1/storage/volumes', data=create_payload2)


def test_volume_duplicate_axrn(fixmgr):
    """Verify creating or renaming a volume which duplicates an axrn, is disallowed"""
    create_payload1 = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload1)

    # Attempt creation of second with same axrn. Verify it is rejected
    create_payload2 = copy.deepcopy(create_payload1)
    with pytest.raises(AXIllegalOperationException):
        http_post('/v1/storage/volumes', data=create_payload2)

    # Create a second volume with different name
    create_payload2['name'] = 'prod-wordpress-blog2'
    created_vol2 = http_post('/v1/storage/volumes', data=create_payload2)
    assert created_vol2['axrn'] == 'vol:/prod-wordpress-blog2'

    # Start workers since we can only rename volumes in active status
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)
    _wait_volume_status(created_vol2['id'], VolumeStatus.ACTIVE)
    created_vol2 = http_get('/v1/storage/volumes/{}'.format(created_vol2['id']))

    # Attempt rename of volume 2 with axrn collision with first
    rename_axrn_collision = {
        'id': created_vol2['id'],
        'name': 'prod-wordpress-blog',
    }
    with pytest.raises(AXIllegalOperationException):
        http_put('/v1/storage/volumes/{}'.format(created_vol2['id']), data=rename_axrn_collision)

    # Verify if we use a new name, there is no error
    rename_axrn = {
        'id': created_vol2['id'],
        'name': 'prod-wordpress-blog3',
    }
    renamed = http_put('/v1/storage/volumes/{}'.format(created_vol2['id']), data=rename_axrn)
    assert renamed['axrn'] == 'vol:/prod-wordpress-blog3'
    assert renamed['name'] == 'prod-wordpress-blog3'
    renamed = http_get('/v1/storage/volumes/{}'.format(created_vol2['id']))
    assert renamed['axrn'] == 'vol:/prod-wordpress-blog3'
    assert renamed['name'] == 'prod-wordpress-blog3'
    # This will ensure that after an update, we don't clobber any existing field values
    assert renamed['resource_id'] == created_vol2['resource_id']
    assert renamed['status_detail'] == created_vol2['status_detail']


def _wait_operations(fixmgr, timeout=30):
    """Helper to wait until fixturemanager completes operations on all volumes"""
    stop_time = time.time() + timeout
    while True:
        if not fixmgr.volumemgr.volume_work_q.unfinished_tasks:
            return
        assert time.time() < stop_time, "Timed out waiting for volume operations to complete"
        time.sleep(.1)

def _wait_volume_status(volume_id, status, timeout=10, poll_interval=0.1):
    """Helper to wait until volume reaches certain status"""
    start_time = time.time()
    while True:
        vol = http_get('/v1/storage/volumes/{}'.format(volume_id))
        if vol['status'] == status:
            break
        if time.time() - start_time > timeout:
            pytest.fail("volume failed to transition to '{}' state".format(status))
        time.sleep(poll_interval)

def _wait_volume_deletion(volume_id, timeout=15, poll_interval=1):
    """Helper to wait until volume is deleted"""
    start_time = time.time()
    while True:
        try:
            http_get('/v1/storage/volumes/{}'.format(volume_id))
        except AXApiResourceNotFound:
            return
        if time.time() - start_time > timeout:
            pytest.fail("volume failed to become deleted")
        time.sleep(poll_interval)

def test_volume_workers_basic(fixmgr):
    """Verifies basic lifecycle of volume worker for creating and deleting a volume"""
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    assert created_vol['status'] == VolumeStatus.INIT, "Created volume not in '{}' status".format(VolumeStatus.INIT)

    _wait_operations(fixmgr)
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    # Mark the volume for deletion
    http_delete('/v1/storage/volumes/{}'.format(created_vol['id']))

    _wait_operations(fixmgr)
    with pytest.raises(AXApiResourceNotFound):
        http_get('/v1/storage/volumes/{}'.format(created_vol['id']))

    # Ensure platform does not know about volume
    plat_get_vol = fixmgr.volumemgr.axsys_client.get_volume(created_vol['id'])
    assert plat_get_vol is None, "Volume not deleted from platform"


def test_volume_axops_user_context_headers(fixmgr):
    """Verifies we can create volumes with appropriate owner/creator when user information is supplied from axops as http headers (and not payload)"""
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    create_payload.pop('owner')
    create_payload.pop('creator')
    user_id = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
    username = "volumetestuser@email.com"

    # First try without the headers, verifies it fails and no volumes were created
    with pytest.raises(AXApiInvalidParam):
        created_vol = http_post('/v1/storage/volumes', data=create_payload)
    volumes = http_get('/v1/storage/volumes')
    assert not volumes['data']

    # Retry creation with HTTP headers
    headers = {
        HTTP_AX_USERID_HEADER: user_id,
        HTTP_AX_USERNAME_HEADER: username,
    }
    created_vol = http_post('/v1/storage/volumes', data=create_payload, headers=headers)
    assert created_vol['creator'] == username, "creator unexpected"
    assert created_vol['owner'] == username, "owner unexpected"

    # Read the created volume
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert get_vol['creator'] == username, "creator unexpected"
    assert get_vol['owner'] == username, "owner unexpected"


def test_volume_ignore_status_changes(fixmgr):
    """Verify we ignore status changes through the update API"""
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    assert created_vol['status'] == VolumeStatus.INIT, "Created volume not in '{}' status".format(VolumeStatus.INIT)
    created_vol['status'] = VolumeStatus.ACTIVE
    updated_vol = http_put('/v1/storage/volumes/{}'.format(created_vol['id']), data=create_payload)
    assert updated_vol['status'] == VolumeStatus.INIT
    updated_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert updated_vol['status'] == VolumeStatus.INIT


def test_volume_workers_serial_requests(fixmgr):
    """Perform create followed immediately by delete. Set workers to 1 have serial request.
    Verify volume is eventually deleted after all operations complete (first worker will requeue the request)"""
    fixmgr.volumemgr.num_workers = 1
    fixmgr.volumemgr.start_workers()
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    _wait_volume_status(created_vol['id'], VolumeStatus.CREATING)
    http_delete('/v1/storage/volumes/{}'.format(created_vol['id']))
    vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert vol['status'] == VolumeStatus.DELETING
    _wait_operations(fixmgr)
    with pytest.raises(AXApiResourceNotFound):
        http_get('/v1/storage/volumes/{}'.format(created_vol['id']))


def test_volume_workers_concurrent_requests(fixmgr):
    """Perform create followed immediately by delete. Set workers to 2 have concurrent workers handling the request.
    Verify volume is eventually deleted after all operations complete (second worker will skip the job, but first worker will requeue the request)"""
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    _wait_volume_status(created_vol['id'], VolumeStatus.CREATING)
    http_delete('/v1/storage/volumes/{}'.format(created_vol['id']))
    vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert vol['status'] == VolumeStatus.DELETING
    _wait_operations(fixmgr)
    with pytest.raises(AXApiResourceNotFound):
        http_get('/v1/storage/volumes/{}'.format(created_vol['id']))

def test_volume_request_assign_release_sync(fixmgr):
    """Verify basic volume lifecycle of request, assignment, release using synchronous requests"""
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'myvol' : {
                'axrn' : 'vol:/'+create_payload['name']
            }
        }
    }
    # First try requesting a volume by name that doesn't exist. This should raise AXApiResourceNotFound (impossible request)
    with pytest.raises(AXApiResourceNotFound) as err:
        http_post('/v1/fixture/requests', data=vol_req_json)
        assert "Impossible request" in err.value.message

    # Now create the volume, but do not allow it to transition to 'active' state (do not start workers).
    created_vol = http_post('/v1/storage/volumes', data=create_payload)

    # While volume is in 'init' stage, fixture request should fail with AXApiResourceNotFound but with a different message than impossible request
    with pytest.raises(AXApiResourceNotFound) as err:
        http_post('/v1/fixture/requests', data=vol_req_json)
        assert "could not allocate resources" in err.value.message

    b4_assignment = fake_volume_atime(fixmgr, created_vol['id'])

    # Start the workers and wait for the volume to become active
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    # Request should be successful and it should have an assignment
    fix_req = http_post('/v1/fixture/requests', data=vol_req_json)
    assert fix_req['vol_assignment'] and fix_req['vol_assignment']['myvol'], "volume was not assigned"
    assert fix_req['vol_assignment']['myvol']['axrn'] == vol_req_json['vol_requirements']['myvol']['axrn']

    # Verify atime of volume is updated as well as referrers field
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert get_vol['atime'] > b4_assignment
    assert len(get_vol['referrers']) == 1
    assert get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']

    # Verify deletes are rejected while volume has referrers
    with pytest.raises(AXIllegalOperationException):
        http_delete('/v1/storage/volumes/{}'.format(created_vol['id']))

    # Release the reservation
    b4_release = fake_volume_atime(fixmgr, created_vol['id'])
    http_delete('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))

    # Verify atime of volume is updated as well as referrers field
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert get_vol['atime'] > b4_release
    assert not get_vol['referrers']


def test_volume_request_assign_release_async(fixmgr):
    """Verify basic volume lifecycle of request, assignment, release using asynchronous requests"""
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'user': 'testuser@email.com',
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'synchronous': False,
        'vol_requirements' : {
            'myvol' : {
                'axrn' : 'vol:/'+create_payload['name']
            },
            'anonvol' : {
                'storage_class' : 'ssd',
                'size_gb' : 10,
            }
        }
    }
    # First try requesting a volume by name that doesn't exist. This should raise AXApiResourceNotFound (impossible request)
    with pytest.raises(AXApiResourceNotFound) as err:
        http_post('/v1/fixture/requests', data=vol_req_json)
        assert "Impossible request" in err.value.message

    # Now create the volume, but do not allow it to transition to 'active' state (do not start workers).
    created_vol = http_post('/v1/storage/volumes', data=create_payload)

    # Make async fixture request while volume is still in 'init' stage.
    # Request should be successful but will not have an assignment
    fix_req = http_post('/v1/fixture/requests', data=vol_req_json)
    assert not fix_req['vol_assignment'], "volume unexpectedly assigned"

    b4_assignment = fake_volume_atime(fixmgr, created_vol['id'])

    # Start the workers, and request processor, and wait for the volume to become active
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    # Wait for volume assignment
    wait_for_assignment(fixmgr, vol_req_json['service_id'])
    fix_req = http_get('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    assert fix_req['vol_assignment'] and fix_req['vol_assignment']['myvol'], "volume was not assigned"
    assert fix_req['vol_assignment']['myvol']['axrn'] == vol_req_json['vol_requirements']['myvol']['axrn']
    expected_anon_vol_axrn = 'vol:/anonymous/root_workflow_id:{}/service_id:{}/anonvol'.format(vol_req_json['root_workflow_id'], vol_req_json['service_id'])
    assert fix_req['vol_assignment']['anonvol']['axrn'] == expected_anon_vol_axrn
    assert fix_req['vol_assignment']['anonvol']['resource_id']

    # Verify atime of named volume is updated as well as referrers field
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert get_vol['atime'] > b4_assignment
    assert len(get_vol['referrers']) == 1
    assert get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']

    # Verify anon volume was created and in active state (we did not notify of assignment when it was in creating status)
    anon_get_vol = http_get('/v1/storage/volumes?anonymous=true')['data'][0]
    assert anon_get_vol['status'] == VolumeStatus.ACTIVE
    assert len(anon_get_vol['referrers']) == 1
    assert anon_get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']

    # Verify deletes are rejected while volume has referrers
    with pytest.raises(AXIllegalOperationException):
        http_delete('/v1/storage/volumes/{}'.format(created_vol['id']))
    with pytest.raises(AXIllegalOperationException):
        http_delete('/v1/storage/volumes/{}'.format(anon_get_vol['id']))

    # Release the reservation
    b4_release = fake_volume_atime(fixmgr, created_vol['id'])
    http_delete('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))

    # Verify atime of named volume is updated as well as referrers field
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert get_vol['atime'] > b4_release
    assert not get_vol['referrers']

    # Verify anonymous volume is deleted
    _wait_operations(fixmgr)
    assert len(http_get('/v1/storage/volumes?anonymous=true')['data']) == 0

def test_volume_retry_operations(fixmgr, monkeypatch):
    """Simulates an error in axmon create_volume and delete_volume. Verify retry worker will attempt retry"""
    orig_create_volume = fixmgr.volumemgr.axsys_client.create_volume
    def _mock_create_volume(*args, **kwargs):
        _mock_create_volume.attempt += 1
        if _mock_create_volume.attempt <= 1:
            raise Exception("Intentional mock create error")
        return orig_create_volume(*args, **kwargs)
    _mock_create_volume.attempt = 0

    orig_delete_volume = fixmgr.volumemgr.axsys_client.delete_volume
    def _mock_delete_volume(*args, **kwargs):
        _mock_delete_volume.attempt += 1
        if _mock_delete_volume.attempt <= 1:
            raise Exception("Intentional mock delete error")
        return orig_delete_volume(*args, **kwargs)
    _mock_delete_volume.attempt = 0

    monkeypatch.setattr(fixmgr.volumemgr.axsys_client, 'create_volume', _mock_create_volume)
    monkeypatch.setattr(fixmgr.volumemgr.axsys_client, 'delete_volume', _mock_delete_volume)

    # Start the workers and wait for the volume to become active
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.retry_interval = 3
    fixmgr.volumemgr.start_workers()

    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)

    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)
    assert _mock_create_volume.attempt == 2, "Unexpected create attempts"

    # Mark the volume for deletion
    http_delete('/v1/storage/volumes/{}'.format(created_vol['id']))

    _wait_volume_deletion(created_vol['id'])
    assert _mock_delete_volume.attempt == 2, "Unexpected delete attempts"


def test_reserve_disabled_volumes(fixmgr):
    """Tests we can set the 'enabled' field of a volume to be False, and it will not be reservable"""
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'myvol' : {
                'axrn' : 'vol:/'+create_payload['name']
            }
        }
    }
    # Start the workers, create the volume, and wait for the volume to become active
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    # Disable the volume
    http_put('/v1/storage/volumes/{}'.format(created_vol['id']), data={'enabled': False})
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert not get_vol['enabled'], "volume still enabled"

    # Sync request should fail. Volume should not be assigned.
    with pytest.raises(AXApiResourceNotFound) as err:
        http_post('/v1/fixture/requests', data=vol_req_json)
        assert "could not allocate resources" in err.value.message
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert not get_vol['referrers'], "volume unexpectedly assigned"

    # Async request should succeed
    vol_req_json['synchronous'] = False
    http_post('/v1/fixture/requests', data=vol_req_json)

    # Verify nothing gets assigned (because volume is still disabled)
    wait_for_processor(fixmgr)
    fix_req = http_get('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    assert not fix_req['vol_assignment'], "Volume unexpectedly assigned"
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert not get_vol['referrers'], "volume unexpectedly assigned"

    # Enable the volume
    http_put('/v1/storage/volumes/{}'.format(created_vol['id']), data={'enabled': True})

    # Verify volume gets assigned
    wait_for_assignment(fixmgr, vol_req_json['service_id'])
    fix_req = http_get('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    assert fix_req['vol_assignment'], "Volume was not assigned"
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert len(get_vol['referrers']) == 1
    assert get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']

    # Release the reservation
    http_delete('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))

    # Verify volume has no more referrers
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert not get_vol['referrers']


def test_verify_volume_names(fixmgr):
    """Supply some invalid values for volume names during create and update. Verify it is rejected"""
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    invalid_names = ['-abc123', 'abc_123', '', 'abc!']

    # Verify we cannot create new volumes with invalid names
    for invalid_name in invalid_names:
        create_payload['name'] = invalid_name
        with pytest.raises(AXApiInvalidParam):
            http_post('/v1/storage/volumes', data=create_payload)

    # Create a volume with valid name
    create_payload['name'] = 'valid-name'
    created_vol = http_post('/v1/storage/volumes', data=create_payload)

    # Verify we cannot rename volumes while in 'init' state
    with pytest.raises(AXIllegalOperationException):
        http_put('/v1/storage/volumes/{}'.format(created_vol['id']), data={'name': 'valid-name2'})

    # Start workers and wait until it is active
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    # Verify we cannot update to an invalid name
    for invalid_name in invalid_names:
        with pytest.raises(AXApiInvalidParam):
            http_put('/v1/storage/volumes/{}'.format(created_vol['id']), data={'name': invalid_name})

    # Rename to valid name and verify axrn change was propogated down to platform
    valid_name2 = 'valid-name2'
    updated_vol = http_put('/v1/storage/volumes/{}'.format(created_vol['id']), data={'name': valid_name2})
    assert 'vol:/'+valid_name2 == updated_vol['axrn']
    plat_vol = fixmgr.volumemgr.axsys_client.get_volume(created_vol['id'])
    plat_axrn = next((x['Value'] for x in plat_vol['Tags'] if x['Key'] == 'axrn'))
    assert 'vol:/'+valid_name2 == plat_axrn


def test_volume_reserve_conflicting_requirements(fixmgr):
    """Verifies fixturemanager rejects a second volume request using the same service_id with different requirements"""
    assert not fixmgr.axdb_client.get_volumes(), "Setup error"
    assert not fixmgr.axdb_client.get_fixture_requests(), "Setup error"
    # Start the workers, and request processor, and wait for two volume to become active
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    create_payload2 = copy.deepcopy(TEST_VOLUMES[1])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    created_vol2 = http_post('/v1/storage/volumes', data=create_payload2)
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)
    _wait_volume_status(created_vol2['id'], VolumeStatus.ACTIVE)

    # Make a request. It should be successful and it should have an assignment
    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'myvol' : {
                'axrn' : 'vol:/'+create_payload['name']
            }
        }
    }
    logger.info("Making initial fixture request")
    fix_req = http_post('/v1/fixture/requests', data=vol_req_json)
    assert fix_req['vol_assignment'] and fix_req['vol_assignment']['myvol'], "volume was not assigned"
    assert fix_req['vol_assignment']['myvol']['axrn'] == vol_req_json['vol_requirements']['myvol']['axrn']

    # Make a second request with same service_id but with different vol_requirements. This should be rejected
    logger.info("Making second fixture request with conflicting requirements")
    vol_req_json2 = copy.deepcopy(vol_req_json)
    vol_req_json2['vol_requirements']['myvol']['axrn'] = 'vol:/'+create_payload2['name']
    with pytest.raises(AXApiInvalidParam) as err:
        fix_req = http_post('/v1/fixture/requests', data=vol_req_json2)
        assert "multiple fixture requests with different requirements" in err.value.message

    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert len(get_vol['referrers']) == 1
    assert get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']

    get_vol2 = http_get('/v1/storage/volumes/{}'.format(created_vol2['id']))
    assert len(get_vol2['referrers']) == 0

def test_volume_reserve_idempotent_sync(fixmgr):
    """Verifies a second, identical synchronous fixture request will return the previous (including any assignments)"""
    # Start the workers, and request processor, and wait for volume to become active
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    # Make a request. It should be successful and it should have an assignment
    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'myvol' : {
                'axrn' : 'vol:/'+create_payload['name']
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=vol_req_json)
    assert fix_req['vol_assignment'] and fix_req['vol_assignment']['myvol'], "volume was not assigned"
    assert fix_req['vol_assignment']['myvol']['axrn'] == vol_req_json['vol_requirements']['myvol']['axrn']
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert len(get_vol['referrers']) == 1
    assert get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']

    # Make a second request with same service_id with same vol_requirements
    fix_req2 = http_post('/v1/fixture/requests', data=vol_req_json)
    assert fix_req == fix_req2

    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert len(get_vol['referrers']) == 1
    assert get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']

def test_volume_reserve_idempotent_async(fixmgr):
    """Verifies a second, identical async fixture request will return the existing request (including any assignments)"""
    # Start the workers, and request processor, and wait for volume to become active
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    # Make a async request (it will not have an assignment from the response payload)
    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': False,
        'vol_requirements' : {
            'myvol' : {
                'axrn' : 'vol:/'+create_payload['name']
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=vol_req_json)
    assert not fix_req['vol_assignment']

    # Ensure it gets assigned
    wait_for_assignment(fixmgr, vol_req_json['service_id'])
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert len(get_vol['referrers']) == 1
    assert get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']

    # Make a second request with same service_id with same vol_requirements
    # The second request will return the existing request which already was assigned.
    fix_req2 = http_post('/v1/fixture/requests', data=vol_req_json)
    assert fix_req2['request_time'] == fix_req['request_time']
    assert fix_req2['vol_assignment']
    assert fix_req2['vol_assignment']['myvol']['axrn'] == vol_req_json['vol_requirements']['myvol']['axrn']

    # Make sure that because we made a second request, fixturemanager will resend the notification
    wait_for_assignment(fixmgr, vol_req_json['service_id'])

def test_reserve_multiple_volumes(fixmgr):
    """Verifies we can perform multiple volume reservations in same request, and assignment is atomic (all or nothing)"""
    # Start the workers, and request processor, and wait for two volume to become active
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    create_payload2 = copy.deepcopy(TEST_VOLUMES[1])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    created_vol2 = http_post('/v1/storage/volumes', data=create_payload2)
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)
    _wait_volume_status(created_vol2['id'], VolumeStatus.ACTIVE)

    # Make a request. It should be successful and it should have an assignment
    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'myvol' : {
                'axrn' : 'vol:/'+create_payload['name']
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=vol_req_json)
    assert fix_req['vol_assignment'] and fix_req['vol_assignment']['myvol'], "volume was not assigned"
    assert fix_req['vol_assignment']['myvol']['axrn'] == vol_req_json['vol_requirements']['myvol']['axrn']

    multi_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'myvol' : {
                'axrn' : 'vol:/'+create_payload['name']
            },
            'myvol2' : {
                'axrn' : 'vol:/'+create_payload2['name']
            }

        }
    }
    # This should fail because only 1 out of 2 volumes are available
    with pytest.raises(AXApiResourceNotFound):
        fix_req = http_post('/v1/fixture/requests', data=multi_req_json)

    # Change it to async request
    multi_req_json['synchronous'] = False
    fix_req = http_post('/v1/fixture/requests', data=multi_req_json)

    # Make sure it does not get assigned (since volume1 is still reserved)
    assert not wait_for_assignment(fixmgr, multi_req_json['service_id'], timeout=2, verify_exists=False)
    get_vol = http_get('/v1/storage/volumes/{}'.format(created_vol['id']))
    assert len(get_vol['referrers']) == 1
    assert get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']
    get_vol2 = http_get('/v1/storage/volumes/{}'.format(created_vol2['id']))
    assert not get_vol2['referrers']

    # release reservation on volume 1. The multi-volume request should now be assigned
    http_delete('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    wait_for_assignment(fixmgr, multi_req_json['service_id'])

def test_anonymous_volume_request_sync(fixmgr):
    """Tests the basic ability of requesting an anonymous volume in synchronous mode"""
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()

    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'user': 'testuser@email.com',
        'root_workflow_id' : str(uuid.uuid4()),
        'application_name' : 'test_app',
        'application_generation' : str(uuid.uuid4()),
        'application_id' : str(uuid.uuid4()),
        'deployment_id' : str(uuid.uuid4()),
        'deployment_name' : 'test_deployment',
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'myvol' : {
                'storage_class' : 'ssd',
                'size_gb' : 10,
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=vol_req_json)
    fix_req_get = http_get('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    assert fix_req == fix_req_get
    assert fix_req['vol_assignment']
    axrn = fix_req['vol_assignment']['myvol']['axrn']
    assert fix_req['vol_assignment']['myvol']['resource_id']

    get_vol = http_get('/v1/storage/volumes')['data'][0]
    assert get_vol['axrn'] == axrn
    assert get_vol['status'] == VolumeStatus.ACTIVE
    assert get_vol['referrers'][0]['service_id'] == vol_req_json['service_id']

    http_delete('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    _wait_volume_deletion(get_vol['id'])

def test_request_fixture_named_and_anonymous_volume_sync(fixmgr):
    """Tests the ability to request a fixture, a named, and anonymous volume together"""
    populate_linux_instances(fixmgr)
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()

    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'user': 'testuser@email.com',
        'root_workflow_id' : str(uuid.uuid4()),
        'application_name' : 'test_app',
        'application_generation' : str(uuid.uuid4()),
        'application_id' : str(uuid.uuid4()),
        'deployment_id' : str(uuid.uuid4()),
        'deployment_name' : 'test_deployment',
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'requirements' : {
            'fix1' : {
                'class': 'Linux'
            }
        },
        'vol_requirements' : {
            'namedvol' : {
                'axrn': 'vol:/'+create_payload['name']
            },
            'anonvol' : {
                'storage_class' : 'ssd',
                'size_gb' : 10,
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=vol_req_json)
    fix_req_get = http_get('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    assert fix_req == fix_req_get
    assert fix_req['vol_assignment']
    assert fix_req['assignment']
    assert fix_req['assignment']['fix1']['class'] == 'Linux'

    vols = http_get('/v1/storage/volumes')['data']
    assert len(vols) == 2
    named_vol = next(v for v in vols if not v['anonymous'])
    anon_vol = next(v for v in vols if v['anonymous'])

    for vol in [named_vol, anon_vol]:
        assert vol['status'] == VolumeStatus.ACTIVE
        assert vol['referrers'][0]['service_id'] == vol_req_json['service_id']

    http_delete('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    _wait_volume_deletion(anon_vol['id'])

    vols = http_get('/v1/storage/volumes')['data']
    assert len(vols) == 1
    assert vols[0]['id'] == created_vol['id']


def test_anonymous_volume_create_delete_race(fixmgr, monkeypatch):
    """In the middle of provisioning an anonymous volume, delete the fixture request. Verify any created volumes are eventually deleted"""
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()

    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    orig_create_volume = fixmgr.volumemgr.axsys_client.create_volume
    def _mock_create_volume(*args, **kwargs):
        time.sleep(4)
        return orig_create_volume(*args, **kwargs)
    monkeypatch.setattr(fixmgr.volumemgr.axsys_client, 'create_volume', _mock_create_volume)

    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'user': 'testuser@email.com',
        'root_workflow_id' : str(uuid.uuid4()),
        'application_name' : 'test_app',
        'application_generation' : str(uuid.uuid4()),
        'application_id' : str(uuid.uuid4()),
        'deployment_id' : str(uuid.uuid4()),
        'deployment_name' : 'test_deployment',
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'namedvol' : {
                'axrn': 'vol:/'+create_payload['name']
            },
            'anonvol' : {
                'storage_class' : 'ssd',
                'size_gb' : 10,
            }
        }
    }

    # Delete the fixture request after two seconds. (We delay the axsys create_volume call by 4 seconds)
    threading.Timer(2, http_delete, args=('/v1/fixture/requests/{}'.format(vol_req_json['service_id']),)).start()

    with pytest.raises(AXIllegalOperationException):
        http_post('/v1/fixture/requests', data=vol_req_json)

    # allow any deletes to complete
    _wait_operations(fixmgr)

    vols = http_get('/v1/storage/volumes')['data']
    assert len(vols) == 1
    assert vols[0]['id'] == created_vol['id']
    assert vols[0]['referrers'] == []

    assert not http_get('/v1/fixture/requests')['data']

def test_get_fixtures_filters(fixmgr):
    """Tests various query filters against fixtures (anonymous, deployment_id)"""
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()

    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    # create a second named vol
    http_post('/v1/storage/volumes', data=copy.deepcopy(TEST_VOLUMES[1]))

    _wait_volume_status(created_vol['id'], VolumeStatus.ACTIVE)

    vol_req_json = {
        'service_id' : str(uuid.uuid4()),
        'user': 'testuser@email.com',
        'root_workflow_id' : str(uuid.uuid4()),
        'application_name' : 'test_app',
        'application_generation' : str(uuid.uuid4()),
        'application_id' : str(uuid.uuid4()),
        'deployment_id' : str(uuid.uuid4()),
        'deployment_name' : 'test_deployment',
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'namedvol' : {
                'axrn': 'vol:/'+create_payload['name']
            },
            'anonvol' : {
                'storage_class' : 'ssd',
                'size_gb' : 10,
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=vol_req_json)
    fix_req_get = http_get('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    assert fix_req == fix_req_get
    vols = http_get('/v1/storage/volumes')['data']
    assert len(vols) == 3
    anon_vol = next(v for v in vols if v['anonymous'])

    # Test anonymous == true filter
    anon_vols = http_get('/v1/storage/volumes?anonymous=tRuE')['data']
    assert len(anon_vols) == 1 and anon_vols[0]['id'] == anon_vol['id']
    # Test anonymous == false filter
    named_vols = http_get('/v1/storage/volumes?anonymous=fAlsE')['data']
    assert len(named_vols) == 2

    # Test deployment_id filter also in conjunction with anonymous filter
    assert not http_get('/v1/storage/volumes?deployment_id={}'.format(str(uuid.uuid4())))['data']
    vols = http_get('/v1/storage/volumes?deployment_id={}'.format(fix_req['service_id']))['data']
    assert len(vols) == 2
    vols = http_get('/v1/storage/volumes?deployment_id={}&anonymous=true'.format(fix_req['service_id']))['data']
    assert len(vols) == 1
    assert vols[0]['anonymous']
    vols = http_get('/v1/storage/volumes?deployment_id={}&anonymous=false'.format(fix_req['service_id']))['data']
    assert len(vols) == 1
    assert not vols[0]['anonymous']

def test_volume_import(fixmgr):
    """Verify ability to import existing volume by supplying resource_id"""
    if not isinstance(fixmgr.volumemgr.axsys_client, MockAxsysClient):
        pytest.skip("test only applicable to mock axsys client")

    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    mock_resource_id = 'vol-abc123'
    create_payload['resource_id'] = mock_resource_id

    # Create the volume and verify it is in 'active' state
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    assert created_vol['status'] == VolumeStatus.ACTIVE, "Created volume not in '{}' status".format(VolumeStatus.ACTIVE)
    assert created_vol['resource_id'] == mock_resource_id, "Created volume did not preserve resource_id"

def test_volume_ignore_unknown_vol_requirements_fields(fixmgr):
    """Verifies fixturemanager ignores fields which are unknown before storing into the requestdb since it can """
    fixmgr.volumemgr.num_workers = 2
    #fixmgr.volumemgr.start_workers()
    #fixmgr.reqproc.start_processor()
    create_payload = copy.deepcopy(TEST_VOLUMES[0])
    create_payload['enabled'] = False
    http_post('/v1/storage/volumes', data=create_payload)

    vol_req_json_1 = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'user': 'testuser@email.com',
        'vol_requirements' : {
            'myvol' : {
                'axrn' : 'vol:/'+create_payload['name'],
                'extra_field': 'blah'
            },
            'myvol2' : {
                'storage_class' : 'ssd',
                'size_gb': 1,
                'extra_field': 'blah'
            }

        }
    }
    vol_req_json_2 = copy.deepcopy(vol_req_json_1)
    vol_req_json_2['vol_requirements']['myvol']['extra_field'] = 'blah2'
    vol_req_json_2['vol_requirements']['myvol2']['extra_field'] = 'blah2'

    fix_req1 = http_post('/v1/fixture/requests', data=vol_req_json_1)
    fix_req2 = http_post('/v1/fixture/requests', data=vol_req_json_2)

    assert fix_req1 == fix_req2
    assert 'extra_field' not in fix_req1['vol_requirements']

def test_volume_reject_size_update(fixmgr):
    """Verifies fixturemanager we reject a size update"""
    create_payload = copy.deepcopy(TEST_VOLUMES[0])

    # Create the volume
    created_vol = http_post('/v1/storage/volumes', data=create_payload)
    assert created_vol['status'] == VolumeStatus.INIT, "Created volume not in '{}' status".format(VolumeStatus.INIT)

    # Update volume with different size
    with pytest.raises(AXIllegalOperationException) as err:
        http_put('/v1/storage/volumes/{}'.format(created_vol['id']), data={'attributes': {'size_gb': 123}})
        assert "cannot be resized" in str(err)

def test_do_not_release_orphaned_deployment_reservations(fixmgr):
    """Verifies we we do not delete orphaned deployment reservations (this is considered too unsafe)"""
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()

    service_id = str(uuid.uuid4())
    vol_req_json = {
        'service_id' : service_id,
        'user': 'testuser@email.com',
        'root_workflow_id' : str(uuid.uuid4()),
        'application_name' : 'test_app',
        'application_generation' : str(uuid.uuid4()),
        'application_id' : str(uuid.uuid4()),
        'deployment_id' : str(uuid.uuid4()),
        'deployment_name' : 'test_deployment',
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'anonvol' : {
                'storage_class' : 'ssd',
                'size_gb' : 10,
            }
        }
    }

    http_post('/v1/fixture/requests', data=vol_req_json)

    vols = http_get('/v1/storage/volumes')['data']
    assert len(vols) == 1
    vol_id = vols[0]['id']

    # run check consistency. this should detect that axamm does not have the deployment but it should not delete the request or volume
    fixmgr.check_consistency()
    assert http_get('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    vol = http_get('/v1/storage/volumes/{}'.format(vol_id))
    assert vol['status'] == VolumeStatus.ACTIVE

    assert fixmgr.reqproc.redis_client_notification.exists("notification:{}".format(service_id))


def test_anon_volume_free_up_axrn(fixmgr, monkeypatch):
    """Ensure when we mark an anonymous volume for deletion, we immediately free up its axrn"""
    fixmgr.volumemgr.num_workers = 2
    fixmgr.volumemgr.start_workers()
    fixmgr.reqproc.start_processor()

    service_id = str(uuid.uuid4())
    vol_req_json = {
        'service_id' : service_id,
        'user': 'testuser@email.com',
        'root_workflow_id' : str(uuid.uuid4()),
        'application_name' : 'test_app',
        'application_generation' : str(uuid.uuid4()),
        'application_id' : str(uuid.uuid4()),
        'deployment_id' : str(uuid.uuid4()),
        'deployment_name' : 'test_deployment',
        'requester' : FIX_REQUESTER_AXAMM,
        'synchronous': True,
        'vol_requirements' : {
            'anonvol' : {
                'storage_class' : 'ssd',
                'size_gb' : 10,
            }
        }
    }

    http_post('/v1/fixture/requests', data=vol_req_json)

    vols = http_get('/v1/storage/volumes')['data']
    assert len(vols) == 1
    vol_id = vols[0]['id']
    start_axrn = vols[0]['axrn']

    # patch the delete_volume call to prevent successful delete
    block_delete = True
    orig_delete_volume = fixmgr.volumemgr.axsys_client.delete_volume
    def _mock_delete_volume(*args, **kwargs):
        if block_delete:
            raise Exception("Intentional mock delete error")
        return orig_delete_volume(*args, **kwargs)
    _mock_delete_volume.attempt = 0
    monkeypatch.setattr(fixmgr.volumemgr.axsys_client, 'delete_volume', _mock_delete_volume)

    # release the volume. this should mark the volume as 'deleting'
    http_delete('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))
    vol = http_get('/v1/storage/volumes/{}'.format(vol_id))
    assert vol['status'] == VolumeStatus.DELETING
    # existing fixture request should be gone
    with pytest.raises(AXApiResourceNotFound):
        http_get('/v1/fixture/requests/{}'.format(vol_req_json['service_id']))

    # ensure the volume still exists, and that we have changed the axrn
    vol = http_get('/v1/storage/volumes/{}'.format(vol_id))
    assert vol['status'] == VolumeStatus.DELETING
    # assert vol['axrn'] != start_axrn

    # make another request using same request payload. there should be two volumes, one in deleting status, the other in active
    http_post('/v1/fixture/requests', data=vol_req_json)
    vols = http_get('/v1/storage/volumes')['data']
    assert len(vols) == 2
    new_vol = next((v for v in vols if v['status'] == VolumeStatus.ACTIVE))
    assert new_vol['axrn'] == start_axrn
