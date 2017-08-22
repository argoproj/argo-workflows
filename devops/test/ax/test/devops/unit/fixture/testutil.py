"""
Fixture test utility functions
"""
import json
import logging
import time

import pytest

from ax.devops.fixture.rest import app
from ax.exceptions import deserialize, AXApiResourceNotFound

app.testing = True
test_client = app.test_client()

logger = logging.getLogger(__name__)

def wait_for_processor(fixmgr, timeout=30):
    """Test helper function to wait for fixture processor to idle"""
    stop_time = time.time() + timeout
    while True:
        if not fixmgr.reqproc._events:
            time.sleep(.1)
            break
        if time.time() > stop_time:
            pytest.fail("Timed out ({}s) waiting for processor to idle".format(timeout))
        time.sleep(.1)

def wait_for_assignment(fixmgr, service_id, timeout=5, verify_exists=True):
    """Waits for a fixture assignment through the notification channel"""
    notification = fixmgr.reqproc.redis_client_notification.brpop(
        'notification:{}'.format(service_id), timeout=timeout)
    if not notification:
        if verify_exists:
            pytest.fail("Notification for {} was not created".format(service_id))
        else:
            return None
    # If assigned, verifies notification payload matches the database
    notification_json = json.loads(notification[1])
    fix_req_json = fixmgr.reqproc.get_fixture_request(service_id).json()
    assert notification_json == fix_req_json, "Notification json did not match request json"
    return fix_req_json

def _request(method, url, data=None, headers=None):
    """Test helper to make a http request to the flask app"""
    method = method.upper()
    if method == 'GET':
        func = test_client.get
    elif method == 'PUT':
        func = test_client.put
    elif method == 'POST':
        func = test_client.post
    elif method == 'DELETE':
        func = test_client.delete
    else:
        raise ValueError("HTTP method {} unsupported".format(method))
    if data:
        data = json.dumps(data)
    resp = func(url, data=data, content_type='application/json', headers=headers)
    if resp.data and resp.mimetype == 'application/json':
        resp_data = resp.data.decode('utf-8')
        try:
            resp_data = json.loads(resp_data)
        except ValueError:
            logger.warning("Response was not JSON:\n%s", resp_data)
            raise
    else:
        resp_data = resp.data
    if resp.status_code >= 400:
        err = deserialize(resp_data)
        if err.code == AXApiResourceNotFound.code:
            assert resp.status_code == 404, "API returned {} but http status code was {} (expected 404)" \
                .format(err.code, resp.status_code)
        raise err
    return resp_data

def http_get(url, **kwargs):
    return _request('GET', url, **kwargs)

def http_put(url, **kwargs):
    return _request('PUT', url, **kwargs)

def http_post(url, **kwargs):
    return _request('POST', url, **kwargs)

def http_delete(url, **kwargs):
    return _request('DELETE', url, **kwargs)

def fake_volume_atime(fixmgr, volume_id, atime=None):
    """Artificially sets atime in database to something else. Default is 1 second in the past"""
    if atime is None:
        atime = int(time.time()-1)
    fixmgr.volumemgr.axdb_client.update_volume({'id': volume_id, 'atime': int(atime*1e6)})
    return atime
