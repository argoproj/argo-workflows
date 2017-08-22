#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#


class AxdbConstants(object):
    """Definition of the constants for AXDB component."""

    TableTypeTimeSeries = 0
    TableTypeKeyValue = 1
    TableTypeTimedKeyValue = 2
    TableTypeCounter = 3

    ColumnTypeString = 0
    ColumnTypeDouble = 1
    ColumnTypeInteger = 2
    ColumnTypeBoolean = 3
    ColumnTypeArray = 4
    ColumnTypeMap = 5
    # ColumnTypeTimestamp = 6
    ColumnTypeUUID = 7
    ColumnTypeOrderedMap = 8
    ColumnTypeTimeUUID = 9
    ColumnTypeSet = 10
    ColumnTypeCounter = 11

    ColumnIndexNone = 0
    ColumnIndexStrong = 1
    ColumnIndexWeak = 2
    ColumnIndexClustering = 3
    ColumnIndexPartition = 4

    TaskStatusWaiting = 'WAITING'
    TaskStatusRunning = 'RUNNING'
    TaskStatusComplete = 'COMPLETE'

    TaskResultSuccess = 'SUCCESS'
    TaskResultFailure = 'FAILURE'
    TaskResultCancelled = 'CANCELLED'
    TaskResultSkipped = 'SKIPPED'

    AXDBQueryMaxTime = "ax_max_time"
    AXDBQueryMinTime = "ax_min_time"
    AXDBQueryMaxEntries = "ax_max_entries"
    AXDBQueryOrderByASC = "ax_orderby_asc"
    AXDBQueryOrderByDESC = "ax_orderby_desc"
    AXDBQuerySessionID = "ax_session_id"
    AXDBQuerySrcInterval = "ax_src_interval"
