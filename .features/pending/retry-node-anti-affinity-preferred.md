Description: Soft (preferred) node anti-affinity for retries
Authors: [somaz](https://github.com/somaz94)
Component: General
Issues: 13969

`retryStrategy.affinity.nodeAntiAffinity` now accepts a `type` field controlling how strictly a retry avoids the hosts earlier attempts ran on.
`Required` is the default and keeps the existing behaviour: previously used hosts are excluded outright, which means a retry stays unschedulable once every eligible host has been tried.
`Preferred` only de-prioritises those hosts, so retries keep running even after every eligible host has failed at least once.
Set `nodeAntiAffinity: {type: Preferred}` when your cluster has fewer eligible nodes than the retry limit.
