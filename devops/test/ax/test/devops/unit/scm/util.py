import logging
import subprocess
import shlex

logger = logging.getLogger(__name__)


def check_call(cmd, shell=None, **kwargs):
    logger.info(cmd)
    if shell:
        subprocess.check_call(cmd, shell=True, **kwargs)
    else:
        subprocess.check_call(shlex.split(cmd), **kwargs)


def check_output(cmd, shell=None, **kwargs):
    logger.info(cmd)
    if shell:
        ret = subprocess.check_output(cmd, shell=True, universal_newlines=True, **kwargs)
    else:
        ret = subprocess.check_output(shlex.split(cmd), universal_newlines=True, **kwargs)
    ret = ret[:-1]
    logger.info("\n%s", ret)
    return ret
