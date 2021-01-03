{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='logEntry', url='', help=''),
  '#withContent':: d.fn(help='', args=[d.arg(name='content', type=d.T.string)]),
  withContent(content): { content: content },
  '#withPodName':: d.fn(help='', args=[d.arg(name='podName', type=d.T.string)]),
  withPodName(podName): { podName: podName },
  '#mixin': 'ignore',
  mixin: self,
}
