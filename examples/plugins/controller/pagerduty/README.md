# Pager Duty

This is an example that sends a Pager Duty event.

You must create a secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pagerduty-controller-plugin
stringData:
  url: https://hooks.slack.com/services/.../.../...
```