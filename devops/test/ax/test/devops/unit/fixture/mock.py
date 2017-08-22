"""
Mock clients for internal services
"""
import copy
import json
import logging
import random
import string
import time
import uuid

from ax.devops.axdb.axops_client import AxopsClient
from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.fixture.request import FixtureRequest
from ax.devops.fixture.common import ServiceStatus
from ax.exceptions import AXIllegalArgumentException, AXException

logger = logging.getLogger('ax.devops.test')

class MockAxopsClient(AxopsClient):
    """Fake AxOps client"""

    def __init__(self, *args, **kwargs):
        super(MockAxopsClient, self).__init__(*args, **kwargs)
        self._categories = {}
        self._templates = {}
        self._fixture_templates = {}
        self._instances = {}
        self._services = {}
        self._deployments = []

    def ping(self):
        return True

    def create_service(self, service):
        service = copy.deepcopy(service)
        service['id'] = str(uuid.uuid4())
        service['status'] = ServiceStatus.WAITING
        if 'user' not in service:
            service['user'] = 'system'
        self._services[service['id']] = service
        return service

    def get_services(self, **kwargs):
        return [copy.deepcopy(s) for s in self._services.values()]

    def get_service(self, service_id):
        return copy.deepcopy(self._services[service_id])

    def search_artifacts(self, params):
        return []

    def get_deployments(self, **kwargs):
        return self._deployments

    def get_fixture_categories(self):
        return self._categories.values()

    def update_fixture_category(self, payload):
        self._categories[payload['id']] = payload

    def get_fixture_instance(self, id):
        return self._instances[id]

    def get_fixture_instances(self):
        return self._instances.values()

    def update_fixture_instance(self, payload):
        self._instances[payload['id']] = payload

    def delete_fixture_instance(self, id):
        self._instances.pop(id, None)

    @classmethod
    def _get_row_by_id(cls, table, row_id):
        """Helper to return a single row by ID or None"""
        results = cls._query_table_helper(table, params={'id': row_id})
        if len(results) == 0:
            return None
        if len(results) > 1:
            raise AXException("Found multiple rows with id: {}".format(row_id))
        return results[0]

    @classmethod
    def _query_table_helper(cls, table, params):
        """Helper to simulate a query against a axdb table with params"""
        results = []
        for row in table.values():
            if params:
                for key, val in params.items():
                    if row[key] != val:
                        break
                else:
                    results.append(copy.deepcopy(row))
            else:
                results.append(copy.deepcopy(row))
        return results


    def get_fixture_templates(self, params):
        return self._query_table_helper(self._fixture_templates, params)

    def get_fixture_template(self, template_id):
        return self._get_row_by_id(self._fixture_templates, template_id)

    def get_fixture_template_by_repo(self, repo, branch, name):
        params = {
            'repo': repo,
            'branch': branch,
            'name': name
        }
        templates = self.get_fixture_templates(params=params)
        if templates:
            return templates[0]
        else:
            return None

    def get_templates(self, repo, branch, name=None):
        params = {
            'repo': repo,
            'branch': branch,
            'name': name
        }
        if name is not None:
            params['name'] = name
        return self._query_table_helper(self._templates, params)

class MockAxdbClient(AxdbClient):
    """Fake Axdb client"""

    def __init__(self, *args, **kwargs):
        super(MockAxdbClient, self).__init__(*args, **kwargs)
        self._volumes = {}
        self._requests = {}
        self._services = {}
        self._fixture_classes = {}
        self._fixture_instances = {}
        self._storage_classes = {}
        self._storage_classes['ssd'] = {
            "id": str(uuid.uuid5(uuid.NAMESPACE_OID, "ssd")),
            "name": "ssd",
            "description": "General purpose SSD volume that balances price and performance for a wide variety of transactional workloads",
            "ctime": int(time.time() * 1e6),
            "mtime": int(time.time() * 1e6),
            "parameters": json.dumps({
                "aws": {
                    "storage_provider_id": str(uuid.uuid5(uuid.NAMESPACE_OID, "ebs")),
                    "storage_provider_name": "ebs",
                    "volume_type": "gp2",
                    "filesystem": "ext4",
                }
            })
        }

    def ping(self):
        return True

    @classmethod
    def _get_row_by_id(cls, table, row_id):
        """Helper to return a single row by ID or None"""
        results = cls._query_table_helper(table, params={'id': row_id})
        if len(results) == 0:
            return None
        if len(results) > 1:
            raise AXException("Found multiple rows with id: {}".format(row_id))
        return results[0]

    @classmethod
    def _query_table_helper(cls, table, params):
        """Helper to simulate a query against a axdb table with params"""
        results = []
        for row in table.values():
            if params:
                for key, val in params.items():
                    if row[key] != val:
                        break
                else:
                    results.append(copy.deepcopy(row))
            else:
                results.append(copy.deepcopy(row))
        return results

    @staticmethod
    def _assert_serialized(doc):
        """Verifies axdb put/post was supplied a serialized document"""
        for val in doc.values():
            assert not isinstance(val, dict), "value to axdb was not serialized"

    def get_storage_class_by_name(self, name):
        """Get a storage class by its name"""
        return self._storage_classes.get(name)

    def create_volume(self, payload):
        payload = copy.deepcopy(payload)
        self._serialize_json_attrs(payload)
        self._volumes[payload['id']] = payload

    def get_volume(self, volume_id):
        """Retrieve a volume by its id"""
        return copy.deepcopy(self._volumes.get(volume_id))

    def get_volumes(self, params=None):
        """Retrieve list of volumes"""
        volumes = []
        for vol in self._volumes.values():
            if params:
                for key, val in params.items():
                    if vol[key] != val:
                        break
                else:
                    volumes.append(copy.deepcopy(vol))
            else:
                volumes.append(copy.deepcopy(vol))
        return volumes

    def get_volume_by_axrn(self, axrn):
        """Retrieve a volume by its axrn"""
        for v in self._volumes.values():
            if v['axrn'] == axrn.lower():
                return copy.deepcopy(v)
        return None

    def update_volume(self, volume):
        """Update a volume"""
        volume_id = volume.get('id')
        if not volume_id:
            raise AXIllegalArgumentException("Volume id required for updates")
        existing = self._volumes.get(volume_id)
        if not existing:
            self._volumes[volume['id']] = volume
        else:
            existing.update(volume)
        self._serialize_json_attrs(self._volumes[volume['id']])

    def delete_volume(self, volume_id):
        """Delete a volume by its id"""
        self._volumes.pop(volume_id, None)

    def _serialize_json_attrs(self, doc):
        # Simulates axdb's current behavior of serializing nulls as empty strings
        for key, val in doc.items():
            if val is None:
                doc[key] = ''

    def _serialize_request_json_attrs(self, doc):
        # Simulates axdb's current behavior of serializing nulls as empty strings
        for field in FixtureRequest.json_fields:
            field_val = doc.get(field)
            if not field_val:
                doc[field] = ''

        # assert that fixturemanager is serializing attributes correctly
        for field in ['attributes', 'status_detail', 'referrers']:
            field_val = doc.get(field)
            assert not isinstance(field_val, dict) and not isinstance(field_val, list), "AXDB called with unserialized json for field: {}, val: {}".format(field, field_val)


    def get_fixture_requests(self, params=None):
        """Retrieve fixture requests"""
        requests = []
        for req in self._requests.values():
            if params and 'assigned' in params and req['assigned'] != params['assigned']:
                continue
            requests.append(copy.deepcopy(req))
        return requests

    def get_fixture_request(self, service_id):
        """Get the fixture request by service_id
        :return: fixture_request if it was still in the queue"""
        return copy.deepcopy(self._requests.get(service_id))

    def create_fixture_request(self, request):
        """Create a fixture request"""
        self._requests[request['service_id']] = copy.deepcopy(request)

    def update_fixture_request(self, request):
        """Updates a fixture request"""
        self._requests[request['service_id']] = copy.deepcopy(request)

    def delete_fixture_request(self, service_id):
        """Delete fixture request from request database"""
        self._requests.pop(service_id, None)

    def create_fixture_instance(self, instance):
        """Inserts a fixture instance into database"""
        if instance['id'] in self._fixture_instances:
            raise AXException()
        instance = copy.deepcopy(instance)
        self._serialize_json_attrs(instance)
        self._fixture_instances[instance['id']] = instance

    def get_fixture_instances(self, params=None):
        """Inserts a fixture instance into database"""
        return self._query_table_helper(self._fixture_instances, params)

    def get_fixture_instance_by_id(self, instance_id):
        """Retreieve an instance by its id"""
        return self._get_row_by_id(self._fixture_instances, instance_id)

    def update_fixture_instance(self, instance):
        """Updates an existing instance into database"""
        instance_id = instance.get('id')
        if not instance_id:
            raise AXIllegalArgumentException("Instance id required for updates")
        existing = self._fixture_instances.get(instance_id)
        if not existing:
            raise AXException("Instance does not exists")
        existing.update(instance)
        self._serialize_json_attrs(instance)

    def delete_fixture_instance(self, instance_id):
        """Delete instance from database"""
        self._fixture_instances.pop(instance_id, None)

    def get_fixture_classes(self, params=None):
        """Retrieve list of classes filtered by params"""
        return self._query_table_helper(self._fixture_classes, params)

    def get_fixture_class_by_name(self, name):
        """Retrieve a fixture class by its name"""
        classes = self.get_fixture_classes(params={'name': name})
        if len(classes) == 0:
            return None
        if len(classes) > 1:
            raise AXException("Found multiple classes with name: {}".format(name))
        return classes[0]

    def get_fixture_class_by_id(self, class_id):
        """Retreieve a fixture class by class id"""
        return self._get_row_by_id(self._fixture_classes, class_id)

    def create_fixture_class(self, fix_class):
        """Inserts a fixture class into database"""
        self._assert_serialized(fix_class)
        assert fix_class['id'] not in self._fixture_classes
        self._fixture_classes[fix_class['id']] = copy.deepcopy(fix_class)

    def update_fixture_class(self, fix_class):
        """Updates an existing fixture class into database"""
        if not fix_class.get('id'):
            raise AXIllegalArgumentException("Class id required for updates")
        assert fix_class['id'] in self._fixture_classes
        fix_class = copy.deepcopy(fix_class)
        self._fixture_classes[fix_class['id']].update(fix_class)

    def delete_fixture_class(self, class_id):
        """Delete fixture class from database"""
        return self._fixture_classes.pop(class_id, None)

    def get_service(self, *args, **kwargs):
        # return a dummy service object
        return {
            'template_name': 'foo',
            'task_id': str(uuid.uuid4())
        }

    def update_service(self, *args, **kwargs):
        pass


class MockAxsysClient(object):

    def __init__(self, *args, **kwargs):
        super(MockAxsysClient, self).__init__(*args, **kwargs)
        self._volumes = {}
        self.delay = 1
        self._in_flight_requests = {}

    def create_volume(self, volume):
        """Create a volume"""
        volume_id = volume['id']
        assert volume_id not in self._in_flight_requests, "fixturemanager made concurrent requests to axsys: {}".format(self._in_flight_requests[volume_id])
        self._in_flight_requests[volume_id] = 'create'
        self._volumes[volume_id] = copy.deepcopy(volume)
        self._volumes[volume_id]['Tags'] = [{'Key': 'axrn', 'Value': volume['axrn']}]
        resource_id = 'vol-' + ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(7))
        self._volumes[volume_id]['VolumeId'] = resource_id
        time.sleep(self.delay)
        del self._in_flight_requests[volume_id]
        return resource_id

    def get_volume(self, volume_id):
        """Get a volume

        Example return value from axmon:
        {'Attachments': [],
        'AvailabilityZone': 'us-west-2a',
        'CreateTime': 'Fri, 14 Apr 2017 04:31:06 GMT',
        'Encrypted': False,
        'Iops': 100,
        'Size': 30,
        'SnapshotId': '',
        'State': 'creating',
        'Tags': [{'Key': 'KubernetesCluster',
                'Value': 'jesse03-6d50262e-0839-11e7-92f7-c0ffeec0ffee'},
                {'Key': 'axrn', 'Value': 'vol:/valid-name'},
                {'Key': 'AXVolumeID',
                'Value': '3d422a72-9c9c-4128-9caf-e9515b451aa2'},
                {'Key': 'id', 'Value': '3d422a72-9c9c-4128-9caf-e9515b451aa2'}],
        'VolumeId': 'vol-06364619c03061afc',
        'VolumeType': 'gp2'}
        """
        time.sleep(self.delay)
        return copy.deepcopy(self._volumes.get(volume_id))

    def update_volume(self, volume):
        """Update a volume"""
        volume_id = volume['id']
        assert volume_id not in self._in_flight_requests, "fixturemanager made concurrent requests to axsys: {}".format(self._in_flight_requests[volume_id])
        self._in_flight_requests[volume_id] = 'update'
        self._volumes[volume_id]['Tags'] = [{'Key': 'axrn', 'Value': volume['axrn']}]
        time.sleep(self.delay)
        del self._in_flight_requests[volume_id]

    def delete_volume(self, volume_id):
        """Delete a volume"""
        assert volume_id not in self._in_flight_requests, "fixturemanager made concurrent requests to axsys: {}".format(self._in_flight_requests[volume_id])
        self._in_flight_requests[volume_id] = 'delete'
        time.sleep(self.delay)
        self._volumes.pop(volume_id, None)
        del self._in_flight_requests[volume_id]
