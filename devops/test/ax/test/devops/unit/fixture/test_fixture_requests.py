"""
Fixture requests unit tests
"""
import copy
import logging
import re
import uuid

import pytest

from .testdata import populate_linux_instances, USER_SESSION_HEADERS
from .testutil import wait_for_processor, wait_for_assignment, http_post, http_get, http_delete, http_put
from ax.exceptions import AXIllegalArgumentException, AXApiResourceNotFound, AXApiInvalidParam
from ax.devops.fixture.request import FixtureRequest
from ax.devops.fixture.common import FIX_REQUESTER_AXWORKFLOWADC

logger = logging.getLogger(__name__)

def test_request_crud(fixmgr):
    """Verify CRUD operations on fixture requests"""
    populate_linux_instances(fixmgr)
    assert not http_get('/v1/fixture/requests')['data'], "Test setup error, fixture requestdb not purged"
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'RHEL 7'
                }
            }
        }
    }
    # Test create
    http_post('/v1/fixture/requests', data=fix_req_json)
    fix_req = http_get('/v1/fixture/requests')['data'][0]
    assert fix_req['service_id'] == fix_req_json['service_id']
    assert fix_req['requirements'] == fix_req_json['requirements']

    # Test get
    fix_req_get = http_get('/v1/fixture/requests/{}'.format(fix_req_json['service_id']))
    assert fix_req == fix_req_get

    # Test update
    fake_assignment = {'fix1' : {'id' : str(uuid.uuid4()), 'name' : 'assigned fixture'}}
    fix_reqdb_get = fixmgr.reqproc.requestdb.get(fix_req_json['service_id'])
    fix_reqdb_get.assignment = fake_assignment
    fixmgr.reqproc.requestdb.update(fix_reqdb_get)
    fix_req_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_json['service_id']))
    assert fix_req_updated['assignment'] == fake_assignment, "Update unsuccessful"

    # Test delete
    http_delete('/v1/fixture/requests/{}'.format(fix_req_json['service_id']))
    assert not http_get('/v1/fixture/requests')['data'], "Deletion failed"
    # Verify delete is idempotent
    http_delete('/v1/fixture/requests/{}'.format(fix_req_json['service_id']))

def test_reserve_and_release(fixmgr):
    """Verifies reservation and release works"""
    populate_linux_instances(fixmgr)
    linux01 = fixmgr.get_fixture_instance(name='linux-01')
    linux02 = fixmgr.get_fixture_instance(name='linux-02')
    fixture_ids = [linux01.id, linux02.id]
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'RHEL 7'
                }
            }
        }
    }
    http_post('/v1/fixture/requests', data=fix_req_json)

    fix_req = fixmgr.reqproc.get_fixture_request(fix_req_json['service_id'])
    fixmgr.reqproc.reserve_instances(fix_req, fixture_ids)
    for fix_name in ['linux-01', 'linux-02']:
        fix_updated = fixmgr.get_fixture_instance(name=fix_name)
        assert fix_updated.referrers, "Reservation failed"
    # Ensure reservation is idempotent
    fixmgr.reqproc.reserve_instances(fix_req, fixture_ids)

    fixmgr.reqproc.release_instances(fix_req, fixture_ids)
    for fix_name in ['linux-01', 'linux-02']:
        fix_updated = fixmgr.get_fixture_instance(name=fix_name)
        assert not fix_updated.referrers, "Release failed"
    # Ensure release is idempotent
    fixmgr.reqproc.release_instances(fix_req, fixture_ids)

def test_reserve_non_existant_fixture(fixmgr):
    """Verifies exception is raised when attempting to reserve non-existent fixture"""
    populate_linux_instances(fixmgr)
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'RHEL 7'
                }
            }
        }
    }
    http_post('/v1/fixture/requests', data=fix_req_json)
    fix_req = fixmgr.reqproc.get_fixture_request(fix_req_json['service_id'])
    with pytest.raises(AXApiResourceNotFound):
        fixmgr.reqproc.reserve_instances(fix_req, [str(uuid.uuid4())])

def test_failed_reservation(fixmgr):
    """Verifies when not all reservations were successful all are released"""
    populate_linux_instances(fixmgr)
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'RHEL 7'
                }
            }
        }
    }
    http_post('/v1/fixture/requests', data=fix_req_json)
    fix_req = fixmgr.reqproc.get_fixture_request(fix_req_json['service_id'])
    linux1 = fixmgr.get_fixture_instance(name='linux-01')
    with pytest.raises(AXApiResourceNotFound):
        fixmgr.reqproc.reserve_instances(fix_req, [linux1.id, str(uuid.uuid4())])
    linux1_updated = fixmgr.get_fixture_instance(name='linux-01')
    assert not linux1_updated.referrers, "Failed reservation did not release fixture"

def test_request_invalid_attributes(fixmgr):
    """If requestor is requesting attributes that don't exist raise error"""
    populate_linux_instances(fixmgr)
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'invalid_attr' : 'RHEL 5'
                }
            }
        }
    }
    with pytest.raises(AXApiInvalidParam) as err:
        http_post('/v1/fixture/requests', data=fix_req_json)
    assert 'does not have attribute' in str(err)

def test_request_invalid_class(fixmgr):
    """If requestor is requesting non-existant class, error is raised"""
    populate_linux_instances(fixmgr)
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'NonExistantClass',
                'attributes' : {
                    'foo' : 'bar'
                }
            }
        }
    }
    with pytest.raises(AXApiInvalidParam) as err:
        http_post('/v1/fixture/requests', data=fix_req_json)
    assert re.search("Class '.*' not found", str(err))

def test_request_invalid_missing_filters(fixmgr):
    """If requestor is requesting with no filters supplied (e.g. requirements, class, or name)"""
    populate_linux_instances(fixmgr)
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'attributes' : {}
            }
        }
    }
    with pytest.raises(AXApiInvalidParam) as err:
        http_post('/v1/fixture/requests', data=fix_req_json)
    assert 'not specified in request' in str(err)

def test_request_impossible_request(fixmgr):
    """Verifies if a request is made where no fixtures can satisfy the request (disabled or otherwise) it raises error"""
    populate_linux_instances(fixmgr)
    impossible_req = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'JesseOS 1.2.3'
                }
            }
        }
    }
    with pytest.raises(AXApiResourceNotFound) as err:
        http_post('/v1/fixture/requests', data=impossible_req)
    assert "Impossible request" in str(err)

    # now create the fixture that can satisfy the request and retry
    satisfying_fixture = {
        'name' : 'jesse-linux',
        'class_name' : 'Linux',
        'attributes' : {
            'os_version' : 'JesseOS 1.2.3',
            'memory_mb' : 8192,
        }
    }
    http_post('/v1/fixture/instances', data=satisfying_fixture, headers=USER_SESSION_HEADERS)
    http_post('/v1/fixture/requests', data=impossible_req)

def _verify_assignment(assignment, expected_assignments):
    """Test helper to verify assignment was expected"""
    for key, val in assignment.items():
        expected_val = expected_assignments[key]
        if isinstance(expected_val, str):
            expected_val = [expected_val]
        assert val in expected_val, "{} was {} which was not in expected values: {}".format(key, val, expected_val)

def test_assignment_algorithm_1(fixmgr):
    """Tests assignment algorithm"""
    candidate_list = [
        ('f1', {'A'}),
    ]
    expected_assignments = {
        'f1' : 'A',
    }
    assignments = fixmgr.reqproc._assign_candidates_helper(candidate_list)
    _verify_assignment(assignments, expected_assignments)

    candidate_list = [
        ('f1', {'A'}),
        ('f2', {'B'}),
    ]
    expected_assignments = {
        'f1' : 'A',
        'f2' : 'B',
    }
    assignments = fixmgr.reqproc._assign_candidates_helper(candidate_list)
    _verify_assignment(assignments, expected_assignments)

def test_assignment_algorithm_2(fixmgr):
    """Tests assignment algorithm"""
    candidate_list = [
        ('f1', {'A', 'C'}),
        ('f2', {'A', 'B', 'C'}),
        ('f3', {'A'}),
        ('f4', {'D'}),
    ]
    expected_assignments = {
        'f1' : 'C',
        'f2' : 'B',
        'f3' : 'A',
        'f4' : 'D',
    }
    # Try multiple times since assignment is random
    for _ in range(5):
        assignments = fixmgr.reqproc._assign_candidates_helper(candidate_list)
        _verify_assignment(assignments, expected_assignments)

def test_assignment_algorithm_3(fixmgr):
    """Tests assignment algorithm"""
    candidate_list = [
        ('f1', {'A', 'B'}),
        ('f2', {'B', 'C'}),
        ('f3', {'C', 'D'}),
        ('f4', {'B', 'A'}),
    ]
    expected_assignments = {
        'f1' : {'A', 'B'},
        'f2' : 'C',
        'f3' : 'D',
        'f4' : {'A', 'B'},
    }
    for _ in range(10):
        assignments = fixmgr.reqproc._assign_candidates_helper(candidate_list)
        _verify_assignment(assignments, expected_assignments)

def test_assignment_algorithm_4(fixmgr):
    """Tests assignment algorithm"""
    candidate_list = [
        ('f1', {'A', 'B', 'C', 'D'}),
        ('f2', {'A', 'B'}),
        ('f3', {'B', 'C'}),
        ('f4', {'C', 'A'}),
    ]
    expected_assignments = {
        'f1' : 'D',
        'f2' : {'A', 'B'},
        'f3' : {'B', 'C'},
        'f4' : {'C', 'A'},
    }
    for _ in range(10):
        assignments = fixmgr.reqproc._assign_candidates_helper(candidate_list)
        _verify_assignment(assignments, expected_assignments)

def test_assignment_algorithm_5(fixmgr):
    """Tests assignment algorithm"""
    candidate_list = [
        ('f1', {'A', 'B', 'C'}),
        ('f2', {'A', 'B', 'C'}),
        ('f3', {'A'}),
        ('f4', {'D'}),
    ]
    expected_assignments = {
        'f1' : {'B', 'C'},
        'f2' : {'B', 'C'},
        'f3' : 'A',
        'f4' : 'D',
    }
    # Try multiple times since assignment is random
    for i in range(5):
        logger.debug("Attempt %s", i)
        assignments = fixmgr.reqproc._assign_candidates_helper(candidate_list)
        _verify_assignment(assignments, expected_assignments)

def test_assignment_algorithm_6(fixmgr):
    """Tests assignment algorithm"""
    candidate_list = [
        ('f1', {'A', 'B'}),
        ('f2', {'B', 'C'}),
        ('f3', {'C', 'D'}),
        ('f4', {'D', 'A'}),
    ]
    expected_assignments = {
        'f1' : {'A', 'B'},
        'f2' : {'B', 'C'},
        'f3' : {'C', 'D'},
        'f4' : {'D', 'A'},
    }
    # Try multiple times since assignment is random
    for i in range(5):
        logger.debug("Attempt %s", i)
        assignments = fixmgr.reqproc._assign_candidates_helper(candidate_list)
        _verify_assignment(assignments, expected_assignments)

def test_impossible_assignment_1(fixmgr):
    """Tests assignment algorithm when assignment is impossible"""
    candidate_list = [
        ('f1', {'A'}),
        ('f2', {'A'}),
    ]
    assert fixmgr.reqproc._assign_candidates_helper(candidate_list) is None

def test_impossible_assignment_2(fixmgr):
    """Tests assignment algorithm when assignment is impossible"""
    candidate_list = [
        ('f1', {'A', 'B', 'C'}),
        ('f2', {'A', 'B', 'C'}),
        ('f3', {'A', 'B', 'C'}),
        ('f4', {'A', 'B', 'C'}),
    ]
    assert fixmgr.reqproc._assign_candidates_helper(candidate_list) is None

def test_impossible_assignment_3(fixmgr):
    """Tests assignment algorithm when assignment is impossible"""
    candidate_list = [
        ('f1', {'A', 'B', 'C'}),
        ('f2', {'A', 'B'}),
        ('f3', {'B', 'C'}),
        ('f4', {'C', 'A'}),
    ]
    assert fixmgr.reqproc._assign_candidates_helper(candidate_list) is None

def test_assign_candidates(fixmgr):
    """Tests internal _assign_candidates method"""
    populate_linux_instances(fixmgr)
    linux03 = fixmgr.get_fixture_instance(name='linux-03')
    candidate_map = {
        'f1' : [linux03]
    }
    assignment = fixmgr.reqproc._assign_candidates(candidate_map)
    assert assignment['f1'].json() == linux03.json()

def test_process_requests_contention(fixmgr):
    """Tests the request processor when there is contention for a finite resource"""
    populate_linux_instances(fixmgr)
    fix_req_json_1 = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'Ubuntu 16.04'
                }
            }
        }
    }
    fix_req_json_2 = copy.deepcopy(fix_req_json_1)
    fix_req_json_2['service_id'] = str(uuid.uuid4())
    fix_req_1 = http_post('/v1/fixture/requests', data=fix_req_json_1)
    fix_req_2 = http_post('/v1/fixture/requests', data=fix_req_json_2)

    assert fixmgr.reqproc.process_requests() == 1, "Assigned unexpected number of requests"
    fix_req_1_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_1['service_id']))
    assert fix_req_1_updated['assignment'] and fix_req_1_updated['assignment']['fix1']['name'] == 'linux-03'
    fix_req_2_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_2['service_id']))
    assert not fix_req_2_updated['assignment'], "Competing fixture request unexpectedly received assignment"

    assert fixmgr.reqproc.process_requests() == 0, "Assigned unexpected number of requests"
    fix_req_1_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_1['service_id']))
    assert fix_req_1_updated['assignment'] and fix_req_1_updated['assignment']['fix1']['name'] == 'linux-03'
    fix_req_2_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_2['service_id']))
    assert not fix_req_2_updated['assignment'], "Competing fixture request unexpectedly received assignment"

def test_process_requests_no_contention(fixmgr):
    """Tests the request processor when there is no contention"""
    populate_linux_instances(fixmgr)
    fix_req_json_1 = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'RHEL 7'
                }
            }
        }
    }
    fix_req_json_2 = copy.deepcopy(fix_req_json_1)
    fix_req_json_2['service_id'] = str(uuid.uuid4())
    fix_req_1 = http_post('/v1/fixture/requests', data=fix_req_json_1)
    fix_req_2 = http_post('/v1/fixture/requests', data=fix_req_json_2)

    assert fixmgr.reqproc.process_requests() == 2, "Assigned unexpected number of requests"
    fix_req_1_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_1['service_id']))
    assert fix_req_1_updated['assignment'], "Fixture was not assigned"
    fix_req_2_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_2['service_id']))
    assert fix_req_2_updated['assignment'], "Fixture was not assigned"
    assert fix_req_1_updated['assignment'] != fix_req_2_updated['assignment'], "Fixture was assigned twice"

    assert fix_req_1_updated['assignment']['fix1']['name'] in ['linux-01', 'linux-02'], "Unexpected assignment"
    assert fix_req_2_updated['assignment']['fix1']['name'] in ['linux-01', 'linux-02'], "Unexpected assignment"

    assert fixmgr.reqproc.process_requests() == 0, "Assigned unexpected number of requests"

def test_request_processor_assignment_flatten(fixmgr):
    """Verifies fixture assignments are flattening attributes into single level dict"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'Ubuntu 16.04'
                }
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=fix_req_json)
    # wait for fixture to be assigned
    fix_req_popped = wait_for_assignment(fixmgr, fix_req['service_id'])
    fix_req_updated = http_get('/v1/fixture/requests/{}'.format(fix_req['service_id']))
    assert fix_req_popped['assignment'] == fix_req_updated['assignment']
    assert fix_req_popped['assignment']['fix1']['name'] in ['linux-03'], "Unexpected assignment"
    assert fix_req_popped['assignment']['fix1']['os_version'] == 'Ubuntu 16.04', "Unexpected assignment"

def test_request_processor_notify_create_request(fixmgr):
    """Verifies when fixture request is created, it will notify the request processor"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()

    fix_req_json_1 = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'RHEL 7'
                }
            }
        }
    }
    fix_req_json_2 = copy.deepcopy(fix_req_json_1)
    fix_req_json_2['service_id'] = str(uuid.uuid4())
    fix_req_1 = http_post('/v1/fixture/requests', data=fix_req_json_1)
    fix_req_2 = http_post('/v1/fixture/requests', data=fix_req_json_2)

    # wait for both fixtures to be assigned
    fix_req_1_popped = wait_for_assignment(fixmgr, fix_req_1['service_id'])
    fix_req_2_popped = wait_for_assignment(fixmgr, fix_req_2['service_id'])

    fix_req_1_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_1['service_id']))
    assert fix_req_1_updated['assignment'], "Fixture was not assigned"
    assert fix_req_1_popped == fix_req_1_updated

    fix_req_2_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_2['service_id']))
    assert fix_req_2_updated['assignment'], "Fixture was not assigned"
    assert fix_req_2_popped == fix_req_2_updated

    assert fix_req_1_updated['assignment'] != fix_req_2_updated['assignment'], "Fixture was assigned twice"
    assert fix_req_1_updated['assignment']['fix1']['name'] in ['linux-01', 'linux-02'], "Unexpected assignment"
    assert fix_req_2_updated['assignment']['fix1']['name'] in ['linux-01', 'linux-02'], "Unexpected assignment"

def test_request_processor_create_request_synchronous(fixmgr):
    """Tests the when the request is marked synchronous, fixture will either be assigned immediately or raise error if not found"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()

    fix_req_json_1 = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'synchronous' : True,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'Ubuntu 16.04'
                }
            }
        }
    }
    fix_req_json_2 = copy.deepcopy(fix_req_json_1)
    fix_req_json_2['service_id'] = str(uuid.uuid4())

    fix_req_1 = http_post('/v1/fixture/requests', data=fix_req_json_1)
    assert fix_req_1['assignment'], "Fixture was not assigned"
    assert fix_req_1['assignment']['fix1']['name'] == 'linux-03', "Unexpected assignment"
    fix_req_1_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_1['service_id']))
    assert fix_req_1['assignment'] == fix_req_1_updated['assignment']

    with pytest.raises(AXApiResourceNotFound):
        http_post('/v1/fixture/requests', data=fix_req_json_2)
    with pytest.raises(AXApiResourceNotFound):
        http_get('/v1/fixture/requests/{}'.format(fix_req_json_2['service_id']))

def test_request_processor_notify_release(fixmgr):
    """Verifies when fixture is released, it will notify the request processor"""
    populate_linux_instances(fixmgr)
    fix_req_json_1 = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'Ubuntu 16.04'
                }
            }
        }
    }
    fix_req_json_2 = copy.deepcopy(fix_req_json_1)
    fix_req_json_2['service_id'] = str(uuid.uuid4())
    fixmgr.reqproc.start_processor()

    fix_req_1 = http_post('/v1/fixture/requests', data=fix_req_json_1)
    fix_req_2 = http_post('/v1/fixture/requests', data=fix_req_json_2)

    fix_req_1_popped = wait_for_assignment(fixmgr, fix_req_1['service_id'])
    assert wait_for_assignment(fixmgr, fix_req_2['service_id'], timeout=1, verify_exists=False) is None

    fix_req_1_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_1['service_id']))
    assert fix_req_1_updated['assignment']['fix1']['name'] == 'linux-03'
    assert fix_req_1_popped == fix_req_1_updated
    fix_req_2_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_2['service_id']))
    assert not fix_req_2_updated['assignment'], "Fixture 2 was unexpectedly assigned"

    fixmgr.reqproc.delete_fixture_request(fix_req_1['service_id'])
    fix_req_2_popped = wait_for_assignment(fixmgr, fix_req_2['service_id'])

    assert fixmgr.reqproc.get_fixture_request(fix_req_1['service_id'], verify_exists=False) is None
    with pytest.raises(AXApiResourceNotFound):
        http_get('/v1/fixture/requests/{}'.format(fix_req_1['service_id']))
    fix_req_2_updated = http_get('/v1/fixture/requests/{}'.format(fix_req_2['service_id']))
    assert fix_req_2_updated['assignment']['fix1']['name'] == 'linux-03'
    assert fix_req_2_updated == fix_req_2_popped

def test_request_processor_notify_create_fixture(fixmgr):
    """Verifies when fixture is created, it will notify the request processor"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'Debian 8'
                }
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=fix_req_json)
    wait_for_processor(fixmgr)
    fix_req_updated = http_get('/v1/fixture/requests/{}'.format(fix_req['service_id']))
    assert not fix_req_updated['assignment']

    # Create the fixture. This should trigger the processor
    linux5_fix_json = {
        'name' : 'linux-05',
        'class_name' : 'Linux',
        'enabled' : True,
        'attributes' : {
            'host' : 'linux-05.host.com',
            'os_version' : 'Debian 8',
            'memory_mb' : 1024,
        }
    }
    linux5_fix = http_post('/v1/fixture/instances', data=linux5_fix_json, headers=USER_SESSION_HEADERS)

    popped_assignment = wait_for_assignment(fixmgr, fix_req['service_id'])
    fix_req_updated = http_get('/v1/fixture/requests/{}'.format(fix_req['service_id']))
    assert fix_req_updated['assignment']['fix1']['name'] == 'linux-05'
    assert fix_req_updated == popped_assignment

    linux5_fix_updated = http_get('/v1/fixture/instances/{}'.format(linux5_fix['id']))
    assert linux5_fix_updated['referrers']

def test_request_processor_notify_update(fixmgr):
    """Verifies when fixture is updated, it will notify the request processor"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()
    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'Debian 8'
                }
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=fix_req_json)
    wait_for_processor(fixmgr)
    fix_req_updated = http_get('/v1/fixture/requests/{}'.format(fix_req['service_id']))
    assert not fix_req_updated['assignment'], "Fixture unexpectedly assigned"

    # update os of some other fixture
    linux1_fix = http_get('/v1/fixture/instances?name=linux-01')['data'][0]
    updates = {'id' : linux1_fix['id'],
               'attributes' : {'os_version' : 'Debian 8'}}
    fixmgr.update_fixture_instance(updates)

    popped_assignment = wait_for_assignment(fixmgr, fix_req['service_id'])
    fix_req_updated = http_get('/v1/fixture/requests/{}'.format(fix_req['service_id']))
    assert fix_req_updated['assignment']['fix1']['name'] == 'linux-01'
    assert fix_req_updated == popped_assignment

    linux1_fix_updated = http_get('/v1/fixture/instances/{}'.format(linux1_fix['id']))
    assert linux1_fix_updated['referrers']

def test_request_processor_filter_disabled(fixmgr):
    """Verify disabled fixtures are not considered by request processor"""
    populate_linux_instances(fixmgr)
    linux3 = http_get('/v1/fixture/instances?name=linux-03')['data'][0]
    # disable fixture
    http_put('/v1/fixture/instances/{}'.format(linux3['id']), data={'enabled': False}, headers=USER_SESSION_HEADERS)

    fix_req_json = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'Ubuntu 16.04'
                }
            }
        }
    }
    http_post('/v1/fixture/requests', data=fix_req_json)

    fixmgr.reqproc.process_requests()
    linux3_updated = http_get('/v1/fixture/instances/{}'.format(linux3['id']))
    assert not linux3_updated['referrers']
    fix_req = http_get('/v1/fixture/requests/{}'.format(fix_req_json['service_id']))
    assert not fix_req['assignment']

    # enable fixture
    http_put('/v1/fixture/instances/{}'.format(linux3['id']), data={'enabled': True}, headers=USER_SESSION_HEADERS)
    fixmgr.reqproc.process_requests()
    linux3_updated = http_get('/v1/fixture/instances/{}'.format(linux3['id']))
    assert linux3_updated['referrers']
    fix_req = http_get('/v1/fixture/requests/{}'.format(fix_req_json['service_id']))
    assert fix_req['assignment'] and fix_req['assignment']['fix1']['name'] == 'linux-03'

def test_concurrency_zero(fixmgr):
    """Verifies when concurrency is zero it can be shared"""
    populate_linux_instances(fixmgr)
    linux3 = http_get('/v1/fixture/instances?name=linux-03')['data'][0]
    # set concurrency 0 (infinite concurrency)
    http_put('/v1/fixture/instances/{}'.format(linux3['id']), data={'concurrency': 0}, headers=USER_SESSION_HEADERS)

    fix_req_json_1 = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'Ubuntu 16.04'
                }
            }
        }
    }
    fix_req_json_2 = copy.deepcopy(fix_req_json_1)
    fix_req_json_2['service_id'] = str(uuid.uuid4())

    http_post('/v1/fixture/requests', data=fix_req_json_1)
    http_post('/v1/fixture/requests', data=fix_req_json_2)
    fixmgr.reqproc.process_requests()

    fix_req1 = http_get('/v1/fixture/requests/{}'.format(fix_req_json_1['service_id']))
    fix_req2 = http_get('/v1/fixture/requests/{}'.format(fix_req_json_2['service_id']))
    assert fix_req1['assignment'] and fix_req1['assignment']['fix1']['name'] == 'linux-03'
    assert fix_req2['assignment'] and fix_req2['assignment']['fix1']['name'] == 'linux-03'

def test_concurrency_semaphore(fixmgr):
    """Verifies when concurrency is two, 2/3 requests will be assigned"""
    populate_linux_instances(fixmgr)
    linux3 = http_get('/v1/fixture/instances?name=linux-03')['data'][0]
    # set concurrency 0 (infinite concurrency)
    http_put('/v1/fixture/instances/{}'.format(linux3['id']), data={'concurrency': 2}, headers=USER_SESSION_HEADERS)

    fix_req_json_1 = {
        'service_id' : str(uuid.uuid4()),
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'Ubuntu 16.04'
                }
            }
        }
    }
    fix_req_json_2 = copy.deepcopy(fix_req_json_1)
    fix_req_json_2['service_id'] = str(uuid.uuid4())
    fix_req_json_3 = copy.deepcopy(fix_req_json_1)
    fix_req_json_3['service_id'] = str(uuid.uuid4())

    http_post('/v1/fixture/requests', data=fix_req_json_1)
    http_post('/v1/fixture/requests', data=fix_req_json_2)
    http_post('/v1/fixture/requests', data=fix_req_json_3)
    fixmgr.reqproc.process_requests()

    fix_req1 = http_get('/v1/fixture/requests/{}'.format(fix_req_json_1['service_id']))
    fix_req2 = http_get('/v1/fixture/requests/{}'.format(fix_req_json_2['service_id']))
    fix_req3 = http_get('/v1/fixture/requests/{}'.format(fix_req_json_3['service_id']))
    assert fix_req1['assignment'] and fix_req1['assignment']['fix1']['name'] == 'linux-03'
    assert fix_req2['assignment'] and fix_req2['assignment']['fix1']['name'] == 'linux-03'
    assert not fix_req3['assignment']

def test_notify_assignments(fixmgr):
    """Tests that we will re-notify all assignments in request_processor (to support crash recovery)"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()

    service_id = str(uuid.uuid4())
    fix_req_json = {
        'service_id' : service_id,
        'root_workflow_id' : str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        'requirements' : {
            'fix1' : {
                'class' : 'Linux',
                'attributes' : {
                    'os_version' : 'RHEL 7'
                }
            }
        }
    }
    fix_req = http_post('/v1/fixture/requests', data=fix_req_json)
    wait_for_assignment(fixmgr, fix_req['service_id'])
    fixmgr.reqproc.redis_client_notification.delete("notification:{}".format(service_id))
    assert not wait_for_assignment(fixmgr, service_id, verify_exists=False, timeout=1), "Setup error"
    fixmgr.reqproc.trigger_processor()
    wait_for_assignment(fixmgr, fix_req['service_id'])

def test_request_default_attributes(fixmgr):
    """Catch a bug where assignment dictionary was being re-used """
    req_json = {
        "service_id": str(uuid.uuid4()),
        "root_workflow_id": str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        "requirements": {
            "fix1": {
                "class": "Linux",
                "attributes": {
                    "os_version": "RHEL 7"
                }
            }
        }
    }
    req1 = FixtureRequest(req_json)
    req2 = FixtureRequest(req_json)
    req1.assignment['foo'] = 'bar'
    assert req2.assignment == {}
    req1.assignment['foo'] = 'bar'

def test_request_by_class_name(fixmgr):
    """Verify we can request by class name"""
    populate_linux_instances(fixmgr)
    fixmgr.reqproc.start_processor()
    req_json = {
        "service_id": str(uuid.uuid4()),
        "root_workflow_id": str(uuid.uuid4()),
        'requester' : FIX_REQUESTER_AXWORKFLOWADC,
        "requirements": {
            "fix1": {
                "class": "Linux",
            }
        }
    }
    http_post('/v1/fixture/requests', data=req_json)
    wait_for_assignment(fixmgr, req_json['service_id'])

