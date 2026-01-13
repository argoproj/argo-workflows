Description: Add configurable limit for semaphore waiting holders display to prevent workflow YAML from becoming too large
Authors: [Shuangkun Tian](https://github.com/shuangkun)
Component: General
Issues: 15236

This feature adds a configurable limit for the number of semaphore waiting holders displayed in the workflow status.
When many workflows are waiting for the same semaphore, displaying all waiting holders causes the workflow YAML to become very large.
Without a limit, the workflow YAML can grow significantly, leading to increased memory usage and potential performance issues.
This feature allows administrators to limit the number of waiting holders displayed in the semaphore status to prevent workflow YAML from becoming too large.

### Configuration

The limit can be configured via the `SEMAPHORE_WAITING_HOLDERS_DISPLAY_LIMIT` environment variable.
The default value is `10`.
Set to `0` or a negative value to disable the limit and show all holders.

### Example

To configure the limit to display only 5 waiting holders:

```yaml
env:
  - name: SEMAPHORE_WAITING_HOLDERS_DISPLAY_LIMIT
    value: "5"
```

When the number of waiting holders exceeds the configured limit, only the first N holders (where N is the limit) will be displayed in the workflow status.
This prevents the workflow YAML from becoming too large and helps reduce memory consumption in environments with many concurrent workflows waiting for semaphores.
