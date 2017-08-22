#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-

"""
# https://developer.atlassian.com/jiradev/jira-apis/webhooks#Webhooks-Addingawebhookasapostfunctiontoaworkflow
# {
# 	"timestamp"
#  	"event"
# 	"user": {
# 			   --> See User shape in table below
# 	},
# 	"issue": {
#                --> See Issue shape in table below
# 	},
# 	"changelog" : {
# 			   --> See Changelog shape in table below
# 	},
# 	"comment" : {
# 			   --> See Comment shape in table below
# 	}
# }
# Example changelog
# "changelog": {
#     "items": [
#         {
#              "toString": "A new summary.",
#              "to": null,
#              "fromString": "What is going on here?????",
#              "from": null,
#              "fieldtype": "jira",
#              "field": "summary"
#          },
#          {
#              "toString": "New Feature",
#              "to": "2",
#              "fromString": "Improvement",
#              "from": "4",
#              "fieldtype": "jira",
#              "field": "issuetype"
#          },
#    ],
#    "id": 10124
# }
"""


def translate_jira_issue_event(payload):
    """
    :param payload:
    :return:
    """
    vendor = 'jira'
    type = payload.get('webhookEvent', '')  # 'jira:issue_deleted', 'jira:issue_updated'
    name = payload.get('issue_event_type_name', '')
    id = payload['issue']['key']
    updated_by = payload['user']['displayName']
    project = payload['issue']['fields']['project']['key']
    summary = payload['issue']['fields']['summary']
    description = payload['issue']['fields']['description']
    if description is None:
        description = ''
    status = payload['issue']['fields']['status']['name']
    # 1:undefined /2: new /3: done /4: indeterminate
    status_category_id = payload['issue']['fields']['status']['statusCategory']['id']

    _changelog = payload.get('changelog', None)
    if _changelog:
        changed_fields = [each['field'] for each in _changelog['items']]
    else:
        changed_fields = []

    old_id = None
    if name == 'issue_moved' and 'Key' in changed_fields:
        key_field = [d for d in payload['changelog']['items'] if d['field'] == 'Key']
        old_id = key_field[0]['fromString']

    event = {
        'vendor': vendor,
        'type': type,
        'name': name,
        'id': id,
        'updated_by': updated_by,
        'project': project,
        'summary': summary,
        'description': description,
        'status': status,
        'status_category_id': status_category_id,
        'old_id': old_id,
        'changed_fields': changed_fields,
        'axdb_content': {'id': id,
                         'project': project,
                         'summary': summary,
                         'description': description,
                         'status': status,
                         'old_id': old_id,
                         }
    }
    return event


def get_webhook_whitelist():
    """Get a list of webhook whitelist
    :return: a list
    """
    default_whitelist = ['0.0.0.0/0']
    return default_whitelist

