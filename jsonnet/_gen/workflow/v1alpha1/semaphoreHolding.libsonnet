{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='semaphoreHolding', url='', help=''),
  '#withHolders':: d.fn(help='Holders stores the list of current holder names in the io.argoproj.workflow.v1alpha1.', args=[d.arg(name='holders', type=d.T.array)]),
  withHolders(holders): { holders: if std.isArray(v=holders) then holders else [holders] },
  '#withHoldersMixin':: d.fn(help='Holders stores the list of current holder names in the io.argoproj.workflow.v1alpha1.\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='holders', type=d.T.array)]),
  withHoldersMixin(holders): { holders+: if std.isArray(v=holders) then holders else [holders] },
  '#withSemaphore':: d.fn(help='Semaphore stores the semaphore name.', args=[d.arg(name='semaphore', type=d.T.string)]),
  withSemaphore(semaphore): { semaphore: semaphore },
  '#mixin': 'ignore',
  mixin: self,
}
