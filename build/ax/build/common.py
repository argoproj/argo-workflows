import abc
import argparse
import getpass
import glob
import logging
import os
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
import threading
import time

from multiprocessing.dummy import Pool

logger = logging.getLogger('ax.build')

SRC_PATH = os.path.realpath(os.path.join(os.path.dirname(__file__), "../../.."))
with open(os.path.join(SRC_PATH, "version.txt"), 'r') as version_file:
    VERSION = version_file.read().strip()

# Hack: We need base registry as long as we do two phased build. Need to clean up.
ARGO_BASE_REGISTRY = os.getenv("ARGO_BASE_REGISTRY", "docker.io")

PYTHON_COMMON_PATH = os.path.join(SRC_PATH, 'common/python')

def product_version(debug=False):
    """Customer visible product version compiled into the resulting binaries.
    
    This version is combined with a commit hash, (e.g. 0.1.1-ce40545) and will be emitted upon service startup
    or using the `--version` option in CLIs. By default (and in production), the product version will also be used 
    as the container image version. However, users can override the container image version during build
    (typically with the string `latest`) in order to ease test and deployment during development.
    """
    cmd = "git -C {} rev-parse --short=7 HEAD".format(SRC_PATH)
    commit_hash = run_cmd(cmd).strip()
    version = '{}-{}'.format(VERSION, commit_hash)
    if debug:
        version += '-debug'
    if run_cmd("git -C {} diff --shortstat".format(SRC_PATH)).strip():
        version += '-dirty'
    return version

def run_cmd(cmd, shell=False, retry=None, retry_interval=None):
    """Wrapper around subprocess.Popen to capture/print output

    :param shell: execute the command in a shell
    :param retry: number of retries to perform if command fails with non-zero return code
    :param retry_interval: interval between retries
    """
    orig_cmd = cmd
    if not shell:
        cmd = shlex.split(cmd)
    attempts = 1 if not retry else 1 + retry
    retry_interval = retry_interval or 10

    for attempt in range(attempts):
        lines = []
        logger.info('$ {}'.format(orig_cmd))
        proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, universal_newlines=True,
                                stderr=subprocess.STDOUT, shell=shell)
        for line in iter(proc.stdout.readline, ''):
            line = line[0:-1] # chop the newline
            logger.info(line)
            lines.append(line)
        proc.stdout.close()
        proc.wait()
        output = '\n'.join(lines)
        if proc.returncode == 0:
            break
        else:
            if attempt+1 < attempts:
                logger.warning("Attempt %s/%s of command: '%s' failed with returncode %s. Retrying in %ss", 
                               attempt+1, attempts, orig_cmd, proc.returncode, retry_interval)
                time.sleep(retry_interval)
    else:
        raise subprocess.CalledProcessError(proc.returncode, cmd, output=output)
    return output


class ProdBuilder(object):
    __metaclass__ = abc.ABCMeta

    def __init__(self, **kwargs):
        # Need to remove default value here after migration is done.
        self.registry = kwargs.get('registry') or os.getenv('ARGO_DEV_REGISTRY')
        assert self.registry is not None
        self.debug = True
        self.version = product_version()
        self.image_version = kwargs.get("image_version") or self.version
        self.image_namespace = kwargs.get("image_namespace") or getpass.getuser()
        self.services = kwargs.get('services') or self.get_default_services()
        self.no_push = kwargs.get('no_push', False)
        self.no_cache = kwargs.get('no_cache', False)
        self.pyinstallerdebug = kwargs.get('pyinstallerdebug', False)

        self.python_version = kwargs.get('python_version', 3)

        # Image to use when using a container to build a PyInstaller spec file
        self.builder_image = None

        # Build without using mapped docker volumes. Will use `docker cp` to copy source into the container instead.
        self._no_volume_map = kwargs.get('no_volume_map', False)
        # `--user` option must be supplied to `docker run` if we are mapping any volumes, in order to prevent permission
        # issues when creating files inside the source tree
        self._user = "{}:{}".format(os.getuid(), os.getgid())
        self._serialized = len(self.services) == 1 or kwargs.get('serialized', False)

    @abc.abstractmethod
    def get_default_services(self):
        """Returns the default services to build"""
        pass

    @abc.abstractmethod
    def get_container_build_path(self, container):
        """Returns the docker build path for a given service name"""
        pass

    def _find_build_scripts(self, path):
        """Returns a tuple to the path of the dockerfile and build.sh script. Takes into consideration of a debug build"""
        # Build checks in order of appearance --debug:
        # ['Dockerfile.debug.in', 'Dockerfile.debug', 'Dockerfile.in', 'Dockerfile']
        dockfile_search = ['Dockerfile.in', 'Dockerfile']
        buildscript_search = ['build.sh']
        if self.debug:
            dockfile_search = ['Dockerfile-debug.in', 'Dockerfile-debug'] + dockfile_search
            buildscript_search = ["build-debug.sh"] + buildscript_search
            
        for dockbase in dockfile_search:
            dockerfile = os.path.join(path, dockbase)
            if os.path.isfile(dockerfile):
                break
        else:
            raise ValueError("Unable to locate Dockerfile under {}".format(path))

        buildscript = None
        for buildbase in buildscript_search:
            build_sh = os.path.join(path, buildbase)
            if os.path.isfile(build_sh) and os.access(build_sh, os.X_OK):
                buildscript = build_sh
                break
        return (dockerfile, buildscript)

    def clean_build_dirs(self, services=None):
        """Cleans the docker_build dir for a list of services"""
        if isinstance(services, str):
            services = [services]
        elif services is None:
            services = self.services
        logger.info("Cleaning docker_build dirs: %s", services)
        for container in services:
            path = self.get_container_build_path(container)
            build_dir = os.path.join(path, "docker_build")
            shutil.rmtree(build_dir, ignore_errors=True)

    def clean_pyc(self, docker_build_dir, excludes=('pyc',)):
        """Purges all pyc files in a docker_build dir so that they will not be included in the container"""
        logger.info('Clean up files in %s: %s', docker_build_dir, excludes)
        for _root, _, _files in os.walk(docker_build_dir, followlinks=False):
            for f in _files:
                if f.endswith(excludes):
                    file_name = os.path.join(_root, f)
                    #logger.info('Remove the file %s', file_name)
                    os.remove(file_name)

    def build_one(self, container, tag=None, no_push=None, macros=None, build_script=None,
                  builder_image=None, force_debug=None):
        """Build a single container

        :param container: name of container to build
        :param tag: apply tag to the container upon build
        :param no_push: do not push the container to registry after build
        :param macros: macro replacement used for dynamically generated Dockerfiles
        :param builder_image: pyinstaller builder image
        """
        threading.current_thread().name = container
        start_time = time.time()
        logger.info("Building %s", container)
        if no_push is None:
            no_push = self.no_push
        if macros is None:
            macros = {}
        path = self.get_container_build_path(container)

        if force_debug is not None:
            logger.info("Found force debug: %s", force_debug)
            self.debug = force_debug

        dockerfile, build_sh = self._find_build_scripts(path)
        if build_script:
            build_sh = build_script
        build_dir = os.path.join(path, "docker_build")
        os.makedirs(build_dir, exist_ok=True)

        builder_image = builder_image or self.builder_image
        if 'BUILDER_IMAGE_ID' not in macros and builder_image:
            # set builder image id for Dockerfiles who wish to base their image on it (typically debug containers)
            macros['BUILDER_IMAGE_ID'] = builder_image

        if 'ARGO_BASE_REGISTRY' not in macros:
            macros['ARGO_BASE_REGISTRY'] = ARGO_BASE_REGISTRY
        if 'ARGO_DEV_REGISTRY' not in macros:
            macros['ARGO_DEV_REGISTRY'] = self.registry

        # The presence of Dockerfile.in indicates the Dockerfile should be dynamically generated.
        # `macros` indicates what should be string replaced when generating the Dockerfile.
        # Generate a temporary Dockerfile (with a randomized name) during the `docker build` 
        dockerfile_temp = None
        with open(dockerfile, 'r') as f:
            dockerfile_contents = f.read()
        for key, val in macros.items():
            dockerfile_contents = dockerfile_contents.replace("%%{}%%".format(key), val)
        dockerfile_temp = tempfile.NamedTemporaryFile(prefix="Dockerfile.", suffix='.tmp', dir=path, delete=True)
        dockerfile_temp.write(str.encode(dockerfile_contents))
        dockerfile_temp.flush()
        dockerfile = dockerfile_temp.name

        try:
            if build_sh:
                run_cmd(build_sh)
            self.clean_pyc(build_dir)
            build_cmd_parts = ['docker', 'build']
            if os.path.basename(dockerfile) != 'Dockerfile':
                build_cmd_parts.append('-f {}'.format(dockerfile))
            if self.no_cache:
                build_cmd_parts.append('--no-cache')
            if tag:
                build_cmd_parts.append('-t {}'.format(tag))
            build_cmd_parts.append(path)
            build_cmd = " ".join(build_cmd_parts)
            out = run_cmd(build_cmd)
            matches = re.findall(r"Successfully built\s+(\w+)", out)
            image_id = matches[-1]
            image_size = int(run_cmd("docker inspect -f '{{{{.Size}}}}' {}".format(image_id)))
        except subprocess.CalledProcessError as cpe:
            if cpe.output:
                logger.error("Build of %s failed at command:\n$ %s\n%s", container, cpe.cmd, cpe.output)
            raise
        finally:
            if dockerfile_temp:
                dockerfile_temp.close()  # will delete file

        self.packaging_test(image_id, build_dir)
        build_time = time.time()

        if no_push or tag is None:
            logger.info("Skip push of %s (tag: %s)", container, tag)
        else:
            logger.info("Pushing: %s", tag)
            # push image with retry
            retry_left = 10
            while True:
                try:
                    run_cmd("docker push %s" % tag)
                    break
                except subprocess.CalledProcessError:
                    retry_left -= 1
                    logger.warning("push %s error, retry_left=%s", tag, retry_left)
                    if retry_left == 0:
                        logger.exception("Build %s failed", container)
                        raise
                    else:
                        time.sleep(2)

        end_time = time.time()
        logger.info("Built %s as %s (tag: %s)", container, image_id, tag)
        return {
            'container': container,
            'image_id': image_id,
            'size_mb': "{0:.1f} MB".format(image_size / (1000 * 1000)),
            'tag': tag,
            'total_time': float("{0:.1f}".format(end_time - start_time)),
            'build_time': float("{0:.1f}".format(build_time - start_time)),
            'push_time': float("{0:.1f}".format(end_time - build_time)),
        }

    def build_version(self):
        """Generates _version.py under prod/common/python/ax/_version.py"""
        version_py_path = os.path.join(PYTHON_COMMON_PATH, "ax/_version.py")
        version_contents = \
            '# This is an automatically generated file\n' \
            '__version__ = "{}"\n' \
            'debug = {}\n'.format(self.version, 'True' if self.debug else 'False')
        with open(version_py_path, 'w') as f:
            f.write(version_contents)
        logger.info("Generated %s", version_py_path)

    def packaging_test(self, image_id, build_dir):
        """Performs a quick test to verify our executables were packaged correctly by PyInstaller.
        
        Searches the `docker_build` directory for anything under the `dist` directory. This directory contains images
        packaged as single-file executables. The executables are sanity checked by running `<tool> --version` on each
        of them to verify it was packaged correctly. This is meant to detect early packaging problems like in AA-886.

        :param image_id: image_id to test
        :param build_dir: docker_build directory for a container
        """
        distpath = os.path.join(build_dir, 'dist')
        if not os.path.exists(distpath):
            return
        test_cmds = ['/ax/bin/{} --version'.format(bin_name) for bin_name in os.listdir(distpath)]
        if not test_cmds:
            return
        smoke_test_cmd = ' && '.join(test_cmds)
        cmd_args = ['docker', 'run', '--rm', '--entrypoint sh', image_id, '-c "{}"'.format(smoke_test_cmd)]
        cmd = ' '.join(cmd_args)
        logger.info("Performing packaging test with cmd: %s", cmd)
        run_cmd(cmd)

    def build(self):
        logger.info("Building %s", self.version)
        self.clean_build_dirs()
        self.build_version()
        result = {}
        success = True
        pool_size = 1 if self._serialized else len(self.services)
        pool = Pool(pool_size)
        async_results = []
        for container in self.services:
            tag = "%s/%s/%s:%s" % (self.registry, self.image_namespace, container, self.image_version)
            async_res = pool.apply_async(self.build_one, args=(container, ), kwds={'tag': tag})
            async_results.append(async_res)
        pool.close()
        pool.join()
        for container, async_res in zip(self.services, async_results):
            try:
                # will raise any exception that occurred
                result[container] = async_res.get()
                self.clean_build_dirs(services=[container])
            except Exception as e:
                logger.exception("Error building %s", container)
                result[container] = str(e)
                success = False
        row_format = "{container:<20} {image_id:<16} {tag:<50} {size_mb:<10} {total_time} ({build_time}/{push_time})"
        headers = row_format.format(container="Container", image_id="Image ID", tag="Tag", size_mb="Size",
                                    total_time="Time", build_time="build", push_time="push")
        separator_len = len(headers)
        print('=' * separator_len)
        print(headers)
        print('-' * separator_len)
        for container, res in result.items():
            if isinstance(res, dict):
                print(row_format.format(**res))
            else:
                print("{:<20} {}".format(container, res))
        print('=' * separator_len)
        return success

    @staticmethod
    def argparser():
        """Return an argument parser"""
        parser = argparse.ArgumentParser(description='AX build')
        parser.add_argument("-r", "--registry", help="Set registry host name.")
        parser.add_argument("-n", "--image-namespace", help="Set image namespace.", default=getpass.getuser())
        parser.add_argument("-v", "--image-version", help="Set image version.")
        parser.add_argument("-s", "--services", type=str, action='append', help="Build list of services.")
        parser.add_argument("-nc", "--no-cache", action="store_true", help="Build without cache.")
        parser.add_argument("-np", "--no-push", action="store_true", help="Build only but don't push.")
        parser.add_argument("-sl", "--serialized", action="store_true", help="Don't build in parallel.")
        parser.add_argument("--debug", action="store_true",
                            help="Build debug version of containers, containing python interpreter and dependencies")
        parser.add_argument("--pyinstallerdebug", action="store_true",
                            help="Run pyinstaller with debug options to troubleshoot packaging errors")
        parser.add_argument("--no-volume-map", action="store_true",
                            help="Do not use volume mapping during build and instead use `docker cp` to copy source into build container")
        return parser

    @classmethod
    def main(cls):
        if sys.version_info <= (3, 4):
            print("Please use python 3.4+")
            sys.exit(1)
        logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s", stream=sys.stdout)
        logging.getLogger("ax").setLevel(logging.DEBUG)
    
        parser = cls.argparser()
        args = parser.parse_args()
        logger.debug(args)
        start_time = time.time()
        ret = cls(**vars(args)).build()
        elapsed = time.time() - start_time
        if ret:
            logger.info("Build succeeded in %ss", elapsed)
        else:
            logger.error("Build failed in %ss", elapsed)
        sys.exit(not ret)
