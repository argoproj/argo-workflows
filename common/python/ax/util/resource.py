#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from math import ceil
from ax.util.const import KiB, MiB, GiB, TiB, KB, MB, GB, TB

literal_to_var = {
    "Ki": KiB,
    "Mi": MiB,
    "Gi": GiB,
    "Ti": TiB,
    "KiB": KiB,
    "MiB": MiB,
    "GiB": GiB,
    "TiB": TiB,
    "KB": KB,
    "MB": MB,
    "GB": GB,
    "TB": TB,
    "K": KB,
    "M": MB,
    "G": GB,
    "T": TB
}

valid_cpu_units = ["m", "mi", "mili", "milicore"]
valid_storage_units = ["KiB", "MiB", "GiB", "TiB", "KB", "MB", "GB", "TB", "K", "M", "G", "T", "Ki", "Mi", "Gi", "Ti"]


class ResourceValueConverter(object):
    def __init__(self, value, target):
        """
        Convert resource values different unit expressions to the value
        we can use generically

        :param value: resource value, i.e. "100 Mi"
        :param target: "cpu", "memory/mem", or "dick"
        """
        self._value = str(value)
        self._target = target.lower()
        assert self._target in ["cpu", "memory", "mem", "disk", "storage"], "Invalid target {}".format(self._target)
        self._raw = self._get_raw_value()

    @property
    def raw(self):
        """ Get value in base unit (milicore for cpu, byte for storage) """
        return self._raw

    def massage(self, func):
        self._raw = func(self._raw)

    def convert(self, unit):
        """ Get value in given unit (CPU core is rounded to 2 digits, memory is rounded to bytes) """
        if self._target == "cpu":
            return self._convert_cpu_unit(unit)
        else:
            return self._convert_storage_unit(unit)

    def _get_raw_value(self):
        """
        For CPU, raw value is in milicore (m), and for Memory, base value is Byte
        """
        if self._target == "cpu":
            return self._get_cpu_raw()
        else:
            return self._get_storage_raw()

    def _get_cpu_raw(self):
        for u in valid_cpu_units:
            if self._value.endswith(u):
                return int(self._value.strip(u))
        # if CPU does not have a unit, we assume the unit is "core" (1000m)
        return int(float(self._value) * 1000)

    def _get_storage_raw(self):
        assert isinstance(self._value, str)
        for u in valid_storage_units:
            if self._value.endswith(u):
                # Lets round it to bytes
                return int(float(self._value.strip(u)) * literal_to_var[u])
        # if for storage, there is not a unit, we assume MiB for now
        return int(float(self._value) * MiB)

    def _convert_cpu_unit(self, unit="m"):
        if unit in valid_cpu_units:
            return int(self._raw)
        else:
            # We might want to do something such as 1.512 CPU,
            # so we round it to milicore accuracy
            return round(float(self._raw) / 1000, 3)

    def _convert_storage_unit(self, unit="MiB"):
        if unit in valid_storage_units:
            return float(self._raw) / literal_to_var[unit]
        else:
            return -1

