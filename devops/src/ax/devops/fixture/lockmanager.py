"""
Library for allowing fine grained locks of a particular resource

Typical usage:

dog_lock_mgr = ResourceLockManager('dogs')
lock_dog = dog_lock_mgr.lock_resource

with lock_dog("Oliver"):
    print("Feed Ollie")

with lock_dog("Snoopy"):
    print("Pet Snoopy")

"""

import collections
import threading
from contextlib import contextmanager

class _ResourceLock(object):
    """Internal wraper around Lock object to additionally keep track of outstanding waiters"""
    def __init__(self):
        self.waiters = 0
        self.lock = threading.Lock()

class ResourceLockManager(object):
    """Manager of locks for a particular resource"""

    def __init__(self, resource_name):
        self.resource_name = resource_name
        self.global_lock = threading.Lock()
        self.resource_locks = collections.defaultdict(lambda: _ResourceLock())

    @contextmanager
    def lock_resource(self, resource_id, timeout=-1):
        """Fine grain locking mechanism against a resource_id"""
        # Get existing or create new lock for the id
        with self.global_lock:
            res_lock = self.resource_locks[resource_id]
            res_lock.waiters += 1

        acquired = res_lock.lock.acquire(timeout=timeout)
        try:
            if not acquired:
                raise Exception("Timed out ({}s) acquiring lock on {}: {}".format(timeout, self.resource_name, resource_id))
            yield
        finally:
            # Release the lock and optionally delete it (if there are no more waiters)
            with self.global_lock:
                res_lock.waiters -= 1
                if res_lock.waiters <= 0:
                    self.resource_locks.pop(resource_id, None)
                if acquired:
                    res_lock.lock.release()
