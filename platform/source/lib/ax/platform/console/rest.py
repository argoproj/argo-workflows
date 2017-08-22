import logging
import re
import select
import time

from flask import Flask, render_template, request, jsonify, make_response, Response
from flask_sockets import Sockets
from geventwebsocket.exceptions import WebSocketError
from werkzeug.exceptions import BadRequest

from ax.exceptions import AXException, AXIllegalArgumentException, AXApiInvalidParam, AXIllegalOperationException, AXApiResourceNotFound
from . import console

logger = logging.getLogger('ax.platform.console')

app = Flask(__name__)
sockets = Sockets(app)

# For some reason, BadRequest is not handled by the generic Exception error handler, so both decorators are needed 
@app.errorhandler(400)
@app.errorhandler(Exception)
def error_handler(error):
    # Our exceptions and error handling is a complete mess. Need to clean this up and standardize across teams
    logger.warning(error)
    if isinstance(error, AXException):
        data = error.json()
        if isinstance(error, (AXIllegalArgumentException, AXApiInvalidParam, AXIllegalOperationException)):
            status_code = 400
        elif isinstance(error, AXApiResourceNotFound):
            status_code = 404
        else:
            status_code = 500
    else:
        if isinstance(error, BadRequest):
            code = AXApiInvalidParam.code
            status_code = error.code
        else:
            logger.exception("Internal error")
            code = "ERR_AX_INTERNAL"
            status_code = 500
        data = {"code" : code,
                "message" : str(error),
                "detail" : ""}
    return make_response(jsonify(data), status_code)


@app.route('/')
def list_containers():
    return render_template('containerlist.html')


@app.route('/ping')
def ping():
    return Response('"pong"', mimetype='application/json')


@app.route('/pods/<service_id>/logs')
@app.route('/deployments/<service_id>/logs')
@app.route('/services/<service_id>/logs')
def logs(service_id):
    return render_template('livelog.html')


@app.route('/pods/<service_id>/console')
@app.route('/deployments/<service_id>/console')
@app.route('/services/<service_id>/console')
def console_page(service_id):
    return render_template('console.html')


@app.route('/api/cluster_info')
def api_get_cluster_info():
    return jsonify(console.get_cluster_info())


@app.route('/v1/pods/<string:pod>', methods=['DELETE'])
def api_stop_pod(pod):
    container = request.args.get('container')
    namespace = request.args.get('namespace')
    if not namespace:
        raise AXApiInvalidParam("namespace required")
    console.kill_pod(namespace, pod, container=container)
    return jsonify()


@sockets.route('/v1/pods/<string:service_id>/logs')
@sockets.route('/v1/services/<string:service_id>/logs')
@sockets.route('/v1/deployments/<string:service_id>/logs')
def api_logs_ws(ws, service_id):
    """Attaches to a running container and sends its output to the client websocket"""
    try:
        container = request.args.get('container')
        pod_name = request.args.get('instance')
        if re.search('/v1/services/', request.path):
            generator = console.service_logs(service_id, container=container)
        elif re.search('/v1/deployments/', request.path):
            generator = console.deployment_logs(service_id, container=container, pod_name=pod_name)
        else:
            namespace = request.args.get('namespace')
            if not namespace:
                raise AXApiInvalidParam("namespace required when retrieving logs by podname")
            generator = console.pod_logs(namespace, service_id, container=container)
        socket_check = time.time()
        for chunk in generator:
            ws.send(chunk)
            if time.time() - socket_check > 5:
                # select is called against the client socket to more quickly and reliably detect
                # whether or not the client websocket has been closed. For this API, a client has
                # no reason to be sending any data, unless it is for a close frame. So if select
                # detects there are readable bytes ready, it indicates the socket has been closed.
                # Peform this check every 5 seconds
                readable, _, _ = select.select([ws.stream.handler.socket], [], [], 0.01)
                if readable:
                    # At this point a close frame was likely received. ws.receive() is called to force
                    # geventwebsocket to detect the closed socket. This should result in a WebSocketError
                    # being raised. Otherwise, any data sent by the client is ignored/discarded.
                    _ = ws.receive()
                socket_check = time.time()
    except WebSocketError as e:
        logger.debug("Client websocket closed (%s): %s", service_id, str(e))
    except Exception as err:
        logger.exception("Unknown error")
        ws.send("Unable to retrieve logs for {}: {}".format(service_id, err))


@app.route('/v1/pods/<string:service_id>/logs')
@app.route('/v1/services/<string:service_id>/logs')
@app.route('/v1/deployments/<string:service_id>/logs')
def api_logs(service_id):
    """Streaming implementation of log API"""
    container = request.args.get('container')
    pod_name = request.args.get('instance')
    if re.search('/v1/services/', request.path):
        generator = console.service_logs(service_id, container=container)
    elif re.search('/v1/deployments/', request.path):
        generator = console.deployment_logs(service_id, container=container, pod_name=pod_name)
    else:
        namespace = request.args.get('namespace')
        if not namespace:
            raise AXApiInvalidParam("namespace required when retrieving logs by podname")
        generator = console.pod_logs(namespace, service_id, container=container)
    return Response(generator, mimetype='text/plain')


@sockets.route('/v1/pods/<string:service_id>/exec')
@sockets.route('/v1/services/<string:service_id>/exec')
@sockets.route('/v1/deployments/<string:service_id>/exec')
def api_exec(ws, service_id):
    """Invokes the docker exec API to run a command inside a running container, where service_id is either a service_id, deployment_id, or pod name"""
    try:
        cmd = request.args.get('cmd')
        container = request.args.get('container')
        term_width = request.args.get('w')
        pod_name = request.args.get('instance')
        if term_width:
            term_width = int(term_width)
        term_height = request.args.get('h')
        if re.search('/v1/services/', request.path):
            console.service_exec_start(ws, service_id, cmd=cmd, container=container, term_width=term_width, term_height=term_height)
        elif re.search('/v1/deployments/', request.path):
            console.deployment_exec_start(ws, service_id, cmd=cmd, pod_name=pod_name, container=container, term_width=term_width, term_height=term_height)
        else:
            namespace = request.args.get('namespace')
            if not namespace:
                raise AXApiInvalidParam("namespace required when retrieving logs by podname")
            console.pod_exec_start(ws, namespace, service_id, cmd=cmd, container=container, term_width=term_width, term_height=term_height)
    except Exception as err:
        logger.exception("Unknown error")
        ws.send("Unable to establish exec session to {}: {}".format(service_id, err))


@app.route('/v1/pods')
def api_get_pods():
    pods = list(console.get_pods(field_selector='status.phase=Running'))
    return jsonify(data=pods)


@app.route('/v1/jobs')
def api_get_jobs():
    jobs = console.get_jobs()
    return jsonify(total=len(jobs), data=jobs)

@app.route('/v1/jobs/<string:service_id>', methods=['DELETE'])
def api_stop_job(service_id):
    return jsonify(console.stop_job(service_id))

@app.route('/v1/jobs/<string:service_id>/pods')
def api_get_job_pods(service_id):
    pods = console.get_job_pods(service_id)
    return jsonify(data=pods)

@app.route('/v1/volumepools')
def api_get_volumepools():
    volumepools = console.get_volumepools()
    return jsonify(data=volumepools)

@app.route('/v1/volumepools/<string:pool_name>', methods=['DELETE'])
def api_delete_volumepool(pool_name):
    logger.info("Deleting volume pool %s", pool_name)
    console.delete_volumepool(pool_name)
    return jsonify({})

@app.route('/v1/volumepools/<string:pool_name>/<string:volume_name>', methods=['DELETE'])
def api_delete_volume(pool_name, volume_name):
    logger.info("Deleting volume %s/%s", pool_name, volume_name)
    console.delete_volume(pool_name, volume_name)
    return jsonify({})
