{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='ttlStrategy', url='', help='TTLStrategy is the strategy for the time to live depending on if the workflow succeeded or failed'),
  '#withSecondsAfterCompletion':: d.fn(help='SecondsAfterCompletion is the number of seconds to live after completion', args=[d.arg(name='secondsAfterCompletion', type=d.T.integer)]),
  withSecondsAfterCompletion(secondsAfterCompletion): { secondsAfterCompletion: secondsAfterCompletion },
  '#withSecondsAfterFailure':: d.fn(help='SecondsAfterFailure is the number of seconds to live after failure', args=[d.arg(name='secondsAfterFailure', type=d.T.integer)]),
  withSecondsAfterFailure(secondsAfterFailure): { secondsAfterFailure: secondsAfterFailure },
  '#withSecondsAfterSuccess':: d.fn(help='SecondsAfterSuccess is the number of seconds to live after success', args=[d.arg(name='secondsAfterSuccess', type=d.T.integer)]),
  withSecondsAfterSuccess(secondsAfterSuccess): { secondsAfterSuccess: secondsAfterSuccess },
  '#mixin': 'ignore',
  mixin: self,
}
