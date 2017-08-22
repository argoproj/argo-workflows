"""
Fixture request datastore abstraction
"""

import logging

from ax.exceptions import AXApiResourceNotFound, AXApiInvalidParam

from .request import FixtureRequest
from .util import pretty_json

logger = logging.getLogger(__name__)

class FixtureRequestDatabase(object):
    """Abstraction on whatever we use as the underlying datastore for fixture requests"""

    def __init__(self, fixturemgr):
        self.fixmgr = fixturemgr

    @property
    def axdb_client(self):
        return self.fixmgr.axdb_client

    def get(self, service_id, verify_exists=True):
        """Get the fixture request by service_id
        :return: fixture_request if it was still in the queue"""
        fix_req_doc = self.axdb_client.get_fixture_request(service_id)
        if not fix_req_doc and verify_exists:
            raise AXApiResourceNotFound("Fixture request for service id {} does not exist".format(service_id))
        return FixtureRequest.deserialize_axdb_doc(fix_req_doc) if fix_req_doc else None

    def add(self, request):
        """Add a fixture request. If already exists, returns the existing request which may already have an assignment"""
        existing_request = self.get(request.service_id, verify_exists=False)
        if not existing_request:
            self.axdb_client.create_fixture_request(request.axdbdoc())
            logger.info("Created fixture request:\n%s", pretty_json(request.json()))
            return request

        # Sanity check to verify request is exactly the same as before
        if existing_request.requirements != request.requirements or existing_request.vol_requirements != request.vol_requirements:
            err = "Service id {} made multiple fixture requests with different requirements".format(request.service_id)
            logger.error(err)
            logger.error("Previous request:\n%s", pretty_json(existing_request.json()))
            logger.error("New request:\n%s", pretty_json(request.json()))
            raise AXApiInvalidParam(err)
        logger.info("Ignoring create fixture request. Already exists:\n%s", pretty_json(request.json()))
        return existing_request

    def update(self, request):
        """Updates a fixture request"""
        existing_request = self.get(request.service_id, verify_exists=False)
        if not existing_request:
            raise AXApiResourceNotFound("Fixture request for service id {} does not exist".format(request.service_id))
        self.axdb_client.update_fixture_request(request.axdbdoc())

    def remove(self, service_id):
        """Removes fixture request from request database
        :param service_id: service
        :return: fixture_request if it was still in the queue"""
        self.axdb_client.delete_fixture_request(service_id)

    def items(self, assigned=None):
        """Returns a list of fixture requests
        :param assigned: whether or not the request is assigned or not
        """
        params = {}
        if assigned is not None:
            params['assigned'] = assigned
        requests = [FixtureRequest.deserialize_axdb_doc(req) for req in self.axdb_client.get_fixture_requests(params=params)]
        # NOTE: priority algorithm goes here. If job priority is implemented, lambda should
        # be a tuple of (priority, request_time)
        return sorted(requests, key=lambda r: r.request_time)

    def initdb(self):
        """Initialize database (drop all keys)"""
        logger.info("Flushing fixture request database")
        for fix_req in self.items():
            self.remove(fix_req.service_id)

