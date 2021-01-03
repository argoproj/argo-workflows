{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='artifactRepositoryRef', url='', help=''),
  '#withConfigMap':: d.fn(help='The name of the config map. Defaults to "artifact-repositories".', args=[d.arg(name='configMap', type=d.T.string)]),
  withConfigMap(configMap): { configMap: configMap },
  '#withKey':: d.fn(help='The config map key. Defaults to the value of the "workflows.argoproj.io/default-artifact-repository" annotation.', args=[d.arg(name='key', type=d.T.string)]),
  withKey(key): { key: key },
  '#mixin': 'ignore',
  mixin: self,
}
