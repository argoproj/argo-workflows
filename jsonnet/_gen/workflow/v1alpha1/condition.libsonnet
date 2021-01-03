{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='condition', url='', help=''),
  '#withMessage':: d.fn(help='Message is the condition message', args=[d.arg(name='message', type=d.T.string)]),
  withMessage(message): { message: message },
  '#withType':: d.fn(help='Type is the type of condition', args=[d.arg(name='type', type=d.T.string)]),
  withType(type): { type: type },
  '#mixin': 'ignore',
  mixin: self,
}
