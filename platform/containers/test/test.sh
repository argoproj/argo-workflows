#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

set -xe

[ -n "$AX_NAMESPACE" ] || exit 1
[ -n "$AX_VERSION" ] || exit 1
[ -n "$AWS_ACCESS_KEY" ] || exit 1
[ -n "$AWS_SECRET_KEY" ] || exit 1
[ -n "$DNS_SERVER" ] || exit 1
[ -n "$DNS_DOMAIN" ] || exit 1

mkdir -p ~/.aws
cat > ~/.aws/config << EOF
[default]
aws_access_key_id = $AWS_ACCESS_KEY
aws_secret_access_key = $AWS_SECRET_KEY
region = us-west-2
EOF

cat > /tmp/resolv.conf << EOF
search $DNS_DOMAIN
nameserver $DNS_SERVER
EOF

sudo cp /tmp/resolv.conf /etc/resolv.conf

python cloud_aws_test.py
cat nosetests.xml | python -c "from xml.etree import ElementTree; import sys; res=ElementTree.parse(sys.stdin).getroot().attrib; print res; assert int(res['failures'])==0; assert int(res['errors'])==0"
exit $?
