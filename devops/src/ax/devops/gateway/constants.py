from ax.devops.utility.utilities import AxEnum


class ScmTypes(AxEnum):
    """SCM types."""

    GIT = 'git'


class ScmVendors(AxEnum):
    """SCM Vendors."""

    BITBUCKET = 'bitbucket'
    GITHUB = 'github'
    GITLAB = 'gitlab'
    GIT = 'git'
    CODECOMMIT = 'codecommit'


class AxEventTypes(AxEnum):
    """Event types in AX."""

    CREATE = 'create'
    PING = 'ping'
    PUSH = 'push'
    PULL_REQUEST = 'pull_request'
    PULL_REQUEST_MERGE = 'pull_request_merge'


class BitBucketEventTypes(AxEnum):
    """Event types of BitBucket."""

    REPO_PUSH = 'repo:push'
    PULL_REQUEST_CREATED = 'pullrequest:created'
    PULL_REQUEST_UPDATED = 'pullrequest:updated'
    PULL_REQUEST_FULFILLED = 'pullrequest:fulfilled'
    PULL_REQUEST_COMMENT_CREATED = 'pullrequest:comment_created'


class GitHubEventTypes(AxEnum):
    """Event types of GitHub."""

    CREATE = 'create'
    PING = 'ping'
    PUSH = 'push'
    PULL_REQUEST = 'pull_request'
    ISSUE_COMMENT = 'issue_comment'


class GitLabEventTypes(AxEnum):
    """Event types of GitLab."""

    PUSH = 'Push Hook'
    MERGE_REQUEST = 'Merge Request Hook'
    NOTES = 'Note Hook'


class AxCommands(AxEnum):
    """Commands supported by AX."""

    RERUN = 'rerun'
    RUN = 'run'


TYPE_BITBUCKET = ScmVendors.BITBUCKET
TYPE_GITHUB = ScmVendors.GITHUB
TYPE_GITLAB = ScmVendors.GITLAB
TYPE_GIT = ScmVendors.GIT
TYPE_CODECOMMIT = ScmVendors.CODECOMMIT
SUPPORTED_TYPES = {
    ScmVendors.BITBUCKET,
    ScmVendors.GITHUB,
    ScmVendors.GITLAB,
    ScmVendors.GIT,
    ScmVendors.CODECOMMIT
}

DEFAULT_CONCURRENCY = 20
DEFAULT_INTERVAL = 30