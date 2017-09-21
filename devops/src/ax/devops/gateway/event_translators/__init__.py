import logging

from ax.devops.ci.constants import ScmVendors
from .bitbucket import BitBucketEventTranslator
from .github import GitHubEventTranslator
from .gitlab import GitLabEventTranslator
from ax.devops.exceptions import UnrecognizableVendor

logger = logging.getLogger(__name__)


class EventTranslator(object):
    """Event translator."""

    TRANSLATORS = {
        ScmVendors.BITBUCKET: BitBucketEventTranslator,
        ScmVendors.GITHUB: GitHubEventTranslator,
        ScmVendors.GITLAB: GitLabEventTranslator
    }

    @classmethod
    def translate(cls, payload, headers=None):
        """Translate an SCM event into AX DevOps event.

        :param payload:
        :param headers:
        :return:
        """
        vendor = cls.detect_vendor(headers)
        translator = cls.TRANSLATORS[vendor]
        return translator.translate(payload, headers)

    @classmethod
    def detect_vendor(cls, headers):
        """Detect the vendor based on event content.

        :param headers:
        :return:
        """
        if headers.get('User-Agent', '').startswith('Bitbucket'):
            return ScmVendors.BITBUCKET
        elif headers.get('User-Agent', '').startswith('GitHub'):
            return ScmVendors.GITHUB
        elif headers.get('X-Gitlab-Event') is not None:
            return ScmVendors.GITLAB
        else:
            raise UnrecognizableVendor('Unrecognizable vendor')
