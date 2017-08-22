"""
Common constants used by fixture
"""
import logging

logger = logging.getLogger(__name__)

DEFAULT_PROCESS_INTERVAL = 5 * 60
DEFAULT_VOLUME_RETRY_INTERVAL = 120

# Time in days in which we will purge deleted fixture instances from the database
DELETED_INSTANCE_GC_DAYS = 3
NANOSECONDS_IN_A_DAY = 24 * 60 * 60 * 1e6

DB_NAME = 'fixturedb'
INSTANCES_COLLECTION_NAME = 'instances'

class InstanceStatus(object):
    """Enum of valid fixture instance statuses"""
    INIT = 'init'
    CREATING = 'creating'
    CREATE_ERROR = 'create_error'
    ACTIVE = 'active'
    OPERATING = 'operating'
    DELETING = 'deleting'
    DELETE_ERROR = 'delete_error'
    DELETED = 'deleted'

INSTANCE_STATUSES = [getattr(InstanceStatus, status) for status in dir(InstanceStatus) if not status.startswith('_')]


class FixtureClassStatus(object):
    ACTIVE = 'active'
    DISCONNECTED = 'disconnected'


class VolumeStatus(object):
    """Enum of valid volume statuses"""
    INIT = 'init'
    CREATING = 'creating'
    ACTIVE = 'active'
    DELETING = 'deleting'


class ServiceStatus(object):
    """Possible service statuses. See: axops/service/service.go"""
    SUCCESS = 0
    WAITING = 1
    RUNNING = 2
    CANCELLING = 3
    FAILED = -1
    CANCELLED = -2
    SKIPPED = -3
    INITIATING = 255

    @staticmethod
    def completed(code):
        return int(code) < 1

# HTTP headers supplied by axops NewSingleHostReverseProxyWithUserContext
HTTP_AX_USERID_HEADER = 'X-AXUserID'
HTTP_AX_USERNAME_HEADER = 'X-AXUsername'

FIX_REQUESTER_AXAMM = "axamm"
FIX_REQUESTER_AXWORKFLOWADC = "axworkflowadc"

# This name is the expected name of the artifact we will look from action containers
# to update attributes of the fixture
ATTRIBUTE_ARTIFACT_NAME = "attributes"

class ReferrersMixin(object):
    """Mixin class to support modification of referrers attribute"""

    def has_referrer(self, service_id):
        """Returns whether or not the service_id is one of the referrers of this volume"""
        return next((True for r in self.referrers if r['service_id'] == service_id), False)

    def add_referrer(self, referrer):
        """Adds a referrer doc to the list of referrers (replaces existing one if it exists)
        Returns True if the list was modified (referrer was added, or replaced the existing entry)"""
        referrers = []
        modified = False
        existed = False
        for ref in self.referrers:
            if ref['service_id'] != referrer['service_id']:
                # preserve list of referrers which are not of this service_id
                referrers.append(ref)
            else:
                existed = True
                # TODO: should this be an assertion?
                logger.warning("%s already had a reservation on %s", referrer, self)
                if ref != referrer:
                    logger.warning("Previous: %s", ref)
                    logger.warning("Current: %s", referrer)
                    referrers.append(referrer)
                    modified = True
                else:
                    referrers.append(ref)
        if not existed:
            referrers.append(referrer)
            modified = True
        if modified:
            self.referrers = referrers
            # TODO: set atime, but make volume atime and instance atime both nanoseconds
        return modified

    def remove_referrer(self, service_id):
        """Removes a referrer doc from the list of referrers. Returns True if the referrer existed and was removed. False if referrer did not have """
        referrers = [r for r in self.referrers if r['service_id'] != service_id]
        if len(referrers) == len(self.referrers):
            logger.warning("%s did not have a reservation for %s", service_id, self)
            return False
        else:
            self.referrers = referrers
            logger.warning("%s removed from referrers of %s", service_id, self)
            # TODO: set atime, but make volume atime and instance atime both nanoseconds
            return True
