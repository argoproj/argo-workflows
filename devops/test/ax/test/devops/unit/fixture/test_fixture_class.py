"""
Unit tests for fixture classes
"""

import pytest

from ax.exceptions import AXApiResourceNotFound, AXIllegalOperationException, AXApiInvalidParam
from ax.devops.fixture.common import FixtureClassStatus, InstanceStatus
from . import testdata
from .testutil import http_get, http_post, http_put, http_delete, wait_for_assignment, fake_volume_atime, wait_for_processor


def test_rest_ping(fixmgr):
    """Verifies ping health check"""
    assert http_get('/v1/fixture/ping') == "pong", "Failed to ping"

def test_fixture_class_create_delete(fixmgr):
    """Tests creating a class from a template and deleting the class"""
    testdata.populate_templates(fixmgr)

    create_payload = {
        'template_id': testdata.TEST_FIXTURE_TEMPLATE['id']
    }
    created_class = http_post('/v1/fixture/classes', data=create_payload)
    for field in ['name', 'description', 'repo', 'branch', 'attributes', 'actions']:
        assert created_class[field] == testdata.TEST_FIXTURE_TEMPLATE[field]

    get_class = http_get('/v1/fixture/classes/{}'.format(created_class['id']))
    assert created_class == get_class

    all_classes = http_get('/v1/fixture/classes')
    assert all_classes['data'][0] == created_class

    # Ensure create is idempotent
    created_class2 = http_post('/v1/fixture/classes', data=create_payload)
    assert created_class2 == get_class

    all_classes = http_get('/v1/fixture/classes')
    assert all_classes['data'][0] == created_class

    res = http_delete('/v1/fixture/classes/{}'.format(created_class['id']))
    assert res['id'] == created_class['id']

    with pytest.raises(AXApiResourceNotFound):
        http_get('/v1/fixture/classes/{}'.format(created_class['id']))

    all_classes = http_get('/v1/fixture/classes')
    assert len(all_classes['data']) == 0

    # Verify delete is idempotent
    http_delete('/v1/fixture/classes/{}'.format(created_class['id']))


def test_fixture_class_create_duplicate_name(fixmgr):
    """Tests creating a class with the same name as a class already enabled"""
    testdata.populate_templates(fixmgr)

    create_payload = {
        'template_id': testdata.TEST_FIXTURE_TEMPLATE['id']
    }
    created_class = http_post('/v1/fixture/classes', data=create_payload)

    # attempt to enable class with same name from different branch
    create_payload2 = {
        'template_id': testdata.TEST_FIXTURE_TEMPLATE_BRANCH2['id']
    }
    with pytest.raises(AXIllegalOperationException):
        http_post('/v1/fixture/classes', data=create_payload2)


def test_fixture_class_update_duplicate_name(fixmgr):
    """Enable two classes perform update on a class using a template with the same name as one that is already existing"""
    testdata.populate_templates(fixmgr)

    http_post('/v1/fixture/classes', data={'template_id': testdata.TEST_FIXTURE_TEMPLATE['id']})
    class2 = http_post('/v1/fixture/classes', data={'template_id': testdata.TEST_FIXTURE_TEMPLATE_NAME2['id']})

    # attempt to update class to a template with same name as one that is already enabled
    update_payload = {
        'template_id': testdata.TEST_FIXTURE_TEMPLATE_BRANCH2['id']
    }
    with pytest.raises(AXIllegalOperationException):
        http_put('/v1/fixture/classes/{}'.format(class2['id']), data=update_payload)

def test_fixture_class_prevent_delete(fixmgr):
    """Ensure we prevent deletion of a fixture class if there are fixture instances"""
    testdata.populate_templates(fixmgr)

    fixclass = http_post('/v1/fixture/classes', data={'template_id': testdata.TEST_FIXTURE_TEMPLATE['id']})

    # create an instance
    create_instance = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {'group': 'dev'},
    }
    http_post('/v1/fixture/instances', data=create_instance, headers=testdata.USER_SESSION_HEADERS)

    with pytest.raises(AXIllegalOperationException):
        http_delete('/v1/fixture/classes/{}'.format(fixclass['id']))


def test_fixture_disconnected_class(fixmgr, monkeypatch):
    """Tests when template disappears, we alert user"""
    testdata.populate_templates(fixmgr)
    template_id = testdata.TEST_FIXTURE_TEMPLATE['id']

    fixclass = http_post('/v1/fixture/classes', data={'template_id': template_id})

    # create an instance
    create_instance = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'status': InstanceStatus.ACTIVE,
        'attributes': {'group': 'qa'},
    }
    instance = http_post('/v1/fixture/instances', data=create_instance, headers=testdata.USER_SESSION_HEADERS)

    # simulate disconnected template
    fixmgr.axops_client._fixture_templates.pop(template_id)

    # notify fixturemanager of the change
    notification_sent = False
    def _mock_send_message_to_notification_center(*args, **kwargs):
        nonlocal notification_sent
        notification_sent = True
    monkeypatch.setattr(fixmgr.notification_center, 'send_message_to_notification_center', _mock_send_message_to_notification_center)

    http_post('/v1/fixture/template_updates')
    assert notification_sent, "notification about disconnected template not sent"

    fixclass_updated = http_get('/v1/fixture/classes/{}'.format(fixclass['id']))
    assert fixclass_updated['status'] == FixtureClassStatus.DISCONNECTED

    # verify we can still perform actions when it is disconnected and cannot find the action template
    def _mock_get_template(*args, **kwargs):
        return None
    monkeypatch.setattr(fixmgr.axops_client, 'get_template', _mock_get_template)
    http_post('/v1/fixture/instances/{}/action'.format(instance['id']), data={'action':'snapshot'}, headers=testdata.USER_SESSION_HEADERS)

    # Add back the template verify status is cleared
    fixmgr.axops_client._fixture_templates[template_id] = testdata.TEST_FIXTURE_TEMPLATE
    monkeypatch.undo()
    http_post('/v1/fixture/template_updates')

    fixclass_updated = http_get('/v1/fixture/classes/{}'.format(fixclass['id']))
    assert fixclass_updated['status'] == FixtureClassStatus.ACTIVE
