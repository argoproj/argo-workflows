Component: Argo Server
Issues: 13114 
Description: Support open custom links in new tab automatically.
Author: [Shuangkun Tian](https://github.com/shuangkun)

Support configuring a custom link to open in a new tab by default.
If target == _blank, open in new tab, if target is null or _self, open in this tab. For example:
```
    - name: Pod Link
      scope: pod
      target: _blank
      url: http://logging-facility?namespace=${metadata.namespace}&podName=${metadata.name}&startedAt=${status.startedAt}&finishedAt=${status.finishedAt}
```

