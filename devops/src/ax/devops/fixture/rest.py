import logging

from flask import Flask, request, jsonify, make_response, Response
from werkzeug.exceptions import BadRequest

from . import common
from .util import generate
from ax.exceptions import AXException, AXIllegalArgumentException, AXIllegalOperationException, \
    AXApiResourceNotFound, AXApiInvalidParam


logger = logging.getLogger(__name__)

app = Flask(__name__)
fixmgr = None

def get_json():
    """Helper to retrieve json from the request body, or raise AXApiInvalidParam if invalid"""
    try:
        return request.get_json(force=True)
    except Exception:
        raise AXApiInvalidParam("Invalid json supplied")

def get_fields(params):
    """Helper to retrieve the fields query param as a comma separated list of fields"""
    field_str = params.pop('fields', None)
    if not field_str:
        return None
    return [f.strip() for f in field_str.split(',')]

# For some reason, BadRequest is not handled by the generic Exception error handler, so both decorators are needed 
@app.errorhandler(400)
@app.errorhandler(Exception)
def error_handler(error):
    # Our exceptions and error handling is a complete mess. Need to clean this up and standardize across teams
    if isinstance(error, AXException):
        data = error.json()
        if isinstance(error, (AXIllegalArgumentException, AXApiInvalidParam, AXIllegalOperationException)):
            status_code = 400
        elif isinstance(error, AXApiResourceNotFound):
            status_code = 404
        else:
            logger.exception("Internal error")
            status_code = 500
    else:
        if isinstance(error, BadRequest):
            logger.exception("Bad request")
            code = AXApiInvalidParam.code
            status_code = error.code
        else:
            logger.exception("Internal error")
            code = "ERR_AX_INTERNAL"
            status_code = 500
        data = {"code" : code,
                "message" : str(error),
                "detail" : ""}
    logger.warning('%s (status_code: %s): %s', error, status_code, data)
    return make_response(jsonify(data), status_code)

@app.route('/v1/fixture/ping', methods=['GET'])
def ping():
    return Response('"pong"', mimetype='application/json')

# Fixture Class CRUD

@app.route('/v1/fixture/classes', methods=['GET'])
def get_fixture_classes():
    params = request.args.copy()
    fields = get_fields(params)
    classes = fixmgr.get_fixture_classes()
    return jsonify(data=[c.json(fields=fields) for c in classes])

@app.route('/v1/fixture/classes', methods=['POST'])
def create_fixture_class():
    payload = get_json()
    template_id = payload.get('template_id')
    if not template_id:
        raise AXApiInvalidParam("Required argument 'template_id' not supplied")
    fix_class = fixmgr.upsert_fixture_class(template_id)
    return jsonify(fix_class.json())

@app.route('/v1/fixture/classes/<class_id>', methods=['GET'])
def get_fixture_class(class_id):
    params = request.args.copy()
    fields = get_fields(params)
    fix_class = fixmgr.get_fixture_class(id=class_id)
    return jsonify(fix_class.json(fields=fields))

@app.route('/v1/fixture/classes/<class_id>', methods=['PUT'])
def update_fixture_class(class_id):
    updates = get_json()
    template_id = updates.get('template_id')
    if not template_id:
        raise AXApiInvalidParam("Required argument 'template_id' not supplied")
    fix_class = fixmgr.update_fixture_class(class_id, template_id)
    return jsonify(fix_class.json())

@app.route('/v1/fixture/classes/<class_id>', methods=['DELETE'])
def delete_fixture_class(class_id):
    fixmgr.delete_fixture_class(class_id)
    return jsonify(id=class_id)

# Fixture CRUD

@app.route('/v1/fixture/instances', methods=['GET'])
def get_fixture_instances():
    params = request.args.to_dict()
    fields = get_fields(params)
    fixtures = fixmgr.query_fixture_instances(query=params)
    return Response(generate(fixtures, fields=fields), content_type='application/json')

@app.route('/v1/fixture/instances', methods=['POST'])
def create_fixture_instance():
    fixture_dict = get_json()
    username = _get_user_context()[1]
    if username:
        fixture_dict['creator'] = username
        fixture_dict['owner'] = username
    fixture = fixmgr.create_fixture_instance(fixture_dict)
    return jsonify(fixture.json())

@app.route('/v1/fixture/instances/<fixture_id>', methods=['GET'])
def get_fixture_instance(fixture_id):
    params = request.args.copy()
    fields = get_fields(params)
    fixture = fixmgr.get_fixture_instance(id=fixture_id)
    return jsonify(fixture.json(fields=fields))

@app.route('/v1/fixture/instances/<fixture_id>', methods=['PUT'])
def update_fixture_instance(fixture_id):
    updates = get_json()
    if 'id' in updates and updates['id'] != fixture_id:
        raise AXApiInvalidParam("Fixture id cannot be updated")
    updates['id'] = fixture_id
    username = _get_user_context()[1]
    fixture = fixmgr.update_fixture_instance(updates, user=username)
    return jsonify(fixture.json())

@app.route('/v1/fixture/instances/<fixture_id>', methods=['DELETE'])
def delete_fixture_instance(fixture_id):
    username = _get_user_context()[1]
    fixture_id = fixmgr.delete_fixture_instance(fixture_id, user=username)
    return jsonify(id=fixture_id)


@app.route('/v1/fixture/instances/<fixture_id>/action', methods=['POST'])
def perform_fixture_instance_action(fixture_id):
    username = _get_user_context()[1]
    payload = get_json()
    if not 'action' in payload:
        raise AXApiInvalidParam("Must specify 'action' field")
    action = payload['action']
    arguments = payload.get('arguments')
    instance = fixmgr.perform_fixture_instance_action(fixture_id, action, user=username, arguments=arguments)
    return jsonify(instance.json())

# Stats endpoints
@app.route('/v1/fixture/summary', methods=['GET'])
def get_fixture_summary():
    params = request.args.to_dict()
    group_by = params.pop('group_by', None)
    return jsonify(fixmgr.get_summary(group_by=group_by, query=params))

# Misc internally used endpoints

@app.route('/v1/fixture/action_result', methods=['POST'])
def process_action_result():
    fixmgr.process_action_result(get_json())
    return jsonify({})

@app.route('/v1/fixture/template_updates', methods=['POST'])
def notify_template_updates():
    try:
        fixmgr.notify_template_updates()
    except Exception:
        logger.exception("Template update failed")
    return jsonify({})

# Fixture Request CRUD

@app.route('/v1/fixture/requests', methods=['GET'])
def get_fixture_requests():
    fix_reqs = fixmgr.reqproc.get_fixture_requests()
    return jsonify(data=[fr.json() for fr in fix_reqs])

@app.route('/v1/fixture/requests/<service_id>', methods=['GET'])
def get_fixture_request(service_id):
    fix_req = fixmgr.reqproc.get_fixture_request(service_id)
    return jsonify(fix_req.json())

@app.route('/v1/fixture/requests', methods=['POST'])
def create_fixture_request():
    fix_req = fixmgr.reqproc.create_fixture_request(get_json())
    return jsonify(fix_req.json())

@app.route('/v1/fixture/requests_mock', methods=['POST'])
def create_fixture_request_mock():
    fix_req = fixmgr.reqproc.create_fixture_request_mock(get_json())
    return jsonify(fix_req.json())

@app.route('/v1/fixture/requests/<service_id>', methods=['DELETE'])
def delete_fixture_request(service_id):
    service_id = fixmgr.reqproc.delete_fixture_request(service_id)
    return jsonify(service_id=service_id)

# Volume CRUD

def _get_user_context():
    user_id = request.headers.get(common.HTTP_AX_USERID_HEADER)
    username = request.headers.get(common.HTTP_AX_USERNAME_HEADER)
    return user_id, username

@app.route('/v1/storage/volumes', methods=['POST'])
def create_volume():
    volume_dict = get_json()
    username = _get_user_context()[1]
    if username:
        volume_dict['creator'] = username
        volume_dict['owner'] = username
    volume = fixmgr.volumemgr.create_volume(volume_dict)
    return jsonify(volume.json())

@app.route('/v1/storage/volumes', methods=['GET'])
def get_volumes():
    anonymous = request.args.get('anonymous') or None
    if anonymous is not None:
        anonymous = True if anonymous.lower() in ['true', 't', '1'] else False
    deployment_id = request.args.get('deployment_id') or None
    volumes = fixmgr.volumemgr.get_volumes(anonymous=anonymous, deployment_id=deployment_id)
    return jsonify(data=[v.json() for v in volumes])

@app.route('/v1/storage/volumes/<volume_id>', methods=['GET'])
def get_volume(volume_id):
    volume = fixmgr.volumemgr.get_volume(volume_id)
    return jsonify(volume.json())

@app.route('/v1/storage/volumes/<volume_id>', methods=['PUT'])
def update_volume(volume_id):
    updates = get_json()
    if 'id' in updates and updates['id'] != volume_id:
        raise AXApiInvalidParam("Volume id cannot be updated")
    updates['id'] = volume_id
    volume = fixmgr.volumemgr.update_volume(updates)
    return jsonify(volume.json())

@app.route('/v1/storage/volumes/<volume_id>', methods=['DELETE'])
def delete_volume(volume_id):
    fixmgr.volumemgr.mark_for_deletion(volume_id)
    return jsonify(volume_id=volume_id)
