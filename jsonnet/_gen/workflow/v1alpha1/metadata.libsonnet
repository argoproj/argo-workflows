{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='metadata', url='', help='Pod metdata'),
  '#withAnnotations':: d.fn(help='', args=[d.arg(name='annotations', type=d.T.object)]),
  withAnnotations(annotations): { annotations: annotations },
  '#withAnnotationsMixin':: d.fn(help='\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='annotations', type=d.T.object)]),
  withAnnotationsMixin(annotations): { annotations+: annotations },
  '#withLabels':: d.fn(help='', args=[d.arg(name='labels', type=d.T.object)]),
  withLabels(labels): { labels: labels },
  '#withLabelsMixin':: d.fn(help='\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='labels', type=d.T.object)]),
  withLabelsMixin(labels): { labels+: labels },
  '#mixin': 'ignore',
  mixin: self,
}
