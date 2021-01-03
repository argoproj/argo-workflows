{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='rawArtifact', url='', help='RawArtifact allows raw string content to be placed as an artifact in a container'),
  '#withData':: d.fn(help='Data is the string contents of the artifact', args=[d.arg(name='data', type=d.T.string)]),
  withData(data): { data: data },
  '#mixin': 'ignore',
  mixin: self,
}
