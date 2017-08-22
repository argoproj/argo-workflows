# Copyright 2015-2017 Applatix, Inc. All rights reserved.

INSTANCE_RESOURCE = {
    't2.nano': [1, 0.5*1024],
    't2.micro': [1, 1*1024],
    't2.small': [1, 2*1024],
    't2.medium': [2, 4*1024],
    't2.large': [2, 8*1024],
    'm4.large': [2, 8*1024],
    'm4.xlarge': [4, 16*1024],
    'm4.2xlarge': [8, 32*1024],
    'm4.4xlarge': [16, 64*1024],
    'm4.10xlarge': [32, 128*1024],
    'm4.16xlarge': [64, 256*1024],
    'm3.medium': [1, 3.75*1024],
    'm3.large': [2, 7.5*1024],
    'm3.xlarge': [4, 15*1024],
    'm3.2xlarge': [8, 30*1024],
    'c3.large': [2, 3.75*1024],
    'c3.xlarge': [4, 7.5*1024],
    'c3.2xlarge': [8, 15*1024],
    'c3.4xlarge': [16, 30*1024],
    'c3.8xlarge': [32, 60*1024]
}

AX_EXECUTOR_RESOURCE = [0.1, 100.0]
MINIMUM_RESOURCE_SCALE = 0.7
