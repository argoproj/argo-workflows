#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import json
import jwt
import logging
import requests
import time

from django.shortcuts import redirect
from rest_framework.mixins import ListModelMixin, CreateModelMixin, RetrieveModelMixin, DestroyModelMixin
from rest_framework.viewsets import GenericViewSet
from rest_framework.decorators import detail_route, list_route
from rest_framework.response import Response

from gateway.settings import LOGGER_NAME
from result.models import Result
from result.serializers import ResultSerializer

from ax.devops.axdb.axdb_client import AxdbClient
from ax.devops.redis.redis_client import RedisClient, DB_RESULT
from ax.devops.kafka.kafka_client import EventNotificationClient
from ax.exceptions import AXApiInvalidParam
from ax.notification_center import FACILITY_PLATFORM, CODE_PLATFORM_ERROR

logger = logging.getLogger('{}.{}'.format(LOGGER_NAME, 'result'))

axdb_client = AxdbClient()
redis_client = RedisClient(host='redis.axsys', db=DB_RESULT, retry_max_attempt=10, retry_wait_fixed=5000)
event_notification_client = EventNotificationClient(FACILITY_PLATFORM)


class ResultViewSet(ListModelMixin, CreateModelMixin, RetrieveModelMixin, DestroyModelMixin, GenericViewSet):
    """View set for result."""

    queryset = Result.objects.all()
    serializer_class = ResultSerializer

    @detail_route(methods=['GET', ])
    def approval(self, request, *args, **kwargs):
        """Save an approval result in redis."""
        token = request.query_params.get('token', None)
        result = jwt.decode(token, 'ax', algorithms=['HS256'])
        result['timestamp'] = int(time.time())

        logger.info("Decode token {}, \n to {}".format(token, json.dumps(result, indent=2)))

        # differentiate key for approval result from the task result
        uuid = result['leaf_id'] + '-axapproval'
        try:
            logger.info("Setting approval result (%s) to Redis ...", uuid)
            try:
                state = axdb_client.get_approval_info(root_id=result['root_id'], leaf_id=result['leaf_id'])
                if state and state[0]['result'] != 'WAITING':
                    return redirect("https://{}/error/404/type/ERR_AX_ILLEGAL_OPERATION;msg=The%20link%20is%20no%20longer%20valid.".format(result['dns']))

                if axdb_client.get_approval_results(leaf_id=result['leaf_id'], user=result['user']):
                    return redirect("https://{}/error/404/type/ERR_AX_ILLEGAL_OPERATION;msg=Response%20has%20already%20been%20submitted.".format(result['dns']))

                # push result to redis (brpop)
                redis_client.rpush(uuid, value=result, encoder=json.dumps)
            except Exception as exc:
                logger.exception(exc)
                pass
            # save result to axdb
            axdb_client.create_approval_results(leaf_id=result['leaf_id'],
                                                root_id=result['root_id'],
                                                result=result['result'],
                                                user=result['user'],
                                                timestamp=result['timestamp'])
        except Exception as e:
            msg = 'Failed to save approval result to Redis: {}'.format(e)
            logger.error(msg)
            raise
        else:
            logger.info('Successfully saved result to Redis')
            return redirect("https://{}/success/201;msg=Response%20has%20been%20submitted%20successfully.".format(result['dns']))

    @list_route(methods=['PUT', ])
    def test_nexus_credential(self, request):
        logger.info('Received testing request (payload: %s)', request.data)
        username = request.data.get('username', "")
        password = request.data.get('password', "")
        port = request.data.get('port', 8081)
        hostname = request.data.get('hostname', None)

        if not hostname:
            raise AXApiInvalidParam('Missing required parameters: Hostname', detail='Missing required parameters, hostname')

        response = requests.get('{}:{}/nexus/service/local/users'.format(hostname, port), auth=(username, password), timeout=10)

        if response.ok:
            return Response({})
        else:
            response.raise_for_status()

    @list_route(methods=['POST', ])
    def redirect_notification_center(self, request):
        logger.info('Received redirecting nc request (payload: %s)', request.data)
        detail = request.data.get('detail', "")
        try:
            event_notification_client.send_message_to_notification_center(CODE_PLATFORM_ERROR, detail={'message': detail})
        except Exception:
            logger.exception("Failed to send event to notification center")
            raise
        return Response({})
