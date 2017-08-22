import cProfile
import heapq
import logging
import random
import re
import string
import sys
import time
from argparse import ArgumentParser
from dateutil.parser import parse
from dateutil.tz import tzutc
from functools import cmp_to_key
from pprint import PrettyPrinter

try:
    from urlparse import urlparse
except ImportError:
    from urllib.parse import urlparse

from ax.devops.exceptions import AXScmException
from ax.version import debug

logger = logging.getLogger(__name__)


class AxPrettyPrinter(PrettyPrinter):
    """Pretty printer that does not wrap texts."""

    def _format(self, object, *args):
        if isinstance(object, str):
            width = self._width
            self._width = sys.maxsize
            try:
                super()._format(object, *args)
            finally:
                self._width = width
        else:
            super()._format(object, *args)


class AxEnum(object):
    """Base AX enumeration class."""

    @classmethod
    def names(cls):
        return set([v for v in vars(cls) if not v.startswith('__')])

    @classmethod
    def values(cls):
        return set([vars(cls)[v] for v in vars(cls) if not v.startswith('__')])


class AxArgumentParser(ArgumentParser):
    """Argument parser that does not do system exit when failure to parse."""

    def error(self, message):
        raise ValueError(message)


def utc(t, type=str):
    """Convert timezone-aware time to UTC time.

    :param t: A time string.
    :param type: Str or int.
    :return: UTC time string (ISO format) if type is str else seconds since epoch.
    """
    dt = parse(t)
    dt_utc = dt.astimezone(tzutc())
    if type == str:
        return dt_utc.strftime('%Y-%m-%dT%H:%M:%S')
    else:
        return int(dt_utc.timestamp())


def get_epoch_time_in_ms():
    """Epoch time in milliseconds"""
    return int(time.time() * 1000)


def parse_repo(repo):
    """Parse repo url into 4-tuple (protocol, vendor, repo_owner, repo_name).

    :param repo:
    :return:
    """
    parsed_url = urlparse(repo)
    protocol, vendor = parsed_url.scheme, parsed_url.hostname
    m = re.match(r'/([a-zA-Z0-9\-]+)/([a-zA-Z0-9_.\-/]+)', parsed_url.path)
    if not m:
        raise AXScmException('Illegal repo URL', detail='Illegal repo URL ({})'.format(repo))
    repo_owner, repo_name = m.groups()
    return protocol, vendor, repo_owner, repo_name


def top_k(lists, k=None, key=None):
    """Find top k entries from n generators with sorted entries.

    :param lists: list of generators.
    :param k: if k is none, simply merge all.
    :param key:
    :return:
    """
    heap = []
    counter = 0
    for i in range(len(lists)):
        try:
            value = next(lists[i])
        except StopIteration:
            continue
        else:
            _key = key(value) if key else value
            heapq.heappush(heap, (_key, i, counter, value))
            counter += 1
    top, visited = [], set()
    while heap and (k is None or len(top) < k):
        _, i, _, value = heapq.heappop(heap)
        if not top or value != top[-1]:
            top.append(value)
        try:
            value = next(lists[i])
        except StopIteration:
            continue
        else:
            _key = key(value) if key else value
            heapq.heappush(heap, (_key, i, counter, value))
            counter += 1
    return top


def sort_str_dictionaries(dictionaries, sorters):
    def cmp(x, y):
        for i in range(len(sorters)):
            sorter = sorters[i]
            if sorter.startswith('-'):
                key, descending = sorter[1:], True
            else:
                key, descending = sorter, False
            if x[key] == y[key]:
                continue
            elif x[key] < y[key]:
                return 1 if descending else -1
            else:
                return -1 if descending else 1
        return 0

    return sorted(dictionaries, key=cmp_to_key(cmp))


def get_error_code(error):
    """Get error code from exception

    :param error:
    :returns:
    """
    try:
        return error.response.json().get('code')
    except Exception as e:
        logger.debug('Unable to extract error code from exception: %s', e)
        return


def retry_on_errors(errors, retry=True, caller=None):
    """Retry or not on certain errors

    :param errors: exceptions or error codes
    :param retry: true to retry on error, false to not retry on error
    :param caller: if supplied, print the caller so that user can correlate
    :returns:
    """

    def _retry_on_errors(error):
        error_code = get_error_code(error)
        _retry = not retry
        for i in range(len(errors)):
            if (type(errors[i]) == str and error_code == errors[i]) or (type(errors[i]) != str and isinstance(error, errors[i])):
                _retry = retry
                break
        logger.debug('Captured error (code: %s, caller: %s) to %s', error_code, caller, 'retry' if _retry else 'not retry')
        return _retry

    return _retry_on_errors


def aggregate_numeric_dictionaries(dict1, dict2):
    """Aggregate 2 dictionaries whose values are numeric

    For any key existed in one dictionary but not the other, 0 will be used as default value.

    :param dict1:
    :param dict2:
    :returns:
    """
    for k in dict2:
        if k not in dict1:
            dict1[k] = dict2[k]
        else:
            dict1[k] += dict2[k]
    return dict1


def random_text(n, lower_or_upper=None):
    """Generate a random length-n string

    :param n:
    :param lower_or_upper: none for both, true for lower, false for upper
    :returns:
    """
    if lower_or_upper is None:
        letters = string.ascii_letters
    elif lower_or_upper:
        letters = string.ascii_lowercase
    else:
        letters = string.ascii_uppercase
    return ''.join(random.choice(letters + string.digits) for _ in range(n))


def ax_profiler(functions=None):
    """Ax profiler

    :param functions:
    :returns:
    """

    def wrap(func):
        def wrapped_func(*args, **kwargs):
            if debug:
                profiler = cProfile.Profile()
                profiler.enable(builtins=False)
                result = func(*args, **kwargs)
                profiler.disable()
                stats = profiler.getstats()
                for i in range(len(stats)):
                    for j in range(len(functions)):
                        file_name, func_name = functions[j]
                        if func_name in stats[i].code.co_name and file_name in stats[i].code.co_filename:
                            logger.warning('Invocation of function (%s) spent %s seconds', stats[i].code.co_name, round(stats[i].totaltime, 3))
                            break
                return result
            else:
                return func(*args, **kwargs)

        return wrapped_func

    return wrap
