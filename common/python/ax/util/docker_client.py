#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module library for docker containers.

Docker has been very inconsistent describing container images. Various terms have been used.
For the purpose of this file, container image name is registry/name:tag.

Registry: Registry host to get image. Must be FQDN with at least one dot ".".
Name: Image name. May contain one or more "/".
Tag: Libral string for tag. Separated from name by ":".
Image: Full name with all above components.
Repo: Registry + "/" + "Name".
"""

import base64
import json
import logging
import time

from threading import Condition
from threading import Lock
from docker import Client
from docker.errors import NotFound, APIError
from future.utils import with_metaclass

from ax.exceptions import AXUnauthorizedException, AXNotFoundException, AXIllegalArgumentException
from ax.platform.exceptions import AXPlatformException
from ax.util.docker_image import DockerImage
from ax.util.docker_registry import DockerRegistry
from ax.util.retry_exception import AXRetry, ax_retry
from ax.util.singleton import Singleton
from ax.util.const import SECONDS_PER_WEEK, SECONDS_PER_DAY, SECONDS_PER_HOUR, SECONDS_PER_MINUTE

logger = logging.getLogger(__name__)

DEFAULT_DOCKER_SOCK = "unix://var/run/docker.sock"


# TODO: Move this out to a generic utility later for executing a function one at
# a time and bonus points for supporting distributed execution
class DockerImageFetcher(with_metaclass(Singleton, object)):

    """
    Utility class that ensures that only a single image
    with the same name is fetched at the same time
    """

    def __init__(self):
        self.image_fetching_dict = {}
        self._lock = Lock()

    def single_executor(self, image_name, func, *args):
        # lock for global dict
        state = None
        self._lock.acquire()
        if image_name not in self.image_fetching_dict:
            logger.info("DockerImageFetcher: %s not being pulled by another thread. Preparing to pull", image_name)
            cv = Condition()
            state = {
                "cv": cv,
                "status": False,
                "detail": "Unknown failure"
            }

            self.image_fetching_dict[image_name] = state
            self._lock.release()

            logger.info("DockerImageFetcher: %s is being fetched now", image_name)
            try:
                status = func(*args)
                state["status"] = status
                if status:
                    state["detail"] = "Success"
            except Exception as e:
                logger.error("DockerImageFetcher: Got exception %s", e)
                state["detail"] = e

            self._lock.acquire()
            self.image_fetching_dict.pop(image_name)
            with cv:
                logger.debug("DockerImageFetcher: Notifying waiter of %s", image_name)
                self._lock.release()
                cv.notify_all()
        else:
            logger.info("DockerImageFetcher: %s already being fetched by another thread", image_name)
            state = self.image_fetching_dict[image_name]
            cv = state["cv"]
            with cv:
                self._lock.release()
                logger.debug("DockerImageFetcher: Waiting to be notified by another thread for %s", image_name)
                cv.wait()

            # At this time I have a reference to 'state' from which i can read
            # the status
            logger.debug("DockerImageFetcher: Got notified from the other thread for %s State %s ", image_name, state)

        if not state["status"]:
            raise AXPlatformException("Error: {}, while fetching image {}".format(state["detail"], image_name))

        return True


class AXDockerClient(object):
    def __init__(self, url=DEFAULT_DOCKER_SOCK):
        self._conn = Client(base_url=url)
        self._retry = AXRetry(retry_exception=(Exception,),
                              success_check=lambda x: x,
                              default=False,
                              success_default=True)

    @property
    def version(self):
        """Cached version information"""
        if hasattr(self, '_version'):
            return self._version
        self._version = self._conn.version()
        return self._version

    @property
    def version_tuple(self):
        """Version tuple of docker daemon (e.g. (1, 11, 2))"""
        return tuple([int(i) for i in self.version['Version'].split('.')])

    # Public APIs
    def start(self, registry, image, tag="latest", **kwargs):
        """
        Start a new container described by image.

        :param registry: The registry to use
        :type registry: DockerRegistry
        :param image: Full image name for this container.
        :param kwargs: Other args passed to container start.
        :return:
        """
        assert "tag" not in kwargs
        assert registry is not None and "Cannot start a container without providing a DockerRegistry"

        if not self._pull_with_caching(registry, image, tag):
            return None

        full_image = registry.servername + "/" + image + ":" + tag
        container = self._create(full_image, **kwargs)
        if container is None:
            return None

        started = self._start(container)
        if started:
            return container
        else:
            self._remove(container["Id"])
            return None

    def stop(self, container, **kwargs):
        """
        Stop sepcified container. Wrapper for docker API and handle exception.
        :param container: (string) Id or name of container.
        :param kwargs: Pass through kwargs for docker. Currently using only timeout.
        """
        if "timeout" not in kwargs:
            kwargs["timeout"] = 1
        self._stop(container, **kwargs)

    def remove(self, container, **kwargs):
        """
        Remove a container.
        :param container: (string) Id or name of container.
        :param kwargs: Pass through for docker.
        :return:
        """
        self._remove(container, **kwargs)

    def run(self, image, cmd, timeout=1200, **kwargs):
        """
        Run a command inside a container and check result.

        Container image is automatically pulled.
        Container will be stopped and removed after comamnd.

        :param image: Container image to run.
        :param cmd: Command inside container. This overwrites docker "command" in kwargs
        :param timeout: Timeout to wait for container.
        :param kwargs: Dict for parameters. It includes AX parameters and pass through ones.
                       All AX parameters start with "ax_" and will be removed before passing to docker create.
                       Currently supported AX parameters:
                         - ax_net_host: Set network mode to "host"
                         - ax_privileged: Run container in privileged mode
        :return: Tuple of: (True/False, return code)
        """
        assert "tag" not in kwargs
        logger.debug("Run %s inside %s on host %s, kwargs %s", cmd, image, self._host, kwargs)

        # Always overwrite command in kwargs.
        kwargs["command"] = cmd

        started = False
        container = {}
        rc = -1
        try:
            container = self._create(image, **kwargs)
            assert container, "Failed to create from %s, %s" % (image, kwargs)

            started = self._start(container)
            assert started, "Failed to start %s, %s" % (image, container)

            rc = self._conn.wait(container, timeout=timeout)
            assert rc == 0, "Command %s failed rc=%s" % (cmd, rc)

        except Exception:
            logger.exception("Failed to run %s in %s on %s", cmd, image, self._host)
            return False, rc

        finally:
            if started:
                self._stop(container["Id"], timeout=1)
            if container:
                self._remove(container["Id"])
        logger.debug("Completed run %s inside %s on %s, rc=%s", cmd, image, self._host, rc)
        return True, rc

    def cache_image(self, registry, name, tag="latest"):
        """
        Cache the image to local registry

        :param registry: The registry to use
        :type registry: DockerRegistry
        :param name: name of repo
        :param tag: repo tag
        """
        fetcher = DockerImageFetcher()
        full_image = registry.servername + "/" + name + ":" + tag
        return fetcher.single_executor(full_image, self._pull_with_caching, registry, name, tag)

    def get_container_uuid(self, name):
        """
        Get UUID for a container.
        """
        try:
            info = self._conn.inspect_container(name)
        except Exception:
            info = {}
        return info.get("Id", None)

    def get_container_version(self, name):
        """
        Get image namespace and version for a running container

        Sample return:
        [
            "docker.example.com/lcj/axagent:latest",
            "docker.local/lcj/axagent:latest"
        ]
        """
        try:
            info = self._conn.inspect_container(name)
        except NotFound:
            return []
        image = info["Image"].split(":")[1]
        info = self._conn.inspect_image(image)
        return info["RepoTags"]

    def exec_cmd(self, container_id, cmd, **kwargs):
        """Executes a command inside a running container and returns its output on completion

        :param container_id: container id
        :param cmd: command to execute
        :return: output from the command
        """
        logger.debug("Executing %s in container %s (kwargs: %s)", cmd, container_id, kwargs)
        try:
            exec_id = self._conn.exec_create(container_id, cmd, **kwargs)
            response = self._conn.exec_start(exec_id)
            return response
        # Docker API can actually return either error at different time.
        except NotFound:
            logger.debug("Container %s not exist on host %s", container_id, self._host)
        except APIError as e:
            if "not running" in str(e):
                logger.debug("Container %s not running on host %s", container_id, self._host)
            else:
                raise

    def exec_kill(self, pid, exec_id=None, container_id=None, signal=None):
        """
        Kill a pid in a container. Optionally checks if exec session is still valid before killing.

        :param pid: pid to kill in the container.
        :param exec_id: perform kill only if exec id is still running.
        :param container_id: perform kill only if exec id is still running.
        :param signal: kill signal to send to process
        """
        if not any([exec_id, container_id]):
            raise ValueError("exec_id or container_id must be supplied")
        pid = int(pid)
        assert pid != -1, "Killing all processes prohibited"
        if exec_id is not None:
            if isinstance(exec_id, dict):
                exec_id = exec_id['Id']
            try:
                exec_info = self._conn.exec_inspect(exec_id)
            except APIError as e:
                logger.warn("Failed to inspect exec session {} for killing. Skipping kill: {}"
                            .format(exec_id, str(e)))
                return
            if container_id:
                if container_id not in exec_info['ContainerID']:
                    raise ValueError("Supplied container id {} mismatched with exec container id: {}"
                                     .format(container_id, exec_info['ContainerID']))
            else:
                container_id = exec_info['ContainerID']
            if not exec_info['Running']:
                logger.debug("Exec session {} no longer running. Skipping kill".format(exec_id))
                return
        # perform kill
        kill_cmd_args = ['kill']
        if signal:
            kill_cmd_args.append('-{}'.format(signal))
        kill_cmd_args.append(str(pid))
        kill_cmd = ' '.join(kill_cmd_args)
        response = self.exec_cmd(container_id, 'sh -c "{} 2>&1; echo $?"'.format(kill_cmd))
        lines = response.splitlines()
        rc = int(lines[-1])
        if rc != 0:
            reason = lines[0] if len(lines) > 1 else "reason unknown"
            logger.warn("Failed to kill pid {} in container {}: {}".format(pid, container_id, reason))
        else:
            logger.debug("Successfully killed pid {} in container {}".format(pid, container_id))

    def containers(self, **kwargs):
        return self._conn.containers(**kwargs)

    def stats(self, name, **kwargs):
        return self._conn.stats(name, **kwargs)

    def clean_graph(self, age=86400):
        """
        Clean graph storage to remove old containers and any unreferenced docker image layers
        """
        # Exit time is in free form string. Parse it. And real coarse time granularity.
        pattern = ["month ago", "months ago", "year ago", "years ago"]
        if age >= SECONDS_PER_MINUTE:
            pattern += ["minutes ago", "minute ago"]
        if age >= SECONDS_PER_HOUR:
            pattern += ["hours ago", "hour ago"]
        if age >= SECONDS_PER_DAY:
            pattern += ["days ago", "day ago"]
        if age >= SECONDS_PER_WEEK:
            pattern += ["weeks ago", "week ago"]

        for c in self._conn.containers(filters={"status": "exited"}):
            if any([p in c["Status"] for p in pattern]):
                try:
                    self._remove(c)
                except Exception:
                    logger.exception("Failed to remove %s", c["Id"])

        for i in self._conn.images():
            if i["RepoTags"][0] == "<none>:<none>" and time.time() > i["Created"] + age:
                try:
                    self._conn.remove_image(i["Id"])
                except Exception:
                    # This is probably OK.
                    logger.debug("Failed to delete %s", i["Id"])

    def search(self, searchstr=None):
        if searchstr is None or searchstr == "":
            raise AXPlatformException("Docker hub search string needs to a non-empty string")
        response = self._conn.search(searchstr)
        return [{"ctime": "", "repo": x['name'], "tag": "latest"} for x in response or []]

    def login(self, registry, username, password):
        """
        Returns a base64 encoded token of username and password
        only if login is successful else it raises exceptions
        """
        try:
            self._conn.login(username, password=password, registry=registry, reauth=True)
        except APIError as e:
            code = e.response.status_code
            if code == 401:
                # on login failure it raises a docker.errors.APIError:
                # 401 Client Error: Unauthorized
                raise AXUnauthorizedException(e.explanation)
            elif code == 404:
                raise AXNotFoundException(e.explanation)
            elif code == 500:
                if "x509: certificate signed by unknown authority" in e.response.text:
                    raise AXIllegalArgumentException("Certificate signed by unknown authority for {}".format(registry))
                else:
                    raise e
            else:
                raise e
        token = base64.b64encode("{}:{}".format(username, password))
        return token

    @staticmethod
    def generate_kubernetes_image_secret(registry, token):
        """
        Create the image pull secret by concatenating the secrets required
        for the passed token
        Args:
            registry: string
            token: base64 encoded

        Returns:
            base64 encoded string used for imagepull secrets
        """
        ret = {
            "auths": {
                registry : {
                    "auth": token
                }
            }
        }
        return base64.b64encode(json.dumps(ret))

    # Internal implementations
    def _pull_with_caching(self, registry, name, tag, **kwargs):
        """
        Pull a new container with AX caching enabled.
        :param registry: DockerRegistry instance.
        :param name: Container short name.
        :param tag: Tag.
        :param kwargs: Other kwargs for pull.
                       Docker API requires tag to be in kwargs.
                       AX needs to process it and enforce tag to be separate.
        :return: True or False
        """
        assert "tag" not in kwargs, "%s" % kwargs

        if registry.user is not None and registry.passwd is not None:
            kwargs["auth_config"] = kwargs.get("auth_config", {"username": registry.user, "password": registry.passwd})

        return self._pull_with_retry(registry.servername, name, tag, **kwargs)

    def _pull_with_retry(self, registry, name, tag, **kwargs):
        return ax_retry(self._pull, self._retry, registry, name, tag, **kwargs)

    def _pull(self, registry, name, tag, **kwargs):
        """
        Do pull. Call docker API and check errors.
        :param registry: Registry host name.
        :param name: Container short name.
        :param tag: Tag.
        :param kwargs: Other pull args.
        :return: True or False.
        """
        # All must be set not empty.
        assert all([registry, name, tag]), "%s, %s, %s" % (registry, name, tag)

        repo = DockerImage(registry=registry, name=name).docker_repo()
        kwargs["tag"] = tag
        try:
            ret = self._conn.pull(repo, stream=True, **kwargs)
        except Exception:
            logger.exception("Failed to pull %s, %s", repo, tag)
            return False

        logger.info("Pull image %s:%s starting", repo, tag)
        # Search pull result to determine status. Must have digest and success message.
        has_digest = False
        has_image = False
        try:
            for l in ret:
                try:
                    progress = json.loads(l)
                    if progress["status"].startswith("Digest:"):
                        has_digest = True
                    if "Image is up to date" in progress["status"] or "Downloaded newer image" in progress["status"]:
                        has_image = True
                except (KeyError, ValueError):
                    logger.debug("Failed to parse pull progress line %s", l)
        except Exception:
            logger.exception("Failed to pull %s:%s", repo, tag)
            return False
        logger.info("Pull image %s:%s result %s %s", repo, tag, has_digest, has_image)
        return has_digest and has_image

    def _push_with_retry(self, registry, name, tag):
        return ax_retry(self._push, self._retry, registry, name, tag)

    def _push(self, registry, name, tag):
        """
        Do push. Call docker API and check errors.
        :param registry: Registry host name.
        :param name: Container short name.
        :param tag: Tag.
        :return: True or False.
        """
        # All must be set not empty.
        assert all([registry, name, tag]), "%s, %s, %s" % (registry, name, tag)

        repo = DockerImage(registry=registry, name=name).docker_repo()
        try:
            ret = self._conn.push(repo, tag, stream=True)
        except Exception:
            logger.exception("Failed to push %s, %s", repo, tag)
            return False

        logger.info("Push image %s:%s starting", repo, tag)
        # Search push result to determine status. Must have digest.
        has_digest = False
        try:
            for l in ret:
                try:
                    progress = json.loads(l)
                    has_digest = progress["status"].startswith("%s: digest:" % tag)
                except (KeyError, ValueError):
                    logger.debug("Failed to parse push progress line %s", l)
        except Exception:
            logger.exception("Failed to push %s:%s", repo, tag)
            return False
        logger.info("Push image %s:%s result %s", repo, tag, has_digest)
        return has_digest

    def _create(self, image, **kwargs):
        """
        Create a new container.

        :param image: (string) Container image with tag
        :param kwargs: AX and docker parameters.
        :return: container or None
        """
        # Docker API has two levels of dict. Top level specify mostly "create" configs.
        # One key at first level is "host_config". This defines second level "run" configs.
        # It's yet another dict. It was specified in docker run API and moved here.
        # We need to set both levels correctly.
        self._validate_config(kwargs)

        kwargs = self._parse_ax_create_config(kwargs)
        kwargs = self._parse_ax_host_config(kwargs)
        kwargs = self._remove_ax_config(kwargs)
        logger.debug("Final kwargs for create %s: %s", image, kwargs)
        try:
            return self._conn.create_container(image, **kwargs)
        except Exception:
            logger.exception("Failed to create container from %s %s", image, kwargs)
            return None

    def _start(self, container):
        try:
            self._conn.start(container)
            return True
        except Exception:
            logger.exception("Failed to start container %s", container)
            return False

    def _stop(self, container, **kwargs):
        try:
            self._conn.stop(container, **kwargs)
        except NotFound:
            pass
        except Exception:
            logger.exception("Failed to stop %s", container)

    def _remove(self, container, **kwargs):
        try:
            self._conn.remove_container(container, v=True, **kwargs)
        except NotFound:
            pass
        except APIError as e:
            if "Conflict" in str(e):
                logger.error("Not removing running container %s", container)
            elif "device or resource busy" in str(e):
                # Work around https://github.com/google/cadvisor/issues/771
                logger.error("Container removal temporary failure. Retrying.")
                retry = AXRetry(retries=10, delay=1, retry_exception=(Exception,), success_exception=(NotFound,))
                ax_retry(self._conn.remove_container, retry, container, v=True, force=True)
            else:
                logger.exception("Failed to remove container %s", container)
        except Exception:
            logger.exception("Failed to remove container %s", container)

    def _validate_config(self, config):
        if "volumes" in config:
            assert isinstance(config["volumes"], list), "Support only list of volumes %s" % config["volumes"]
        if "host_config" in config and "Binds" in config["host_config"]:
            assert isinstance(config["host"]["Binds"], list), "Support only list of volumes %s" % config["host"]["Binds"]
        if "ports" in config:
            assert isinstance(config["ports"], list), "Support only list of ports %s" % config["ports"]
        if "host_config" in config and "port_bindings" in config["host_config"]:
            assert isinstance(config["host"]["PortBindings"], dict), "Support only dict of port_bindings %s" % config["host"]["PortBindings"]
        if "environment" in config:
            assert isinstance(config["environment"], list), "Support only list of environments %s" % config["environment"]

    def _parse_ax_create_config(self, config):
        if config.get("ax_daemon", False):
            config["detach"] = True

        if "ax_volumes" in config:
            axv = [v.split(":")[1] for v in config["ax_volumes"]]
            if "volumes" in config:
                assert isinstance(config["volumes"], list), "must be list {}".format(config["volumes"])
                config["volumes"] += axv
            else:
                config["volumes"] = axv

        if "ax_ports" in config:
            config["ports"] = config["ax_ports"].keys()

        return config

    def _parse_ax_host_config(self, config):
        ax_config = {}
        if config.get("ax_net_host", False):
            ax_config["network_mode"] = "host"

        if config.get("ax_privileged", False):
            ax_config["privileged"] = True

        if config.get("ax_host_namespace", False):
            ax_config["pid_mode"] = "host"

        if config.get("ax_daemon", False):
            ax_config["restart_policy"] = {"MaximumRetryCount": 0, "Name": "unless-stopped"}

        if "ax_volumes" in config:
            if "binds" in ax_config:
                assert isinstance(ax_config["binds"], list), "must be list {}".format(ax_config["binds"])
                ax_config["binds"] += config["ax_volumes"]
            else:
                ax_config["binds"] = config["ax_volumes"]

        if "ax_ports" in config:
            ax_config["port_bindings"] = config["ax_ports"]

        ax_host_config = self._conn.create_host_config(**ax_config)
        if "host_config" in config:
            config["host_config"].update(ax_host_config)
        else:
            config["host_config"] = ax_host_config
        return config

    def _remove_ax_config(self, config):
        for key in config.keys():
            if key.startswith("ax_"):
                del config[key]
        return config


class DockerHub(DockerRegistry):

    def __init__(self):
        pass

    def list_images(self, searchstr=None):
        return AXDockerClient().search(searchstr=searchstr)

    def delete_images(self, images):
        raise AXPlatformException("Do not support deleting images from docker hub")


