import logging
import time
from ax.devops.axdb.axops_client import AxopsClient

logger = logging.getLogger(__name__)


class MockAxopsClient(AxopsClient):
    """Fake AxOps client"""

    def __init__(self, *args, **kwargs):
        super(MockAxopsClient, self).__init__(*args, **kwargs)

    def ping(self):
        return True

    def get_policy(self):
        return \
            [
                {
                    "id": "47efd907-d427-5d72-5a16-b870ad8690e2",
                    "name": "Argo CI Policy",
                    "description": "Policy to trigger build for all events",
                    "repo": "https://repo.org/company/prod.git",
                    "branch": "master",
                    "template": "Argo CI",
                    "enabled": True,
                    "notifications": [
                        {
                            "whom": [
                                "committer",
                                "author"
                            ],
                            "when": [
                                "on_start",
                                "on_success",
                                "on_failure"
                            ]
                        },
                        {
                            "whom": [
                                "channelhashid@company.slack.com"
                            ],
                            "when": [
                                "on_failure"
                            ]
                        }
                    ],
                    "when": [
                        {
                            "event": "on_push",
                            "target_branches": [
                                "master"
                            ]
                        },
                        {
                            "event": "on_pull_request",
                            "target_branches": [
                                "master"
                            ]
                        },
                        {
                            "event": "on_pull_request_merge",
                            "target_branches": [
                                "master"
                            ]
                        },
                        {
                            "event": "on_cron",
                            "target_branches": [
                                "master"
                            ],
                            "schedule": "* * * * *",
                            "timezone": "UTC"
                        },
                        {
                            "event": "on_cron",
                            "target_branches": [
                                ".*", "master"
                            ],
                            "schedule": "0 0 * * *",
                            "timezone": "US/Pacific"
                        }
                    ],
                    "parameters": {
                        "namespace": "axsys",
                        "version": "staging"
                    }
                }
            ]
