import logging
import os
import re
import requests
from ax.devops.exceptions import AXNexusException
from ax.devops.axdb.axops_client import AxopsClient
from requests_toolbelt import MultipartEncoder
from urllib.parse import urlencode, urlparse

logger = logging.getLogger(__name__)

axops_client = AxopsClient()


class Artifact(object):
    """
    Generic class describing an artifact
    """
    def __init__(self, group, artifact='', version='', classifier='', extension=''):
        self.group = group
        self.artifact = artifact
        self.version = version
        self.classifier = classifier
        self.extension = extension

    def get_coordinates_string(self):
        return '{group}:{artifact}:{version}:{classifier}:{extension}'.format(group=self.group, artifact=self.artifact,
                                                                              version=self.version,
                                                                              classifier=self.classifier,
                                                                              extension=self.extension)

    def __repr__(self):
        return self.get_coordinates_string()


class NexusArtifact(Artifact):
    """
    Artifact for upload to repository
    """
    def __init__(self, group, local_path, artifact='', version='', classifier='', extension=''):
        self.local_path = local_path

        artifact_detected, version_detected, extension_detected = self.detect_name_ver_ext()

        if not artifact:
            artifact = artifact_detected
            if not artifact:
                raise AXNexusException("artifact cannot be empty")

        if not version:
            version = version_detected
            if not version:
                raise AXNexusException("version cannot be empty")

        if not extension:
            extension = extension_detected
            if not extension:
                raise AXNexusException("extension cannot be empty")

        super(NexusArtifact, self).__init__(group=group, artifact=artifact, version=version, classifier=classifier,
                                            extension=extension)

    def detect_name_ver_ext(self):
        base_name = os.path.basename(self.local_path)
        result = re.match('^(?# name)(.*?)-(?=\d)(?# version)(\d.*)\.(?# extension)([^.]+)$', base_name)

        if result is None:
            return None, None, None

        name, version, extension = result.group(1), result.group(2), result.group(3)
        logger.debug('name: %s, version: %s, extension: %s', name, version, extension)
        return name, version, extension


class RepoClient(object):
    """
    A generic repository client
    """
    def __init__(self, repo_url, username=None, password=None, timeout=None):
        self.repo_url = repo_url
        self.username = username
        self.password = password

    def download(self, repo_id, remote_artifacts):
        """Load an artifact from the remote repository.

        :param repo_id: repository name
        :param remote_artifacts: list of artifact names
        :return:
        """
        raise NotImplementedError

    def publish(self, local_artifacts, repo_id):
        """Publish an artifact from local to the remote repository.

        :param local_artifacts: list of artifacts to upload
        :param repo_id: local path of the artifact to upload
        :return:
        """
        raise NotImplementedError


class NexusClient(object):
    """
    A Nexus (OSS) repository client
    """
    def __init__(self, repo_url, port_number=8081, username=None, password=None, timeout=None):
        if 'http:' in repo_url or 'https:' in repo_url:
            repo_url = urlparse(repo_url).hostname
        self.port_number = port_number

        if not username or not password:
            try:
                nexus_list = axops_client.get_tools(type='nexus')
                for item in nexus_list:
                    if repo_url in item['url']:
                        self.repo_url = item['url']
                        self.username = item['username']
                        self.password = item['password']
                        self.port_number = item.get('port', 8081)
            except Exception:
                logger.exception("Cannot find the configured Nexus repository. Please check Congigurations -> Connect artifact repositories")
                raise AXNexusException

        if self.repo_url is None or self.username is None or self.password is None:
            logger.error("Cannot find the Nexus repository using %s as hostname. Please check Congigurations -> Connect artifact repositories", repo_url)
            raise AXNexusException

    def _upload_artifact(self, artifact, repo_id):
        filename = os.path.basename(artifact.local_path)
        logger.info("Will try to upload %s", artifact)
        if not os.path.isfile(artifact.local_path):
            raise AXNexusException("cannot find file from local path, {}".format(artifact.local_path))

        with open(artifact.local_path, 'rb') as f:
            data = (
                ('r', repo_id),
                ('hasPom', 'false'),
                ('e', artifact.extension),
                ('g', artifact.group),
                ('a', artifact.artifact),
                ('v', artifact.version),
                ('p', artifact.extension),
                ('file', (filename, f, 'text/plain'))
            )

            m = MultipartEncoder(fields=data)
            headers = {'Content-Type': m.content_type}

            if self.port_number:
                hostname = "{}:{}".format(self.repo_url, self.port_number)
            else:
                hostname = self.repo_url
            response = requests.post('{}/nexus/service/local/artifact/maven/content'.format(hostname),
                                     data=m, headers=headers, auth=(self.username, self.password))
            return response

    def _download_artifact(self, artifact, repo_id):
        params = {
            'r': repo_id,
            'g': artifact.group,
            'a': artifact.artifact,
            'v': artifact.version,
            'e': artifact.extension,
        }

        if artifact.classifier:
            params['c'] = artifact.classifier

        if self.port_number:
            hostname = "{}:{}".format(self.repo_url, self.port_number)
        else:
            hostname = self.repo_url
        url = "%(hostname)s/nexus/service/local/artifact/maven/redirect?%(qs)s" % {
            'hostname': hostname,
            'qs': urlencode(params),
        }

        if os.path.isdir(artifact.local_path):
            temp_name = '{}-{}.{}'.format(artifact.artifact, artifact.version, artifact.extension)
            local_filename = os.path.join(artifact.local_path, temp_name)
        else:
            local_filename = artifact.local_path

        dir_name = os.path.dirname(os.path.abspath(local_filename))
        if not os.path.isdir(dir_name):
            os.makedirs(dir_name)

        logger.info("Request url, %s", url)
        try:
            r = requests.get(url, stream=True)
            with open(local_filename, 'wb') as f:
                for chunk in r.iter_content(chunk_size=1024):
                    if chunk:
                        f.write(chunk)
            r.raise_for_status()
            return local_filename
        except Exception:
            logger.exception("Failed to download, %s", local_filename)
            return None
