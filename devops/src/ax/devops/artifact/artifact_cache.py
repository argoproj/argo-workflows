# Copyright 2015-2016 Applatix, Inc. All rights reserved.

import logging
import threading
import time
import random
from collections import deque

logger = logging.getLogger(__name__)


class ArtifactFileObject(object):
    def __init__(self, key, file_size, file_path, s3_path):
        self.key = key
        self.file_size = file_size
        self.file_path = file_path
        self.s3_path = s3_path
        # TODO use a read-write lock: https://majid.info/blog/a-reader-writer-lock-for-python/
        self._file_lock = threading.Lock()

    def __repr__(self):
        return "key: {}, file_size: {}, file_path: {}, s3_path: {}".format(self.key, self.file_size, self.file_path, self.s3_path)

    def read_file(self, file_name):
        # TODO
        logger.info("Reading file %s from %s", file_name, self.file_path)
        time.sleep(random.randint(0, 2))
        return file_name

    def write_file(self):
        # TODO
        logger.info("Write tar file %s from s3 (%s) and save to path %s", self.key, self.s3_path, self.file_path)
        time.sleep(random.randint(0, 8))

    def remove_file(self):
        # TODO
        logger.info("Delete file %s from local disk space %s", self.key, self.file_path)
        time.sleep(random.randint(0, 4))

    def get_lock(self):
        return self._file_lock


class ArtifactCache(object):
    def __init__(self, max_size):
        self._max_size = max_size
        self._current_size = 0
        self.key_dict = dict()
        self.key_queue = deque()
        self._cache_lock = threading.Lock()

    def _remove(self, file_size):
        while self._current_size + file_size > self._max_size:
            to_delete_object = self.key_queue[0]
            with to_delete_object.get_lock():
                to_delete_object.remove_file()
                self._current_size -= to_delete_object.file_size
                self.key_dict.pop(to_delete_object.key, None)
                self.key_queue.popleft()

    def get_file(self, key, file_size, file_path, s3_path, file_name):
        if file_size > self._max_size:
            logger.info("File size %s bigger than max_size %s", file_size, self._max_size)
            raise Exception
        self._cache_lock.acquire()
        self._remove(file_size)
        if key not in self.key_dict:
            new_file_object = ArtifactFileObject(key=key, file_size=file_size, file_path=file_path, s3_path=s3_path)
            with new_file_object.get_lock():
                self.key_dict[key] = new_file_object
                self.key_queue.append(new_file_object)
                self._current_size += file_size
                self._cache_lock.release()
                new_file_object.write_file()
                return new_file_object.read_file(file_name=file_name)
        else:
            read_object = self.key_dict[key]
            with read_object.get_lock():
                self._cache_lock.release()
                return read_object.read_file(file_name=file_name)

