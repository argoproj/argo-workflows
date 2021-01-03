{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='event', url='', help=''),
  '#withSelector':: d.fn(help='Selector (https://github.com/antonmedv/expr) that we must must match the io.argoproj.workflow.v1alpha1. E.g. `payload.message == "test"`', args=[d.arg(name='selector', type=d.T.string)]),
  withSelector(selector): { selector: selector },
  '#mixin': 'ignore',
  mixin: self,
}
