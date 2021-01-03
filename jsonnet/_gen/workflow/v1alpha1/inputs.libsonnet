{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='inputs', url='', help='Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another'),
  '#withArtifacts':: d.fn(help='Artifact are a list of artifacts passed as inputs', args=[d.arg(name='artifacts', type=d.T.array)]),
  withArtifacts(artifacts): { artifacts: if std.isArray(v=artifacts) then artifacts else [artifacts] },
  '#withArtifactsMixin':: d.fn(help='Artifact are a list of artifacts passed as inputs\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='artifacts', type=d.T.array)]),
  withArtifactsMixin(artifacts): { artifacts+: if std.isArray(v=artifacts) then artifacts else [artifacts] },
  '#withParameters':: d.fn(help='Parameters are a list of parameters passed as inputs', args=[d.arg(name='parameters', type=d.T.array)]),
  withParameters(parameters): { parameters: if std.isArray(v=parameters) then parameters else [parameters] },
  '#withParametersMixin':: d.fn(help='Parameters are a list of parameters passed as inputs\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='parameters', type=d.T.array)]),
  withParametersMixin(parameters): { parameters+: if std.isArray(v=parameters) then parameters else [parameters] },
  '#mixin': 'ignore',
  mixin: self,
}
