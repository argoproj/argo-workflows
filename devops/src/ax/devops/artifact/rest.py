# Copyright 2015-2016 Applatix, Inc. All rights reserved.

import logging

from flask import Flask, make_response, Response, request, redirect
from flask import jsonify as original_jsonify
from requests.exceptions import HTTPError

from ax.exceptions import AXException, AXApiResourceNotFound, AXApiInvalidParam, AXApiInternalError
from ax.devops.artifact.constants import RETENTION_TAG_USER_LOG

app = Flask(__name__)

artifact_manager = None

app.logger.setLevel(logging.DEBUG)


def jsonify(*args, **kwargs):
    return ax_make_response(original_jsonify(*args, **kwargs), 200)


def ax_make_response(rv, status, headers=None):
    new_headers = {"Pragma": "no-cache", "Cache-Control": "no-cache"}
    request_uuid = request.headers.get("X-Request-UUID", None)
    if request_uuid:
        new_headers["X-Request-UUID-Echo"] = request_uuid
    if headers:
        headers.update(new_headers)
    else:
        headers = new_headers
    return make_response(rv, status, headers)


@app.errorhandler(400)
@app.errorhandler(404)
@app.errorhandler(500)
@app.errorhandler(Exception)
def error_handler(error):
    if isinstance(error, AXException):
        data = error.json()
        if isinstance(error, AXApiInvalidParam):
            status_code = 400
        elif isinstance(error, AXApiResourceNotFound):
            status_code = 404
        else:
            status_code = 500
        f = app.logger.exception if status_code == 500 else app.logger.error
        f('Error occurred: code = %s, message = %s, detail = %s', data.get('code'), data.get('message'), data.get('detail'))
    else:
        # If exception is HTTPError, we need to examine its status code to determine
        # if it is client-side error or server-side error
        if isinstance(error, HTTPError):
            if error.response.status_code == 404:
                _error = AXApiResourceNotFound
            elif 400 <= error.response.status_code < 500:
                _error = AXApiInvalidParam
            else:
                _error = AXApiInternalError
        else:
            _error = AXApiInternalError
        data = {
            'code': _error.code,
            'message': str(error),
            'detail': str(error)
        }
        status_code = _error.status_code
        f = app.logger.exception if status_code == 500 else app.logger.error
        f('Error occurred: code = %s, message = %s, detail = %s', data.get('code'), data.get('message'), data.get('detail'))
    return ax_make_response(original_jsonify(data), status_code)


###################
# Debug endpoints #
###################


@app.route('/v1/ping', methods=['GET'])
def ping():
    """Ping artifact manager"""
    return Response('"pong"', mimetype='application/json')


@app.route('/v1/overview', methods=['GET'])
def overview():
    """Overview of artifact manager for debugging purpose"""
    return jsonify(artifact_manager.show())


####################
# Policy endpoints #
####################


@app.route('/v1/retention_policies', methods=['GET'])
def get_policies():
    """Get artifact retention policies"""
    policies = artifact_manager.get_retention_policies()
    return jsonify({'data': policies})


@app.route('/v1/retention_policies/<name>', methods=['GET'])
def get_policy(name):
    """Get artifact retention policy with policy name"""
    policies = artifact_manager.get_retention_policies(tag_name=name)
    if not policies:
        raise AXApiResourceNotFound('Policy not found', detail='Policy with name ({}) cannot be found'.format(name))
    return jsonify({'data': policies})


@app.route('/v1/retention_policies', methods=['POST'])
def add_policy():
    """Add artifact retention policy"""
    policy_json = request.get_json(force=True)
    if 'name' not in policy_json or 'policy' not in policy_json:
        raise AXApiInvalidParam('Missing required parameters', detail='Missing required parameters ("name", "policy")')
    description = policy_json.get('description')
    result = artifact_manager.add_retention_policy(tag_name=policy_json['name'], policy=policy_json['policy'], description=description)
    return jsonify(result)


@app.route('/v1/retention_policies/<name>', methods=['PUT'])
def update_policy(name):
    """Update artifact retention policy"""
    policy_json = request.get_json(force=True)
    policy = policy_json.get('policy', None)
    description = policy_json.get('description', None)
    result = artifact_manager.update_retention_policy(tag_name=name, policy=policy, description=description)
    return jsonify(result)


@app.route('/v1/retention_policies/<name>', methods=['DELETE'])
def delete_policy(name):
    """Delete artifact retention policy with policy name"""
    result = artifact_manager.delete_retention_policy(tag_name=name)
    return jsonify(result)


######################
# Artifact endpoints #
######################


@app.route('/v1/artifacts', methods=['POST'])
def create_artifact():
    """Create an artifact"""
    artifact = request.get_json()
    artifact_manager.create_artifact(artifact)
    return jsonify(artifact)


@app.route('/v1/artifacts', methods=['GET'])
def get_artifacts():
    """Search/retrieve/browse/download artifact(s)"""
    action = request.args.get('action')
    if not action:
        raise AXApiInvalidParam('Missing required parameter', 'Missing required parameter (action)')
    if action == 'search':
        return _search_artifacts(request.args)
    elif action == 'retrieve':
        return _retrieve_artifact(request.args)
    elif action == 'browse':
        return _browse_artifact(request.args)
    elif action == 'download':
        return _download_artifact(request.args)
    elif action == 'list_tags':
        return _list_tags(request.args)
    elif action == 'get_usage':
        return _get_usage()
    else:
        raise AXApiInvalidParam('Invalid parameter value', 'Unsupported action ({})'.format(action))


@app.route('/v1/artifacts/<artifact_id>', methods=['GET'])
def get_artifact(artifact_id):
    artifact = artifact_manager.get_artifact(artifact_id)  # Requested by AA-3063
    return jsonify(artifact)


@app.route('/v1/artifacts', methods=['PUT'])
def update_artifacts():
    """Delete/restore/tag/untag artifact(s)"""
    payload = request.get_json(force=True)
    action = payload.get('action')
    if not action:
        raise AXApiInvalidParam('Missing required parameter', 'Missing required parameter (action)')
    if action == 'delete':
        return _delete_artifacts(payload)
    elif action == 'restore':
        return _restore_artifacts(payload)
    elif action == 'tag':
        return _tag_artifacts(payload)
    elif action == 'untag':
        return _untag_artifacts(payload)
    elif action == 'update_retention':
        return _update_retention(payload)
    elif action == 'clean':
        return _clean_artifacts()
    else:
        raise AXApiInvalidParam('Invalid parameter value', 'Unsupported action ({})'.format(action))


def _clean_artifacts():
    """Manually trigger retention background thread from s3"""
    artifact_manager._trigger_processor()
    return jsonify({})


def _search_artifacts(params):
    """Search artifacts

    :param params:
    :returns:
    """
    parsed_params = {}
    for k in ('artifact_id', 'full_path', 'name', 'service_instance_id', 'workflow_id', 'is_alias'):
        if k in request.args:
            parsed_params[k] = params[k]
    for k in ('deleted', 'retention_tags', 'tags'):
        if k in request.args:
            parsed_params[k] = params[k].split(',')
    artifacts = artifact_manager.get_artifacts(**parsed_params)
    return jsonify({'data': artifacts})


def _retrieve_artifact(params):
    """Retrieve an artifact

    :param params:
    :returns:
    """
    if 'artifact_id' not in params:
        raise AXApiInvalidParam('Missing required parameter', 'Missing required parameter (artifact_id)')
    artifact = artifact_manager.get_artifact(request.args['artifact_id'])
    return jsonify(artifact)


def _browse_artifact(params):
    """Browse an artifact

    :param params:
    :returns:
    """
    if 'artifact_id' not in params:
        raise AXApiInvalidParam('Missing required parameter', 'Missing required parameter (artifact_id)')
    structure = artifact_manager.browse_artifact(request.args['artifact_id'])
    return jsonify(structure)


def _download_artifact(params):
    """Download an artifact

    :param params:
    :returns:
    """
    if 'artifact_id' in params:
        _params = {'artifact_id': params['artifact_id']}
    elif 'service_instance_id' in params:
        if 'name' in params:
            _params = {
                'service_instance_id': params['service_instance_id'],
                'name': params['name']
            }
        elif 'retention_tags' in params and params['retention_tags'] == RETENTION_TAG_USER_LOG:
            _params = {
                'service_instance_id': params['service_instance_id'],
                'retention_tags': params['retention_tags']
            }
        elif 'retention_tags' in params and params['retention_tags'] != RETENTION_TAG_USER_LOG:
            raise AXApiInvalidParam('Invalid parameter value',
                                    'Can only download {} artifacts when "name" is not supplied'.format(RETENTION_TAG_USER_LOG))
        else:
            raise AXApiInvalidParam('Missing required parameter',
                                    'Must supply either "name" or "retention_tags" when supplying "service_instance_id"')
    elif 'workflow_id' in params:
        if 'full_path' not in params or 'name' not in params:
            raise AXApiInvalidParam('Missing required parameter',
                                    'Must supply both "full_path" and "name" when supplying "workflow_id"')
        _params = {
            'workflow_id': params['workflow_id'],
            'full_path': params['full_path'],
            'name': params['name']
        }
    else:
        raise AXApiInvalidParam('Missing required parameter', 'Must supply "artifact_id", "service_instance_id", or "workflow_id"')
    location, content = artifact_manager.download_artifact_by_query(**_params)
    if location:
        return redirect(location, code=302)
    if content:
        return Response(content)
    else:
        raise AXApiInternalError('Internal Error')


def _list_tags(params):
    """List tags"""
    return jsonify({'data': artifact_manager.get_tags(params)})


def _get_usage():
    """Get usage"""
    artifact_nums = artifact_manager.get_artifact_nums()
    artifact_size = artifact_manager.get_artifact_size()
    return jsonify({'data': {'artifact_nums': artifact_nums, 'artifact_size': artifact_size}})


def _delete_artifacts(payload):
    """Delete artifacts

    :param payload:
    :returns:
    """
    if 'artifact_id' in payload:
        params = {
            'artifact_id': payload['artifact_id'],
            'deleted_by': payload.get('deleted_by')
        }
    elif 'workflow_ids' in payload:
        params = {
            'workflow_ids': set(payload['workflow_ids']),
            'retention_tag': payload.get('retention_tag'),
            'deleted_by': payload.get('deleted_by')
        }
    else:
        raise AXApiInvalidParam('Missing required parameter', 'Must supply either "artifact_id" or "workflow_ids"')
    artifact_manager.delete_artifacts(**params)
    return jsonify({})


def _restore_artifacts(payload):
    """Restore artifacts

    :param payload:
    :returns:
    """
    if 'artifact_id' in payload:
        params = {
            'artifact_id': payload['artifact_id']
        }
        artifact_manager.restore_artifact(**params)
    elif 'workflow_ids' in payload:
        params = {
            'workflow_ids': set(payload['workflow_ids']),
            'retention_tag': payload.get('retention_tag')
        }
        artifact_manager.restore_artifacts(**params)
    else:
        raise AXApiInvalidParam('Missing required parameter', 'Must supply either "artifact_id" or "workflow_ids"')
    return jsonify({})


def _tag_artifacts(payload):
    """Tag artifacts

    :param payload:
    :returns:
    """
    if 'tag' not in payload or 'workflow_ids' not in payload:
        raise AXApiInvalidParam('Missing required parameters', 'Must supply "tag" and "workflow_ids"')
    artifact_tags = payload['tag'].split(',')
    if not artifact_tags:
        return jsonify({})
    for workflow_id in payload['workflow_ids']:
        artifact_manager.tag_workflow(workflow_id, artifact_tags)
    return jsonify({})


def _untag_artifacts(payload):
    """Untag artifacts

    :param payload:
    :returns:
    """
    if 'tag' not in payload or 'workflow_ids' not in payload:
        raise AXApiInvalidParam('Missing required parameters', 'Must supply "tag" and "workflow_ids"')
    artifact_tags = payload['tag'].split(',')
    if not artifact_tags:
        return jsonify({})
    for workflow_id in payload['workflow_ids']:
        artifact_manager.untag_workflow(workflow_id, artifact_tags)
    return jsonify({})


def _update_retention(payload):
    """Update retention tag

    :param payload:
    :returns:
    """
    if 'retention_tag' not in payload or 'artifact_id' not in payload:
        raise AXApiInvalidParam('Missing required parameters', 'Must supply "retention_tag" and "artifact_id"')
    artifact_manager.update_artifact_retention_tag(payload['artifact_id'], payload['retention_tag'])
    return jsonify({})
