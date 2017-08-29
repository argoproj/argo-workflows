import logging
import shlex

from ax.devops.ci.constants import AxCommands
from ax.devops.exceptions import InvalidCommand
from ax.devops.kafka.kafka_client import EventNotificationClient
from ax.devops.utility.utilities import AxArgumentParser
from ax.notification_center import FACILITY_GATEWAY, CODE_JOB_CI_INVALID_COMMAND

logger = logging.getLogger(__name__)


class CommandParser(object):
    """Parser for AX command."""

    def __init__(self):
        self.parser = AxArgumentParser()
        subparsers = self.parser.add_subparsers(dest='command')
        # Parser for rerun
        rerun_subparser = subparsers.add_parser('rerun')
        rerun_subparser.add_argument('-a', '--all', action='store_true', dest='rerun_all', default=False)
        # Parser for run
        run_subparser = subparsers.add_parser('run')
        run_subparser.add_argument('template', type=str)
        run_subparser.add_argument('-p', '--param', action='append', dest='parameters', default=None)

    def parse(self, command):
        try:
            args = self.parser.parse_args(shlex.split(command)[1:])
        except ValueError:
            raise InvalidCommand('Given command ({}) is invalid'.format(command))
        else:
            return vars(args)


class BaseEventTranslator(object):
    """Base event translator."""

    parser = CommandParser()
    event_notification_client = EventNotificationClient(FACILITY_GATEWAY)

    @classmethod
    def _parse_command(cls, texts):
        """Extract command out of text.

        Currently, we support the following commands:
        (1) /ax rerun                                                           # Rerun failed service templates (subject to policy enforcement)
        (2) /ax rerun -a/--all                                                  # Rerun all service templates (subject to policy enforcement)
        (3) /ax run "AX Workflow Test"                                          # Run a specific service template
        (4) /ax run "AX Workflow Test" -p/--param namespace=staging             # Run a specific service template by supplying some parameters

        :param texts:
        :return:
        """
        texts = texts.splitlines()
        commands = []
        for i in range(len(texts)):
            text = texts[i].strip()
            if not text.startswith('/ax '):  # Not a command
                continue
            try:
                args = cls.parser.parse(text)
            except InvalidCommand as e:
                logger.warning('Failed to parse command: %s', e)
                cls.event_notification_client.send_message_to_notification_center(CODE_JOB_CI_INVALID_COMMAND, detail={'command': text})
                continue
            else:
                command = {
                    'command': args['command']
                }
                if command['command'] == AxCommands.RUN:
                    command['template'] = args['template']
                    command['parameters'] = {}
                    if args['parameters']:
                        for param in args['parameters']:
                            key, value = param.split('=')
                            command['parameters'][key] = value
                else:
                    command['rerun_all'] = args['rerun_all']
                commands.append(command)
        return commands
