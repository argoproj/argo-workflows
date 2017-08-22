#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import boto3
import logging
import subprocess
import time

from botocore.client import ClientError


logger = logging.getLogger(__name__)


class ServerCert(object):

    def __init__(self, name, client=None):
        self.name = name
        if not client:
            self._client = boto3.client("iam")
        else:
            self._client = client

    def generate_certificate(self):
        def _create_new_cert():
            # generate a self-signed certificate
            subprocess.check_call(
                'openssl req -nodes -x509 -newkey rsa:4096 -keyout /tmp/key.pem -out /tmp/cert.crt -days 365 -subj "/C=CA"',
                shell=True)

            # in some case we hit a race condition that key and cert files are not ready after openssl return
            time.sleep(10)

            # upload it
            with open("/tmp/key.pem") as f:
                key = f.read()

            with open("/tmp/cert.crt") as f:
                cert = f.read()

            arn = self.upload_certificate(cert, key)
            return arn

        try:
            logger.info("Looking for server certificate %s", self.name)
            rsp = self._client.get_server_certificate(ServerCertificateName=self.name)
            arn = rsp["ServerCertificate"]["ServerCertificateMetadata"]["Arn"]
        except ClientError as ce:
            if "NoSuchEntity" in str(ce):
                logger.info("Cannot find server certificate, generating new one")
                arn = _create_new_cert()
            else:
                raise ce

        logger.info("ARN for certificate is %s", arn)
        return arn

    def upload_certificate(self, cert, key):
        logger.info("Uploading certificate {} to IAM".format(self.name))
        try:
            ret = self._client.upload_server_certificate(ServerCertificateName=self.name, CertificateBody=cert, PrivateKey=key)
        except ClientError as e:
            if e.response["ResponseMetadata"]["HTTPStatusCode"] == 409:
                self.delete_certificate()
                ret = self._client.upload_server_certificate(ServerCertificateName=self.name, CertificateBody=cert, PrivateKey=key)
            else:
                raise e

        return ret["ServerCertificateMetadata"]["Arn"]

    def delete_certificate(self):
        logger.info("Deleting existing certificate {} from IAM".format(self.name))

        try:
            return self._client.delete_server_certificate(ServerCertificateName=self.name)
        except ClientError as e:
            if "NoSuchEntity" in str(e):
                pass
            else:
                logger.exception("Failed to delete server certificate %s. Error: %s", self.name, e)
                if "AccessDenied" in str(e):
                    msg =  "\tPlease refer to latest Argo documentation and update your aws"
                    msg += "\tCross Account Role policy to latest version. Meanwhile, please manually"
                    msg += "\tdelete the certification after uninstall finishes using the following"
                    msg += "\tcommand:\n"
                    msg += "\taws iam delete-server-certificate --server-certificate-name {}\n".format(self.name)
                    logger.warning(msg)


def delete_server_certificate(aws_profile, name):
    client = boto3.Session(profile_name=aws_profile).client("iam")
    s = ServerCert(name, client=client)
    s.delete_certificate()
