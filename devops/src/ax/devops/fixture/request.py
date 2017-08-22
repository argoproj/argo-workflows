"""
FixtureRequest model
"""
import copy
import json
import time

import voluptuous
from voluptuous import Schema, Any, Required, REMOVE_EXTRA, Optional, Invalid

from ax.exceptions import AXApiInvalidParam
from .common import FIX_REQUESTER_AXAMM
from .util import humanize_error

vol_requirement = Schema({
    Optional('axrn') : str,
    Optional('storage_class') : str,
    Optional('size_gb') : Any(int, str),
}, extra=REMOVE_EXTRA)

def VolRequirementsDict(attr_def):
    try:
        items = attr_def.items()
    except Exception:
        raise Invalid("vol_requirements is not a dictionary")
    vol_requirements = {}
    for ref_name, req in items:
        vol_requirements[ref_name] = vol_requirement(req)
    return vol_requirements

request_schema = Schema({
    Required('service_id'): str,
    Required('requester'): str,
    Required('user', default=None): Any(None, str),
    Required('application_id', default=None): Any(None, str),
    Required('application_name', default=None): Any(None, str),
    Required('application_generation', default=None): Any(None, str),
    Required('deployment_name', default=None): Any(None, str),
    Required('root_workflow_id'): str,
    Required('requirements', default=dict) : dict,
    Required('vol_requirements', default=dict) : VolRequirementsDict,
    Required('request_time', default=None): Any(int, None),
    Required('assignment', default=dict): Any(dict, None),
    Required('vol_assignment', default=dict): Any(dict, None),
    Required('assignment_time', default=None): Any(int, None),
}, extra=REMOVE_EXTRA)


class FixtureRequest(object):
    """
    A FixtureRequest class.

    service_id is the reservation key in which the request will be made against.
    It will either the step level service id of a workflow, or the stable deployment_id.

    JSON representation:
    {
        "service_id": "89e36c66-230e-4ef4-ac6d-1a23f8559bf9",
        "assigned": true,
        "application_id": "9cfafca1-b360-4397-a266-76cd00941648",
        "application_name": "claudia",
        "application_generation": "193dd1ff-be21-49c6-ac4a-d92c14da1593",
        "deployment_name": "claudia-ingestd",
        "owner" : "axamm",
        "user" : "tester@email.com",
        "root_workflow_id": "48c294d6-1bdc-474a-af0b-f5595b6417c6",
        "request_time": 1492202993154442,
        "requirements": {
            "fix1": {
                "class": "Linux",
                "attributes": {
                    "os_version": "RHEL 7"
                }
            }
        },
        "vol_requirements": {
            "db": {
                "axrn": "vol:/prod-wordpress-blog"
            }
        },
        "synchronous" : false,
        "assignment_time": 1492203000106595,
        "assignment": {
            "fix1": {
                "id": "d4576a94-bd66-4155-b2cc-f2302bd4adbe",
                "name": "linux-01"
                "class": "Linux",
                "description": "",
                "host": "linux-01.host.com",
                "os_version": "RHEL 7",
                "memory_mb": 4096,
            }
        }
        "vol_assignment": {
            "db": {
                "axrn": "vol:/prod-wordpress-blog"
                "id" : "ebfa071a-8710-4c4c-97a8-6d24112558db",
                "resource_id" : "vol-053c0473b82cd474e",
                "storage_provider_id" : "aa-bb-cc",
                "storage_provider_name" : "EBS",
                "volume_id" : "vol-053c0473b82cd474e",
                "volume_type" : "io1",
                "iops" : "30k"
                "filesystem" : "ext4",
                "size_gb" : "30"
            }
        },
    }
    """

    json_fields = ['requirements', 'vol_requirements', 'assignment', 'vol_assignment']
    time_fields = ['request_time', 'assignment_time']

    def __init__(self, request):
        """
        :param request: fixture request dictionary
        """
        request = copy.deepcopy(request)
        try:
            request = request_schema(request)
        except voluptuous.Error as err:
            raise AXApiInvalidParam(humanize_error(str(err)))
        self.service_id = request['service_id']
        self.application_id = request['application_id'] or None
        self.application_name = request['application_name'] or None
        self.application_generation = request['application_generation'] or None
        self.deployment_name = request['deployment_name'] or None
        self.requester = request['requester']
        self.user = request['user']
        self.root_workflow_id = request['root_workflow_id']
        self.assignment = request['assignment']
        self.vol_assignment = request['vol_assignment']
        self.request_time = int(request['request_time']) if request['request_time'] else int(time.time() * 1e6)
        self.assignment_time = int(request['assignment_time']) if request['assignment_time'] else None
        # Validate/normalize the requirements
        self.requirements = {}
        for req_name, req in request['requirements'].items():
            if 'name' not in req and 'class' not in req and not req.get('attributes'):
                raise AXApiInvalidParam("name, class and/or attributes not specified in request of {}".format(req_name))
            normalized_req = {}
            for attr_name, req_value in req.items():
                normalized_req[attr_name.lower()] = req_value
            self.requirements[req_name] = normalized_req

        for req_name, req in request['vol_requirements'].items():
            if 'axrn' in req:
                # named volume request
                continue
            # anonymous volume request
            if not self.user:
                raise AXApiInvalidParam("Username must be supplied when requesting anonymous volumes")
            if self.requester == FIX_REQUESTER_AXAMM and (not self.application_name or not self.deployment_name):
                raise AXApiInvalidParam("Anonymous volume requests for deployments must supply application and deployment name")
            if not req.get('storage_class'):
                raise AXApiInvalidParam("Anonymous volume request of '{}' did not specify 'storage_class'".format(req_name))
            if not req.get('size_gb'):
                raise AXApiInvalidParam("Anonymous volume request of '{}' did not specify 'size_gb'".format(req_name))
            try:
                int(req['size_gb'])
            except ValueError:
                raise AXApiInvalidParam("Anonymous volume request of '{}' specified non numeric value for 'size_gb'".format(req_name))
        self.vol_requirements = request['vol_requirements']
        if not self.requirements and not self.vol_requirements:
            raise AXApiInvalidParam("Fixture request had no requirements or vol_requirements")

        self.notification_channel = "notification:{}".format(self.service_id)

    def __str__(self):
        return "Request {} {} (workflow: {})".format(self.requester, self.service_id, self.root_workflow_id)

    @property
    def assigned(self):
        return bool(self.assignment or self.vol_assignment)

    def json(self):
        return {
            'service_id' : self.service_id,
            'assigned' : self.assigned,
            'application_id' : self.application_id,
            'application_name' : self.application_name,
            'application_generation' : self.application_generation,
            'deployment_name' : self.deployment_name,
            'requester' : self.requester,
            'user' : self.user,
            'root_workflow_id' : self.root_workflow_id,
            'requirements' : self.requirements,
            'vol_requirements' : self.vol_requirements,
            'request_time' : self.request_time,
            'assignment' : self.assignment,
            'vol_assignment' : self.vol_assignment,
            'assignment_time' : self.assignment_time,
        }

    def axdbdoc(self):
        """Returns a json string serializable to axdb"""
        doc = self.json()
        for json_field in self.json_fields:
            doc[json_field] = json.dumps(doc[json_field])
        for time_field in self.time_fields:
            doc[time_field] = int(doc[time_field]) if doc[time_field] else 0
        return doc

    def referrer(self):
        """Returns a 'referrer' structure, which will be stored in the 'referrers' field of volumes and fixture instances"""
        return {
            'service_id' : self.service_id,
            'requester' : self.requester,
            'user' : self.user,
            'application_id' : self.application_id,
            'application_name' : self.application_name,
            'application_generation' : self.application_generation,
            'deployment_name' : self.deployment_name,
            'root_workflow_id' : self.root_workflow_id,
        }

    @classmethod
    def deserialize_axdb_doc(cls, axdbdoc):
        """Deserializes a fixture request document retrieved from axdb into a FixtureRequest object"""
        doc = copy.deepcopy(axdbdoc)
        for json_field in cls.json_fields:
            doc[json_field] = json.loads(doc[json_field]) if doc[json_field] else None
        for time_field in cls.time_fields:
            doc[time_field] = doc[time_field] if doc[time_field] else None
        return FixtureRequest(doc)
