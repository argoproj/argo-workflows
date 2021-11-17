# Slack

This is an example that sends a Slack message.

You must create a secret named `slack-executor-plugin` with a single `stringData` field for the `url`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: slack-executor-plugin
stringData:
  url: https://hooks.slack.com/services/.../.../...
```