#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2015 Applatix, Inc. All rights reserved.
#
import os
import logging
import traceback
from inspect import isfunction

from flask import Flask, request, make_response, render_template
from flask import jsonify as original_jsonify
from gevent import pywsgi

from ax.exceptions import AXException, AXIllegalOperationException, AXServiceTemporarilyUnavailableException, AXIllegalArgumentException
from werkzeug.exceptions import BadRequest

from .adc_main import ADC, ADC_DEFAULT_PORT

dir_path = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'templates')
_app = Flask("ADC", template_folder=dir_path)
adc = None


def jsonify(*args, **kwargs):
    return ax_make_response(original_jsonify(*args, **kwargs), 200)


def ax_make_response(rv, status, headers=None):
    new_headers = {"Pragma": "no-cache",
                   "Cache-Control": "no-cache"}

    request_uuid = request.headers.get("X-Request-UUID", None)
    if request_uuid:
        new_headers["X-Request-UUID-Echo"] = request_uuid

    if headers:
        headers.update(new_headers)
    else:
        headers = new_headers

    return make_response(rv, status, headers)


def get_adc_url_root():
    url_root = ''.join(request.url.partition("/v1/adc/")[:2])
    return url_root


@_app.errorhandler(Exception)
def error_handler(error):
    if isinstance(error, AXException):
        if isinstance(error, AXIllegalArgumentException):
            status_code = 400
        elif isinstance(error, AXServiceTemporarilyUnavailableException):
            status_code = 503
        elif isinstance(error, AXIllegalOperationException):
            status_code = 500
        else:
            status_code = 500

        error_dict = error.json()
        _app.logger.exception(error_dict)
        return ax_make_response(original_jsonify(error_dict), status_code)
    data = {"message": str(error), "backtrace": str(traceback.format_exc())}
    if isinstance(error, ValueError):
        _app.logger.warn(str(error))
        status_code = 400
    else:
        _app.logger.exception("Internal error")
        status_code = 500
    return ax_make_response(original_jsonify(data), status_code)


@_app.route('/')
@_app.route("/v1/adc/help")
def adc_api_help():
    """
    Print out this help.
    """
    _help_msg["headers"] = str(request.headers)
    return jsonify(_help_msg)


@_app.route("/v1/adc/ping")
def adc_api_ping():
    """
    Ping to check whether server is alive.
    """
    return jsonify({"status": "OK"})


@_app.route("/v1/adc/state")
def adc_api_state():
    """
    Return current ADC state
    """
    return jsonify(adc.get_state())


@_app.route("/v1/adc/state", methods=['POST'])
def adc_api_set_state():
    """
    Set ADC state.
    """
    try:
        input_json = request.get_json(force=True)
        new_state = input_json["state"]
    except Exception:
        raise ValueError("Invalid json: {}".format(request.get_data()))

    return jsonify(adc.request_set_state(new_state=new_state))


@_app.route("/v1/adc/test")
def adc_api_test():
    """
    Test API.
    """
    return jsonify({"state": adc.state})


@_app.route("/v1/adc/version")
def adc_api_version():
    """
    Return current ADC version.
    """
    return jsonify({"version": adc.version})


@_app.route("/v1/adc/workflows", methods=['GET'])
def adc_api_workflows():
    """
    Return current active ADC workflows
    """
    default_recent_second = 600
    try:
        recent_seconds = request.args.get('recent', str(default_recent_second))
        recent_seconds = int(recent_seconds)
    except Exception:
        recent_seconds = default_recent_second

    try:
        verbose = bool(request.args.get('verbose', 'False').lower() not in {'false', '0'})
    except Exception:
        verbose = False
    return jsonify(adc.workflows_show(recent_seconds=recent_seconds, verbose=verbose, url_root=get_adc_url_root()))


@_app.route("/v1/adc/workflows/all", methods=['DELETE'])
def adc_api_workflows_delete():
    """
    Return current active ADC workflows
    """

    return jsonify(adc.workflows_delete())


@_app.route("/v1/adc/workflows/<workflow_id>", methods=['GET'])
def adc_api_workflow_show(workflow_id):
    """
    Show a single workflow

    Return a dict of single workflow
    """
    state_only = request.args.get('state_only', 'False').lower() not in {'false', '0'}
    workflow_dict = adc.workflow_show(workflow_id, state_only=state_only)
    return jsonify(workflow_dict)


@_app.route("/v1/adc/workflow_create_random", methods=['POST'])
def adc_api_workflow_create_random():
    """
    Create or add a random workflow instance.
    """
    try:
        input_json = request.get_json(force=True)
    except Exception:
        input_json = None

    return jsonify(adc.workflow_create_random(input_json=input_json))


@_app.route("/v1/adc/workflows", methods=['POST'])
def adc_api_workflow_create():
    """
    Create or add a new workflow instance.
    """
    workflow_json = request.get_json(force=True)

    return jsonify(adc.workflow_create(workflow_json=workflow_json))


@_app.route("/v1/adc/workflows/<workflow_id>", methods=['DELETE'])
def adc_api_workflow_delete(workflow_id):
    """
    Delete a single workflow
    """
    force_deletiong = request.args.get('force', 'False').lower()

    return jsonify(adc.workflow_delete(workflow_id=workflow_id, force=force_deletiong=='true'))


# remove in next upgrade
@_app.route("/v1/adc/notification/workflow_done", methods=['POST'])
def adc_api_notification_workflow_done():
    """
    notification of finish of a workflow
    """
    json = request.get_json(force=True)
    return jsonify(adc.notification_workflow(json))


@_app.route("/v1/adc/notification/workflow", methods=['POST'])
def adc_api_notification_workflow():
    """
    notification of finish of a workflow
    """
    json = request.get_json(force=True)
    return jsonify(adc.notification_workflow(json))


@_app.route("/v1/adc/notification/resource", methods=['POST'])
def adc_api_notification_resource():
    """
    notification of system resource
    """
    json = request.get_json(force=True)
    return jsonify(adc.notification_resource(json))


@_app.route("/v1/adc/d3_1/<workflow_id>", methods=['GET'])
def adc_api_workflow_d3_1_show(workflow_id):
    """
    Get the d3 json object for workflow tree
    """
    workflow_dict = adc.workflow_d3_1_format(workflow_id)
    return jsonify(workflow_dict)


@_app.route("/v1/adc/d3_2/<workflow_id>", methods=['GET'])
def adc_api_workflow_d3_2_show(workflow_id):
    """
    Get the d3 json object for workflow tree
    """
    workflow_dict = adc.workflow_d3_2_format(workflow_id)
    return jsonify(workflow_dict)


@_app.route("/v1/adc/d3_3/<workflow_id>", methods=['GET'])
def adc_api_workflow_d3_3_show(workflow_id):
    """
    Get the d3 json object for workflow tree
    """
    workflow_dict = adc.workflow_d3_3_format(workflow_id)
    return jsonify(workflow_dict)


@_app.route("/v1/adc/workflows_v/<workflow_id>", methods=['GET'])
def adc_api_visual(workflow_id):
    """
    Visualize the workflow tree
    """
    return render_template("index.html", v='1', adc_url_root=get_adc_url_root(), workflow_id=workflow_id)


@_app.route("/v1/adc/workflows_v2/<workflow_id>", methods=['GET'])
def adc_api_visual_2(workflow_id):
    """
    Visualize the workflow tree
    """
    return render_template("index.html", v='2', adc_url_root=get_adc_url_root(), workflow_id=workflow_id)


@_app.route("/v1/adc/workflows_v3/<workflow_id>", methods=['GET'])
def adc_api_visual_3(workflow_id):
    """
    Visualize the workflow tree
    """
    return render_template("index.html", v='3', adc_url_root=get_adc_url_root(), workflow_id=workflow_id)


@_app.route("/v1/adc/workflows/<workflow_id>/events", methods=['GET'])
def adc_api_workflow_event(workflow_id):
    """
    Show workflow node events that reported to kafka.
    """
    return jsonify(adc.workflow_event_show(workflow_id))


@_app.route("/v1/adc/resource", methods=['PUT'])
def adc_api_resource_reserve_resource():
    """
    Reserve resource.
    """
    json = request.get_json(force=True)
    return jsonify(adc.resource_reserve_resource(json))


@_app.route("/v1/adc/resource/<resource_id>", methods=['DELETE'])
def adc_api_resource_release_resource(resource_id):
    """
    Release resource.
    """
    return jsonify(adc.resource_release_resource(resource_id))


def _get_optional_arguments(*args):
    """
    Returns a tuple of optional param values from a request payload
    """
    try:
        data = request.get_json(force=True) or {}
    except BadRequest:
        raise ValueError("Invalid json: {}".format(request.get_data()))
    return tuple(map(lambda arg: data[arg] if arg in data else None, args))


def adc_rest_start(port=None):
    """
    Start Flask http server
    """
    if port is None:
        port = ADC_DEFAULT_PORT

    global adc
    adc = ADC()
    adc.set_port(port=port)

    _app.logger.setLevel(logging.DEBUG)
    http_server = pywsgi.WSGIServer(('', port), _app)
    _app.logger.info("ADC %s started (port: %s)",
                     adc.version, port)
    http_server.start()


# Must be last in this module to make help work.
_all_locals = dict(locals())
_help_msg = {}
for f in _all_locals:
    if isfunction(_all_locals[f]) and f.startswith("adc_api_"):
        _help_msg[f.replace("adc_api_", "")] = _all_locals[f].__doc__.splitlines()
