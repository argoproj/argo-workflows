Component: General
Issues: 13114
Description: Fast cache workflows to avoid reconciling outdated objects.
Author: [Shuangkun Tian](https://github.com/shuangkun)

Use a thread-safe cache.Store to cache the latest workflow. On read, compare fast-cache and informer resourceVersion and use the newer. Removed legacy resourceVersion-only tracking.
