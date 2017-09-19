"""
Kubernetes REST API client built on top of generated swagger client

The client is intended to be used both inside and outside a Kubernetes cluster
"""
import json
import logging
import os
import re
import ssl
import sys

from ax.cloud import Cloud
from ax.exceptions import AXKubeApiException, AXNotFoundException, AXConflictException
from ax.kubernetes.ax_kube_dict import KUBE_NO_NAMESPACE_SET
from ax.util.singleton import Singleton
from future.utils import with_metaclass
import requests
from retrying import retry
import urllib3
import websocket
import yaml

from . import swagger_client
from .swagger_client.rest import ApiException


try:
    from urllib.parse import urlencode
except ImportError:
    from urllib import urlencode



logger = logging.getLogger(__name__)

# These paths will be mounted in all Kubernetes pods
# See: http://kubernetes.io/docs/user-guide/accessing-the-cluster/#accessing-the-api-from-a-pod
KUBE_CACRT_HOST = 'kubernetes.default'
KUBE_CACRT_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
KUBE_TOKEN_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/token"
API_V1_BASE = "/api/v1/"
API_EXT_BASE = "/apis/extensions/v1beta1/"
API_AUTO_SCALE_BASE = "/apis/autoscaling/v1/"
API_BATCH_BASE = "/apis/batch/v1/"
DEFAULT_KUBECONFIG = "~/.kube/config"
py3env = sys.version_info[0] >= 3


class GCPToken(with_metaclass(Singleton, object)):
    """
    Simple helper to get GCP token.
    Make it singleton to cache token and to save number of API calls.
    """
    # TODO: This is not needed after we upgrade kubernetes client.
    def __init__(self):
        self._cred = None

    @property
    def token(self):
        """
        Get oauth token from ADC.
        ADC must be set before calling this.
        https://developers.google.com/identity/protocols/application-default-credentials
        """
        from google.auth import default as default_credential
        from google.cloud.storage import Client as gcs

        project = None
        if self._cred is None:
            self._cred, project = default_credential()
        if self._cred.expiry is None:
            # Token is not refreshed unless we make a real call.
            # List buckets to force refresh token.
            for b in gcs(credentials=self._cred, project=project).list_buckets():
                logger.debug("bucket %s", b)
        # It is possible that token was still valid and expires in less than 10 minutes.
        # Platform update will generally take longer than this.
        # This is not handled right now.
        return self._cred.token


def parse_kubernetes_exception(func):
    def swagger_exception_handler(*args, **kwargs):
        try:
            return func(*args, **kwargs)
        except ApiException as e:
            if e.status == 404:
                raise AXNotFoundException(e.reason, detail=e.body)
            elif e.status == 409:
                raise AXConflictException(e.reason, detail=e.body)
            raise AXKubeApiException(e.reason, detail=e.body)

    return swagger_exception_handler


def retry_exp_not_apiexception(func):

    def raise_apiexception_else_retry(e):
        if isinstance(e, swagger_client.rest.ApiException):
            return False
        return True

    @parse_kubernetes_exception
    @retry(retry_on_exception=raise_apiexception_else_retry,
           wait_exponential_multiplier=100,
           stop_max_attempt_number=10)
    def wrapped_func(*args, **kwargs):
        return func(*args, **kwargs)

    return wrapped_func


def retry_not_exists(func):

    @parse_kubernetes_exception
    @retry(wait_exponential_multiplier=100,
           stop_max_attempt_number=10)
    def wrapped_func(*args, **kwargs):
        try:
            return func(*args, **kwargs)
        except swagger_client.rest.ApiException as e:
            if e.status == 409:
                return None
            else:
                raise e

    return wrapped_func


def retry_unless_not_found(func):

    @parse_kubernetes_exception
    @retry(wait_exponential_multiplier=100,
           stop_max_attempt_number=10)
    def wrapped_func(*args, **kwargs):
        try:
            return func(*args, **kwargs)
        except swagger_client.rest.ApiException as e:
            if e.status == 404:
                return None
            else:
                raise e

    return wrapped_func


def retry_unless(status_code=None, swallow_code=None):
    """
    Don't retry if error code is in status_code, but raise exception
    Don't retry if error code is in swallow_code, no exception is raised as well
    """

    def retry_internal(func):

        def raise_exception_unless(e):
            if status_code is None:
                return True
            if isinstance(e, swagger_client.rest.ApiException) and e.status in status_code:
                return False
            return True

        @parse_kubernetes_exception
        @retry(retry_on_exception=raise_exception_unless,
               wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def wrapped_func(*args, **kwargs):
            try:
                return func(*args, **kwargs)
            except swagger_client.rest.ApiException as e:
                # logger.debug("Func {} has exception {}".format(func, e))
                if swallow_code is not None and e.status in swallow_code:
                    return None
                else:
                    raise e

        return wrapped_func

    return retry_internal


def retry_if(status_code):
    """
    Retry only when error code is in status_code, else, raise exception without retrying
    :param status_code:
    :return:
    """
    def retry_internal(func):

        def raise_exception_unless(e):
            if status_code is None:
                return False
            if isinstance(e, swagger_client.rest.ApiException) and e.status in status_code:
                return True
            return False

        @parse_kubernetes_exception
        @retry(retry_on_exception=raise_exception_unless,
               wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def wrapped_func(*args, **kwargs):
            return func(*args, **kwargs)

        return wrapped_func

    return retry_internal


def parse_swagger_object(obj, rule, func):
    """
    This function will parse an the passed object based on the rule.
    The grammar for the rule is simple
        ATTR = "string"
        PERIOD = .
        RULE = <ATTR> | <ATTR><PERIOD><RULE>
    which forms rules such as "spec.template.port". In this example,
    The object passed must have an attribute called "spec" and the
    object that is pointed to by "spec" must have an attribute called
    "template". In this manner when the final attribute is found, then
    the passed function is applied to the object at that attribute and
    the result of the function is new object that is set
    Args:
        obj: The object to parse
        rule: string
        func: func(obj) is the prototype needed here
    """
    (attrib, _, rest_of_rule) = rule.partition(".")
    attrib_obj = getattr(obj, attrib, None)
    if attrib_obj is None:
        # logger.warn("Attribute {} not found in object {}".format(attrib, obj))
        return

    if rest_of_rule == "" and func is not None:
        # time to apply the func
        setattr(obj, attrib, func(attrib_obj))
        return

    if isinstance(attrib_obj, list):
        for list_obj in attrib_obj:
            parse_swagger_object(list_obj, rest_of_rule, func)
        return
    else:
        return parse_swagger_object(attrib_obj, rest_of_rule, func)


def return_swagger_subobject(obj, rule):
    (attrib, _, rest_of_rule) = rule.partition(".")
    attrib_obj = getattr(obj, attrib, None)
    if attrib_obj is None:
        return None

    if rest_of_rule == "":
        return attrib_obj

    if isinstance(attrib_obj, list):
        ret = []
        for list_obj in attrib_obj:
            ret.append(return_swagger_subobject(list_obj, rest_of_rule))
        return ret
    else:
        return return_swagger_subobject(attrib_obj, rest_of_rule)


class KubernetesApiClient(object):
    # All kubernetes objects
    item_v1 = frozenset(["configmaps", "endpoints", "events", "limitranges", "namespaces", "nodes",
                         "persistentvolumeclaims", "persistentvolumes", "pods",
                         "podtemplates", "replicationcontrollers", "resourcequotas",
                         "secrets", "serviceaccounts", "services"])

    item_v1_beta = frozenset(["daemonsets", "deployments", "ingresses", "replicasets", "networkpolicies",
                              "thirdpartyresources"])
    item_v1_auto_scale = frozenset(["horizontalpodautoscalers"])
    item_v1_batch = frozenset(["jobs"])

    def __init__(self, host=None, port=None, username='admin', password=None, verify_ssl=None, config_file=None,
                 use_proxy=False):
        """Dynamically determine host, port and credential information from current environment

        :param host: hostname or IP of API server
        :param port: port of API server
        :param username: username for basic http authentication (required if token unavailable)
        :param password: password for basic http authentication (required if token unavailable)
        :param verify_ssl: whether we trust self-signed certificate
        """
        self.in_pod = bool(os.getenv('KUBERNETES_SERVICE_HOST'))
        self.session = requests.Session()
        if use_proxy:
            self.host = host or "127.0.0.1"
            self.port = port or "8001"
            self.url = "http://{ip}:{port}".format(ip=self.host, port=self.port)
            self.token = None
        else:
            session_verify = verify_ssl
            if self.in_pod and not config_file:
                self.host = host or 'kubernetes.default'
                self.port = port or int(os.getenv('KUBERNETES_SERVICE_PORT', 443))
                with open(KUBE_TOKEN_PATH) as f:
                    self.token = 'Bearer ' + f.read().strip()
                swagger_client.configuration.ssl_ca_cert = KUBE_CACRT_PATH
                if verify_ssl is None:
                    verify_ssl = bool(self.host == KUBE_CACRT_HOST)
                    session_verify = KUBE_CACRT_PATH
            else:
                self.port = port or 443
                if verify_ssl is None:
                    verify_ssl = False
                    session_verify = False
                if host and password:
                    self.host = host
                    self.token = urllib3.util.make_headers(basic_auth=username + ':' + password).get('authorization')
                else:
                    config_path = os.path.expanduser(config_file if config_file else DEFAULT_KUBECONFIG)
                    if not os.path.isfile(config_path):
                        raise ValueError("Unable to dynamically determine host/credentials from ~/.kube/config")
                    with open(config_path) as f:
                        kube_config = yaml.load(f)
                    cred_info = self._parse_config(kube_config)
                    self.host = cred_info['host']
                    self.token = cred_info['token']

            self.url = 'https://{}:{}'.format(self.host, self.port)
            # NOTE: options set on the configuration singleton (e.g. verify_ssl),
            # must be set *before* instantiation of ApiClient for it to take effect.
            swagger_client.configuration.verify_ssl = verify_ssl
            self.session.verify = session_verify

        self.swag_client = swagger_client.ApiClient(self.url)
        if self.token:
            self.swag_client.set_default_header('Authorization', self.token)
            self.session.headers['Authorization'] = self.token
        # Add modules as attributes of this client
        self.api = swagger_client.Apiv1Api(self.swag_client)
        self.version = swagger_client.VersionApi(self.swag_client)
        self.batchv = swagger_client.Apisbatchv1Api(self.swag_client)
        self.apisappsv1beta1_api = swagger_client.Apisappsv1beta1Api(self.swag_client)
        self.extensionsv1beta1 = swagger_client.Apisextensionsv1beta1Api(self.swag_client)
        self.apisextensionsv1beta1_api = swagger_client.Apisextensionsv1beta1Api(self.swag_client)

        # Add additional (apps, autoscaling, batch, etc...)
        for attr in dir(swagger_client):
            match = re.match(r"^Apis(\w+)Api$", attr)
            if match:
                api_class = getattr(swagger_client, attr)
                api_name = match.group(1)
                api_instance = api_class(self.swag_client)
                setattr(self, api_name, api_instance)

    def get_log(self, namespace, pod, **kwargs):
        """Read log of the specified Pod and return a streaming requests response object

        :param namespace: namespace of pod
        :param pod: name of pod
        :param container: The container for which to stream logs. Defaults to only container if there is one container in the pod.
        :param follow: Follow the log stream of the pod. Defaults to false.
        :param previous: Return previous terminated container logs. Defaults to false.
        :param tail_lines: If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime
        """
        params = []
        for (param, val) in kwargs.items():
            if val is not None:
                if type(val) == bool:
                    params.append((param, str(val).lower()))
                else:
                    params.append((param, str(val)))
        query = urlencode(params, doseq=True)
        url = "{}/api/v1/namespaces/{}/pods/{}/log?{}".format(self.url, namespace, pod, query)
        logger.debug("Retrieving %s logs: GET %s", pod, url)
        response = self.session.get(url, stream=True)
        response.raise_for_status()
        return response

    def exec_start(self, namespace, pod, commands, **kwargs):
        """Start an exec session to a pod container and return the raw websocket
        
        :param namespace: namespace of pod
        :param pod: name of pod
        :parma command: remote command to execute as argv array. Not executed within a shell.
        :param stdin: Redirect the standard input stream of the pod for this call. Defaults to false.
        :param stdout: Redirect the standard output stream of the pod for this call. Defaults to true.
        :param stderr: Redirect the standard error stream of the pod for this call. Defaults to true.
        :param tty: if true indicates that a tty will be allocated for the exec call. Defaults to false.
        :param container: Container in which to execute the command. Defaults to only container if there is only one container in the pod.
        """
        # Kubernetes implements a higher level protocol on top of the websocket connection. The first byte of each
        # frame indicates integer value of the stream. 0:stdin, 1:stdout, 2:sterr, 3:ctrl.
        # See: https://github.com/kubernetes/kubernetes/pull/13885
        if isinstance(commands, str):
            commands = [commands]
        params = []
        for param, val in kwargs.items():
            if val is not None:
                if type(val) == bool:
                    params.append((param, str(val).lower()))
                else:
                    params.append((param, str(val)))
        for cmd_arg in commands:
            params.append(('command', cmd_arg))
        query = urlencode(params, doseq=True)
        url = "wss://{}:{}/api/v1/namespaces/{}/pods/{}/exec?{}".format(self.host, self.port, namespace, pod, query)
        logger.debug("Establishing exec websocket to: %s", url)
        ws = websocket.WebSocket(sslopt={"cert_reqs": ssl.CERT_NONE})
        ws.connect(url, header={"Authorization" : self.token})
        return ws

    def exec_cmd(self, *args, **kwargs):
        """Execute a command and return a generator of its output, or entire output as a string

        :param namespace: namespace of pod
        :param pod: name of pod
        :parma command: remote command to execute as argv array. Not executed within a shell.
        :param container: Container in which to execute the command. Defaults to only container if there is only one container in the pod.
        :param stdin: Redirect the standard input stream of the pod for this call. Defaults to false.
        :param stdout: Redirect the standard output stream of the pod for this call. Defaults to true.
        :param stderr: Redirect the standard error stream of the pod for this call. Defaults to true.
        :param tty: if true indicates that a tty will be allocated for the exec call. Defaults to false.
        :param stream: If true, returns a generator of the output, rather than the output as a string. Defaults to false.
        """
        stream = kwargs.pop('stream', False)
        output_gen = self._exec_cmd_stream(*args, **kwargs)
        if stream:
            return output_gen
        else:
            return ''.join([msg for msg in output_gen])

    def _exec_cmd_stream(self, *args, **kwargs):
        """Generator to stream output of a command executed in a pod"""
        ws = self.exec_start(*args, **kwargs)
        try:
            while True:
                opcode, data = ws.recv_data()
                # Recieved close frame See: https://tools.ietf.org/html/rfc6455#section-5.5.1
                if opcode == websocket.ABNF.OPCODE_CLOSE:
                    break

                if sys.version_info[0] < 3:  # handle python 2
                    stream = ord(data[0])
                    msg = data[1:]
                else:
                    stream = data[0]
                    msg = str(data[1:], 'utf-8', errors="replace")

                if msg:
                    if stream == 3:
                        logger.debug("Received the following str on ctrl plane: %s", msg)
                    else:
                        yield msg
        finally:
            ws.close()

    def exec_kill(self, namespace, pod, pid, container=None, signal=None):
        """Kill a pid in a container

        :param namespace: namespace of pod
        :param pod: name of pod
        :param pid: pid to kill in the container.
        :param container: container name to perform kill.
        :param signal: kill signal to send to process
        """
        logger.info("Killing pid %s in pod %s", pid, pod)
        pid = int(pid)
        assert pid != -1, "Killing all processes prohibited"
        # perform kill
        signal_str = '-{} '.format(signal) if signal else ''
        cmd_args = [
            'sh',
            '-c',
            "kill {}{} 2>&1; echo $?".format(signal_str, pid)
        ]
        response = self.exec_cmd(namespace, pod, cmd_args, container=container, stdout=True, stderr=True)
        lines = response.splitlines()
        rc = int(lines[-1])
        if rc != 0:
            reason = lines[0] if len(lines) > 1 else "reason unknown"
            logger.warning("Failed to kill pid %s in container %s: %s", pid, container, reason)
        else:
            logger.debug("Successfully killed pid %s in container %s", pid, container)

    def _parse_config(self, kube_config):
        """Return config information from current kubernetes context"""
        cred_info = {}
        # cluster info
        context = next(c['context'] for c in kube_config['contexts'] if c['name'] == kube_config['current-context'])
        cred_info.update(context)
        context_cluster = next(c['cluster'] for c in kube_config['clusters'] if c['name'] == context['cluster'])
        cred_info['host'] = context_cluster['server'].split('/')[-1]
        cred_info['certificate-authority-data'] = context_cluster['certificate-authority-data']
        # user info
        context_user = next(c['user'] for c in kube_config['users'] if c['name'] == context['user'])
        if Cloud().target_cloud_aws():
            if 'token' in context_user:
                cred_info['token'] = "Bearer " + context_user['token']
            else:
                cred_info['token'] = urllib3.util.make_headers(basic_auth=context_user['username'] + \
                                                               ':' + context_user['password']).get('authorization')
        elif Cloud().target_cloud_gcp():
            cred_info['token'] = GCPToken().token
        return cred_info

    @staticmethod
    def _stream_helper(response):
        """
        Generate iterables from kubernetes stream API

        :param response: HTTP get response
        :return: iterables with json objects
        """
        try:
            response.raise_for_status()
        except requests.exceptions.HTTPError as e:
            raise AXKubeApiException(str(e))

        # this assumes that there is no newline character
        # within one json object. Note Kubernetes does retusn
        # json that contains "\" and "n", but thats two characters
        for line in response.iter_lines(chunk_size=None):
            try:
                if py3env:
                    yield json.loads(line.decode("utf-8"))
                else:
                    yield json.loads(line)
            except Exception as e:
                msg = str(e) + " Content: " + line
                raise AXKubeApiException(msg)

    def _generate_api_params(self, **kwargs):
        data = {}
        valid_params = frozenset(["label_selector", "timeout_seconds", "field_selector"])

        # Try to keep argument name same as that of other swagger
        # generated apis
        swagger_to_kube = {
            "label_selector": "labelSelector",
            "timeout_seconds": "timeoutSeconds",
            "field_selector": "fieldSelector"
        }
        for k, v in kwargs.items():
            if k not in valid_params:
                raise AXKubeApiException("Stream API only support {} for now".format(str(valid_params)))
            data[swagger_to_kube[k]] = v
        return data

    def _generate_api_endpoint(self, item=None, namespace=None, name=None):
        api_tail = ""
        if item in self.item_v1:
            api_base = API_V1_BASE
        elif item in self.item_v1_beta:
            api_base = API_EXT_BASE
        elif item in self.item_v1_auto_scale:
            api_base = API_AUTO_SCALE_BASE
        else:
            api_base = API_BATCH_BASE
        api_base += "watch/"

        if name:
            if not namespace and item not in KUBE_NO_NAMESPACE_SET:
                namespace = "default"
            api_tail = "/" + name

        if namespace:
            if item in KUBE_NO_NAMESPACE_SET:
                raise AXKubeApiException("Cannot watch namespaced {}".format(item))
            api_base += "namespaces/{namespace}/".format(namespace=namespace)

        endpoint = api_base + item + api_tail
        return endpoint

    def _call_api(self, endpoint, **kwargs):
        api_url = self.url + endpoint
        data = self._generate_api_params(**kwargs)
        return self._stream_helper(
            self.session.get(api_url, stream=True, params=data)
        )

    def watch(self, item, namespace=None, name=None, **kwargs):
        """
        For API v1, we can watch (and watch namespaced)
        1. configmaps
        2. endpoints
        3. events
        4. limitranges
        5. namespaces (should NOT specify namespace)
        6. nodes (should NOT specify namespace)
        7. persistentvolumeclaims
        8. persistentvolumes (should NOT specify namespace)
        9. pods
        10. podtemplates
        11. replicationcontrollers
        12. resourcequotas
        13. secrets
        14. serviceaccounts
        15. services

        For API extensions, we can watch (and watch namespaced)
        1. daemonsets
        2. deployments
        3. horizontalpodautoscalers
        4. ingresses
        5. jobs
        6. replicasets

        We can watch object by specifying its name. In this case, if namespace is not
        provided explicitly, we will watch the namespace "default"

        We can specify query parameters "labelSelector, timeoutSeconds and fieldSelector". Others are
        either not relevant to us or is not supported by Kubernetes

        Example:
        label = [
           "key1=value1",
           "key2=value2"
        ]
        timeout = 10
        pods = client.watch(item="pods", label_selector=label, timeout_seconds=timeout)

        Only get events about Pods. Note that the value of field_selector is case sensitive.
        events = client.watch(item="events", field_selector="involvedObject.kind=Pod")

        :param item string: one of the items specified above
        :param namespace string: kubernetes namespace
        :param name string: name of the object
        :param kwargs string: label_selector or timeout_seconds
        :return:
        """
        if not self.validate_object(item):
            msg = "Invalid item: {}. Supported items are {}, {}, {} with autoscaling API and {} with batch API".format(
                item, str(self.item_v1), str(self.item_v1_beta),
                str(self.item_v1_auto_scale), str(self.item_v1_batch)
            )
            raise AXKubeApiException(msg)

        endpoint = self._generate_api_endpoint(item, namespace, name)
        return self._call_api(endpoint, **kwargs)

    @classmethod
    def validate_object(cls, obj):
        return (obj in cls.item_v1) or (obj in cls.item_v1_beta) or \
               (obj in cls.item_v1_auto_scale) or (obj in cls.item_v1_batch)

    @staticmethod
    def reason_for_exception(e):
        assert isinstance(e, ApiException) and "Instance must be of type ApiException"
        body = json.loads(e.body)
        return body['reason']


class KubernetesApiClientWrapper(object):
    """
    This class simply stores a kubernetes client object and has a method
    to retrieve this object
    """
    def __init__(self, client):
        assert isinstance(client, KubernetesApiClient), "Client needs to be of type KubernetesApiClient but got type is {}".format(type(client))
        self._client = client

    def get_k8s_client(self):
        return self._client

