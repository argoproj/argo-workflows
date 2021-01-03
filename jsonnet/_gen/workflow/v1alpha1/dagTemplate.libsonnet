{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='dagTemplate', url='', help='DAGTemplate is a template subtype for directed acyclic graph templates'),
  '#withFailFast':: d.fn(help='This flag is for DAG logic. The DAG logic has a built-in "fail fast" feature to stop scheduling new steps, as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completed before failing the DAG itself. The FailFast flag default is true,  if set to false, it will allow a DAG to run all branches of the DAG to completion (either success or failure), regardless of the failed outcomes of branches in the DAG. More info and example about this feature at https://github.com/argoproj/argo/issues/1442', args=[d.arg(name='failFast', type=d.T.boolean)]),
  withFailFast(failFast): { failFast: failFast },
  '#withTarget':: d.fn(help='Target are one or more names of targets to execute in a DAG', args=[d.arg(name='target', type=d.T.string)]),
  withTarget(target): { target: target },
  '#withTasks':: d.fn(help='Tasks are a list of DAG tasks', args=[d.arg(name='tasks', type=d.T.array)]),
  withTasks(tasks): { tasks: if std.isArray(v=tasks) then tasks else [tasks] },
  '#withTasksMixin':: d.fn(help='Tasks are a list of DAG tasks\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='tasks', type=d.T.array)]),
  withTasksMixin(tasks): { tasks+: if std.isArray(v=tasks) then tasks else [tasks] },
  '#mixin': 'ignore',
  mixin: self,
}
