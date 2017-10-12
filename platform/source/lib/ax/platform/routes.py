#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
This module is tasked with creating routes (kubernetes services and ingress rules)
"""

import boto3
import logging
import time
import yaml

from botocore.client import Config

from ax.exceptions import AXConflictException, AXNotFoundException, AXIllegalArgumentException
from ax.kubernetes import swagger_client
from ax.kubernetes.client import parse_kubernetes_exception, KubernetesApiClient, retry_unless, KubernetesApiClientWrapper
from ax.kubernetes.kube_object import KubeObject
from ax.platform.operations import Operation
from ax.platform.resources import AXResource
from ax.platform.cloudprovider.aws import Route53, Route53HostedZone
from ax.util.validators import hostname_validator
from ax.util.hash import generate_hash

from retrying import retry

logger = logging.getLogger(__name__)


def raise_apiexception_else_retry(e):
    if isinstance(e, swagger_client.rest.ApiException):
        logger.exception(e)
        return False
    return True


class ServiceOperation(Operation):
    """
    Forbid ServiceOperations to conflict with each other
    """

    def __init__(self, obj):
        token = "{}/{}".format(obj.namespace, obj.name)
        super(ServiceOperation, self).__init__(token=token)

    @staticmethod
    def prettyname():
        return "ServiceOperation"


class ServiceEndpoint(KubernetesApiClientWrapper):
    """
    Expose a service using ELB
    XXX: This will be removed when EA for Deployment is removed.
         DO NOT USE THIS CLASS IN ANY NEW CODE
    """
    def __init__(self, name, namespace="axuser", client=None):
        """
        Args:
            name: String. Needs to be 24 characters and valid dns chars only
        """
        self.name = name
        self.namespace = namespace
        _client = client
        if client is None:
            _client = KubernetesApiClient(use_proxy=True)

        super(ServiceEndpoint, self).__init__(_client)


    def create(self, appname, port_spec, layer7=False, type='LoadBalancer'):
        """

        Args:
            appname: The pod/s that will be selected by this load balancer
            port_spec: dict of { "name": string, "port": number, "containerPort": number }
        """
        with ServiceOperation(self):
            svc = self._get_from_provider()
            if svc:
                self._update(svc, appname, port_spec, layer7)
            else:
                self._create(appname, port_spec, layer7, type)

    def _create(self, appname, port_spec, layer7, type):

        srv = swagger_client.V1Service()
        srv.metadata = swagger_client.V1ObjectMeta()
        srv.metadata.name = self.name
        srv.metadata.labels = {
            "app": appname,
            "tier": "deployment",
            "role": "user"
        }
        if layer7:
            srv.metadata.annotations = {
                "service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "http"
            }

        spec = swagger_client.V1ServiceSpec()
        spec.selector = {
            'app': appname
        }
        spec.type = type
        spec.ports = []

        for p in port_spec:
            port = swagger_client.V1ServicePort()
            port.name = p["name"]
            port.port = p["port"]
            port.target_port = p["containerPort"]
            spec.ports.append(port)

        srv.spec = spec

        @parse_kubernetes_exception
        @retry(retry_on_exception=raise_apiexception_else_retry,
               wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def create_in_provider(service):
            client = self.get_k8s_client()
            client.api.create_namespaced_service(service, self.namespace)

        create_in_provider(srv)

    def _update(self, curr, appname, port_spec, layer7):
        state = {
            "metadata": {
                "labels": {
                    "app": appname
                }
            },
            "spec": {
                "ports": [],
                "selector": {
                    "app": appname
                }
            }
        }
        if layer7:
            state["metadata"]["annotations"] = {
                "service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "http"
            }
        for p in port_spec:
            port = {
                "name": p["name"],
                "port": p["port"],
                "targetPort": p["containerPort"]
            }
            state["spec"]["ports"].append(port)

        @parse_kubernetes_exception
        @retry(wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def update_in_provider(s):
            client = self.get_k8s_client()
            client.api.patch_namespaced_service(s, self.namespace, self.name)

        update_in_provider(state)

    def exists(self):
        with ServiceOperation(self):
            svc = self._get_from_provider()
            if svc is None:
                return False
            else:
                return True

    def delete(self):
        @parse_kubernetes_exception
        @retry(wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def delete_in_provider():
            client = self.get_k8s_client()
            try:
                client.api.delete_namespaced_service(self.namespace, self.name)
            except swagger_client.rest.ApiException as e:
                if e.status != 404:
                    raise e

        with ServiceOperation(self):
            delete_in_provider()

    def get_addrs(self):
        """
        Returns:
            list of hostnames associated with this load balancer
        """
        with ServiceOperation(self):
            svc = self._get_from_provider(use_proxy=False)

        ret = []
        ing_list = svc.status.load_balancer.ingress
        if not ing_list:
            return ret

        for ing in ing_list:
            # this will be a list of v1.LoadBalancerIngress
            if ing.hostname:
                ret.append(ing.hostname)

        return ret

    @parse_kubernetes_exception
    @retry(wait_exponential_multiplier=100,
           stop_max_attempt_number=10)
    def _get_from_provider(self, use_proxy=True):
        try:
            client = self.get_k8s_client()
            svc = client.api.read_namespaced_service(self.namespace, self.name)
            return svc
        except swagger_client.rest.ApiException as e:
            if e.status != 404:
                raise e

        return None


class NginxIngressController(object):

    def __init__(self, name, namespace="axuser"):
        self.name = name
        self.namespace = namespace
        self.cmap_name = "{}-conf".format(self.name)

    def create(self, node_selector=None):
        from ax.kubernetes.swagger_client import ApiClient
        converter = ApiClient()

        client = KubernetesApiClient(use_proxy=True)

        with open("/ax/config/service/nginx-ingress-template.yml.in") as f:
            deployment_spec = yaml.load(f)

        spec = converter._ApiClient__deserialize(deployment_spec, 'V1beta1Deployment')
        spec.metadata.name = self.name
        spec.metadata.labels = {
            "app": self.name,
            "tier": "platform",
            "role": "axcritical"
        }

        spec.spec.selector.match_labels = spec.metadata.labels
        spec.spec.template.metadata.labels = spec.metadata.labels
        spec.spec.template.spec.containers[0].args.append("--ingress-class={}".format(self.name))
        spec.spec.template.spec.containers[0].args.append("--nginx-configmap=$(POD_NAMESPACE)/{}".format(self.cmap_name))
        spec.spec.template.spec.node_selector = node_selector

        # define the config map
        cmap = swagger_client.V1ConfigMap()
        cmap.metadata = swagger_client.V1ObjectMeta()
        cmap.metadata.name = self.cmap_name
        cmap.data = {
            "server-name-hash-bucket-size": "512",
            "server-name-hash-max-size": "512"
        }

        @retry_unless()
        def create_config_map():
            try:
                client.api.replace_namespaced_config_map(cmap, self.namespace, self.cmap_name)
            except swagger_client.rest.ApiException as ee:
                if ee.status == 404:
                    client.api.create_namespaced_config_map(cmap, self.namespace)
                else:
                    raise ee

        @retry_unless()
        def create_in_provider():
            logger.debug("Creating deployment in provider")
            try:
                client.extensionsvbeta.replace_namespaced_deployment(spec, self.namespace, self.name)
            except swagger_client.rest.ApiException as ee:
                if ee.status == 404:
                    client.extensionsvbeta.create_namespaced_deployment(spec, self.namespace)
                else:
                    raise ee

        s = ServiceEndpoint(self.name, self.namespace)
        ports = [
            {
                "name": "http",
                "port": 80,
                "containerPort": 80
            },
            {
                "name": "https",
                "port": 443,
                "containerPort": 443
            }
        ]
        try:
            s.create(self.name, ports, layer7=True)
        except Exception as e:
            logger.debug("Got exception while trying to create ServiceEndpoint {}".format(e))
            raise e

        create_config_map()
        create_in_provider()

    def exists(self):

        s = ServiceEndpoint(self.name, self.namespace)
        client = KubernetesApiClient(use_proxy=True)

        @retry_unless(swallow_code=[404])
        def _get_from_provider():
            return client.extensionsvbeta.read_namespaced_deployment(self.namespace, self.name)

        @retry_unless(swallow_code=[404])
        def _get_configmap_from_provider():
            return client.api.read_namespaced_config_map(self.namespace, self.cmap_name)

        if _get_from_provider() and _get_configmap_from_provider() and s.exists():
            return True

        return False

    def delete(self):

        client = KubernetesApiClient(use_proxy=True)
        options = swagger_client.V1DeleteOptions()
        options.grace_period_seconds = 1

        @parse_kubernetes_exception
        @retry(wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def delete_in_provider():
            try:
                client.extensionsvbeta.delete_namespaced_deployment(options, self.namespace, self.name)
                client.extensionsvbeta.deletecollection_namespaced_replica_set(self.namespace, label_selector="app={}".format(self.name))
                client.api.deletecollection_namespaced_pod(self.namespace, label_selector="app={}".format(self.name))
            except swagger_client.rest.ApiException as e:
                if e.status != 404:
                    raise e

        delete_in_provider()
        time.sleep(2)

        s = ServiceEndpoint(self.name, self.namespace)
        s.delete()


class IngressProvider(object):
    EXTERNAL = "all-deployments"
    INTERNAL = "int-deployments"


class IngressRuleOperation(Operation):
    """
    Forbid IngressRuleOperations to conflict with each other
    """
    def __init__(self, obj):
        token = "{}/{}".format(obj.namespace, obj.name)
        super(IngressRuleOperation, self).__init__(token=token)

    @staticmethod
    def prettyname():
        return "IngressRuleOperation"


class IngressRules(AXResource):

    @staticmethod
    def create_object_from_info(info):
        name = info["name"]
        ingress_class = info["ingress_class"]
        namespace = info["namespace"]
        return IngressRules(name, ingress_class, namespace)

    def __init__(self, name, ingress_class, namespace="axuser"):
        self.name = name
        self.ingress_class = ingress_class
        self.namespace = namespace
        self.client = KubernetesApiClient(use_proxy=True)
        self.ax_meta = {}

    def __str__(self):
        return "Ingress Rule {}/{} of class {}".format(self.namespace, self.name, self.ingress_class)

    def create(self, service, ports, host_mapping=None, whitelist_cidrs=[]):
        ing = self._get_from_provider()
        if ing is None:
            self._delete_in_provider()

        ing = swagger_client.V1beta1Ingress()
        ing.metadata = swagger_client.V1ObjectMeta()
        ing.metadata.name = self.name
        ing.metadata.annotations = {
            "kubernetes.io/ingress.class": self.ingress_class,
            "ingress.kubernetes.io/whitelist-source-range":  ",".join(whitelist_cidrs)
        }

        ing.spec = swagger_client.V1beta1IngressSpec()
        ing.spec.rules = []

        r = swagger_client.V1beta1IngressRule()

        if host_mapping:
            r.host = host_mapping
        r.http = swagger_client.V1beta1HTTPIngressRuleValue()
        r.http.paths = []

        # only one rule with multiple paths for now
        ing.spec.rules.append(r)

        # build the metadata for the AXResource
        self.ax_meta = {
            "name": self.name,
            "namespace": self.namespace,
            "ingress_class": self.ingress_class,
            "ingress_to": {
                "service": service,
                "ports": ports,
                "whitelist": whitelist_cidrs
            }
        }
        AXResource.set_ax_meta(ing, self.ax_meta)

        self._rules_to_ingress(service, r.http.paths, ports)
        with IngressRuleOperation(self):
            self._create_in_provider(ing)

    def delete(self):
        with IngressRuleOperation(self):
            self._delete_in_provider()

    def exists(self):
        with IngressRuleOperation(self):
            return self._get_from_provider() is not None

    def get_resource_name(self):
        return self.name

    def get_resource_info(self):
        return self.ax_meta

    def status(self):
        return {}

    @retry_unless(swallow_code=[404])
    def _get_from_provider(self):
        return self.client.apisextensionsv1beta1_api.read_namespaced_ingress(self.namespace, self.name)

    @retry_unless(status_code=[409, 404, 422])
    def _create_in_provider(self, ing):
        # 409 - conflict (already exists)
        # 404 - namespace not found
        # 422 - unprocessable entity
        try:
            self.client.apisextensionsv1beta1_api.create_namespaced_ingress(ing, self.namespace)
        except swagger_client.rest.ApiException as e:
            if e.status == 409:
                self.client.apisextensionsv1beta1_api.replace_namespaced_ingress(ing, self.namespace, self.name)
            else:
                raise e

    @retry_unless(swallow_code=[404])
    def _delete_in_provider(self):
        options = swagger_client.V1DeleteOptions()
        options.grace_period_seconds = 1
        self.client.apisextensionsv1beta1_api.delete_namespaced_ingress(options, self.namespace, self.name)

    def _rules_to_ingress(self, service, paths, ports):
        for port in ports or []:
            if 'urlPath' not in port:
                continue
            path = swagger_client.V1beta1HTTPIngressPath()
            path.path = port["urlPath"]
            path.backend = swagger_client.V1beta1IngressBackend()
            path.backend.service_name = service
            path.backend.service_port = int(port["port"])
            paths.append(path)


class InternalRouteOperation(Operation):
    """
    Forbid InternalRouteOperations to conflict with each other
    """

    def __init__(self, route):
        token = "{}/{}".format(route.namespace, route.name)
        super(InternalRouteOperation, self).__init__(token=token)

    @staticmethod
    def prettyname():
        return "InternalRouteOperation"


class InternalRoute(AXResource):
    """
    Internal Route is a kubernetes service object
    """
    def __init__(self, name, namespace, client=None):
        self.name = name
        self.namespace = namespace
        if client is None:
            self.client = KubernetesApiClient(use_proxy=True)
        else:
            self.client = client
        self.ax_meta = {}

    def __str__(self):
        return "InternalRoute {}/{}".format(self.namespace, self.name)

    @staticmethod
    def create_object_from_info(info):
        name = info["name"]
        namespace = info["application"]
        return InternalRoute(name, namespace)

    def create(self, port_spec, selector=None, owner=None):
        """
        Create a kubernetes service with port_spec. The port_spec is a list of ports
        with the following in each port spec {name: string, port: int, target_port: int}
        Args:
            port_spec: list of ports as described above
            selector: a string that has a dict of key:value strings. This dict is used to match labels on pods
            owner: string that describes an owner.

        If the object already exists with a different owner a conflict will result that will be raised
        If the object exists with the same owner, it will be silently overwritten
        """
        @retry_unless(status_code=[404, 422])
        def create_in_provider(spec):
            # 404: namespace not found
            # 422 unprocessable entity
            self.client.api.create_namespaced_service(spec, self.namespace)

        with InternalRouteOperation(self):
            # we acquire this lock to prevent a race between concurrent but conflicting creation
            # of the same internal route. For example, this can happen when two deployments in the same
            # namespace use the same name for the internal route are being created simultaneously
            status = self._get_from_provider()
            if status is not None:
                old_owner = self._get_owner_of_route()
                if old_owner is not None and old_owner != owner:
                    raise AXConflictException("Route {}.{} exists with owner {}. The requested owner is {}".
                                              format(self.name, self.namespace, old_owner, owner))
                self._delete_in_provider()

            # now generate spec and create in provider
            k8s_spec = self._generate_spec(port_spec, selector, owner)
            create_in_provider(k8s_spec)

    def delete(self):
        """
        Delete this service. Is idempotent. Does not raise any error if the route does not exist
        """
        with InternalRouteOperation(self):
            self._delete_in_provider()

    def exists(self):
        with InternalRouteOperation(self):
            return self._get_from_provider() is not None

    def get_resource_name(self):
        return self.name

    def get_resource_info(self):
        return self.ax_meta

    def status(self, with_loadbalancer_info=False):
        # Locking so that a create is not meddling with status
        s = self._get_from_provider()
        if s is None:
            raise AXNotFoundException("Could not find Route {}.{}".format(self.name, self.namespace))
        ep = self._get_ep_from_provider()
        if ep is None:
            raise AXNotFoundException("Could not find Route Endpoint for {}.{}".format(self.name, self.namespace))

        field_map = {
            "name": "metadata.name",
            "application": "metadata.namespace",
            "ip": "spec.cluster_ip"
        }

        if with_loadbalancer_info:
            field_map["loadbalancer"] = "status.load_balancer.ingress.hostname"

        ret = KubeObject.swagger_obj_extract(s, field_map)
        field_map = {
            "ips": "subsets.addresses.ip"
        }
        ret["endpoints"] = KubeObject.swagger_obj_extract(ep, field_map)
        return ret

    def _get_owner_of_route(self):
        return self.ax_meta.get("owner", None)

    @retry_unless(swallow_code=[404])
    def _get_from_provider(self):
        status = self.client.api.read_namespaced_service(self.namespace, self.name)
        if status is not None:
            self.ax_meta = AXResource.get_ax_meta(status)
        return status

    @retry_unless(swallow_code=[404])
    def _get_ep_from_provider(self):
        return self.client.api.read_namespaced_endpoints(self.namespace, self.name)

    @retry_unless(swallow_code=[404])
    def _delete_in_provider(self):
        self.client.api.delete_namespaced_service(self.namespace, self.name)

    def _generate_spec(self, port_spec, selector, owner):
        # generate port spec
        k8s_ports = []

        for port in port_spec or []:
            p = swagger_client.V1ServicePort()
            p.port = int(port["port"])
            p.name = "port-{}".format(p.port)
            p.target_port = int(port["target_port"])
            if "node_port" in port:
                p.node_port = port["node_port"]
            k8s_ports.append(p)

        # service spec
        srv_spec = swagger_client.V1ServiceSpec()
        srv_spec.ports = k8s_ports
        srv_spec.selector = selector
        srv_spec.type = "ClusterIP"

        # generate meta
        meta = swagger_client.V1ObjectMeta()
        meta.name = self.name

        meta.labels = {
            "srv": self.name,
            "tier": "deployment",
            "role": "user"
        }
        meta.annotations = {}

        self.ax_meta = {
            "name": self.name,
            "application": self.namespace,
            "owner": owner,
            "port_spec": port_spec,
            "selector": selector
        }

        # finally the service
        srv = swagger_client.V1Service()
        srv.metadata = meta
        srv.spec = srv_spec

        AXResource.set_ax_meta(srv, self.ax_meta)

        return srv


class NodeRoute(InternalRoute):
    """
    This object creates node port routes for deployments
    """
    def create(self, port_spec, selector=None, owner=None):

        @retry_unless(status_code=[404, 422])
        def create_in_provider(spec):
            # 404: namespace not found
            # 422 unprocessable entity
            self.client.api.create_namespaced_service(spec, self.namespace)

        # now generate spec and create in provider
        k8s_spec = self._generate_spec(port_spec, selector, owner)
        k8s_spec.spec.type = "NodePort"

        create_in_provider(k8s_spec)


class DnsObject(AXResource):
    """
    This is a wrapper on top of route53 object to manage it as an AXResource
    """

    @staticmethod
    def create_object_from_info(info):
        name = info["name"]
        domain = info["domain"]
        return DnsObject("{}.{}".format(name, domain))

    def __str__(self):
        return "DNS Object {}.{}".format(self.name, self.domain)

    def __init__(self, dnsname):
        if not hostname_validator(dnsname):
            raise AXIllegalArgumentException("dns name {} is illegal".format(dnsname))

        (self.name, _, self.domain) = dnsname.partition(".")
        self.ax_meta = {}

        config = Config(connect_timeout=60, read_timeout=60)
        boto_client = boto3.client("route53", config=config)
        client = Route53(boto_client)
        self.zone = Route53HostedZone(client, self.domain)

    # idempotent
    def create_alias(self, elb_addr, elb_name=None):
        self.zone.create_alias_record(self.name, elb_addr, elb_name=elb_name)
        self.ax_meta = {
            "name": self.name,
            "domain": self.domain,
            "domain_points_to": {
                "elb_addr": elb_addr,
            }
        }

    # idempotent
    def delete(self):
        self.zone.delete_record(self.name, "A")

    def status(self):
        return {}

    def get_resource_name(self):
        return self.name

    def get_resource_info(self):
        return self.ax_meta


class ExternalRouteVisibility(object):
    VISIBILITY_WORLD = "world"
    VISIBILITY_ORGANIZATION = "organization"


class ExternalRoute(AXResource):
    """
    This class combines internal routes and ingress rule and dns entries
    to create an external route to an application
    """
    @staticmethod
    def create_object_from_info(info):
        dns_name = info["dns_name"]
        application = info["application"]
        deployment_selector = info["deployment_selector"]
        target_port = info["target_port"]
        whitelist = info["whitelist"]
        visibility = info["visibility"]
        eroute = ExternalRoute(dns_name, application, deployment_selector, target_port, whitelist=whitelist, visibility=visibility)
        elb_addr = info.get("elb_addr", None)
        if elb_addr:
            eroute.ax_meta["elb_addr"] = elb_addr
        return eroute

    def __init__(self, dns_name, application, deployment_selector, target_port, whitelist=["0.0.0.0/0"], visibility=ExternalRouteVisibility.VISIBILITY_WORLD, name=None):
        self._object_name = name
        if not name:
            self._object_name = generate_hash("{}-{}-{}".format(dns_name, application, deployment_selector))
        self.dns_name = dns_name[:-1] if dns_name.endswith(".") else dns_name
        self.application = application
        self.deployment_selector = deployment_selector
        self.target_port = target_port
        self.whitelist = whitelist

        assert visibility == ExternalRouteVisibility.VISIBILITY_WORLD or visibility == ExternalRouteVisibility.VISIBILITY_ORGANIZATION, \
            "External routes can only have visibility of \"{}\" or \"{}\"".format(ExternalRouteVisibility.VISIBILITY_WORLD, ExternalRouteVisibility.VISIBILITY_ORGANIZATION)
        self.visibility = visibility

        self.ax_meta = {
            "dns_name": self.dns_name,
            "application": self.application,
            "deployment_selector": self.deployment_selector,
            "target_port": self.target_port,
            "whitelist": self.whitelist,
            "visibility": self.visibility
        }
        self.ingress_target = IngressProvider.EXTERNAL if self.visibility == ExternalRouteVisibility.VISIBILITY_WORLD else IngressProvider.INTERNAL

    def create(self, elb_addr=None, elb_name=None):

        # create an internal route to deployment using selector
        ir = InternalRoute(self._object_name, self.application)
        ir.create([{"name": "externalroute", "port": self.target_port, "target_port": self.target_port}],
                  selector=self.deployment_selector)

        # create ingress rule
        ing = IngressRules(self._object_name, self.ingress_target, namespace=self.application)
        ing.create(self._object_name, [{"urlPath": "/", "port": self.target_port}], self.dns_name, whitelist_cidrs=self.whitelist)

        if elb_addr:
            self.ax_meta["elb_addr"] = elb_addr
            # create dns record
            dns = DnsObject(self.dns_name)
            dns.create_alias(elb_addr, elb_name=elb_name)

        return self._object_name

    def delete(self):
        ir = InternalRoute(self._object_name, self.application)
        ir.delete()

        ing = IngressRules(self._object_name, self.ingress_target, namespace=self.application)
        ing.delete()

        if self.ax_meta.get("elb_addr", None) is not None:
            dns = DnsObject(self.dns_name)
            dns.delete()

    def exists(self):
        assert False, "exists() is not implemented for ExternalRoute"

    def status(self):
        assert False, "status() is not implemented for ExternalRoute"

    def get_resource_name(self):
        return self._object_name

    def get_resource_info(self):
        return self.ax_meta
