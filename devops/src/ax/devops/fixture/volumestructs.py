"""
Volume structures
"""
import copy
import json
import logging
import re
import time
import uuid

import voluptuous
from voluptuous import Schema, Any, Required, Invalid, REMOVE_EXTRA

from ax.exceptions import AXApiInvalidParam, AXException, AXIllegalOperationException
from .common import VolumeStatus, ReferrersMixin
from .util import humanize_error

logger = logging.getLogger(__name__)

"""
AXRNs are Argo resource names, which uniquely identifies a resource within a cluster
A valid axrn is <resource_type>:<resource_name>
Only volumes use axrns for now. Volumes are either named, or anonymous. 

Named Volumes:
Named volumes have AXRNs which look like a single level absolute path (e.g. vol:/mydatabase). This implies that
volume names are globally unique. In the future, to support named volumes at the user scope, we will need to invent
a convention that incorporates user information. Something like: vol:/user:jesse@email.com/mydatabase. For now
we only support named volumes at the global scope.

Anonymous Volumes:
Anonymous volumes incorporate the application and deployment name into the axrn.
"""
# resource type (e.g. vol:/) with alphanumeric or dashes, beginning with letter
VOL_NAME_REGEX = re.compile(r"^[a-z][a-z0-9-]*$")

volume_schema = Schema({
    Required('name', default=None): Any(str, None),
    Required('ctime', default=lambda: int(time.time())) : int,
    Required('mtime', default=lambda: int(time.time())) : int,
    Required('atime', default=lambda: int(time.time())) : int,
    Required('anonymous', default=False): bool,
    Required('storage_provider'): str,
    Required('storage_provider_id'): str,
    Required('storage_class'): str,
    Required('storage_class_id'): str,
    Required('enabled', default=True): bool,
    Required('axrn', default=None): Any(str, None),
    Required('owner'): str,
    Required('creator'): str,
    Required('status', default=VolumeStatus.INIT): str,
    Required('status_detail', default=None): Any(dict, None),
    Required('concurrency', default=1): int,
    Required('referrers', default=list) : list,
    Required('resource_id', default=None): Any(str, None),
    Required('attributes', default=dict) : dict,
}, extra=REMOVE_EXTRA)

def validate_storage_class_parameters(parameters):
    aws_params = parameters.get('aws')
    if not aws_params:
        raise Invalid("Storage class missing 'aws' parameters")
    for attr_name in ['storage_provider_name', 'storage_provider_id']:
        if attr_name not in aws_params:
            raise Invalid("Storage class missing {} parameter".format(attr_name))
    return parameters

storage_class_schema = Schema({
    Required('id'): str,
    Required('name'): str,
    Required('description', default=''): str,
    Required('ctime', default=lambda: int(time.time())) : int,
    Required('mtime', default=lambda: int(time.time())) : int,
    Required('parameters'): validate_storage_class_parameters,
}, extra=REMOVE_EXTRA)

class Volume(ReferrersMixin):
    """A Volume class as

    JSON representation:
    {
        "anonymous":false,
        "atime":1493226202,
        "attributes":{
            "filesystem":"ext4",
            "free_bytes":0,
            "size_gb":"10",
            "volume_type":"gp2"
        },
        "axrn":"vol:/test2",
        "concurrency":1,
        "creator":"tester@email.com",
        "ctime":1493079088,
        "enabled":true,
        "id":"8d65fcbd-9c0e-4dc3-8f2f-7b3ca2758b4a",
        "mtime":1493237594,
        "name":"test2",
        "owner":"tester@email.com",
        "referrers":[
            {
                "application_generation":"3c1cd4ea-2aa2-11e7-b6a0-0a58c0a8992f",
                "application_id":"46cea5ab-5bf4-5225-6302-3ae6086c04c8",
                "application_name":"test-app",
                "deployment_name":"deploy",
                "owner":"axamm",
                "root_workflow_id":"38ac9369-2aa2-11e7-ab75-0a58c0a89a23",
                "service_id":"ae23b80a-5513-5e09-5c93-95f3d7674eb9"
            }
        ],
        "resource_id":null,
        "status":"active",
        "status_detail":null,
        "storage_class":"ssd",
        "storage_class_id":"786f1236-06f0-54ca-76a6-f249eb3639c9",
        "storage_provider":"ebs",
        "storage_provider_id":"357aff9d-02f7-5345-651f-676fe2faaa6f"
    }
    """

    # whitelist of fields that user can set upon creation
    create_fields = frozenset(['name', 'storage_class', 'enabled', 'owner', 'creator', 'concurrency', 'attributes', 'resource_id'])
    # whitelist of fields that are user updatable
    mutable_fields = frozenset(['name', 'enabled', 'attributes'])
    json_fields = ['attributes', 'referrers', 'status_detail']
    null_fields = ['status_detail', 'resource_id']

    def __init__(self, volume):
        """Volume object model. Normalizes missing/incompatible attributes

        :param volume: volume dictionary
        """
        self.id = volume.get('id') or str(uuid.uuid4())
        volume = copy.deepcopy(volume)
        try:
            volume = volume_schema(volume)
        except voluptuous.Error as err:
            raise AXApiInvalidParam(humanize_error(str(err)))
        self.anonymous = volume['anonymous']
        if self.anonymous:
            if volume['name']:
                raise AXApiInvalidParam("Anonymous volumes cannot have names")
            self.name = None
        else:
            if not volume['name'] or not VOL_NAME_REGEX.match(volume['name']):
                raise AXApiInvalidParam("Invalid volume name: '{}'. Names must be alphanumeric or dashes beginning with a letter".format(volume['name']))
            self.name = volume['name'].lower()
        self.ctime = volume['ctime']
        self.mtime = volume['mtime']
        self.atime = volume['atime']
        self.storage_provider = volume['storage_provider'].lower()
        self.storage_provider_id = volume['storage_provider_id']
        self.storage_class = volume['storage_class'].lower()
        self.storage_class_id = volume['storage_class_id']
        self.enabled = volume['enabled']
        self.axrn = 'vol:/{}'.format(self.name) if not volume['axrn'] else volume['axrn'].lower()
        self.owner = volume['owner']
        self.creator = volume['creator']
        self.status = volume['status']
        self.status_detail = volume['status_detail']
        self.concurrency = volume['concurrency']
        self.referrers = volume['referrers']
        self.resource_id = volume['resource_id']
        self.attributes = {}
        for attr_name, val in volume['attributes'].items():
            self.attributes[attr_name.lower()] = val
        if self.storage_provider == 'ebs':
            if self.concurrency != 1:
                raise AXApiInvalidParam("Concurrency of EBS volumes must be 1")
            try:
                int(self.attributes['size_gb'])
            except ValueError:
                raise AXApiInvalidParam("EBS volume specified non-numeric value for 'size_gb': '{}'".format(self.attributes['size_gb']))
            except KeyError:
                raise AXApiInvalidParam("EBS volume did not specify 'size_gb'")
        # NOTE: the following logic only works for global, named volumes and will need to change if we support user scoped volumes
        if not self.anonymous and self.axrn != 'vol:/'+self.name:
            raise AXException("Volume name {} does not match axrn {}".format(self.name, self.axrn))

    def __str__(self):
        return 'Volume {} ({})'.format(self.id, self.axrn)

    def json(self):
        return {
            'id': self.id,
            'name': self.name,
            'ctime': self.ctime,
            'mtime': self.mtime,
            'atime': self.atime,
            'anonymous': self.anonymous,
            'storage_provider': self.storage_provider,
            'storage_provider_id': self.storage_provider_id,
            'storage_class': self.storage_class,
            'storage_class_id': self.storage_class_id,
            'enabled': self.enabled,
            'axrn': self.axrn,
            'owner': self.owner,
            'creator': self.creator,
            'status': self.status,
            'status_detail': self.status_detail,
            'concurrency': self.concurrency,
            'referrers': self.referrers,
            'resource_id': self.resource_id,
            'attributes': self.attributes,
        }

    def is_reservable(self, service_id):
        """Returns whether or not the volume is able to be served by the service_id"""
        if not self.enabled:
            logger.warning("%s unable to be reserved: volume disabled", self)
            return False
        if not self.anonymous:
            if self.status != VolumeStatus.ACTIVE:
                logger.warning("%s unable to be reserved: volume status: %s. status_detail: %s", self, self.status, self.status_detail)
                return False
        else:
            if self.status == VolumeStatus.DELETING:
                # anonymous volumes are able to be reserved while still in init/creating stage
                logger.warning("%s unable to be reserved: volume status: %s. status_detail: %s", self, self.status, self.status_detail)
                return False
        if self.concurrency > 0:
            if self.has_referrer(service_id):
                return True
            if len(self.referrers) >= self.concurrency:
                logger.warning("%s unable to be reserved: %s/%s slots in use", self, len(self.referrers), self.concurrency)
                return False
        return True

    def mark_for_deletion(self):
        """Sets volume status to deleting"""
        if len(self.referrers) > 0:
            raise AXIllegalOperationException("Unable to mark volume for deletion: volume is currently in use")
        if self.status != VolumeStatus.DELETING:
            self.status = VolumeStatus.DELETING
            self.mtime = int(time.time())
            if self.anonymous:
                # When marking anonymous volumes for deletion, we also update the axrn to something random so that
                # the original axrn can be used immediately after this call. This is needed because volume deletion
                # is asynchronous, and it is possible that another anonymous volume request comes in which we will
                # attempt to use the same axrn before the volume processor has a chance to delete the volume. This
                # can happen when user terminates a deployment, then immediately recreates it, and kubernetes/aws
                # is slow to detatch the volume from the host/perform the delete.
                self.axrn = "{}-deleting-{}".format(self.axrn, self.mtime)
            logger.info("Marked %s for deletion", self)
        else:
            logger.debug("%s already marked for deletion", self)

    def axdbdoc(self):
        """Normalizes json document for storage to axdb"""
        doc = self.json()
        Volume.serialize_axdb_doc(doc)
        return doc

    def axmondoc(self):
        """Normalizes json document for volume create/update requests to axmon"""
        doc = copy.deepcopy(self.attributes)
        doc['id'] = self.id
        doc['axrn'] = self.axrn
        doc['storage_provider_name'] = self.storage_provider
        doc['resource_id'] = self.resource_id
        return doc

    @classmethod
    def deserialize_axdb_doc(cls, doc):
        """Deserializes an axdb volume json to a Volume object

        NOTE: Due to AA-2789, storing null values in the axdb is not practical, since during retrieval, axdb will always
        serialize nulls to empty strings. Luckily, fixturemanager does not have a need for distinguishing empty strings
        from null, but it is proper to correct this behavior to be consistent from API stand point
        """
        for time_field in ['ctime', 'mtime', 'atime']:
            doc[time_field] = int(doc[time_field] / 1e6)
        for null_field in cls.null_fields:
            doc[null_field] = doc.get(null_field) or None
        for json_field in cls.json_fields:
            val = doc.get(json_field)
            if val:
                doc[json_field] = json.loads(val)
        return Volume(doc)

    @classmethod
    def serialize_axdb_doc(cls, doc):
        """Serializes a dictionary to a json structure suitable for storing to axdb"""
        # time fields are stored in nanoseconds in axdb
        for time_field in ['ctime', 'mtime', 'atime']:
            if time_field in doc:
                doc[time_field] = int(doc[time_field] * 1e6)
        for null_field in cls.null_fields:
            if null_field in doc:
                doc[null_field] = doc.get(null_field) or None
        for json_field in cls.json_fields:
            if json_field in doc:
                val = doc.get(json_field)
                if val is not None:
                    doc[json_field] = json.dumps(val)
        return doc


class StorageClass(object):
    """StorageClass object model"""

    json_fields = ['parameters']

    def __init__(self, storage_class):
        """StorageClass object model. Normalizes missing/incompatible attributes

        :param storage_class: storage_class dictionary
        """
        storage_class = copy.deepcopy(storage_class)
        try:
            storage_class = storage_class_schema(storage_class)
        except voluptuous.Error as err:
            raise AXApiInvalidParam(humanize_error(str(err)))
        self.id = storage_class['id']
        self.name = storage_class['name']
        self.description = storage_class['description']
        self.ctime = storage_class['ctime']
        self.mtime = storage_class['mtime']
        self.parameters = storage_class['parameters']

    def __str__(self):
        return 'StorageClass {} ({})'.format(self.id, self.name)

    def json(self):
        return {
            'id': self.id,
            'name': self.name,
            'description': self.description,
            'ctime': self.ctime,
            'mtime': self.mtime,
            'parameters': self.parameters,
        }
    @classmethod
    def deserialize_axdb_doc(cls, axdbdoc):
        """Deserializes an axdb storage class json to a StorageClass object"""
        if axdbdoc is None:
            return None
        doc = copy.deepcopy(axdbdoc)
        for time_field in ['ctime', 'mtime']:
            doc[time_field] = int(axdbdoc[time_field] / 1e6)
        for field in cls.json_fields:
            doc[field] = json.loads(axdbdoc[field])
        return StorageClass(doc)
