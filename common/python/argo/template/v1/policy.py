
from .base import BaseTemplate

class PolicyTemplate(BaseTemplate):

    def __init__(self):
        super(PolicyTemplate, self).__init__()
        self.template = None
        self.arguments = {}
        self.notifications = []
        self.when = []


class Notification(object):

    def __init__(self):
        self.whom = []
        self.when = []


class When(object):

    def __init__(self):
        self.event = None
        self.schedule = None
        self.timezone = None
