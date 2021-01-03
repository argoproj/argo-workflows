{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='executorConfig', url='', help='ExecutorConfig holds configurations of an executor container.'),
  '#withServiceAccountName':: d.fn(help='ServiceAccountName specifies the service account name of the executor container.', args=[d.arg(name='serviceAccountName', type=d.T.string)]),
  withServiceAccountName(serviceAccountName): { serviceAccountName: serviceAccountName },
  '#mixin': 'ignore',
  mixin: self,
}
