{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='link', url='', help='A link to another app.'),
  '#withName':: d.fn(help='The name of the link, E.g. "Workflow Logs" or "Pod Logs"', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withScope':: d.fn(help='Either "workflow" or "pod"', args=[d.arg(name='scope', type=d.T.string)]),
  withScope(scope): { scope: scope },
  '#withUrl':: d.fn(help='The URL. May contain "${metadata.namespace}", "${metadata.name}", "${status.startedAt}" and "${status.finishedAt}".', args=[d.arg(name='url', type=d.T.string)]),
  withUrl(url): { url: url },
  '#mixin': 'ignore',
  mixin: self,
}
