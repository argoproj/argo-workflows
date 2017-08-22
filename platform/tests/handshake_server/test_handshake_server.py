#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import os
import sys
import subprocess
import random
import string
import shlex
from multiprocessing.pool import ThreadPool

from .conftest import TEST_SOCK

PWD = os.path.dirname(__file__)
if sys.platform == "darwin":
    bin_name = "handshake_mac"
else:
    # assume linux
    bin_name = "handshake"
hs_cli = os.path.join(PWD, bin_name)
download_cmd = "bash -c 'wget -P {pwd} https://s3-us-west-1.amazonaws.com/ax-public/static_tools/ax_handshake_ax/{bin_name} && chmod +x {hs_cli}'".format(pwd=PWD, hs_cli=hs_cli, bin_name=bin_name)
subprocess.check_call(shlex.split(download_cmd))

def _do_handshake():
    msg_len = random.randint(1, 511)
    msg = ''.join(random.SystemRandom().choice(string.ascii_uppercase + string.digits + string.ascii_lowercase)
                  for _ in range(msg_len))
    output_raw = subprocess.check_output([hs_cli, TEST_SOCK, msg, str(len(msg))])
    output = output_raw.decode("utf-8")
    if msg != output:
        print("Sent != Received. Raw: \"{}\"\nOutput: \"{}\"\nSent: \"{}\"".format(output_raw, output, msg))
    return msg == output


def test_single_handshake(handshake):
    for _ in range(20):
        assert _do_handshake()


def test_parallel_handshake_10(handshake):
    conn_cnt = 10
    pool = ThreadPool(conn_cnt)
    rst = []

    for _ in range(conn_cnt):
        rst.append(pool.apply_async(_do_handshake))

    pool.close()
    pool.join()
    successful = 0
    for r in rst:
        if r.get():
            successful += 1
    print("{} out of {} successful".format(successful, conn_cnt))
    assert successful == conn_cnt


def test_parallel_handshake_50(handshake):
    conn_cnt = 50
    pool = ThreadPool(conn_cnt)
    rst = []

    for _ in range(conn_cnt):
        rst.append(pool.apply_async(_do_handshake))

    pool.close()
    pool.join()

    successful = 0
    for r in rst:
        if r.get():
            successful += 1
    print("{} out of {} successful".format(successful, conn_cnt))
    assert successful == conn_cnt


def test_parallel_handshake_120(handshake):
    conn_cnt = 120
    pool = ThreadPool(conn_cnt)
    rst = []

    for _ in range(conn_cnt):
        rst.append(pool.apply_async(_do_handshake))

    pool.close()
    pool.join()

    successful = 0
    for r in rst:
        if r.get():
            successful += 1
    print("{} out of {} successful".format(successful, conn_cnt))
    assert successful == conn_cnt
