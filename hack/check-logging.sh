#!/usr/bin/env sh
# This script will return an error if the current branch introduces unstructured logging statements, such as:
#
# Errorf/Infof/Warningf/Warnf/Debugf
#
# Unstructured logging is not machine readable, so it is not possible to build reports from it, or efficiently query on it.
#
# Most production system will not be logging at debug level, so why bad Debugf?
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

set -eu

from=$(git merge-base --fork-point master)
exitCode=0
exitCode=$(git diff "$from" | grep '^+' | grep -v '\(fmt\|errors\).Errorf' | grep -c '\(Debug\|Info\|Warn\|Warning\|Error\)f' || echo 0)
if [ $exitCode -gt 0 ]; then
  for file in $(git diff --name-only "$from" | grep '\.go$' ); do
    # https://github.community/t/annotations-how-to-create-them/18387/2
    grep -n '\(Debug\|Info\|Warn\|Warning\|Error\)f' "$file" | grep -v '\(fmt\|errors\).Errorf' | cut -f1 -d: | sed "s|\(.*\)|:error file=$file,line=\1,col=0::Infof/Errorf etc are banned. Logging must be structured. Instead, use WithField, WithError, Info, and Error.|" >&2 || true
  done
fi

echo $exitCode
exit $exitCode