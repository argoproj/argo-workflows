Description: Add a configurable limit for the number of semaphore waiting holders displayed in workflow status
Authors: [Shuangkun Tian](https://github.com/shuangkun)
Component: General
Issues: 15236

When many workflows are waiting on the same semaphore, recording every waiting holder in the workflow status makes the workflow object very large.
This increases memory usage and can degrade performance.

This feature limits the number of waiting holders written to the semaphore status.
It is configurable via the `SEMAPHORE_WAITING_HOLDERS_DISPLAY_LIMIT` environment variable, defaulting to `10`.
Set it to `0` or a negative value to disable the limit and record all holders.

```yaml
env:
  - name: SEMAPHORE_WAITING_HOLDERS_DISPLAY_LIMIT
    value: "5"
```

When the number of waiting holders exceeds the limit, only the first N holders are recorded in the status.
