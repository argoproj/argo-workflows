# Slack

This is an example that sends a Slack message.

You must create a secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: slack-executor-plugin
stringData:
  URL: https://hooks.slack.com/services/.../.../...
```