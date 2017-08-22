#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2015 Applatix, Inc. All rights reserved.
#

import argparse
import logging

from ax.kubernetes import swagger_client

from ax.exceptions import AXException

logger = logging.getLogger(__name__)

def gen_docker_command(image, cmd):
    run_cmd = ["axdockerrun", "--rm", "--net", "host"]
    run_cmd += ["-e", "DOCKER_HOSTNAME=`hostname -s`"]
    run_cmd += [image]
    if isinstance(cmd, basestring):
        run_cmd += ['sh -c "' + cmd + '"']
    else:
        run_cmd += ['sh -c "'] + cmd + ['"']
    return run_cmd

def get_docker_login_command(reg_server, user, password, email="none"):
    cmd = ["docker", "login"]
    cmd += ["-u", user]
    cmd += ["-p", password]
    cmd += ["-e", email]
    cmd += [reg_server]
    return cmd

def docker_options_to_envvar(cmds):
    kubeenv = []
    parser = argparse.ArgumentParser()
    parser.add_argument("-e", "--env", action='append')
    args, unknown = parser.parse_known_args(cmds)
    for env in args.env or []:
        name, value = env.split("=", 1)
        var = swagger_client.V1EnvVar()
        var.name = name
        var.value = value
        kubeenv.append(var)

    return kubeenv, unknown

def docker_options_to_ports(cmds):
    kubeports = []
    parser = argparse.ArgumentParser()
    parser.add_argument("-p", "--publish", action='append')
    args, unknown = parser.parse_known_args(cmds)
    for mapping in args.publish or []:
        ip_ports = mapping.split(":")
        if len(ip_ports) == 3:
            # format is hostip:hostport:containerport
            cp = swagger_client.V1ContainerPort()
            cp.host_ip = ip_ports[0]
            cp.host_port = ip_ports[1]
            cp.container_port = ip_ports[2]
            kubeports.append(cp)
            logger.warn("-p specification in docker_options is not needed unless you want to map the port to the specific hostname")
        elif len(ip_ports) == 2:
            # format is hostport:containerport
            cp = swagger_client.V1ContainerPort()
            cp.host_port = ip_ports[0]
            cp.container_port = ip_ports[1]
            kubeports.append(cp)
            logger.warn("-p specification in docker_options is not needed unless you want to map the port to the specific hostname")
        else:
            raise AXException("Port specification {} does not match format in docker run command".format(mapping))

    return kubeports, unknown

def docker_options_to_volumes(cmds):
    """
    Parse docker volumes mounts and return true if docker support is required
    Args:
        opts: string

    Returns: boolean
    """
    parser = argparse.ArgumentParser()
    parser.add_argument("-v", "--volume", action='append')
    args, unknown = parser.parse_known_args(cmds)
    for vol in args.volume or []:
        # [host-src:]container-dest[:<options>]
        vals = vol.split(":")
        if len(vals) == 2 and vals[0] == "/var/run/docker.sock":
            return True, unknown

    return False, unknown
