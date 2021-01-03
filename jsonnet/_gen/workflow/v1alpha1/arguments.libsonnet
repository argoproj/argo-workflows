{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='arguments', url='', help='Arguments to a template'),
  '#withArtifacts':: d.fn(help='Artifacts is the list of artifacts to pass to the template or workflow', args=[d.arg(name='artifacts', type=d.T.array)]),
  withArtifacts(artifacts): { artifacts: if std.isArray(v=artifacts) then artifacts else [artifacts] },
  '#withArtifactsMixin':: d.fn(help='Artifacts is the list of artifacts to pass to the template or workflow\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='artifacts', type=d.T.array)]),
  withArtifactsMixin(artifacts): { artifacts+: if std.isArray(v=artifacts) then artifacts else [artifacts] },
  '#withParameters':: d.fn(help='Parameters is the list of parameters to pass to the template or workflow', args=[d.arg(name='parameters', type=d.T.array)]),
  withParameters(parameters): { parameters: if std.isArray(v=parameters) then parameters else [parameters] },
  '#withParametersMixin':: d.fn(help='Parameters is the list of parameters to pass to the template or workflow\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='parameters', type=d.T.array)]),
  withParametersMixin(parameters): { parameters+: if std.isArray(v=parameters) then parameters else [parameters] },
  '#mixin': 'ignore',
  mixin: self,
}
