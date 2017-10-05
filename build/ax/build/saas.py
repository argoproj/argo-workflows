import logging
import os
import shutil

from .common import ProdBuilder, SRC_PATH

logger = logging.getLogger('ax.build')

SAAS_PATH = os.path.join(SRC_PATH, "saas")
CONFIG_DIR = os.path.join(SAAS_PATH, "common/config")

PRODUCTION_SERVICES = [
    "axamm",
    "axdb",
    "axnc",
    "axops",
    "argocli",
    "zookeeper",
    "kafka"
]

tools_services = [
    "zookeeper",
    "kafka"
]

class SaasBuilder(ProdBuilder):

    def __init__(self, **kwargs):
        self._build_local = kwargs.pop("build_local", False)
        super(SaasBuilder, self).__init__(**kwargs)

    def get_default_services(self):
        return PRODUCTION_SERVICES

    def get_container_build_path(self, container):
        if container in tools_services:
            return os.path.join(SAAS_PATH, "tools", container)
        else:
            return os.path.join(SAAS_PATH, container)

    def build_one(self, container, **kwargs):
        path = self.get_container_build_path(container)
        build_dir = os.path.join(path, "docker_build")
        shutil.rmtree(build_dir, ignore_errors=True)
        os.mkdir(build_dir)
        if not self._build_local:
            build_script = os.path.join(path, "build-in-contr.sh")
        else:
            build_script = os.path.join(path, "build.sh")
        if not os.path.isfile(build_script):
            build_script = None
        ret = super(SaasBuilder, self).build_one(container, build_script=build_script, **kwargs)
        shutil.rmtree(build_dir, ignore_errors=True)
        return ret

    @staticmethod
    def argparser():
        parser = super(SaasBuilder, SaasBuilder).argparser()
        parser.add_argument("-bl", "--build-local", action="store_true", help="Build locally without container")
        return parser
