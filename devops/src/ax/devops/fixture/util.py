"""
Utility functions to support fixturemanager
"""
import json
import logging
import re
import threading

from ax.exceptions import AXIllegalArgumentException

logger = logging.getLogger(__name__)

ATTRIBUTE_PARAM_REGEX = re.compile(r"%%fixture\.(\w+)%%")

class TimerThread(threading.Thread):
    """Thread subclass which will run the target at periodic intervals, until stopped or exception is hit"""

    def __init__(self, interval, ignore_errors=False, *args, **kwargs):
        threading.Thread.__init__(self, *args, **kwargs)
        self.stopped = threading.Event()
        self.interval = interval
        self.ignore_errors = ignore_errors

    def run(self):
        while not self.stopped.wait(self.interval):
            try:
                self._target(*self._args, **self._kwargs)
            except Exception:
                if self.ignore_errors:
                    logger.exception("%s had error. Error ignored", self.name)
                else:
                    logger.exception("%s had error. Timer thread terminating", self.name)
                    raise

# returns a pretty formatted json string. Use this in place of pprint.pformat so that log output prints valid json
pretty_json = lambda doc: json.dumps(doc, sort_keys=True, indent=4, separators=(',', ': '))

def substitute_attributes(value, instance):
    """Substitutes occurrences %%attributes.attr_name%% with instance attributes"""
    for attr_name, attr_val in instance.attributes.items():
        value = value.replace("%%attributes.{}%%".format(attr_name), str(attr_val))
    for field in ['id', 'name', 'status']:
        value = value.replace("%%attributes.{}%%".format(field), getattr(instance, field))
    value = value.replace("%%attributes.class%%", instance.class_name)
    remaining = re.search(r"(%%attributes\.(\w+)%%)", value)
    if remaining:
        raise AXIllegalArgumentException("Template parameters had unresolvable attributes: {}".format(remaining.group(1)))
    return value

def new_status_detail(code, message, detail=""):
    """Convenience function to return a new status detail struct"""
    return {
        "code": code,
        "message": message,
        "detail": detail
    }

def humanize_error(errmsg):
    """Massage voluptuous' error to be friendlier to the end user"""
    match = re.search(r"(.*) for dictionary value @ data\['(.*)']", errmsg)
    if match:
        return "Invalid value for attribute '{}': {}".format(match.group(2), match.group(1))
    match = re.search(r"(.*) @ data\['(.*)']", errmsg)
    if match:
        return "{}: {}".format(match.group(1), match.group(2))
    return errmsg

def generate(item_generator, fields=None):
    """
    A lagging generator to stream JSON so we don't have to hold everything in memory
    This is a little tricky, as we need to omit the last comma to make valid JSON.
    See: https://blog.al4.co.nz/2016/01/streaming-json-with-flask/
    Assumes the presence of .json() method in the item
    """
    try:
        prev_item = next(item_generator)  # get first result
    except StopIteration:
        # StopIteration here means the length was zero, so yield a valid, empty doc and stop
        yield '{"data": []}'
        raise StopIteration
    # We have some data. First, yield the opening json
    yield '{"data": ['
    # Iterate over the releases
    for item in item_generator:
        yield json.dumps(prev_item.json(fields=fields)) + ', '
        prev_item = item
    # Now yield the last iteration without comma but with the closing brackets
    yield json.dumps(prev_item.json(fields=fields)) + ']}'
