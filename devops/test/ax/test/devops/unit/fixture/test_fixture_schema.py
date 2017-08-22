"""
Tests schema changes to fixture classes
"""

import pytest

from ax.exceptions import AXApiResourceNotFound, AXIllegalOperationException, AXApiInvalidParam
from . import testdata
from .testutil import http_get, http_post, http_put, http_delete, wait_for_assignment, fake_volume_atime, wait_for_processor

def test_fixture_schema_new_class_name(fixmgr):
    """Ensure we can rename a fixture class (by rebasing) and its instances will receive the new name"""
    testdata.populate_templates(fixmgr)

    fixclass = http_post('/v1/fixture/classes', data={'template_id': testdata.TEST_FIXTURE_TEMPLATE['id']})

    # create an instance
    create_instance = {
        'name': 'instance1',
        'class_name': testdata.TEST_FIXTURE_TEMPLATE['name'],
        'attributes': {'group': 'dev'},
    }
    instance = http_post('/v1/fixture/instances', data=create_instance, headers=testdata.USER_SESSION_HEADERS)

    # reassociate class with different name
    http_put('/v1/fixture/classes/{}'.format(fixclass['id']),
             data={'template_id': testdata.TEST_FIXTURE_TEMPLATE_NAME2['id']})

    # verify the created instance has the new name
    instance_get = http_get('/v1/fixture/instances/{}'.format(instance['id']))
    assert instance_get['class_name'] == testdata.TEST_FIXTURE_TEMPLATE_NAME2['name']


def test_fixture_schema_migration(fixmgr):
    """Tests when schema is updated with new/removed attributes, change is reflected in instances"""
    testdata.populate_templates(fixmgr)
    template_id = testdata.SCHEMA_TEST_1['id']

    fixclass = http_post('/v1/fixture/classes', data={'template_id': template_id})

    for field in ['to_delete', 'str_to_int', 'str_to_arr']:
        assert field in fixclass['attributes'], "setup error"

    # create an instance
    create_instance = {
        'name': 'instance1',
        'class_name': testdata.SCHEMA_TEST_1['name'],
    }
    instance = http_post('/v1/fixture/instances', data=create_instance, headers=testdata.USER_SESSION_HEADERS)
    for field in ['to_delete', 'str_to_int', 'str_to_arr']:
        assert field in instance['attributes'], "setup error"

    # simulate a schema change and notify fixturemanager of the change
    fixmgr.axops_client._fixture_templates[template_id] = testdata.SCHEMA_TEST_2
    http_post('/v1/fixture/template_updates')

    # verify fixture class was updated
    fixclass_updated = http_get('/v1/fixture/classes/{}'.format(fixclass['id']))
    assert 'to_delete' not in fixclass_updated['attributes']
    assert fixclass_updated['attributes']['str_to_int']['type'] == 'int'
    assert fixclass_updated['attributes']['str_to_arr']['type'] == 'string' and fixclass_updated['attributes']['str_to_arr']['flags'] == 'array'

    # verify instance reflects schema change (attributes deleted)
    instance_get = http_get('/v1/fixture/instances/{}'.format(instance['id']), headers=testdata.USER_SESSION_HEADERS)
    for field in ['to_delete', 'str_to_int', 'str_to_arr']:
        assert field not in instance_get['attributes'], "{} was not deleted".format(field)

    # create another instance
    create_instance2 = {
        'name': 'instance2',
        'class_name': testdata.SCHEMA_TEST_1['name'],
    }
    instance2 = http_post('/v1/fixture/instances', data=create_instance2, headers=testdata.USER_SESSION_HEADERS)
    for field in ['new_field', 'str_to_int', 'str_to_arr']:
        assert field in instance2['attributes']
    assert 'to_delete' not in instance2


def test_fixture_schema_migration_axdb_error(fixmgr, monkeypatch):
    """Verifies if axdb error happens when persisting batch updates to instances, instance state will still be consistent and next template update will apply the update"""
    testdata.populate_templates(fixmgr)
    template_id = testdata.SCHEMA_TEST_1['id']

    fixclass = http_post('/v1/fixture/classes', data={'template_id': template_id})

    for field in ['to_delete', 'str_to_int', 'str_to_arr']:
        assert field in fixclass['attributes'], "setup error"

    # create an instance
    create_instance = {
        'name': 'instance1',
        'class_name': testdata.SCHEMA_TEST_1['name'],
    }
    instance = http_post('/v1/fixture/instances', data=create_instance, headers=testdata.USER_SESSION_HEADERS)
    for field in ['to_delete', 'str_to_int', 'str_to_arr']:
        assert field in instance['attributes'], "setup error"

    # simulate axdb error when persisting an instance to axdb
    def _raise_error(*args, **kwargs):
        raise Exception("Simulated request error")
    monkeypatch.setattr(fixmgr.axdb_client, 'update_fixture_instance', _raise_error)

    # simulate a schema change and notify fixturemanager of the change
    fixmgr.axops_client._fixture_templates[template_id] = testdata.SCHEMA_TEST_2
    http_post('/v1/fixture/template_updates')

    # verify fixture class was not updated (because we update class as the last step)
    fixclass_updated = http_get('/v1/fixture/classes/{}'.format(fixclass['id']))
    assert 'to_delete' in fixclass_updated['attributes']
    assert fixclass_updated['attributes']['str_to_int']['type'] == 'string'
    assert fixclass_updated['attributes']['str_to_arr']['type'] == 'string' and not fixclass_updated['attributes']['str_to_arr'].get('flags')

    # verify instance does not reflect schema change
    instance_get = http_get('/v1/fixture/instances/{}'.format(instance['id']), headers=testdata.USER_SESSION_HEADERS)
    for field in ['to_delete', 'str_to_int', 'str_to_arr']:
        assert field in instance_get['attributes'], "field was unexpectedly deleted"

    # Undo error injection and notify fixturemanager about a schema change
    monkeypatch.undo()
    fixmgr.notify_template_updates()

    # verify fixture class was updated this time
    fixclass_updated = http_get('/v1/fixture/classes/{}'.format(fixclass['id']))
    assert 'to_delete' not in fixclass_updated['attributes']
    assert fixclass_updated['attributes']['str_to_int']['type'] == 'int'
    assert fixclass_updated['attributes']['str_to_arr']['type'] == 'string' and fixclass_updated['attributes']['str_to_arr']['flags'] == 'array'

    # verify instance reflects schema change
    instance_get = http_get('/v1/fixture/instances/{}'.format(instance['id']), headers=testdata.USER_SESSION_HEADERS)
    for field in ['to_delete', 'str_to_int', 'str_to_arr']:
        assert field not in instance_get['attributes'], "{} was not deleted".format(field)

