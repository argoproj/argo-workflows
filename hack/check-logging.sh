#!/usr/bin/env sh
set -eu

echoError() {
  // https://github.community/t/annotations-how-to-create-them/18387/2
  echo "::error file=$1,line=$2,col=$3::$4"
}

from=$(git merge-base --fork-point master)
exitCode=0
for file in $(git diff --name-only "$from" | grep '\.go$' ); do
  git diff "$from" -- "$file" | grep '^+' | grep -c '\(Debug\|Info\|Warn\|Warning\|Error\)f' || exitCode=1
  grep -n '\(Debug\|Info\|Warn\|Warning\|Error\)f' "$file" | cut -f1 -d: | sed "s|\(.*\)|:error file=$file,line=\1,col=0::Infof/Errorf etc are banned. Logging must be structured. Instead, use WithField, WithError, Info, and Error.|"
done

exit $exitCode