"""Test data for fixture tests"""
import copy
import uuid

import pytest
from .mock import MockAxopsClient
from . import testutil

from ax.devops.fixture.common import HTTP_AX_USERID_HEADER, HTTP_AX_USERNAME_HEADER

def populate_templates(fixmgr):
    if not isinstance(fixmgr.axops_client, MockAxopsClient):
        pytest.skip("test only applicable to mock axops client")
    template_ids = []
    for temp_dict in TEST_FIXTURE_TEMPLATES:
        temp_dict = copy.deepcopy(temp_dict)
        fixmgr.axops_client._fixture_templates[temp_dict['id']] = copy.deepcopy(temp_dict)
        template_ids.append(temp_dict['id'])

    for temp_dict in TEST_TEMPLATES:
        temp_dict = copy.deepcopy(temp_dict)
        fixmgr.axops_client._templates[temp_dict['id']] = copy.deepcopy(temp_dict)
    return template_ids

def populate_class(fixmgr):
    populate_templates(fixmgr)
    testutil.http_post('/v1/fixture/classes', data={'template_id': TEST_FIXTURE_TEMPLATE['id']})

def populate_linux_instances(fixmgr):
    populate_templates(fixmgr)
    testutil.http_post('/v1/fixture/classes', data={'template_id': LINUX_SCHEMA['id']})
    for create_payload in TEST_LINUX_INSTANCES:
        testutil.http_post('/v1/fixture/instances', data=create_payload, headers=USER_SESSION_HEADERS)

USER_SESSION_HEADERS = {
    HTTP_AX_USERID_HEADER: str(uuid.uuid4()),
    HTTP_AX_USERNAME_HEADER: 'testuser@email.com',
}

TEST_ACTION_TEMPLATE = {
    "id":"f28a8744-ac5f-53bb-75d7-f7647cb330ab",
    "type":"service_template",
    "subtype":"container",
    "name":"test-fixture-action",
    "repo":"https://repo.org/company/prod.git",
    "branch":"master",
    "revision":"8b2a1c12d1cc67da114bb1f1ede4ba7419f9c376",
    "cost":0,
    "container":{
        "resources":{
            "mem_mib":64,
            "cpu_cores":0.02
        },
        "image":"debian:8.5",
        "docker_options":"",
        "command":"echo 'performing action %%ACTION%% instance_type: %%INSTANCE_TYPE%%'; sleep 30; echo '%%ATTRIBUTES%%' > /tmp/fix_attrs.json; if [ %%ACTION%% = fail ] ; then exit 1; fi"
    },
    "inputs":{
        "parameters":{
            "ACTION":{

            },
            "ATTRIBUTES":{
                "default":"{\"ip_address\": \"1.2.3.4\"}"
            },
            "INSTANCE_TYPE":{
                "default":" "
            }
        }
    },
    "outputs":{
        "artifacts":{
            "attributes":{
                "path":"/tmp/fix_attrs.json"
            }
        }
    },
    "jobs_fail":0,
    "jobs_success":0
}

TEST_ACTION_TEMPLATE_BRANCH2 = copy.deepcopy(TEST_ACTION_TEMPLATE)
TEST_ACTION_TEMPLATE_BRANCH2['branch'] = 'dev'
TEST_ACTION_TEMPLATE_BRANCH2['id'] = str(uuid.uuid4())

TEST_TEMPLATES = [
    TEST_ACTION_TEMPLATE,
    TEST_ACTION_TEMPLATE_BRANCH2,
]

# NOTE: This json was copied from axops fixture template payload so is is representative of how they will be returned from axops API
TEST_FIXTURE_TEMPLATE = {
    "id": "2f36d9ff-cf65-56cc-5c2a-b4d13355fd11",
    "repo": "https://repo.org/company/prod.git",
    "branch": "master",
    "revision": "8b2a1c12d1cc67da114bb1f1ede4ba7419f9c376",
    "name":"test-fixture",
    "description":"fixture for test purposes",
    "attributes":{
        "cpu_cores":{
            "type":"int",
            "options":[
                1,
                2,
                4,
                8
            ],
            "default":1
        },
        "disable_nightly":{
            "type":"bool"
        },
        "group":{
            "type":"string",
            "flags":"required",
            "options":[
                "dev",
                "qa",
                "prod"
            ]
        },
        "instance_type":{
            "type":"string",
            "flags":"required",
            "options":[
                "m3.medium",
                "m3.large",
                "m3.xlarge",
                "m3.2xlarge"
            ],
            "default":"m3.large"
        },
        "ip_address":{
            "type":"string"
        },
        "memory_gib":{
            "type":"int",
            "default":4
        },
        "tags":{
            "type":"string",
            "flags":"array"
        }
    },
    "actions":{
        "bad_attributes":{
            "template":"test-fixture-action",
            "parameters":{
                "ACTION":"success",
                "ATTRIBUTES":"{\"memory_gib\": \"foo\"}"
            }
        },
        "create":{
            "template":"test-fixture-action",
            "parameters":{
                "ACTION":"create",
                "INSTANCE_TYPE":"%%fixture.instance_type%%"
            }
        },
        "delete":{
            "template":"test-fixture-action",
            "parameters":{
                "ACTION":"delete"
            }
        },
        "health_check_fail":{
            "template":"test-fixture-action",
            "parameters":{
                "ACTION":"fail"
            },
            "on_failure":"disable"
        },
        "resume":{
            "template":"test-fixture-action",
            "parameters":{
                "ACTION":"suspend"
            },
            "on_success":"enable"
        },
        "snapshot":{
            "template":"test-fixture-action",
            "parameters":{
                "ACTION":"snapshot"
            }
        },
        "suspend":{
            "template":"test-fixture-action",
            "parameters":{
                "ACTION":"suspend"
            },
            "on_success":"disable"
        }
    }
}

# Schema to test changes to the schema to see if we handle it properly
SCHEMA_TEST_1 = {
    "id": str(uuid.uuid4()),
    "repo": "https://repo.org/company/prod.git",
    "branch": "master",
    "revision": "8b2a1c12d1cc67da114bb1f1ede4ba7419f9c376",
    "name":"schema-test-1",
    "description":"Schema Test 1",
    "attributes":{
        "float_attr":{"type":"float", "default":3.141592654},
        "int_attr":{"type":"int", "default":1234},
        "bool_attr":{"type":"bool", "default":False},
        "str_attr":{"type":"string", "default":"foo"},
        "float_array":{"type":"float", "default":[3.141592654], "flags":"array"},
        "int_array":{"type":"int", "default":[1234], "flags":"array"},
        "bool_array":{"type":"bool", "default":[False], "flags":"array"},
        "str_array":{"type":"string", "default":["foo bar"], "flags":"array"},
        "to_delete":{"type":"string", "default":"val to be deleted"}, # field which will be deleted
        "str_to_int":{"type":"string", "default":"string val"}, # field which will change form str to int
        "str_to_arr":{"type":"string", "default":"string val"}, # field which will change form str to array of str
    }
}

# this second schema simulates the user changing his class schema with new/changed/removed attributes and pushing the template change
SCHEMA_TEST_2 = {
    "id": SCHEMA_TEST_1['id'],
    "repo": "https://repo.org/company/prod.git",
    "branch": "master",
    "revision": "8993ff50a54716133032046dbe65256c544922a1",
    "name":"schema-test-1",
    "description":"Schema Test 2",
    "attributes":{
        "float_attr":{"type":"float", "default":3.141592654},
        "int_attr":{"type":"int", "default":1234},
        "bool_attr":{"type":"bool", "default":False},
        "str_attr":{"type":"string", "default":"foo"},
        "float_array":{"type":"float", "default":[3.141592654], "flags":"array"},
        "int_array":{"type":"int", "default":[1234], "flags":"array"},
        "bool_array":{"type":"bool", "default":[False], "flags":"array"},
        "str_array":{"type":"string", "default":["foo"], "flags":"array"},
        "str_to_int":{"type":"int", "default":0},
        "str_to_arr":{"type":"string", "default":["string val"], "flags":"array"},
        "new_field":{"type":"string", "default":"val to be deleted"}, # field which was added
    }
}

LINUX_SCHEMA = {
    "id": str(uuid.uuid4()),
    "repo": "https://repo.org/company/prod.git",
    "branch": "master",
    "revision": "8993ff50a54716133032046dbe65256c544922a1",
    "name":"Linux",
    "description":"Linux Schema",
    "attributes":{
        "host":{"type":"string"},
        "os_version":{"type":"string"},
        "memory_mb":{"type":"int"},
        "int_array":{"type":"int", "flags":"array"},
    }
}



# Same template as test-fixture, but in dev branch
TEST_FIXTURE_TEMPLATE_BRANCH2 = copy.deepcopy(TEST_FIXTURE_TEMPLATE)
TEST_FIXTURE_TEMPLATE_BRANCH2['branch'] = 'dev'
TEST_FIXTURE_TEMPLATE_BRANCH2['id'] = str(uuid.uuid4())

# Same template as test-fixture, but with different name
TEST_FIXTURE_TEMPLATE_NAME2 = copy.deepcopy(TEST_FIXTURE_TEMPLATE)
TEST_FIXTURE_TEMPLATE_NAME2['name'] = "test-fixture2"
TEST_FIXTURE_TEMPLATE_NAME2['id'] = str(uuid.uuid4())

TEST_FIXTURE_TEMPLATES = [
    TEST_FIXTURE_TEMPLATE,
    TEST_FIXTURE_TEMPLATE_BRANCH2,
    TEST_FIXTURE_TEMPLATE_NAME2,
    SCHEMA_TEST_1,
    LINUX_SCHEMA,
]

TEST_LINUX_INSTANCES = [
    {
        'name' : 'linux-01',
        'class_name' : 'Linux',
        'attributes' : {
            'host' : 'linux-01.host.com',
            'os_version' : 'RHEL 7',
            'memory_mb' : 4096,
            'int_array' : [0],
        }
    },
    {
        'name' : 'linux-02',
        'class_name' : 'Linux',
        'attributes' : {
            'host' : 'linux-02.host.com',
            'os_version' : 'RHEL 7',
            'memory_mb' : 8192,
            'int_array' : [1],
        }
    },
    {
        'name' : 'linux-03',
        'class_name' : 'Linux',
        'attributes' : {
            'host' : 'linux-03.host.com',
            'os_version' : 'Ubuntu 16.04',
            'memory_mb' : 8192,
            'int_array' : [1],
        }
    },
    {
        'name' : 'linux-04',
        'class_name' : 'Linux',
        'enabled' : False,
        'attributes' : {
            'host' : 'linux-04.host.com',
            'os_version' : 'Debian 8',
            'memory_mb' : 1024,
        }
    }
]

_DISABLED = [
    {
        'name' : 'datatypetest-01',
        'class_name' : 'DataTypeCategory',
        'attributes' : {
            'int_type' :              0,
            'string_type' :           "foo",
            'bool_type' :             False,
            'float_type' :            3.141592654,
            'optional_int_type' :     None,
            'optional_string_type' :  None,
            'optional_bool_type' :    None,
            'optional_float_type' :   None,
            'int_array' :             [0],
            'string_array' :          [""],
            'bool_array' :            [False],
            'float_array' :           [3.141592654],
            'optional_int_array' :    [],
            'optional_string_array' : [],
            'optional_bool_array' :   [],
            'optional_float_array' :  [],
        }
    },
    {
        'name' : 'datatypetest-02',
        'class_name' : 'DataTypeCategory',
        'attributes' : {
            'int_type' :              0,
            'string_type' :           "foo",
            'bool_type' :             False,
            'float_type' :            3.141592654,
            'optional_int_type' :     0,
            'optional_string_type' :  "",
            'optional_bool_type' :    False,
            'optional_float_type' :   3.141592654,
            'int_array' :             [0],
            'string_array' :          [""],
            'bool_array' :            [False],
            'float_array' :           [3.141592654],
            'optional_int_array' :    [0],
            'optional_string_array' : [""],
            'optional_bool_array' :   [False],
            'optional_float_array' :  [0.0],
        }
    },
    {
        'name' : 'datatypetest-03',
        'class_name' : 'DataTypeCategory',
        'attributes' : {
            'int_type' :              3,
            'string_type' :           "bar",
            'bool_type' :             True,
            'float_type' :            2.71828,
            'optional_int_type' :     3,
            'optional_string_type' :  "bar",
            'optional_bool_type' :    True,
            'optional_float_type' :   2.71828,
            'int_array' :             [3],
            'string_array' :          ["bar"],
            'bool_array' :            [True],
            'float_array' :           [2.71828],
            'optional_int_array' :    [3],
            'optional_string_array' : ["bar"],
            'optional_bool_array' :   [True],
            'optional_float_array' :  [2.71828],
        }
    },
    {
        'name' : 'datatypetest-04',
        'class_name' : 'DataTypeCategory',
        'attributes' : {
            'int_type' :              4,
            'string_type' :           "baz",
            'bool_type' :             True,
            'float_type' :            4.99,
            'optional_int_type' :     4,
            'optional_string_type' :  "baz",
            'optional_bool_type' :    True,
            'optional_float_type' :   4.99,
            'int_array' :             [4],
            'string_array' :          ["baz"],
            'bool_array' :            [True],
            'float_array' :           [4.99],
            'optional_int_array' :    [4],
            'optional_string_array' : ["baz"],
            'optional_bool_array' :   [True],
            'optional_float_array' :  [4.0],
        }
    },
]

TEST_VOLUMES = [
    {
        "name" : "prod-wordpress-blog",
        "storage_class" : "ssd",
        "owner" : "testuser@email.com",
        "creator" : "testuser@email.com",
        "attributes" : {
            "size_gb" : 30
        }
    },
    {
        "name" : "staging-wordpress-blog",
        "storage_class" : "ssd",
        "owner" : "testuser@email.com",
        "creator" : "testuser@email.com",
        "attributes" : {
            "size_gb" : 30
        }
    }
]
