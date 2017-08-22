#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import random

from ax.platform.applet.plm_pool import PodLogManagerPool
from multiprocessing.pool import ThreadPool
from .mock import PodLogManagerMock
from .utils import generate_random_string

import ax.platform.applet.plm_pool
ax.platform.applet.plm_pool.PodLogManager = PodLogManagerMock


def test_add_remove_plm():
    pool = PodLogManagerPool()
    app_name = "test-app-1"
    app_id = "app123456789"
    dep_name = "test-dep-1"
    dep_id = "dep123456789"
    pod_name = "test-pod-1"
    to_add = [("test-container-1", "123456789"), ("test-container-2", "987654321")]

    # Should avoid creating repeated PLMs
    pool.create_or_update_pod_log_manager(app_name, app_id, dep_name, dep_id, pod_name, to_add)
    pool.create_or_update_pod_log_manager(app_name, app_id, dep_name, dep_id, pod_name, to_add)
    containers = pool.get_containers()
    assert {"123456789", "987654321"} == set(containers)

    # Re-delete should be noops
    pool.remove_pod_log_manager(app_name, pod_name)
    pool.remove_pod_log_manager(app_name, pod_name)
    assert not pool.get_containers()
    assert pool.get_plm_number() == 0


def test_update_plm():
    pool = PodLogManagerPool()
    app_name = "test-app-1"
    app_id = "app123456789"
    dep_name = "test-dep-1"
    dep_id = "dep123456789"
    pod_name = "test-pod-1"
    to_add = [("test-container-1", "111111111"), ("test-container-2", "222222222")]

    # Should avoid creating repeated PLMs
    pool.create_or_update_pod_log_manager(app_name, app_id, dep_name, dep_id, pod_name, to_add, None)

    to_add_2 = [("test-container-3", "333333333"), ("test-container-4", "444444444")]
    to_remove = ["111111111"]
    pool.create_or_update_pod_log_manager(app_name, app_id, dep_name, dep_id, pod_name, to_add_2, to_remove)

    containers = pool.get_containers()
    desired_remainings = ["222222222", "333333333", "444444444"]
    assert set(desired_remainings) == set(containers)

    pool.create_or_update_pod_log_manager(app_name, app_id, dep_name, dep_id, pod_name, None, desired_remainings)
    assert not pool.get_containers()

    pool.remove_pod_log_manager(app_name, pod_name)
    assert pool.get_plm_number() == 0


def test_parallel_update():
    pool = PodLogManagerPool()
    app_name = "test-app-1"
    app_id = "app123456789"
    dep_name = "test-dep-1"
    dep_id = "dep123456789"
    pod_name = "test-pod-1"
    container_ids = []

    def _update_pool(plmpool, app_name, app_id, dep_name, dep_id, pod_name, add, delete):
        plmpool.create_or_update_pod_log_manager(app_name, app_id, dep_name, dep_id, pod_name, add, delete)

    trail_num = random.randint(2, 20)
    t_pool = ThreadPool(trail_num)

    for _ in range(trail_num):
        to_add = []
        for _ in range(5):
            cid = generate_random_string(digits=True, rand_len=16)
            to_add.append(("test-container-{}".format(cid), cid))
            container_ids.append(cid)
        t_pool.apply_async(_update_pool, args=(pool, app_name, app_id, dep_name, dep_id, pod_name, to_add, None))

    t_pool.close()
    t_pool.join()
    cid_in_pool = pool.get_containers()
    assert set(container_ids) == set(cid_in_pool)

    pool.remove_pod_log_manager(app_name, pod_name)
    assert not pool.get_containers()
    assert pool.get_plm_number() == 0

