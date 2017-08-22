"""
Unit tests for correcting database inconsistency
"""
import logging
import uuid

import pytest

from .testdata import TEST_LINUX_INSTANCES, populate_linux_instances
from .testutil import wait_for_assignment, http_get, http_put, http_post
from ax.devops.fixture.instance import FixtureInstance, lock_instance
from ax.devops.fixture.common import FIX_REQUESTER_AXWORKFLOWADC, ServiceStatus
from ax.exceptions import AXException

logger = logging.getLogger(__name__)

_mock_categories = None

def test_initdb(fixmgr, monkeypatch):
    """Verifies FixtureManager::initdb() populates its instances from axdb"""
    populate_linux_instances(fixmgr)
    instances = [f.axdbdoc() for f in fixmgr.query_fixture_instances()]

    def _mock_get_instances():
        return instances

    fixmgr.instances.drop()
    assert not list(fixmgr.query_fixture_instances()), "Setup error"

    monkeypatch.setattr(fixmgr.axops_client, 'get_fixture_instances', _mock_get_instances)
    fixmgr.initdb()
    instances = list(fixmgr.query_fixture_instances())
    assert instances, "Fixture instances not populated from axops"
    expected_fix_names = set([f['name'] for f in TEST_LINUX_INSTANCES])
    actual_fix_names = set([f.name for f in instances])
    assert actual_fix_names == expected_fix_names

    for f in instances:
        assert f.attributes

def test_axdb_update_failure(fixmgr, monkeypatch):
    """Tests if axdb is temporarily down when persisting update, we undo any changes to the cache"""
    populate_linux_instances(fixmgr)
    def _raise_error(*args, **kwargs):
        raise Exception("Simulated request error")
    monkeypatch.setattr(fixmgr.axdb_client, 'update_fixture_instance', _raise_error)

    linux01 = http_get('/v1/fixture/instances?name=linux-01')['data'][0]
    with pytest.raises(AXException):
        http_put('/v1/fixture/instances/{}'.format(linux01['id']), data={'enabled': False})

    assert http_get('/v1/fixture/instances?name=linux-01')['data'][0]['enabled'] == True

    # Undo error injection and try again
    monkeypatch.undo()
    http_put('/v1/fixture/instances/{}'.format(linux01['id']), data={'enabled': False})
    assert http_get('/v1/fixture/instances?name=linux-01')['data'][0]['enabled'] == False


def test_repair_extra_referrers(fixmgr):
    """Verifies we correct referrers of instances when fixture request is gone"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'name' : 'linux-01'
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=fix_req_json)
    # wait for fixture to be assigned
    assignment = wait_for_assignment(fixmgr, fix_req['service_id'])
    fix = assignment['assignment']['fix1']
    assert http_get('/v1/fixture/instances/{}'.format(fix['id']))['referrers'], "Setup error"
    # remove fixture request from backend
    fixmgr.reqproc.requestdb.remove(fix_req['service_id'])
    assert http_get('/v1/fixture/instances/{}'.format(fix['id']))['referrers'], "Setup error"
    fixmgr.check_consistency()
    assert not http_get('/v1/fixture/instances/{}'.format(fix['id']))['referrers']

def test_repair_missing_referrers(fixmgr, monkeypatch):
    """Verifies we will add missing referrers for assigned fixture requests"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()
    root_workflow_id = str(uuid.uuid4())
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : root_workflow_id,
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'name' : 'linux-01'
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=fix_req_json)
    # wait for fixture to be assigned
    assignment = wait_for_assignment(fixmgr, fix_req['service_id'])
    fix = assignment['assignment']['fix1']
    assert http_get('/v1/fixture/instances/{}'.format(fix['id']))['referrers'], "Setup error"

    # artificially empty out the referrers
    instance = fixmgr.get_fixture_instance(id=fix['id'])
    instance.referrers = []
    with lock_instance(fix['id']):
        fixmgr._persist_instance_updates(instance)
    assert not http_get('/v1/fixture/instances/{}'.format(fix['id']))['referrers'], "Setup error"

    # simulates an active, running job
    def _mock_get_services(*args, **kwargs):
        return [{'id' : root_workflow_id, 'status' : ServiceStatus.RUNNING}]
    monkeypatch.setattr(fixmgr.axops_client, 'get_services', _mock_get_services)

    fixmgr.check_consistency()
    assert http_get('/v1/fixture/instances/{}'.format(fix['id']))['referrers'], "referrers not corrected"

def test_release_orphaned_reservations(fixmgr):
    """Verifies we can delete fixture requests/assignments for completed jobs"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'name' : 'linux-01'
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=fix_req_json)
    # wait for fixture to be assigned
    assignment = wait_for_assignment(fixmgr, fix_req['service_id'])
    fix = assignment['assignment']['fix1']
    assert http_get('/v1/fixture/instances/{}'.format(fix['id']))['referrers'], "Setup error"

    fixmgr.check_consistency()
    assert not http_get('/v1/fixture/instances/{}'.format(fix['id']))['referrers'], "referrers not corrected"

def test_axops_release_orphaned_assignment_channels(fixmgr):
    """Verifies we delete assignment channels that no longer have an associated fixture request"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()
    running_service_id = str(uuid.uuid4())
    root_workflow_id = str(uuid.uuid4())
    # simulates an active, running job
    fixmgr.axops_client._services[root_workflow_id] = {'id' : root_workflow_id, 'status' : ServiceStatus.RUNNING}

    fix_req_json = {
        'service_id' : running_service_id,
        'root_workflow_id' : root_workflow_id,
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'name' : 'linux-01'
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=fix_req_json)
    # add a bogus, orphaned assignment channel
    orphaned_service_id = str(uuid.uuid4())
    fixmgr.reqproc.redis_client_notification.rpush('notification:' + orphaned_service_id, "asdf")
    fixmgr.check_consistency()
    # Verify the bogus notification channel was deleted
    assert not wait_for_assignment(fixmgr, orphaned_service_id, timeout=1, verify_exists=False)
    # but it should not have deleted the valid channel
    assert wait_for_assignment(fixmgr, fix_req['service_id'])
