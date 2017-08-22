from ax.exceptions import AXException


class AXScmException(AXException):
    code = "ERR_AX_SCM"


class UnrecognizableEventType(AXException):
    code = 'ERR_DEVOPS_UNRECOGNIZABLE_EVENT_TYPE'


class UnrecognizableVendor(AXException):
    code = 'ERR_DEVOPS_UNRECOGNIZABLE_VENDOR'


class UnsupportedSCMType(AXException):
    code = 'ERR_DEVOPS_UNSUPPORTED_SCM_TYPE'


class YamlUpdateError(AXException):
    code = 'ERR_DEVOPS_YAML_UPDATE_ERROR'


class UnknownRepository(AXException):
    code = 'ERR_DEVOPS_UNKNOWN_REPOSITORY'


class InvalidCommand(AXException):
    code = 'ERR_DEVOPS_INVALID_COMMAND'


class AXNexusException(AXException):
    code = "ERR_AX_NEXUS"


class AXJFrogException(AXException):
    code = "ERR_AX_JFROG"
