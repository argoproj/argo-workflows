#!/usr/bin/env bash
set -eu -o pipefail

TAG_NAME=$1
FILE=$2
NAME=$(basename "$FILE")

ASSET_URL=$(curl -fs -u"$GITHUB_TOKEN:" https://api.github.com/repos/argoproj/argo/releases | jq -r "map(select(.tag_name==\"$TAG_NAME\")) | .[0] | .assets | map(select(.name=\"$NAME\")) | .[0] | .url")

if [ "$ASSET_URL" != "null" ]; then
  echo "deleting existing asset $NAME"
  curl -fs -u"$GITHUB_TOKEN:" "$ASSET_URL" -XDELETE
fi

UPLOAD_URL=$(curl -fs -u"$GITHUB_TOKEN:" https://api.github.com/repos/argoproj/argo/releases | jq -r "map(select(.tag_name==\"$TAG_NAME\")) | map(.upload_url) | .[0]" | sed 's/{.*//')

echo "uploading $(du -sh "$FILE")"

curl \
    -f \
    --progress-bar \
    -u "$GITHUB_TOKEN:" \
    -H "Content-Type: application/octet-stream" \
    --data-binary @"$FILE" \
    "$UPLOAD_URL?name=$NAME" \
    -o /dev/null
