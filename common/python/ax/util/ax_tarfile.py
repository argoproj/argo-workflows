#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import argparse
import copy
import fnmatch
import json
import logging
import os
import re
import sys
import tarfile

from six import string_types

logger = logging.getLogger(__name__)


class AXTarfile(object):
    def __init__(self, excludes=None, use_regex=True):
        self._use_regex = use_regex
        if isinstance(excludes, list):
            excludes_processed = []
            for exclude in excludes:
                if exclude is not None:
                    if self.is_unicode(exclude):
                        excludes_processed.append(exclude.encode('utf-8'))
                    else:
                        assert isinstance(exclude, str), "bad exclude {} {}".format(exclude, type(exclude))
                        excludes_processed.append(exclude)
            if self._use_regex:
                self._excludes = []
                for exclude in excludes_processed:
                    # try to valid the reg
                    if exclude:
                        try:
                            c = re.compile(exclude)
                            self._excludes.append(c)
                        except Exception:
                            logger.error("invalid regex string %s", exclude)
            else:
                self._excludes = excludes_processed
        else:
            self._excludes = None

    def _filter_counter_reset(self):
        self.num_files = 0
        self.num_dir = 0
        self.num_symlink = 0
        self.num_other = 0
        self.num_byte = 0
        self.num_byte_skip = 0
        self.num_skip = 0

    def _filter_match(self, pattern, fn):
        if not self._use_regex:
            try:
                match = fnmatch.fnmatch(fn, pattern)
                return match
            except Exception:
                logger.exception("fnmatch %s %s", fn, pattern)
                return False
        try:
            result = pattern.match(fn)
        except Exception:
            return False

        if result:
            return True
        else:
            return False

    def _filter_common(self, tarinfo):
        name = tarinfo.name
        n = os.path.basename(name)
        if self._excludes is None:
            return tarinfo

        for exclude in self._excludes:
            if exclude is None:
                continue
            if self._filter_match(pattern=exclude, fn=n):
                self.num_byte_skip += tarinfo.size
                self.num_skip += 1
                logger.info("skip %s because of %s", name, exclude)
                return None
        return tarinfo

    def _filter_create(self, tarinfo):
        tarinfo = self._filter_common(tarinfo)
        if tarinfo:
            self.num_byte += tarinfo.size
            if tarinfo.isfile():
                self.num_files += 1
            elif tarinfo.isdir():
                self.num_dir += 1
            else:
                self.num_other += 1
        return tarinfo

    def _add(self, tar, name, arcname, recursive=True):
        tarinfo = tar.gettarinfo(name, arcname)

        if arcname is None:
            arcname = name

        if tarinfo is None:
            logger.error("tarfile: Unsupported type %s", name)
            return

        tarinfo = self._filter_create(tarinfo)
        if tarinfo is None:
            logger.info("tarfile: Excluded %s", name)
            return

        # Append the tar header and data to the archive.
        if tarinfo.isreg():
            with open(name, "rb") as f:
                tar.addfile(tarinfo, f)

        elif tarinfo.isdir():
            tar.addfile(tarinfo)
            if recursive:
                for f in os.listdir(name):
                    # listdir has problem with unicode names and generate bytes in some cases.
                    # Map it back to unicode.
                    f = f.decode("charmap")
                    try:
                        self._add(tar=tar, name=os.path.join(name, f),
                                  arcname=os.path.join(arcname, f),
                                  recursive=recursive)
                    except Exception:
                        logger.exception("cannot add %s %s", name, f)
        else:
            tar.addfile(tarinfo)

    def tar_gen(self, tar_name, dirname, compression_mode=None, recursive=True):
        if self.is_unicode(tar_name):
            tar_name = tar_name.encode('utf-8')
        if self.is_unicode(dirname):
            dirname = dirname.encode('utf-8')

        if not os.path.isabs(dirname):
            logger.error("src direct not a absolute path %s", dirname)
            # not a absolute path
            return False
        dirname = os.path.realpath(dirname)

        if dirname == "/":
            # is root
            arcname = 'root'
        else:
            arcname = os.path.basename(dirname)

        open_mode = "w"
        if isinstance(compression_mode, string_types):
            open_mode += ":" + compression_mode
        tar = tarfile.open(tar_name, open_mode)

        self._filter_counter_reset()
        try:
            self._add(tar=tar, name=dirname, arcname=arcname, recursive=recursive)
            tar.close()
            ret = True
        except Exception:
            logger.exception("cannot gen tar %s from %s, %s", tar_name, dirname, arcname)
            ret = False
        finally:
            tar.close()
        return ret

    def _filter_extract(self, tarinfo, arcname):
        name = tarinfo.name
        if os.path.isabs(name):
            logger.error("skip absolute path %s", name)
            return
        if name.startswith('..'):
            logger.error("skip %s", name)
            return
        if self._filter_common(tarinfo) is None:
            return None
        newtarinfo = copy.copy(tarinfo)
        if arcname:
            newname = newtarinfo.name
            names = newname.split("/", 1)
            newtarinfo.name = arcname
            if len(names) == 2 and names[1]:
                newtarinfo.name = os.path.join(newtarinfo.name, names[1])
        return newtarinfo

    def tar_extract(self, tar_name, dirname, compression_mode=None):
        if self.is_unicode(tar_name):
            tar_name = tar_name.encode('utf-8')
        if self.is_unicode(dirname):
            dirname = dirname.encode('utf-8')

        if not os.path.isabs(dirname):
            # not a absolute path
            logger.error("dest dir not a absolute path %s", dirname)
            return False

        dirname = os.path.realpath(dirname)
        if dirname == '/':
            # dirname is root
            arcname = ''
        else:
            dirname, arcname = os.path.split(dirname)

        open_mode = "r"
        if isinstance(compression_mode, string_types):
            open_mode += ":" + compression_mode

        self._filter_counter_reset()

        tar = tarfile.open(tar_name, open_mode)
        success = True
        try:
            for tarinfo in tar:
                try:
                    newtarinfo = self._filter_extract(tarinfo, arcname)
                except Exception:
                    logger.exception("filter exception %s %s", tarinfo, arcname)
                    success = False
                    continue
                if newtarinfo is None:
                    continue
                name = newtarinfo.name
                new_name = os.path.join(dirname, name)
                try:
                    if os.path.exists(new_name):
                        try:
                            os.chmod(new_name, newtarinfo.mode)
                        except Exception:
                            pass
                        try:
                            os.remove(new_name)
                        except Exception:
                            pass
                except Exception:
                    logger.exception("exists exception, %s", new_name)
                try:
                    tar.extract(newtarinfo, path=dirname)
                except Exception:
                    logger.exception("dir: %s name: %s", dirname, name)
                    success = False
        except Exception:
            logger.exception("bad tar file=%s mode=%s", tar_name, open_mode)
            success = False

        tar.close()
        return success

    def tar_get_info(self, tar_name, compression_mode=None):
        if self.is_unicode(tar_name):
            tar_name = tar_name.encode('utf-8')

        open_mode = "r"
        if isinstance(compression_mode, string_types):
            open_mode += ":" + compression_mode

        success = True
        self._filter_counter_reset()
        tar = tarfile.open(tar_name, open_mode)
        try:
            for tarinfo in tar:
                self.num_byte += tarinfo.size
                if tarinfo.isfile():
                    self.num_files += 1
                elif tarinfo.isdir():
                    self.num_dir += 1
                elif tarinfo.issym():
                    self.num_symlink += 1
                else:
                    self.num_other += 1
        except Exception:
            logger.exception("tar_get_info exception tar_name=%s comp=%s", tar_name, compression_mode)
            success = False

        tar.close()
        return success

    def get_tar_structure(self, tar_name, compression_mode=None):
        if self.is_unicode(tar_name):
            tar_name = tar_name.encode('utf-8')

        open_mode = "r"
        if isinstance(compression_mode, string_types):
            open_mode += ":" + compression_mode

        tar = tarfile.open(tar_name, open_mode)
        result = dict()
        try:
            for tarinfo in tar:
                current = result
                all_parts = self.split_all(tarinfo.name)
                for part in all_parts[:-1]:
                    if part not in current:
                        current[part] = dict()
                    current = current[part]
                if all_parts:
                    if tarinfo.isdir():
                        current[all_parts[-1]] = dict()
                    else:
                        current[all_parts[-1]] = 1
            return json.dumps(result)
        except Exception:
            logger.exception("get_tar_structure exception tar_name=%s comp=%s", tar_name, compression_mode)
            return None

    @staticmethod
    def split_all(path):
        allparts = []
        while 1:
            parts = os.path.split(path)
            if parts[0] == path:  # sentinel for absolute paths
                allparts.insert(0, parts[0])
                break
            elif parts[1] == path:  # sentinel for relative paths
                allparts.insert(0, parts[1])
                break
            else:
                path = parts[0]
                allparts.insert(0, parts[1])
        return allparts

    @staticmethod
    def is_unicode(test_string):
        if sys.version_info < (3, 0):
            if isinstance(test_string, unicode):
                return True
        return False


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='AX tarfile',
                                     formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    parser.add_argument('--dir', help='directory')
    parser.add_argument('--tar', help='tarfile')
    parser.add_argument('--exclude', help='excludes list')
    parser.add_argument('--compression-mode', default="gz", help='compression mode')
    parser.add_argument('--create', action="store_true", help='create tarfile')
    parser.add_argument('--extract', action="store_true", help='extract tarfile')

    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(message)s")
    logging.getLogger("ax").setLevel(logging.DEBUG)

    args = parser.parse_args()

    axtarfile = AXTarfile([args.exclude])
    if args.create:
        axtarfile.tar_gen(args.tar, args.dir, args.compression_mode)
    elif args.extract:
        axtarfile.tar_extract(args.tar, args.dir, args.compression_mode)
