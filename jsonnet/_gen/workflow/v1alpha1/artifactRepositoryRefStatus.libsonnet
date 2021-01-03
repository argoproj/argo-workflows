{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='artifactRepositoryRefStatus', url='', help=''),
  '#withConfigMap':: d.fn(help='The name of the config map. Defaults to "artifact-repositories".', args=[d.arg(name='configMap', type=d.T.string)]),
  withConfigMap(configMap): { configMap: configMap },
  '#withDefault':: d.fn(help='If this ref represents the default artifact repository, rather than a config map.', args=[d.arg(name='default', type=d.T.boolean)]),
  withDefault(default): { default: default },
  '#withKey':: d.fn(help='The config map key. Defaults to the value of the "workflows.argoproj.io/default-artifact-repository" annotation.', args=[d.arg(name='key', type=d.T.string)]),
  withKey(key): { key: key },
  '#withNamespace':: d.fn(help="The namespace of the config map. Defaults to the workflow's namespace, or the controller's namespace (if found).", args=[d.arg(name='namespace', type=d.T.string)]),
  withNamespace(namespace): { namespace: namespace },
  '#mixin': 'ignore',
  mixin: self,
}
