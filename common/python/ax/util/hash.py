#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import hashlib
import uuid

from hashids import Hashids

DEFAULT_ALPHABET= "abcdefghijklmnopqrstuvwxyz"

def generate_hash(data, alphabet=DEFAULT_ALPHABET):
    """
    Currently this uses md5 to generate a 32 byte string. This
    a 256 bit value and then use hashid to convert this to a hash using
    the alphabet above
    Args:
        data: string
        alphabet: the alphabet to use for the hash
    """
    hashids = Hashids(alphabet=alphabet)
    md5 = hashlib.md5(data)
    return hashids.encode_hex(md5.hexdigest())

