#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import logging
import uuid

try:
    from urllib import urlencode
except ImportError:
    from urllib.parse import urlencode

import requests

REQUESTS_GET = 'get'
REQUESTS_POST = 'post'
REQUESTS_PUT = 'put'
REQUESTS_DELETE = 'delete'

logger = logging.getLogger('ax.devops.axrequests')
logging.getLogger("requests").setLevel(logging.WARNING)
logging.getLogger("urllib3").setLevel(logging.WARNING)


class AxRequests(object):
    """AX request."""

    def __init__(self, host=None, port=8080, version=None, protocol='http', url=None, username=None, password=None, timeout=30, ssl_verify=True):
        """Initialize host.

        :param host:
        :param port:
        :param version:
        :param protocol:
        :param url:
        :param username:
        :param password:
        :param timeout:
        :param ssl_verify: Verify SSL certificate or not.
        :return:
        """
        if url:
            self.url = url
        elif version is not None:
            self.url = '{}://{}:{}{}'.format(protocol, host, port, '/{}'.format(version))
        else:
            self.url = '{}://{}:{}'.format(protocol, host, port)
        self.auth = (username, password)
        self.timeout = timeout
        self.ssl_verify = ssl_verify

    def __str__(self):
        return 'Connection: {}'.format(self.url)

    def _run_requests(self, method, path, params=None, data=None, headers=None, auth=None, raise_exception=True, value_only=False, stream=None):
        """Send request.

        :param method:
        :param path:
        :param params:
        :param data:
        :param headers:
        :param auth:
        :param raise_exception:
        :param value_only:
        :return:
        """
        url = '{}{}'.format(self.url, path)
        headers = headers or {'Content-Type': 'application/json'}
        auth = auth or self.auth
        kwargs = {}
        if not self.ssl_verify:
            kwargs['verify'] = False
        if stream is not None:
            kwargs['stream'] = stream
        request_uuid = str(uuid.uuid1())
        headers["X-Request-UUID"] = request_uuid
        logger.debug('{} {}{}'.format(method.upper(), url, ('?' + urlencode(params, True)) if params else ''))
        try:
            response = requests.request(method, url, params=params, data=data, headers=headers, auth=auth, timeout=self.timeout, **kwargs)
        except requests.exceptions.RequestException as e:
            logger.error('Unexpected exception occurred during request: %s', e)
            raise
        logger.debug('Response status: %s (%s %s)', response.status_code, response.request.method, response.url)
        request_uuid_echo = response.headers.get("X-Request-UUID-Echo", None)
        if request_uuid_echo:
            assert request_uuid == request_uuid_echo, "X-Request-UUID not matching, {} vs {}".format(request_uuid, request_uuid_echo)
        else:
            logger.debug("no request_uuid_echo, %s", response.headers)
        # Raise exception if status code indicates a failure
        if response.status_code >= 400:
            logger.error('Request failed (status: %s, reason: %s)', response.status_code, response.text)
        if raise_exception:
            response.raise_for_status()
        if value_only:
            return self._parse_response(response)
        else:
            return response

    @staticmethod
    def _parse_response(response):
        """Parse response.

        :param response:
        :return:
        """
        if 'application/json' not in response.headers['content-type'].lower():
            return response.text
        else:
            return response.json()

    def get(self, path, params=None, **kwargs):
        """Get.

        :param path:
        :param params:
        :param kwargs:
        :return:
        """
        return self._run_requests(REQUESTS_GET, path, params=params, **kwargs)

    def post(self, path, params=None, data=None, **kwargs):
        """Post.

        :param path:
        :param params:
        :param data:
        :param kwargs:
        :return:
        """
        return self._run_requests(REQUESTS_POST, path, params=params, data=data, **kwargs)

    def put(self, path, params=None, data=None, **kwargs):
        """Put.

        :param path:
        :param params:
        :param data:
        :param kwargs:
        :return:
        """
        return self._run_requests(REQUESTS_PUT, path, params=params, data=data, **kwargs)

    def delete(self, path, params=None, **kwargs):
        """Delete.

        :param path:
        :param params:
        :param kwargs:
        :return:
        """
        return self._run_requests(REQUESTS_DELETE, path, params=params, **kwargs)
