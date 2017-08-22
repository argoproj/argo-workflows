#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#

import logging
import smtplib
import json
from . import myapp, db
from .models import Configuration
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.header import Header
from email.utils import formataddr
from flask import jsonify, make_response, request
from flask_restful import Api, Resource, abort
from voluptuous import Schema, MultipleInvalid, Required, Optional
from ax.devops.axdb.axdb_client import AxdbClient
from ax.exceptions import AXIllegalOperationException, \
    AXIllegalArgumentException

logger = logging.getLogger(__name__)
logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s")
logging.getLogger("ax").setLevel(logging.DEBUG)

# Schema for configuration
configuration_schema = Schema({
    Required('id'): str,
    Required('type'): str,
    Optional('nickname', default=None): str,
    Optional('active', default=True): bool,
}, extra=True)

# Schema for smtp
smtp_schema = Schema({
    Required('id'): str,
    Required('type'): str,
    Optional('nickname', default=None): str,
    Required('url'): str,
    Required('admin_address'): str,
    Optional('port', default=None): int,
    Required('use_tls'): bool,
    Optional('username', default=None): str,
    Optional('password', default=None): str,
    Optional('timeout', default=30): int,
    Optional('active', default=True): bool,
}, extra=True)

# Schema for email
# The smtp field could be provided. If so, it will override the smtp in the db.
email_schema = Schema({
    Required('to'): [str],
    Optional('from', default=None): str,
    Required('subject'): str,
    Required('body'): str,
    Optional('smtp'): smtp_schema,
    Optional('html', default=False): bool,
    Optional('display_name'): str,
}, extra=True)


@myapp.errorhandler(404)
def page_not_found(e):
    return make_response(jsonify(AXIllegalOperationException(str(e)).json()),
                         404)


def init_db():
    results = AxdbClient().retry_request('get', '/axops/tools',
                                         params={'type': 'smtp'},
                                         value_only=True)
    assert isinstance(results, list)
    logger.info("Start initiating database (sqlite3) using data from axdb")
    for result in results:
        logger.info("Config information from axdb, %s", result)
        configuration = json.loads(result['config'])
        config, config_detail = Common.parse_smtp_argument(configuration)
        new_config = Configuration(id=config['id'],
                                   type=config['type'],
                                   active=config['active'],
                                   nickname=config['nickname'],
                                   config=json.dumps(config_detail))
        db.session.add(new_config)
        db.session.commit()
    logger.info("Finish initiating database (sqlite3) using data from axdb")


class Common(object):
    """Parse for all types of configuration arguments"""
    # Validate the argument for respected type
    @classmethod
    def parse_smtp_argument(cls, argument, test=False):
        try:
            if test and 'id' not in argument:
                argument['id'] = "NA"
            config = configuration_schema(argument)
            config_detail = config
            if config['type'] == 'smtp':
                config_detail = smtp_schema(argument)
                if not config_detail['port']:
                    if config_detail['use_tls']:
                        config_detail['port'] = 465
                    else:
                        config_detail['port'] = 25
            else:
                error_msg = \
                    "Unsupported notification type: {}".format(config['type'])
                logger.error(error_msg)
                abort(400, **(AXIllegalArgumentException(error_msg).json()))
            return config, config_detail
        except MultipleInvalid or Exception as exc:
            logger.exception("Fail to parse argument. %s", str(exc))
            abort(400, **(AXIllegalArgumentException(str(exc)).json()))

    @staticmethod
    def hide_password(argument):
        if isinstance(argument, dict):
            argument_copy = argument.copy()
            if 'username' in argument_copy:
                argument_copy['username'] = '******'
            if 'password' in argument_copy:
                argument_copy['password'] = '******'
            return argument_copy
        return argument

    @staticmethod
    def pretty_json(argument):
        if isinstance(argument, dict):
            return json.dumps(argument, indent=4, sort_keys=True)
        return str(argument)


class EmailNotificationAPI(Resource):
    """API for sending emails. Only POST method will be provided."""

    @staticmethod
    def send_mail_smtp(smtp_info, email_content=None, test=False):
        """A function that sends email using SMTP
        :param email_content: details about the particular email to be sent
        :param smtp_info: SMTP configuration to use
        :param test: If used for testing configuration
        """
        logger.info("SMTP info: %s",
                    Common.pretty_json(Common.hide_password(smtp_info)))
        mail_server = smtplib.SMTP(smtp_info['url'],
                                   smtp_info['port'],
                                   timeout=smtp_info['timeout'])
        if smtp_info['use_tls']:
            mail_server.ehlo()
            mail_server.starttls()
            mail_server.ehlo()
        if smtp_info['username'] or smtp_info['password']:
            mail_server.login(smtp_info['username'], smtp_info['password'])

        if not test:
            msg = MIMEMultipart()
            if not email_content['from']:
                if 'display_name' in email_content:
                    msg['From'] = formataddr((str(Header(email_content['display_name'],
                                                         'utf-8')), smtp_info['admin_address']))
                else:
                    msg['From'] = formataddr((str(Header('Argo',
                                                         'utf-8')), smtp_info['admin_address']))
            else:
                if 'display_name' in email_content:
                    msg['From'] = formataddr((str(Header(email_content['display_name'],
                                                         'utf-8')), email_content['from']))
                else:
                    msg['From'] = email_content['from']
            msg['To'] = ", ".join(email_content['to'])
            msg['Subject'] = email_content['subject']

            if email_content.get('html', False):
                body = MIMEText(email_content['body'], 'html')
            else:
                body = MIMEText(email_content['body'], 'plain')
            msg.attach(body)
            logger.info("Email Content %s", msg)
            mail_server.sendmail(msg['From'], email_content['to'], msg.as_string())
        mail_server.quit()

    @classmethod
    def post(cls):
        """Email API for sending emails.
        Take json argument contains the email information
        """
        logger.info("Received email sending request: %s",
                    Common.pretty_json(request.json))
        email_info = {}
        try:
            email_info = email_schema(request.json)
        except MultipleInvalid or Exception as exc:
            logger.exception("Fail to parse argument. %s", str(exc))
            abort(400, **(AXIllegalArgumentException(str(exc)).json()))

        # If customized smtp configuration is supplied in the payload
        if email_info.get('smtp', None):
            smtp_info = email_info['smtp']
            try:
                cls.send_mail_smtp(smtp_info,
                                   email_content=email_info, test=False)
            except Exception as exc:
                logger.exception("Fail to send email. %s", str(exc))
                abort(400, **(AXIllegalOperationException(str(exc)).json()))
            return {}
        # Using the configs in the database
        else:
            smtp_result = Configuration.query.filter_by(active=True,
                                                        type='smtp')
            if smtp_result.count() == 0:
                error_msg = \
                    "No active SMTP Server found. Please configure " \
                    "it in the configuration page."
                logger.exception(error_msg)
                abort(400, **(AXIllegalOperationException(error_msg).json()))

            error_msg = ""
            for smtp_info in smtp_result:
                try:
                    cls.send_mail_smtp(json.loads(smtp_info.config),
                                       email_content=email_info, test=False)
                    return {}
                except Exception as exc:
                    error_msg += str(exc)
                    logger.exception("Fail to send email. %s", str(exc))

            logger.error("Fail to send email. %s", error_msg)
            msg = "Fail to send email using the current SMTP configuration settings. " \
                  "Please check the Notification section in Configurations."

            abort(400, **(AXIllegalOperationException(message=msg, detail=error_msg).json()))


class SMTPConfigurationListAPI(Resource):
    """API for listing SMTP configuration. GET and POST method are provided"""

    @classmethod
    def get(cls):
        logger.info("Received GET config list request.")
        config_result = Configuration.query.order_by(Configuration.id.asc())
        return {'Configurations': [item.as_dict() for item in config_result]}

    @classmethod
    def post(cls):
        logger.info("Received POST config list request.")
        config, config_detail = Common.parse_smtp_argument(request.json)
        logger.info("POST request arguments are %s",
                    Common.pretty_json(Common.hide_password(config_detail)))

        try:
            new_config = Configuration(id=config['id'],
                                       type=config['type'],
                                       active=config['active'],
                                       nickname=config['nickname'],
                                       config=json.dumps(config_detail))
            db.session.add(new_config)
            db.session.commit()
            return config_detail
        except Exception as exc:
            logger.exception("Internal Error. %s", str(exc))
            abort(400, message="Internal Error {}".format(str(exc)))

    @classmethod
    def put(cls):
        """The PUT function will create the new configuration if the id does
        not exist or update the configuration if the id does exist."""
        logger.info("Received PUT config list request.")
        config_result = {}
        try:
            config_result = Configuration.query.filter(
                Configuration.id == request.json['id'])
        except Exception as exc:
            logger.exception(str(exc))
            abort(400, **(AXIllegalArgumentException(str(exc)).json()))

        if config_result.count() == 0:
            return SMTPConfigurationListAPI.post()
        else:
            return SMTPConfigurationAPI.put(request.json['id'])


class SMTPConfigurationAPI(Resource):
    """API for SMTP configuration. GET, PUT, DELETE method are provided"""

    @classmethod
    def check_entry_existed(cls, uuid):
        config_result = Configuration.query.filter(Configuration.id == uuid)\
            .with_entities(Configuration.config)
        if config_result.count() == 0:
            logger.exception("Configuration with id=%s not existed", uuid)
            abort(400, **(AXIllegalOperationException(
                "Configuration with id={} not existed".format(uuid)).json()))
        return config_result

    @classmethod
    def get(cls, uuid):
        logger.info("Received GET configuration request, id=%s", uuid)
        config_result = cls.check_entry_existed(uuid)
        return config_result.first()

    @classmethod
    def delete(cls, uuid):
        logger.info("Received DELETE configuration request, id=%s", uuid)
        cls.check_entry_existed(uuid)
        try:
            Configuration.query.filter_by(id=uuid).delete()
            db.session.commit()
            return {}
        except Exception as exc:
            logger.exception("Internal Error. %s", str(exc))
            abort(400, message="Internal Error {}".format(str(exc)))

    @classmethod
    def put(cls, uuid):
        logger.info("Received PUT configuration request, id=%s", uuid)
        config, config_detail = Common.parse_smtp_argument(request.json)
        cls.check_entry_existed(uuid)
        logger.info("PUT request arguments are %s",
                    str(Common.hide_password(config_detail)))
        try:
            Configuration.query.filter(Configuration.id == uuid).update({
                'type': config_detail['type'],
                'active': config_detail['active'],
                'nickname': config_detail['nickname'],
                'config': json.dumps(config_detail)})
            db.session.commit()
            return config_detail
        except Exception as exc:
            logger.exception("Internal Error. %s", str(exc))
            abort(400, message="Internal Error {}".format(str(exc)))


class SMTPTestAPI(Resource):
    """API for Test SMTP configuration. POST method is provided"""

    @classmethod
    def post(cls):
        logger.info("Received POST configuration test request")
        config, config_detail = Common.parse_smtp_argument(request.json,
                                                           test=True)
        if config['type'] == 'smtp':
            try:
                EmailNotificationAPI.send_mail_smtp(config_detail, test=True)
            except Exception as exc:
                logger.exception("Internal Error. %s", str(exc))
                abort(400, **(AXIllegalArgumentException(str(exc)).json()))
        return config_detail


class HelloWorld(Resource):
    @staticmethod
    def get():
        return {'hello': 'world'}

api = Api(myapp)
api.add_resource(HelloWorld, '/', '/ping', '/v1/ping')
api.add_resource(EmailNotificationAPI, '/v1/notifications/email',
                 endpoint='email')
api.add_resource(SMTPConfigurationListAPI, '/v1/notifications/configurations',
                 endpoint='configurations')
api.add_resource(SMTPTestAPI, '/v1/notifications/configurations/test',
                 endpoint='test')
api.add_resource(SMTPConfigurationAPI,
                 '/v1/notifications/configurations/<string:uuid>',
                 endpoint='configuration')
