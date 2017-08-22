from django.conf.urls import url, include

from rest_framework import routers

from gateway.views import hello_world, resource_not_found
from axjira.api import events, JiraIssueViewSet, JiraIssueTypeViewSet, JiraProjectViewSet, JiraUserViewSet, JiraWebhookViewSet
from result.api import ResultViewSet
from scm.api import branches, commit, commits, files, SCMViewSet


class AXRouter(routers.DefaultRouter):
    def get_urls(self):
        urls_static = [
            url(r'^$', hello_world),
            url(r'^jira/events$', events),
            url(r'^scm/branches$', branches),
            url(r'^scm/commits$', commits),
            url(r'^scm/commits/(?P<pk>[a-z0-9]+)$', commit),
            url(r'^scm/files$', files)
        ]
        urls_dynamic = super(AXRouter, self).get_urls()
        return urls_static + urls_dynamic


router_v1 = AXRouter(trailing_slash=False)
router_v1.register(r'jira/issues', JiraIssueViewSet, base_name='jira-issue')
router_v1.register(r'jira/issuetypes', JiraIssueTypeViewSet, base_name='jira-issuetype')
router_v1.register(r'jira/projects', JiraProjectViewSet, base_name='jira-project')
router_v1.register(r'jira/users', JiraUserViewSet, base_name='jira-user')
router_v1.register(r'jira/webhooks', JiraWebhookViewSet, base_name='jira-webhook')
router_v1.register(r'results', ResultViewSet)
router_v1.register(r'scm', SCMViewSet)

urlpatterns = [
    url(r'^$', hello_world),
    url(r'^v1/', include(router_v1.urls, namespace='v1')),
    url(r'^.*$', resource_not_found)
]
