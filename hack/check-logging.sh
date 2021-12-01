#!/usr/bin/env sh
# This script will return and error if a branch introduces unstructured logging statements, such as:
#
# Errorf/Infof/Warningf/Warnf/Debugf
#
# Unstructured logging is not machine readable, so it is not possible to build reports from it.
#
# Most production system will not be logging at debug level, so why bad Debugf?
#
# * You may change debug logging to info.
# * To encourage best practice.
#
set -eu

from=$(git merge-base --fork-point master)
exitCode=0
for file in $(git diff --name-only "$from" | grep '\.go$' ); do
  git diff "$from" -- "$file" | grep '^+' | grep -c '\(Debug\|Info\|Warn\|Warning\|Error\)f' || exitCode=1
  # https://github.community/t/annotations-how-to-create-them/18387/2
  grep -n '\(Debug\|Info\|Warn\|Warning\|Error\)f' "$file" | cut -f1 -d: | sed "s|\(.*\)|:error file=$file,line=\1,col=0::Infof/Errorf etc are banned. Logging must be structured. Instead, use WithField, WithError, Info, and Error.|"
done

exit $exitCode