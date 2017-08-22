import argparse
import logging
import os
import sys

from ax.version import __version__
from ax.devops.exceptions import AXScmException
from .factory import create_scm_client

logger = logging.getLogger('ax.devops.scm')


def main(args=None):
    if not os.path.isfile('/.dockerenv'):
        # Enforce running as container since it changes user environment files
        print("Must be run from with container environment")

    common_parser = argparse.ArgumentParser(add_help=False)
    common_parser.add_argument('--loglevel', type=int, default=logging.INFO, help="Log level")
    common_parser.add_argument('--cmd_timeout', type=int, default=None, help="Maximum time a command can run before timing out")

    parser = argparse.ArgumentParser(description="General purpose source code management utility", parents=[common_parser])
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    subparsers = parser.add_subparsers(dest="command", title="Commands")

    clone_parser = subparsers.add_parser('clone', help="Clone a repository", parents=[common_parser])
    clone_parser.add_argument('repo')
    clone_parser.add_argument('path')
    commit_branch_group = clone_parser.add_mutually_exclusive_group()
    commit_branch_group.add_argument('--commit')
    commit_branch_group.add_argument('--branch')
    clone_parser.add_argument('--username')
    clone_parser.add_argument('--password')
    clone_parser.add_argument('--merge', help="Attempt merge of supplied commit or branch into the workspace following the clone")
    clone_parser.add_argument('--author', help="Author (e.g. \"John Doe <john@email.com>\") for the merge commit")
    clone_parser.add_argument('--push', action='store_true', help="Push the changes after merge")

    push_parser = subparsers.add_parser('push', help="Push outgoing commits from a local repo", parents=[common_parser])
    push_parser.add_argument('path')

    parsed = parser.parse_args(args=args)

    logging.basicConfig(stream=sys.stdout, level=parsed.loglevel,
                        format="%(asctime)s %(levelname)5s %(name)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S")
    rc = 0
    scm_client = None
    try:
        if parsed.command == 'clone':
            scm_client = create_scm_client(path=parsed.path, repo=parsed.repo, username=parsed.username,
                                           password=parsed.password, cmd_timeout=parsed.cmd_timeout)
            scm_client.clone(commit=parsed.commit, branch=parsed.branch)
            if parsed.merge:
                scm_client.merge(parsed.merge, author=parsed.author)
                if parsed.push:
                    scm_client.push()
        elif parsed.command == 'push':
            scm_client = create_scm_client(path=parsed.path)
            scm_client.push()
        else:
            parser.print_help()
            rc = 1
    except Exception as e:
        rc = 1
        if isinstance(e, AXScmException):
            logger.error("%s failed: %s", parsed.command, e)
        else:
            logger.exception("Unexpected checkout failure: %s", e)
    finally:
        if scm_client:
            scm_client.remove_credentials()
    if rc:
        sys.exit(rc)
