#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module library to call a function with retry and handle exception
"""

import time
import logging

logger = logging.getLogger(__name__)


class AXRetry(object):
    def __init__(self,
                 retries=2,
                 delay=0.1,
                 backoff=True,
                 success_exception=None,
                 retry_exception=None,
                 success_check=None,
                 default=None,
                 success_default=None,
                 logging=True):
        """
        Retry object to retry function calls.
        We will retry function several times and have backed off delay.
        For each function call,
            if hit success_exception, it's considered success and default value is returned without retry.
            if hit retry_exception, it's ignored and retried.
            if uncaught exception is hit, raise it
            if no exception but success_check returned True, treat it same as success_exception but return actual ret.
        If all retries failed, return default value.

        :param retries: Number of retries.
        :param delay: Number of seconds to delay for first retry. Delay will multiplied by count i for each loop.
        :param success_exception: Exception considered as success, return immediately.
        :param retry_exception: Exception to ignore and retry.
        :param success_check: Function to process underlying return value and determine success or not.
                              It should be:
                              def success_check(value):
                                  return True or False
        :param default: Default value to return if still failed after retry
        :param success_default: Default value to return if success_exception is hit
        :param logging: Whether to log when exception hits
        """
        assert retries > 0, "Invalid retries %s" % retries
        assert delay > 0, "Invalid delay %s" % delay
        self.retries = retries
        self.delay = delay
        self.backoff = backoff
        self.success_exception = () if success_exception is None else success_exception
        self.retry_exception = () if retry_exception is None else retry_exception
        self.success_check = success_check
        self.default = default
        self.success_default = success_default
        self.logging = logging
        assert set(self.success_exception).isdisjoint(set(self.retry_exception)), \
            "Overlapping exception %s %s" % (success_exception, retry_exception)

    def __repr__(self):
        return "Retry policy (%s %s exceptions %s %s check %s defaults %s, %s)" % (self.retries,
                                                                                   self.delay,
                                                                                   self.success_exception,
                                                                                   self.retry_exception,
                                                                                   self.success_check,
                                                                                   self.default,
                                                                                   self.success_default)


def ax_retry(func, retry, *args, **kwargs):
    """
    Call a function with retry and exception handling.

    :param func: Function
    :param retry: Retry policy as defined above
    :param args: Pass-through args
    :param kwargs: Pass-through kwargs
    :return:
    """
    for i in range(retry.retries):
        delay = retry.delay * (i + 1) if retry.backoff else retry.delay
        try:
            ret = func(*args, **kwargs)
            if retry.success_check and not retry.success_check(ret):
                # This is considered failure. Retry.
                logger.debug("Retry #%s: %s(%s, %s), %s returned %s", i, func, args, kwargs, retry, ret)
                time.sleep(delay)
                continue
            return ret
        except retry.success_exception:
            # This is success return. We don't have return value here.
            logger.debug("Exception %s is OK", retry.success_exception)
            return retry.success_default
        except retry.retry_exception as e:
            # Ignore this error and retry. Log for debugging now.
            if retry.logging:
                logger.exception("Retry #%s: %s(%s,%s), %s, %s", i, func, args, kwargs, retry, e)
            time.sleep(delay)
    else:
        return retry.default


if __name__ == "__main__":

    logging.basicConfig()
    logging.getLogger("ax").setLevel(logging.DEBUG)

    def test1():
        global count
        count += 1
        assert 0

    def test2(ret):
        global count
        count += 1
        return ret

    def check(val):
        return True if val == "Good" else False

    count = 0
    retry = AXRetry(retries=10, delay=0.1, retry_exception=(AssertionError,))
    ts = time.time()
    ax_retry(test1, retry)
    assert time.time() > ts + 4
    assert count == 10

    count = 0
    retry = AXRetry(retries=10, delay=0.1, success_exception=(AssertionError,))
    ts = time.time()
    ax_retry(test1, retry)
    assert time.time() < ts + 2
    assert count == 1

    count = 0
    retry = AXRetry(success_check=check)
    ret = ax_retry(test2, retry, "Good")
    assert count == 1
    assert ret == "Good"

    count = 0
    retry = AXRetry(retries=5, success_check=check, default="Default")
    ret = ax_retry(test2, retry, "Bad")
    assert count == 5
    assert ret == "Default"

    try:
        retry = AXRetry(success_exception=(AssertionError,), retry_exception=(AssertionError,))
    except AssertionError as e:
        print("Caught assertion {}".format(str(e)))
    else:
        assert 0

    print("Above assertions expected.")
    print("All tests passed.")
