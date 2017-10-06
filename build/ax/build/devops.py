
import glob
import logging
import os
import re
import shutil

from concurrent.futures import ThreadPoolExecutor, wait

from ax.build.common import ProdBuilder, SRC_PATH, ARGO_BASE_REGISTRY, PYTHON_COMMON_PATH, run_cmd

logger = logging.getLogger('ax.build')

DEVOPS_CONTAINERS_PATH = os.path.join(SRC_PATH, "devops/builds")

DEVOPS_BUILDER_IMAGE = '{}/argobase/axdevopsbuilder:v8'.format(ARGO_BASE_REGISTRY)
DEBIAN_BUILDER_IMAGE = '{}/argobase/axplatbuilder-debian:v15'.format(ARGO_BASE_REGISTRY)

class DevOpsModules(object):
    TEMPLATES = 'templates'
    INFRASTRUCTURE = 'infrastructure'
    DEVOPS = 'devops'
    DEBUG = 'debug'
    WORKFLOW = 'workflow'

ALL_MODULES = [getattr(DevOpsModules, m) for m in dir(DevOpsModules) if not m.startswith('_')]
DEFAULT_MODULES = [DevOpsModules.INFRASTRUCTURE, DevOpsModules.DEVOPS, DevOpsModules.WORKFLOW]

BASE_CONTAINERS = [
    'axdevopsbuilder',
    'axdevopsbuilder-debian'
]


class DevOpsBuilder(ProdBuilder):

    def __init__(self, **kwargs):
        modules = kwargs.pop('modules', None)
        services = kwargs.pop('services', None) or []
        if modules:
            invalid_modules = list(set(modules) - set(ALL_MODULES))
            if invalid_modules:
                raise ValueError("Invalid module(s): {}".format(invalid_modules))
            services = list(set(services + self._get_module_services(modules)))
        elif not services:
            services = self.get_default_services()
        super(DevOpsBuilder, self).__init__(services=services, python_version=3, **kwargs)
        self.builder_image = kwargs.get('builder') or DEVOPS_BUILDER_IMAGE
        if kwargs.get('no_builder'):
            logger.info("BUILDING base images first {}".format(BASE_CONTAINERS))
            bak_services = self.services
            self.services = BASE_CONTAINERS
            self.build()
            logger.info("Base images built successfully")
            self.builder_image = "%s/%s/%s:%s" % (self.registry, self.image_namespace, "axdevopsbuilder", self.image_version)
            self.services = bak_services

    def get_default_services(self):
        """Default services to build"""
        return self._get_module_services(DEFAULT_MODULES)

    def _get_module_services(self, modules):
        if isinstance(modules, str):
            modules = [modules]
        services = set()
        for module in modules:
            container_dirs = glob.glob('{}/{}/*/Dockerfile*'.format(DEVOPS_CONTAINERS_PATH, module))
            services.update(set([os.path.basename(os.path.dirname(dock_file)) for dock_file in container_dirs]))
        return list(services)

    def get_container_build_path(self, container):
        paths = glob.glob("{}/*/{}".format(DEVOPS_CONTAINERS_PATH, container))
        assert len(paths) == 1, "Found {} paths for container {}: {}".format(len(paths), container, paths)
        return paths[0]

    def pull_latest_templates(self):
        """Determine which templates need to be pulled, based on the services requested to be built"""
        templates = set()
        for container in self.services:
            build_path = self.get_container_build_path(container)
            (docker_file, _) = self._find_build_scripts(build_path)
            with open(docker_file, 'r') as f:
                docker_file_contents = f.read()
            template_containers = self._get_module_services(DevOpsModules.TEMPLATES)
            match = re.search("FROM {}/({})".format(ARGO_BASE_REGISTRY, '|'.join(template_containers)), docker_file_contents, re.IGNORECASE)
            if match:
                templates.add(match.group(1).lower())
        if templates:
            logger.info("Pulling templates: %s", templates)
            with ThreadPoolExecutor(max_workers=len(templates)) as executor:
                future_res = []
                for template in templates:
                    image_name = "{}/{}".format(ARGO_BASE_REGISTRY, template)
                    res = executor.submit(run_cmd, "docker pull {}".format(image_name))
                    future_res.append(res)
                wait(future_res)
                _ = [each.result() for each in future_res]
        else:
            logger.info("No templates required to be pulled")

    def _copy_source_code(self, container_build_path):
        """
        Copy source code to the container build directory
        :param container_build_path:
        """
        docker_build_dir = "{}/docker_build".format(container_build_path)
        os.makedirs(docker_build_dir)

        # Workaround due to the python3.4 shutil.copytree have the bug on NFS
        # dereference symlink
        run_cmd('cp -LR {} {}/src'.format(PYTHON_COMMON_PATH, docker_build_dir))
        run_cmd('cp -LR {}/devops/requirements {}'.format(SRC_PATH, docker_build_dir))

        # Issue #306 devops component requires platform_client. This is only needed from release 1.0.1
        # if not re.search('devopsbuilder', container_build_path):
        #     # DevOps source code is not protected (for the time being). Platform code is.
        #     # Do not copy platform code into devops containers to maintain this protection
        #     # and prevent exposure of our platform source to customers.
        #     shutil.rmtree("{}/src/ax/platform".format(docker_build_dir))
        #     shutil.rmtree("{}/src/ax/platform_client".format(docker_build_dir))

    def build_one(self, container, **kwargs):
        path = self.get_container_build_path(container)
        self._copy_source_code(path)
        builder_image = self.builder_image
        if container in ['fixturemanager', 'workflow']:
            # fixturemanager uses debian because mongodb on alpine is not well supported
            builder_image = DEBIAN_BUILDER_IMAGE
        return super(DevOpsBuilder, self).build_one(container, builder_image=builder_image, **kwargs)

    def build(self, *args, **kwargs):
        self.pull_latest_templates()
        logger.info("Frozen requirements:")
        run_cmd("docker run --rm {} pip freeze".format(self.builder_image or DEVOPS_BUILDER_IMAGE))
        ret = super(DevOpsBuilder, self).build(*args, **kwargs)
        return ret

    @staticmethod
    def argparser():
        parser = super(DevOpsBuilder, DevOpsBuilder).argparser()
        parser.add_argument('-m', '--module', action='append', dest='modules', default=[],
                            choices=ALL_MODULES, help='Repeatedly add a module to be built')
        grp = parser.add_mutually_exclusive_group()
        grp.add_argument('--builder', default=DEVOPS_BUILDER_IMAGE,
                            help='Use a supplied builder image instead of default ({})'.format(DEVOPS_BUILDER_IMAGE))
        grp.add_argument('--no-builder', action='store_true', help='Do not use a prebuilt base image')
        return parser
