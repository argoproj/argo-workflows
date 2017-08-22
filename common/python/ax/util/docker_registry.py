#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2015 Applatix, Inc. All rights reserved.
#

import json
import logging
import requests

from dateutil import parser
from ax.platform.component_config import SoftwareInfo
from ax.platform.exceptions import AXPlatformException
from ax.util.docker_image import DockerImage

logger = logging.getLogger(__name__)

class DockerRegistry(object):

    def __init__(self, servername, user=None, passwd=None, cert=None):
        self.servername = servername
        self.user = user
        self.passwd = passwd
        self.cert = cert
        self._epoch = parser.parse("1970-01-01T00:00:00.000000Z")

    def list_images(self, searchstr=None):
        """
        :returns: List of strings that are in the format servername/image-name:tag
        """
        retval = []
        artifacts = self._get_catalog()
        for artifact in artifacts:
            if searchstr is not None and searchstr not in artifact:
                continue
            logger.debug("Artifact selected {}".format(artifact))
            tags = self._get_tags(artifact)
            for tag in tags:
                time_from_epoch = self._get_manifest(artifact, tag)
                if time_from_epoch is None:
                    time_from_epoch = long(0)

                retval.append({
                    "repo": "{}/{}".format(self.servername, artifact),
                    "tag": tag,
                    "ctime": time_from_epoch
                })

        return retval

    def delete_images(self, images):

        for image in images or []:
            d_img = DockerImage(fullname=image)
            _server, artifact, tag = d_img.docker_names()

            if _server != self.servername:
                raise AXPlatformException("Delete only supports deletion from {} registry".format(self.servername))

            digest = self._get_digest(artifact, tag)
            if digest is None:
                raise AXPlatformException("Could not find digest for image {}:{}".format(artifact, tag))

            # now try to delete
            if not self._delete_digest(artifact, tag, digest):
                raise AXPlatformException("Could not delete image {}:{} with digest {}".format(artifact, tag, digest))

    def _get_auth_object(self):
        if self.user is not None and self.passwd is not None:
            auth = requests.auth.HTTPBasicAuth(self.user, self.passwd)
            return auth
        return None

    def _get_catalog(self):
        """
        :returns: a list of image-names
        """
        url = "https://{}/v2/_catalog".format(self.servername)

        ret_list = []
        # we have a while loop to handle pagination
        try:
            while True:
                logger.debug("Getting images from %s", url)
                r = requests.get(url, auth=self._get_auth_object(), verify=self.cert)
                if not r.ok:
                    logger.error("Got error {} status code from server {} in get catalog".format(
                        r.status_code, self.servername))
                    return []

                data = r.json()
                if 'repositories' in data:
                    logger.debug("Repos %s", data['repositories'])
                    logger.debug("Links %s", r.links)
                    ret_list.extend(data['repositories'])

                    if 'next' in r.links:
                        logger.debug("Pagination in images %s", r.links['next'])
                        url = "https://{}{}".format(self.servername, r.links['next']['url'])
                    else:
                        logger.debug("End of pages")
                        return ret_list
                else:
                    logger.debug("Did not find repositories in returned data %s", data)

        except Exception as e:
            logger.error("Exception in getting catalog {}".format(e))

        return []

    def _get_tags(self, artifact):
        """
        :returns: a list of tags for the artifact
        """
        url = "https://{}/v2/{}/tags/list".format(self.servername, artifact)
        try:
            r = requests.get(url, auth=self._get_auth_object(), verify=self.cert)
            if not r.ok:
                logger.error("Got error {} status code from server {} for artifact {} tags".format(
                    r.status_code, self.servername, artifact))
                return []

            data = r.json()
            if 'tags' in data:
                return data['tags']

        except Exception as e:
            logger.error("Exception in getting catalog {}".format(e))

        return []

    def _get_manifest(self, artifact, tag):
        """
        Extracts some information from the manifest requested and returns it
        """
        url = "https://{}/v2/{}/manifests/{}".format(self.servername, artifact, tag)
        try:
            r = requests.get(url, auth=self._get_auth_object(), verify=self.cert)
            if not r.ok:
                logger.error("Got error {} status code from server {} for artifact {} manifest {}".format(
                    r.status_code, self.servername, artifact, tag))
                return None

            data = r.json()

            # get creation date, v1compat is a json dict in a string form
            v1str = data["history"][0]["v1Compatibility"]
            v1dict = json.loads(v1str)

            # created is in zulu time
            created = parser.parse(v1dict["created"])
            td = (created - self._epoch)

            return long(td.microseconds + (td.seconds + td.days * 24 * 3600) * 10**6)

        except Exception as e:
            logger.error("Exception in getting manifest %s", e)

        return None

    def _get_digest(self, artifact, tag):
        url = "https://{}/v2/{}/manifests/{}".format(self.servername, artifact, tag)
        # this is required as per docker documentation to get the correct SHA
        # digest for the artifact, tag combo.
        headers = {'Accept': 'application/vnd.docker.distribution.manifest.v2+json'}
        try:
            logger.debug("Getting digest for %s headers %s", url, headers)
            r = requests.head(url, auth=self._get_auth_object(), verify=self.cert, headers=headers)
            if not r.ok:
                return None
            logger.debug("Returned %s", r.headers)
            field = 'docker-content-digest'
            if field not in r.headers:
                return None

            return r.headers[field]
        except Exception as e:
            logger.error("Exception %s in getting digest for %s:%s", e, artifact, tag)

        return None

    def _delete_digest(self, artifact, tag, digest):
        url = "https://{}/v2/{}/manifests/{}".format(self.servername, artifact, digest)
        try:
            r = requests.delete(url, auth=self._get_auth_object(), verify=self.cert)
            return r.ok
        except Exception as e:
            logger.error("Exception %s in getting digest for %s:%s digest %s", e, artifact, tag, digest)

        return False
