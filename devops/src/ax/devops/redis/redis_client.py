#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import logging
import re
import redis
import time
from redis.exceptions import ConnectionError, TimeoutError
from retrying import retry

logger = logging.getLogger(__name__)

DB_DEFAULT = 0
DB_RESULT = 2
DB_REPORTING = 10

REDIS_HOST = "redis.axsys"


class RedisClient(object):
    """AX Redis client."""

    MAX_ATTEMPT = 1
    WAIT_FIXED = 5000

    def __init__(self, host=None, port=None, db=None, password=None, retry_max_attempt=1, retry_wait_fixed=5000, **kwargs):
        """Initialize connection to Redis server.

        :param host:
        :param port:
        :param db:
        :param password:
        :param kwargs:
        :return:
        """
        self.host = host or 'localhost'
        self.port = port or 6379
        self.db = db or 0
        self.password = password
        RedisClient.MAX_ATTEMPT = retry_max_attempt
        RedisClient.WAIT_FIXED = retry_wait_fixed
        self.client = redis.StrictRedis(host=self.host, port=self.port, db=self.db,
                                        password=self.password, decode_responses=True, **kwargs)

    def wait(self, timeout=None):
        """Wait redis server to be available.

        :param timeout: Wait forever if none.
        :return:
        """
        start_time = time.time()
        while True:
            connected = self.ping()
            if connected:
                logger.info('Successfully connected to Redis server')
                return
            if timeout is not None and time.time() - start_time > timeout:
                raise TimeoutError('Failed to connect to Redis server in %s seconds', timeout)
            logger.warning('Unable to connect to Redis server, retry')
            time.sleep(5)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def ping(self):
        """Ping redis server."""
        try:
            self.client.ping()
        except (ConnectionError, TimeoutError):
            return False
        else:
            return True

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def set(self, key, value, expire=None, encoder=None):
        """Create a key-value pair.

        :param key:
        :param value:
        :param expire:
        :param encoder: Encode value before writing to redis.
        :return:
        """
        if encoder is not None:
            value = encoder(value)
        if expire is None:
            self.client.set(key, value)
        else:
            self.client.setex(key, expire, value)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def get(self, key, decoder=None):
        """Get value of a key.

        :param key:
        :param decoder: If expected return value is not a string, we need to decode the value (e.g. decode to an integer).
        :return:
        """
        value = self.client.get(key)
        if decoder is not None and value is not None:
            value = decoder(value)
        return value

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def delete(self, key):
        """Delete a key.

        :param key:
        :return:
        """
        self.client.delete(key)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def keys(self, regex=None):
        """Get keys by regular expression.

        :param regex:
        :return:
        """
        keys = self.client.keys()
        if regex is not None:
            keys = [key for key in keys if re.match(regex, key)]
        return keys

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def expired(self, key):
        """Check if key is expired.

        :param key:
        :return: Boolean.
        """
        return not self.client.pttl(key)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def exists(self, key):
        """Check if key exists.

        :param key:
        :return: Boolean.
        """
        return key in self.client.keys()

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def flushall(self):
        """Flush all keys on all DB.

        :return:
        """
        return self.client.flushall()

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def flush(self):
        """Flush all keys on current DB.

        :return:
        """
        return self.client.flushdb()

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def hset(self, hash, key, value, encoder=None):
        """Set a key in a hash.

        :param hash:
        :param key:
        :param value:
        :param encoder:
        :return:
        """
        if encoder is not None:
            value = encoder(value)
        return self.client.hset(hash, key, value)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def hget(self, hash, key, decoder=None):
        """Get a key in a hash.

        :param hash:
        :param key:
        :param decoder:
        :return:
        """
        value = self.client.hget(hash, key)
        if decoder is not None and value is not None:
            value = decoder(value)
        return value

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def hdel(self, hash, key):
        """Delete a key in a hash.

        :param hash:
        :param key:
        :return:
        """
        return self.client.hdel(hash, key)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def hexists(self, hash, key):
        """Test if key exists in a hash.

        :param hash:
        :param key:
        :return:
        """
        return self.client.exists(hash) and key in self.client.hkeys(hash)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def hsetall(self, hash, mapping, encoder=None):
        """Set a hash.

        :param hash:
        :param mapping:
        :param encoder:
        :return:
        """
        self.client.delete(hash)
        if encoder is not None:
            encoded_mapping = {}
            for k in mapping:
                encoded_mapping[k] = encoder(mapping[k])
        else:
            encoded_mapping = mapping
        return self.client.hmset(hash, encoded_mapping)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def hgetall(self, hash, decoder=None):
        """Get a hash.

        :param hash:
        :param decoder:
        :return:
        """
        mapping = self.client.hgetall(hash)
        if decoder is not None:
            for k in mapping:
                mapping[k] = decoder(mapping[k])
        return mapping

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def hdelall(self, hash):
        """Delete a hash.

        :param hash:
        :return:
        """
        return self.client.delete(hash)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def hsetex(self, hash, expire):
        """Set expiration of a hash.

        :param hash:
        :param expire:
        :return:
        """
        return self.client.expire(hash, expire)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def hkeys(self, hash, regex=None):
        """Get all keys in a hash.

        :param hash:
        :param regex:
        :return:
        """
        keys = self.client.hkeys(hash)
        if regex is not None:
            keys = [key for key in keys if re.match(regex, key)]
        return keys

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def rename(self, src, dst):
        """Rename a key.

        :param src:
        :param dst:
        :return:
        """
        return self.client.rename(src, dst)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def brpop(self, key, timeout=0):
        """BRPOP a list.

        :param key:
        :param timeout:
        :return:
        """
        return self.client.brpop(key, timeout)

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def rpush(self, key, value, expire=None, encoder=None):
        """Push to the right end of the list.

        :param key:
        :param value:
        :param expire:
        :param encoder:
        :return:
        """
        if encoder is not None:
            value = encoder(value)
        ret = self.client.rpush(key, value)
        if expire is not None:
            self.client.expire(name=key, time=expire)
        return ret

    @retry(stop_max_attempt_number=MAX_ATTEMPT, wait_fixed=WAIT_FIXED)
    def lrange(self, key, start, end, decoder=None):
        """Retrieve range of the list between start index and end index

        :param key:
        :param start:
        :param end:
        :param decoder:
        :return:
        """
        value = self.client.lrange(key, start, end)
        if decoder is not None and value is not None:
            result = []
            for item in value:
                result.append(decoder(item))
            return result
        return value
