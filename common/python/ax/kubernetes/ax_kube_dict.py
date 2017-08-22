#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
This module defines Argo related kubernetes object names and other constants
"""

from ax.kubernetes.swagger_client import *


class AXNameSpaces(object):
    """
    Namespaces Argo uses
    """
    KUBE_SYSTEM = "kube-system"
    AXSYS = "axsys"
    AXUSER = "axuser"
    DEFAULT = "default"


class KubeKind(object):
    """
    "Kind" field in kubernetes API objects.

    Note that "HorizontalPodAutoscaler" and "PodTemplate" are not
    confirmed neither in Kubernetes documentation nor by calling
    actual API.
    """
    CONFIGMAP = "ConfigMap"
    ENDPOINTS = "Endpoints"
    EVENT = "Event"
    LIMITRANGE = "LimitRange"
    NAMESPACE = "Namespace"
    NODE = "Node"
    PVC = "PersistentVolumeClaim"
    PV = "PersistentVolume"
    POD = "Pod"
    PODTEMPLATE = "PodTemplate"
    RC = "ReplicationController"
    RESOURCE_QUOTA = "ResourceQuota"
    SECRET = "Secret"
    SERVICE_ACCOUNT = "ServiceAccount"
    SERVICE = "Service"
    DAEMONSET = "DaemonSet"
    DEPLOYMENT = "Deployment"
    INGRESS = "Ingress"
    REPLICASET = "ReplicaSet"
    HPA = "HorizontalPodAutoscaler"
    JOB = "Job"
    STATEFULSET = "StatefulSet"
    TPR = "ThirdPartyResource"


class KubeApiObjKind(object):
    """
    Kubernetes object name used for API endpoint
    """
    CONFIGMAP = "configmaps"
    ENDPOINTS = "endpoints"
    EVENT = "events"
    LIMITRANGE = "limitranges"
    NAMESPACE = "namespaces"
    NODE = "nodes"
    PVC = "persistentvolumeclaims"
    PV = "persistentvolumes"
    POD = "pods"
    PODTEMPLATE = "podtemplates"
    RC = "replicationcontrollers"
    RESOURCE_QUOTA = "resourcequotas"
    SECRET = "secrets"
    SERVICE_ACCOUNT = "serviceaccounts"
    SERVICE = "services"
    DAEMONSET = "daemonsets"
    DEPLOYMENT = "deployments"
    INGRESS = "ingresses"
    REPLICASET = "replicasets"
    HPA = "horizontalpodautoscalers"
    JOB = "jobs"
    TPR = "thirdpartyresources"

# This is a map between the "kind" field in kube object and
# name literals of swagger object. We are only listing the
# ones we use. This is useful to deserialize object using
# swagger client
KubeKindToV1KubeSwaggerObject = {
    "Service": ("V1Service", V1Service),
    "Deployment": ("V1beta1Deployment", V1beta1Deployment),
    "StatefulSet": ("V1beta1StatefulSet", V1beta1StatefulSet),
    "DaemonSet": ("V1beta1DaemonSet", V1beta1DaemonSet),
    "PersistentVolumeClaim": ("V1PersistentVolumeClaim", V1PersistentVolumeClaim),
    "Secret": ("V1Secret", V1Secret),
    "Pod": ("V1Pod", V1Pod),
    "Node": ("V1Node", V1Node),
    "Job": ("V1Job", V1Job),
    "Namespace": ("V1Namespace", V1Namespace),
    "Event": ("V1Event", V1Event),
    "ConfigMap": ("V1ConfigMap", V1ConfigMap)
}


KUBE_NO_NAMESPACE_SET = frozenset([KubeKind.NAMESPACE, KubeApiObjKind.NAMESPACE, KubeKind.PV, KubeApiObjKind.PV,
                                   KubeKind.NODE, KubeApiObjKind.NODE, KubeKind.TPR, KubeApiObjKind.TPR])

KubeKindToKubeApiObjKind = {}
KubeApiObjKindToKubeKind = {}

for attr in dir(KubeKind):
    if "__" not in attr and attr in dir(KubeApiObjKind):
        KubeKindToKubeApiObjKind[getattr(KubeKind, attr)] = getattr(KubeApiObjKind, attr)

for attr in dir(KubeApiObjKind):
    if "__" not in attr and attr in dir(KubeKind):
        KubeApiObjKindToKubeKind[getattr(KubeApiObjKind, attr)] = getattr(KubeKind, attr)
