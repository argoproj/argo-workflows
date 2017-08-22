import logging

from flask import Flask, jsonify, make_response, Response
from werkzeug.exceptions import BadRequest
from ax.exceptions import AXException, AXIllegalArgumentException, AXIllegalOperationException, \
    AXApiResourceNotFound, AXApiInvalidParam

logger = logging.getLogger(__name__)

app = Flask(__name__)
jobScheduler = None


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
            status_code = 500
    else:
        if isinstance(error, BadRequest):
            code = AXApiInvalidParam.code
            status_code = error.code
        else:
            logger.exception("Internal error")
            code = "ERR_AX_INTERNAL"
            status_code = 500
        data = {"code": code,
                "message": str(error),
                "detail": ""}
    return make_response(jsonify(data), status_code)


@app.route('/v1/scheduler/ping', methods=['GET'])
def ping():
    """
    Ping job scheduler.
    """
    return Response('"pong"', mimetype='application/json')


@app.route('/v1/scheduler/show', methods=['GET'])
def get_schedules():
    """
    Show scheduled jobs in the current scheduler.
    """
    return jsonify(jobScheduler.get_schedules())


@app.route('/v1/scheduler/refresh', methods=['GET'])
def refresh():
    """
    Refresh the internal cron scheduler.
    """
    return jsonify(jobScheduler.refresh_scheduler())
