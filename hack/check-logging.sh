#!/usr/bin/env sh
# This script will return an error if the current branch introduces unstructured logging statements, such as:
#
# Errorf/Warningf/Warnf/Infof/Debugf
#
# Unstructured logging is not machine readable, so it is not possible to build reports from it, or efficiently query on it.
#
# Most production system will not be logging at debug level, so why ban Debugf?
#
# * You may change debug logging to info.
# * To encourage best practice.
#
# What to do if really must format your log messages.
#
# I've tried hard to think of any occasion when I'd prefer unstructured logging. I can't think of any times, but here
# might be some edge cases.
#
# As a last resort, use `log.Info(fmt.Sprintf(""))`.

set -eux

from=$(git merge-base --fork-point origin/master)
count=$(git diff "$from" -- '*.go' | grep '^+' | grep -v '\(fmt\|errors\).Errorf' | grep -c '\(Debug\|Info\|Warn\|Warning\|Error\)f' || true)

if [ $count -gt 0 ]; then
  echo 'Errorf/Warningf/Warnf/Infof/Debugf are banned. Use structured logging, e.g. log.WithError(err).Error() or log.WithField().Info().' >&2
  exit 1
fi
