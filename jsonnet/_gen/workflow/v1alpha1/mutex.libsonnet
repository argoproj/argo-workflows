{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='mutex', url='', help='Mutex holds Mutex configuration'),
  '#withName':: d.fn(help='name of the mutex', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
