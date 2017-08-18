# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import subprocess
import sys
import traceback

from anytree import Node
from prompt_toolkit.shortcuts import confirm
from .cli import AbstractPrompt
from .installer import InstallPrompts, UninstallPrompts, ResumePrompts, PausePrompts

HELP="""
install     : Interactive argo cluster install
uninstall   : Interactive argo cluster uninstall
pause       : Stop a running cluster
resume      : Start a stopped cluster
advanced    : Non-interactive command line access to cluster operations
help        : This menu
exit        : Exit this interactive shell or press Ctrl+D
"""


class MainPrompt(AbstractPrompt):

    def __init__(self):
       self.root = Node("main",
                         prompt=u'What do you want to do today',
                         values=[u'install', u'uninstall', u'help', u'pause', u'resume', u'advanced', u'exit'],
                         help=u'Type "help" to see help. If you get an error when running a command type \'advanced\' use argocluster command directly',
                         function=self._handle_command
                        )

    def get_root(self):
        return self.root

    def get_history(self):
        return None

    def get_header(self):
        return u'Main Menu'

    @staticmethod
    def _handle_command(cmd):
        _m = {
            "install": InstallPrompts,
            "uninstall": UninstallPrompts,
            "pause": PausePrompts,
            "resume": ResumePrompts
        }
        if cmd in _m:
            prompter = _m[cmd]()
            prompter.run_cli()
            command = prompter.get_argocluster_command()
            print "Running the following command. It will take some time\n\n{}\n\n".format(command)
            answer = confirm(u'Are you sure you want to perform the action? (y/n) ')
            if answer:
                try:
                    subprocess.check_call(command.split(" "))
                except Exception as e:
                    traceback.print_exc()
                    print "Got the following exception {} when trying to run command {}.".format(e, command)
                    print "\nYou can drop into the advanced shell by typing 'advanced' and add more options to the command"
        elif cmd == "help":
            print(HELP)
        elif cmd == "advanced":
            print ("\n\nDropping you to a bash shell where you can type 'argocluster help' for info. Type 'exit' when done\n\n")
            subprocess.call(['bash'])
        elif cmd == "exit":
            raise EOFError("Pressed exit")
        return cmd, True


if __name__ == "__main__":
    m = MainPrompt()
    while True:
        try:
            m.run_cli()
        except (EOFError, KeyboardInterrupt):
            sys.exit()
