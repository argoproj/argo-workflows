# Copyright 2015-2017 Applatix, Inc. All rights reserved.

import argparse
import logging
import sys

from ax.version import __version__
from ax.devops.nexus.repoclients import NexusClient, NexusArtifact

logger = logging.getLogger(__name__)


def main(args=None):

    common_parser = argparse.ArgumentParser(add_help=False)
    common_parser.add_argument('--loglevel', type=int, default=logging.INFO, help="Log level")
    common_parser.add_argument('--repo', default=None, help="Nexus repository repo address")
    common_parser.add_argument('--port', default=8081, help="Nexus repository port number")
    common_parser.add_argument('--username', default=None, help="Username")
    common_parser.add_argument('--password', default=None, help="Password")

    parser = argparse.ArgumentParser(description="Argo Nexus Repository Helper", parents=[common_parser])
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--nexus_host', required=True)
    subparsers = parser.add_subparsers(dest="command", title="Commands")

    upload_parser = subparsers.add_parser('upload', help="Upload an artifact from Argo to Nexus Repository", parents=[common_parser])
    upload_parser.add_argument('--group', required=True)
    upload_parser.add_argument('--artifact')
    upload_parser.add_argument('--art_version')
    upload_parser.add_argument('--extension')
    upload_parser.add_argument('--local_path')
    upload_parser.add_argument('--repo_id', required=True)

    download_parser = subparsers.add_parser('download', help='Download an artifact from Nexus Repository to Argo', parents=[common_parser])
    download_parser.add_argument('--group', required=True)
    download_parser.add_argument('--artifact')
    download_parser.add_argument('--art_version')
    download_parser.add_argument('--extension')
    download_parser.add_argument('--local_path')
    download_parser.add_argument('--repo_id', required=True)

    parsed = parser.parse_args(args=args)

    logging.basicConfig(stream=sys.stdout, level=parsed.loglevel,
                        format="%(asctime)s %(levelname)5s %(name)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S")
    rc = 0
    try:
        repo = NexusClient(repo_url=parsed.nexus_host, port_number=parsed.port, username=parsed.username, password=parsed.password)
        art = NexusArtifact(group=parsed.group, local_path=parsed.local_path,
                            artifact=parsed.artifact, version=parsed.art_version, extension=parsed.extension)

        if parsed.command == 'upload':
            response = repo._upload_artifact(artifact=art, repo_id=parsed.repo_id)
            logger.info("Status code %s", response.status_code)
            logger.info("Response message %s", response.text)
            response.raise_for_status()
            logger.info("Successfully upload artifact to Nexus.")
        elif parsed.command == 'download':
            response = repo._download_artifact(artifact=art, repo_id=parsed.repo_id)
            if response is None:
                rc = 1
            if rc == 0:
                logger.info("Successfully download artifact to Nexus.")
        else:
            parser.print_help()
            rc = 1
    except Exception as e:
        rc = 1
        logger.exception("Failure: %s", str(e))

    if rc:
        sys.exit(rc)
