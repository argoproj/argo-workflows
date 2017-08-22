"""
Module to provide unified version string based on commit hash 

There are three scenarios we need to handle when caller imports ax.version.__version__:

 1) application has been compiled/bundled to an executable via PyInstaller
 2) application source has been copied/packaged into a container
 3) a script is being invoked directly from a workspace

In cases 1 and 2, it is expected that an auto-generated _version.py has been created during build, and can be imported. 
In case 3, we read version.txt from source root as the version and append '-workspace'
"""

import os
import sys

_version_txt = os.path.realpath(os.path.join(os.path.dirname(__file__), "../../../version.txt"))
_in_workspace = not getattr(sys, 'frozen', False) and os.path.isfile(_version_txt)

if _in_workspace:
    with open(_version_txt, 'r') as f:
        __version__ = f.read().strip() + '-workspace'
    debug = False
else:
    from ._version import __version__, debug
