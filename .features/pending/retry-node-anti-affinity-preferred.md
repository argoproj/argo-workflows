Description: Soft (preferred) node anti-affinity for retries
Authors: [somaz](https://github.com/somaz94)
Component: General
Issues: 13969

`retryStrategy.affinity.nodeAntiAffinity` now accepts a `type` field controlling how strictly a retry avoids the hosts earlier attempts ran on.
`Required` is the default and keeps the existing behaviour: previously used hosts are excluded outright, which means a retry stays unschedulable once every eligible host has been tried.
`Preferred` only de-prioritises those hosts, so a retry can still be scheduled onto a previously tried host once every eligible host has been tried, provided no other scheduling constraint blocks it.
Set `nodeAntiAffinity: {type: Preferred}` when your cluster has fewer eligible nodes than the retry limit.
