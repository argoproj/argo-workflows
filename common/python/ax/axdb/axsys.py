#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

from ax.axdb.axdb import AxdbConstants

config_table = {
    "AppName": "axsys",
    "Name": "config",
    "Type": AxdbConstants.TableTypeKeyValue,
    "Columns": {
        "key": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexPartition},
        "value": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone}
    }
}
config_table_path = "axsys/config"

host_table = \
 {"AppName": "axsys", "Name":"host", "Type": AxdbConstants.TableTypeKeyValue, "Columns":
  {"hostid": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexPartition},
   "name": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "status": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "private_ip": {"Type": AxdbConstants.ColumnTypeArray, "Index": AxdbConstants.ColumnIndexNone},
   "public_ip": {"Type": AxdbConstants.ColumnTypeArray, "Index": AxdbConstants.ColumnIndexNone},
   "mem": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "cpu": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "ecu": {"Type": AxdbConstants.ColumnTypeDouble, "Index": AxdbConstants.ColumnIndexNone},
   "disk": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "model": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "network": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   }
  }
host_table_path = "axsys/host"

# Common container info table for both live and historic container data
container_columns = {
    "hostid": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexPartition},
    "containerid": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexClustering},
    "serviceid": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
    "costid": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
    "name": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
    "type": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
    "mem": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
    "cpu": {"Type": AxdbConstants.ColumnTypeDouble, "Index": AxdbConstants.ColumnIndexNone}
}

container_table = {
    "AppName": "axsys", "Name":"container", "Type": AxdbConstants.TableTypeKeyValue, "Columns": container_columns
}
container_table_path = "axsys/container"

container_history_table = {
    "AppName": "axsys", "Name":"container_history", "Type": AxdbConstants.TableTypeKeyValue, "Columns": container_columns
}
container_history_table_path = "axsys/container_history"

host_usage_table = \
 {"AppName": "axsys", "Name":"host_usage", "Type": AxdbConstants.TableTypeTimeSeries, "Columns":
  {"hostid": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexPartition},
   "hostname": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexStrong},
   "cpu": {"Type": AxdbConstants.ColumnTypeDouble, "Index": AxdbConstants.ColumnIndexNone},
   "cpu_total": {"Type": AxdbConstants.ColumnTypeDouble, "Index": AxdbConstants.ColumnIndexNone},
   "cpu_percent": {"Type": AxdbConstants.ColumnTypeDouble, "Index": AxdbConstants.ColumnIndexNone},
   "mem": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "mem_percent": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "network": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "disk": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "io": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   },
  "Stats": {"cpu": 1}
  }
host_usage_table_path = "axsys/host_usage"

container_usage_table = \
 {"AppName": "axsys", "Name":"container_usage", "Type": AxdbConstants.TableTypeTimeSeries, "Columns":
  {"hostid": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexPartition},
   "containerid": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexPartition},
   "containername": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexStrong},
   "cpu": {"Type": AxdbConstants.ColumnTypeDouble, "Index": AxdbConstants.ColumnIndexNone},
   "cpu_total": {"Type": AxdbConstants.ColumnTypeDouble, "Index": AxdbConstants.ColumnIndexNone},
   "cpu_percent": {"Type": AxdbConstants.ColumnTypeDouble, "Index": AxdbConstants.ColumnIndexNone},
   "mem": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "mem_percent": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "network": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "disk": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "io": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   },
  "Stats": {"cpu": 1}
  }
container_usage_table_path = "axsys/container_usage"

artifacts_table = \
 {"AppName": "axsys", "Name":"artifacts", "Type": AxdbConstants.TableTypeTimeSeries, "Columns":
  {"artifact_id": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexPartition},
   "service_instance_id": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexStrong},
   "name": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "is_alias": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "description": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "src_path": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "src_name": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "excludes": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "storage_method": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "storage_path": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "inline_storage": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "compression_mode": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "symlink_mode": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "archive_mode": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "num_byte": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "num_file": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "num_dir": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "num_symlink": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "num_other": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "num_skip_byte": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "num_skip": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "stored_byte": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   "meta": {"Type": AxdbConstants.ColumnTypeString, "Index": AxdbConstants.ColumnIndexNone},
   "timestamp": {"Type": AxdbConstants.ColumnTypeInteger, "Index": AxdbConstants.ColumnIndexNone},
   }
  }
artifacts_table_path = "axsys/artifacts"
