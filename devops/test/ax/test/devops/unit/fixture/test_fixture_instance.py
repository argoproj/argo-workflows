"""
Unit tests for fixture classes
"""
import time
import uuid

import pytest

from ax.exceptions import AXApiResourceNotFound, AXIllegalOperationException, AXApiInvalidParam
from ax.devops.fixture.common import InstanceStatus, ServiceStatus
from . import testdata
from .testutil import http_get, http_post, http_put, http_delete, wait_for_assignment, fake_volume_atime, wait_for_processor

def _create_instance(fixmgr):
    testdata.populate_class(fixmgr)
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {'group': 'dev'},
    }
    created_instance = http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)
    # ensure we set status to creating and operation is not null
    assert created_instance['status'] == InstanceStatus.CREATING
    assert created_instance['operation']
    service = fixmgr.axops_client.get_service(created_instance['operation']['id'])
    assert created_instance['operation']['id'] == service['id']
    return created_instance, service

def _create_instance_action_success(fixmgr, name='instance1'):
    testdata.populate_class(fixmgr)
    create_payload = {
        'name': name,
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {'group': 'dev'},
    }
    created_instance = http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)
    # ensure we set status to creating and operation is not null
    assert created_instance['status'] == InstanceStatus.CREATING
    assert created_instance['operation']
    service = fixmgr.axops_client.get_service(created_instance['operation']['id'])
    assert created_instance['operation']['id'] == service['id']

    # notify fixture manager it was successful
    service['status'] = ServiceStatus.SUCCESS
    http_post('/v1/fixture/action_result', data=service)

    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.ACTIVE
    assert not get_instance['operation']
    return get_instance


def test_fixture_instance_create_action_success(fixmgr):
    """Perform a fixture create with successful create action"""
    created_instance, service = _create_instance(fixmgr)

    # do not allow fixture to mark active or be deleted while it is operating
    with pytest.raises(AXIllegalOperationException):
        http_put('/v1/fixture/instances/{}'.format(created_instance['id']), data={'status': InstanceStatus.ACTIVE})
    with pytest.raises(AXIllegalOperationException):
        http_delete('/v1/fixture/instances/{}'.format(created_instance['id']))

    # notify fixture manager it was successful
    service['status'] = ServiceStatus.SUCCESS
    http_post('/v1/fixture/action_result', data=service)

    # now the fixture should be in active state without operation
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.ACTIVE
    assert not get_instance['operation']


def test_fixture_instance_create_action_failure(fixmgr):
    """Perform a fixture create with failed create action"""
    created_instance, service = _create_instance(fixmgr)

    # do not allow fixture to mark active or be deleted while it is operating
    with pytest.raises(AXIllegalOperationException):
        http_put('/v1/fixture/instances/{}'.format(created_instance['id']), data={'status': InstanceStatus.ACTIVE})
    with pytest.raises(AXIllegalOperationException):
        http_delete('/v1/fixture/instances/{}'.format(created_instance['id']))

    # notify fixture manager action failed
    service['status'] = ServiceStatus.FAILED
    http_post('/v1/fixture/action_result', data=service)

    # now the fixture should be in create_error state without operation
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.CREATE_ERROR
    assert not get_instance['operation']

    # allow us mark active from create error
    http_put('/v1/fixture/instances/{}'.format(created_instance['id']), data={'status': InstanceStatus.ACTIVE})
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.ACTIVE


def test_fixture_instance_create_mark_active(fixmgr):
    """Tests ability to create a fixture but bypass create action by setting status to active"""
    testdata.populate_class(fixmgr)
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'status': InstanceStatus.ACTIVE,
        'attributes': {'group': 'dev'},
    }
    created_instance = http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)
    assert created_instance['status'] == InstanceStatus.ACTIVE
    assert not created_instance['operation']

    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.ACTIVE
    assert not get_instance['operation']


def test_fixture_instance_delete_action_success(fixmgr):
    """Test behavior when delete is performed and delete was successful"""
    created_instance = _create_instance_action_success(fixmgr)
    http_delete('/v1/fixture/instances/{}'.format(created_instance['id']))

    # now the fixture should be in deleting with operation
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.DELETING and get_instance['operation']

    # do not allow mark deleted while deleting
    with pytest.raises(AXIllegalOperationException):
        http_put('/v1/fixture/instances/{}'.format(created_instance['id']), data={'status': InstanceStatus.DELETED})

    # notify fixture manager delete success
    service = fixmgr.axops_client.get_service(get_instance['operation']['id'])
    service['status'] = ServiceStatus.SUCCESS
    http_post('/v1/fixture/action_result', data=service)

    # now the fixture should be in deleted state without operation
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.DELETED and not get_instance['operation']

    # verify idempotent delete
    http_delete('/v1/fixture/instances/{}'.format(created_instance['id']))

def test_fixture_instance_delete_action_failure(fixmgr):
    """Test behavior when delete is performed and delete failed"""
    created_instance = _create_instance_action_success(fixmgr)
    http_delete('/v1/fixture/instances/{}'.format(created_instance['id']))

    # now the fixture should be in deleting with operation
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.DELETING and get_instance['operation']

    # do not allow mark deleted while deleting
    with pytest.raises(AXIllegalOperationException):
        http_put('/v1/fixture/instances/{}'.format(created_instance['id']), data={'status': InstanceStatus.DELETED})

    # notify fixture manager delete failed
    service = fixmgr.axops_client.get_service(get_instance['operation']['id'])
    service['status'] = ServiceStatus.FAILED
    http_post('/v1/fixture/action_result', data=service)

    # now the fixture should be in delete_error state without operation
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.DELETE_ERROR and not get_instance['operation']

    # allow us to mark instance as deleted from delete_error state
    http_put('/v1/fixture/instances/{}'.format(created_instance['id']), data={'status': InstanceStatus.DELETED})
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.DELETED and not get_instance['operation']


def test_fixture_instance_mark_deleted(fixmgr):
    """Tests ability to mark a fixture as deleted from active state"""
    created_instance = _create_instance_action_success(fixmgr)
    http_put('/v1/fixture/instances/{}'.format(created_instance['id']), data={'status': InstanceStatus.DELETED})
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['status'] == InstanceStatus.DELETED and not get_instance['operation']


def test_fixture_instance_name_uniqueness_create(fixmgr):
    """Try to create two instances with the same name. The second should fail"""
    created_instance = _create_instance_action_success(fixmgr)
    with pytest.raises(AXApiInvalidParam):
        _create_instance_action_success(fixmgr)
    instances = http_get('/v1/fixture/instances')['data']
    assert len(instances) == 1
    assert instances[0]['id'] == created_instance['id']


def test_fixture_instance_name_uniqueness_rename(fixmgr):
    """Try to rename an instance with existing name, the update should fail"""
    created_instance1 = _create_instance_action_success(fixmgr, name='instance1')
    created_instance2 = _create_instance_action_success(fixmgr, name='instance2')
    with pytest.raises(AXApiInvalidParam):
        http_put('/v1/fixture/instances/{}'.format(created_instance2['id']), data={'name': 'instance1'})
    get_instance2 = http_get('/v1/fixture/instances/{}'.format(created_instance2['id']))
    assert get_instance2['name'] == 'instance2'


def test_fixture_instance_action_on_success_disable_enable(fixmgr):
    """Tests ability to disable/enable"""
    created_instance = _create_instance_action_success(fixmgr)

    # perform 'suspend' action
    suspend_instance = http_post('/v1/fixture/instances/{}/action'.format(created_instance['id']), data={'action': 'suspend'}, headers=testdata.USER_SESSION_HEADERS)
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert suspend_instance == get_instance
    assert suspend_instance['enabled']

    # simulate 'suspend' action success
    service = fixmgr.axops_client.get_service(get_instance['operation']['id'])
    service['status'] = ServiceStatus.SUCCESS
    http_post('/v1/fixture/action_result', data=service)

    # Verify on_success: disable is honored
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['enabled'] == False

    # perform 'resume' action and simulate success
    suspend_instance = http_post('/v1/fixture/instances/{}/action'.format(created_instance['id']), data={'action': 'resume'}, headers=testdata.USER_SESSION_HEADERS)
    service = fixmgr.axops_client.get_service(suspend_instance['operation']['id'])
    service['status'] = ServiceStatus.SUCCESS
    http_post('/v1/fixture/action_result', data=service)

    # Verify on_success: enable is honored
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['enabled'] == True


def test_fixture_instance_action_on_failure_disable(fixmgr):
    """Tests ability to disable/enable"""
    created_instance = _create_instance_action_success(fixmgr)

    # perform 'health_check_fail' action
    suspend_instance = http_post('/v1/fixture/instances/{}/action'.format(created_instance['id']), data={'action': 'health_check_fail'}, headers=testdata.USER_SESSION_HEADERS)
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert suspend_instance == get_instance
    assert suspend_instance['enabled']

    # simulate 'health_check_fail' action success. this should not enabled state
    service = fixmgr.axops_client.get_service(get_instance['operation']['id'])
    service['status'] = ServiceStatus.SUCCESS
    http_post('/v1/fixture/action_result', data=service)
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['enabled'] == True

    # perform 'health_check_fail' action
    suspend_instance = http_post('/v1/fixture/instances/{}/action'.format(created_instance['id']), data={'action': 'health_check_fail'}, headers=testdata.USER_SESSION_HEADERS)
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert suspend_instance == get_instance
    assert suspend_instance['enabled']

    # simulate 'health_check_fail' action success. verify we changed enabled to be false
    service = fixmgr.axops_client.get_service(get_instance['operation']['id'])
    service['status'] = ServiceStatus.FAILED
    http_post('/v1/fixture/action_result', data=service)
    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['enabled'] == False


def test_fixture_attribute_required(fixmgr):
    """Verify we raise error if we do not supply a required attribute"""
    testdata.populate_class(fixmgr)
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
    }
    with pytest.raises(AXApiInvalidParam):
        http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)


def test_fixture_attribute_invalid_option(fixmgr):
    """Verify we raise error if attribute was not a valid option"""
    testdata.populate_class(fixmgr)
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'group': 'random'
    }
    with pytest.raises(AXApiInvalidParam):
        http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)

def test_fixture_attribute_default_value(fixmgr):
    """Verify default values are set when not supplied"""
    testdata.populate_class(fixmgr)
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {
            'group': 'dev',
        }
    }
    created_instance = http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)
    assert created_instance['attributes']['cpu_cores'] == 1
    assert created_instance['attributes']['instance_type'] == 'm3.large'
    assert created_instance['attributes']['memory_gib'] == 4

def test_fixture_attribute_parse_strings(fixmgr):
    """Verify strings can be supplied instead of ints/floats/bools"""
    testdata.populate_class(fixmgr)
    tags = ['foo', 1, True, 3.141592654]
    parsed_tags = ['foo', '1', 'true', '3.141592654']
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {
            'group': 'dev',
            'cpu_cores': '2',
            'disable_nightly': 'false',
            'tags': tags,
        }
    }
    created_instance = http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)
    assert created_instance['attributes']['cpu_cores'] == 2
    assert created_instance['attributes']['disable_nightly'] == False
    assert created_instance['attributes']['tags'] == parsed_tags

def test_fixture_attribute_array(fixmgr):
    """Verify attributes of type array is supported"""
    testdata.populate_class(fixmgr)
    tags = ['foo', 'bar']
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {
            'group': 'dev',
            'tags': tags,
        }
    }
    created_instance = http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)
    assert created_instance['attributes']['tags'] == tags


def test_fixture_attribute_invalid_types(fixmgr):
    """Verifies when user supplies invalid data types, fixture creation fails"""
    testdata.populate_class(fixmgr)
    invalid_types = [
        # invalid ints
        ('cpu_cores', True),
        ('cpu_cores', 'foo'),
        ('cpu_cores', 3.141),
        # invalid bools
        ('disable_nightly', 1),
        ('disable_nightly', 'foo'),
        ('disable_nightly', 3.141),
        # non array
        ('tags', 'foo')
    ]
    for field, val in invalid_types:
        create_payload = {
            'name': 'instance1',
            'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
            'attributes': {
                'group': 'dev',
            }
        }
        create_payload['attributes'][field] = val
        with pytest.raises(AXApiInvalidParam):
            http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)


def test_fixture_attributes_ignore_extra_fields(fixmgr):
    """Verifies manager ignores extra fields supplied during fixture creation"""
    testdata.populate_class(fixmgr)
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {
            'group': 'dev',
            'extra': 'field',
        }
    }
    created_instance = http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)
    assert 'extra' not in created_instance['attributes']

    updated_instance = http_put('/v1/fixture/instances/{}'.format(created_instance['id']),
                                data=create_payload, headers=testdata.USER_SESSION_HEADERS)
    assert 'extra' not in updated_instance['attributes']


def test_fixture_attributes_updates(fixmgr):
    """Verifies during update, we ignore fields which cannot be updated, and we update mtime"""
    testdata.populate_class(fixmgr)
    created_instance = _create_instance_action_success(fixmgr)
    with pytest.raises(AXApiInvalidParam):
        updated_instance = http_put('/v1/fixture/instances/{}'.format(created_instance['id']),
                                    data={'id': str(uuid.uuid4())}, headers=testdata.USER_SESSION_HEADERS)

    new_time = 1234567
    updates = {
        'class_id' : str(uuid.uuid4()),
        'class_name' : 'foo',
        'attributes' : {
        },
        'mtime': new_time,
        'ctime': new_time,
        'atime': new_time,
        'status_detail' : {'foo': 'bar'},
        'operation': {'foo': 'bar'},
        'referrers': [{'foo': 'bar'}],
        'description': "new description",
        'enabled': False,
        'disable_reason': "because",
        'concurrency': 0,
    }
    # sleep 1 second so that we can ensure mtime changes
    time.sleep(1.2)
    updated_instance = http_put('/v1/fixture/instances/{}'.format(created_instance['id']),
                                data=updates, headers=testdata.USER_SESSION_HEADERS)
    # ensure updates to the following fields were not allowed, including ctime/atime
    for field in ['class_name', 'class_id', 'attributes', 'status_detail', 'operation', 'referrers', 'ctime', 'atime']:
        assert updated_instance[field] == created_instance[field]
    # ensure user cannot explicity set times
    for field in ['mtime', 'ctime', 'atime']:
        assert updated_instance[field] != new_time
    # ensure other fields were updated, and mtime was updated
    for field in ['description', 'enabled', 'disable_reason', 'concurrency', 'mtime']:
        assert updated_instance[field] != created_instance[field]


def test_fixture_attributes_as_artifact(fixmgr, monkeypatch):
    """Verifies if service in the operation has an attributes artifact, we can download and update the attributes of the instance"""
    testdata.populate_class(fixmgr)
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {'group': 'dev'},
    }
    created_instance = http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)

    artifact_id = str(uuid.uuid4())
    def _mock_search_artifacts(*args, **kwargs):
        return [{'artifact_id': artifact_id}]

    def _mock_download_artifact_json(aid):
        assert aid == artifact_id
        return {'ip_address': '1.2.3.4'}

    monkeypatch.setattr(fixmgr.axops_client, 'search_artifacts', _mock_search_artifacts)
    monkeypatch.setattr(fixmgr, 'download_artifact_json', _mock_download_artifact_json)

    service = fixmgr.axops_client.get_service(created_instance['operation']['id'])
    service['status'] = ServiceStatus.SUCCESS
    http_post('/v1/fixture/action_result', data=service)

    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['attributes']['ip_address'] == '1.2.3.4'


def test_fixture_attributes_bad_artifact(fixmgr, monkeypatch):
    """Verifies if the job produced a bad artifact, we send an alert and do not accept the artifact"""
    testdata.populate_class(fixmgr)
    create_payload = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {'group': 'dev'},
    }
    created_instance = http_post('/v1/fixture/instances', data=create_payload, headers=testdata.USER_SESSION_HEADERS)

    artifact_id = str(uuid.uuid4())
    def _mock_search_artifacts(*args, **kwargs):
        return [{'artifact_id': artifact_id}]

    def _mock_download_artifact_json(aid):
        assert aid == artifact_id
        return {
            'ip_address': '1.2.3.4',
            'disable_nightly': 0,
            'memory_gib': "1234",
        }

    notification_sent = False
    def _mock_send_message_to_notification_center(*args, **kwargs):
        nonlocal notification_sent
        notification_sent = True

    monkeypatch.setattr(fixmgr.axops_client, 'search_artifacts', _mock_search_artifacts)
    monkeypatch.setattr(fixmgr, 'download_artifact_json', _mock_download_artifact_json)
    monkeypatch.setattr(fixmgr.notification_center, 'send_message_to_notification_center', _mock_send_message_to_notification_center)

    service = fixmgr.axops_client.get_service(created_instance['operation']['id'])
    service['status'] = ServiceStatus.FAILED
    http_post('/v1/fixture/action_result', data=service)

    get_instance = http_get('/v1/fixture/instances/{}'.format(created_instance['id']))
    assert get_instance['attributes']['ip_address'] == '1.2.3.4'
    assert 'disable_nightly' not in get_instance['attributes']
    assert notification_sent, "Notification was not sent"
    assert get_instance['attributes']['memory_gib'] == 1234, "parser was not lenient to a numeric string value"

def test_fixture_instance_query(fixmgr):
    """Tests various querying capabilities against instances"""
    testdata.populate_linux_instances(fixmgr)

    # check exact match or in array
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : 4096}))
    assert len(res) == 1
    assert res[0].name == 'linux-01'
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : [4096]}))
    assert len(res) == 1
    assert res[0].name == 'linux-01'
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : [1024, 4096]}))
    assert len(res) == 2
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : '1024,4096'}))
    assert len(res) == 2

    # check int array
    res = list(fixmgr.query_fixture_instances({'attributes.int_array' : 0}))
    assert len(res) == 1
    assert res[0].name == 'linux-01'
    res = list(fixmgr.query_fixture_instances({'attributes.int_array' : [0]}))
    assert len(res) == 1
    assert res[0].name == 'linux-01'
    res = list(fixmgr.query_fixture_instances({'attributes.int_array' : [0, 1]}))
    assert len(res) == 3

    # check when nothing exists
    res = list(fixmgr.query_fixture_instances({'attributes.int_type' : 123}))
    assert not res
    res = list(fixmgr.query_fixture_instances({'attributes.int_type' : [123]}))
    assert not res

    res = list(fixmgr.query_fixture_instances({'attributes.int_array' : 123}))
    assert not res
    res = list(fixmgr.query_fixture_instances({'attributes.int_array' : [123]}))
    assert not res

    # check for null and empty list values
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : None}))
    assert not res
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : [None]}))
    assert not res
    res = list(fixmgr.query_fixture_instances({'attributes.int_array' : None}))
    assert not res
    res = list(fixmgr.query_fixture_instances({'attributes.int_array' : [None]}))
    assert not res
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : []}))
    assert not res
    res = list(fixmgr.query_fixture_instances({'attributes.int_array' : []}))
    assert not res

    # check operators
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : 'lte:1024'}))
    assert len(res) == 1 and res[0].name == 'linux-04'
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : 'lt:1024'}))
    assert not res
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : 'gte:8192'}))
    assert len(res) == 2
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : 'gt:8192'}))
    assert not res
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : 'ne:1024'}))
    assert len(res) == 3
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : 'eq:1024'}))
    assert len(res) == 1
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : 'in:1023,1024,1025'}))
    assert len(res) == 1
    res = list(fixmgr.query_fixture_instances({'attributes.memory_mb' : 'nin:1023,1024,1025'}))
    assert len(res) == 3

def test_fixture_summary(fixmgr):
    """Tests fixture summary endpoint"""
    # populate instances
    testdata.populate_linux_instances(fixmgr)
    testdata.populate_class(fixmgr)
    _create_instance_action_success(fixmgr)

    # mark one instance as deleted (summary should exclude the disabled instance)
    linux01 = http_get('/v1/fixture/instances?name=linux-01')['data'][0]
    http_put('/v1/fixture/instances/{}'.format(linux01['id']), data={'status': InstanceStatus.DELETED})

    # if no filters supplied, will fall into a "all" group
    summary = http_get('/v1/fixture/summary')
    assert summary['all']['total'] == 4
    assert summary['all']['available'] == 3

    # verify we can group by class_name
    summary = http_get('/v1/fixture/summary?group_by=class_name')
    assert summary['class_name:Linux']['total'] == 3
    assert summary['class_name:Linux']['available'] == 2
    assert summary['class_name:test-fixture']['total'] == 1
    assert summary['class_name:test-fixture']['available'] == 1

    # verify we can use a filter
    summary = http_get('/v1/fixture/summary?class_name=test-fixture')
    assert summary['all']['available'] == 1
    assert summary['all']['total'] == 1

    # verify we can use a filter + grouping
    summary = http_get('/v1/fixture/summary?class_name=Linux&group_by=attributes.os_version')
    expected_val = {
        'attributes.os_version:RHEL 7': {'available': 1, 'total': 1},
        'attributes.os_version:Ubuntu 16.04': {'available': 1, 'total': 1},
        'attributes.os_version:Debian 8': {'available': 0, 'total': 1}
    }
    assert summary == expected_val

    # verify we can group by multiple values
    summary = http_get('/v1/fixture/summary?class_name=Linux&group_by=attributes.os_version,attributes.memory_mb')
    expected_val = {
        'attributes.memory_mb:1024,attributes.os_version:Debian 8': {'total': 1, 'available': 0},
        'attributes.memory_mb:8192,attributes.os_version:Ubuntu 16.04': {'total': 1, 'available': 1},
        'attributes.memory_mb:8192,attributes.os_version:RHEL 7': {'total': 1, 'available': 1}
    }
    assert summary == expected_val
