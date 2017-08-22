#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import logging
import random

from multiprocessing.pool import ThreadPool

from ax.platform.applet.appdb import ApplicationRecord
from ax.platform.applet.consts import *
from .utils import generate_random_string

logging.basicConfig(level=logging.DEBUG)


def generate_db_record(num=5):
    record_to_add = []
    record_to_delete = []
    for _ in range(num):
        app_name = "sample-app-{}".format(generate_random_string(digits=True, rand_len=5))
        app_id = "{}".format(generate_random_string(digits=True, ascii_lower=True, rand_len=10))
        pod_name = "sample-pod-{}".format(generate_random_string(digits=True, rand_len=5))
        dep_name = "sample-dep-{}".format(generate_random_string(digits=True, rand_len=5))
        dep_id = "{}".format(generate_random_string(digits=True, ascii_lower=True, rand_len=10))
        container_name = "sample-container-{}".format(generate_random_string(digits=True, rand_len=5))
        container_id = "{}".format(generate_random_string(digits=True, rand_len=36))
        record_to_add.append((CUR_RECORD_VERSION, app_name, app_id, dep_name, dep_id,
                              pod_name, container_name, container_id, None))
        record_to_delete.append((CUR_RECORD_VERSION, app_name, pod_name, container_id))
    return record_to_add, record_to_delete


def test_db_insert_delete(app_record):
    assert isinstance(app_record, ApplicationRecord)
    record_to_add, record_to_delete = generate_db_record(num=random.randint(1, 100))
    app_record.refresh_db_record(to_add=record_to_add)
    record_dict = app_record.load_from_db()

    for key in record_dict:
        app = record_dict[key]["app"]
        aid = record_dict[key]["aid"]
        dep = record_dict[key]["dep"]
        did = record_dict[key]["did"]
        pod = record_dict[key]["pod"]
        for c in record_dict[key]["containers"]:
            record = (CUR_RECORD_VERSION, app, aid, dep, did, pod, c["name"], c["id"], None)
            assert record in record_to_add
            record_to_add.remove(record)
    assert not record_to_add

    app_record.refresh_db_record(to_delete=record_to_delete)
    assert not app_record.load_from_db()


def test_db_get(app_record):
    assert isinstance(app_record, ApplicationRecord)
    record_to_add, record_to_delete = generate_db_record(num=random.randint(1, 100))
    app_record.refresh_db_record(to_add=record_to_add)

    for (v, aname, aid, dname, did, pname, cname, cid, last) in record_to_add:
        assert last == app_record.get_last_done(aname, pname, cid)

    app_record.refresh_db_record(to_delete=record_to_delete)


def test_db_modify(app_record):
    assert isinstance(app_record, ApplicationRecord)
    record_to_add, record_to_delete = generate_db_record(num=random.randint(1, 100))
    app_record.refresh_db_record(to_add=record_to_add)

    for (v, aname, aid, dname, did, pname, cname, cid, last) in record_to_add:
        log = generate_random_string(ascii_lower=True, digits=True, suffix=".log.gz", rand_len=20)
        app_record.record_done_log(aname, pname, cid, log)
        assert log == app_record.get_last_done(aname, pname, cid)

    app_record.refresh_db_record(to_delete=record_to_delete)


def test_insert_parallel(app_record):
    worker_num = random.randint(2, 10)
    records = []
    all_added = []
    pool = ThreadPool(worker_num)
    rst = []

    def _db_insert(add):
        db = ApplicationRecord(db="/tmp/example.db")
        print("Adding {} to db {}\n".format(add, db))
        db.refresh_db_record(to_add=add)

    for _ in range(worker_num):
        to_add, to_delete = generate_db_record(num=random.randint(2, 50))
        records.append((to_add, to_delete))
        all_added.extend(to_add)
        rst.append(pool.apply_async(_db_insert, args=(to_add,)))

    pool.close()
    pool.join()

    record_dict = app_record.load_from_db()
    for key in record_dict:
        app = record_dict[key]["app"]
        aid = record_dict[key]["aid"]
        dep = record_dict[key]["dep"]
        did = record_dict[key]["did"]
        pod = record_dict[key]["pod"]
        for c in record_dict[key]["containers"]:
            record = (CUR_RECORD_VERSION, app, aid, dep, did, pod, c["name"], c["id"], None)
            assert record in all_added
            all_added.remove(record)
    assert not all_added
    
    for (_, to_delete) in records:
        app_record.refresh_db_record(to_delete=to_delete)

    assert not app_record.load_from_db()




