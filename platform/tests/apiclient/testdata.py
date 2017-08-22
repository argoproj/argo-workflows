# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

TEST_KUBE_WATCH_API_ENDPOINTS_V1 = {
    # There are 42 v1 watch API endpoints
    # http://kubernetes.io/docs/api-reference/v1/operations/
    "/api/v1/watch/configmaps",
    "/api/v1/watch/endpoints",
    "/api/v1/watch/events",
    "/api/v1/watch/limitranges",
    "/api/v1/watch/namespaces",
    "/api/v1/watch/namespaces/{namespace}/configmaps",
    "/api/v1/watch/namespaces/{namespace}/configmaps/{name}",
    "/api/v1/watch/namespaces/{namespace}/endpoints",
    "/api/v1/watch/namespaces/{namespace}/endpoints/{name}",
    "/api/v1/watch/namespaces/{namespace}/events",
    "/api/v1/watch/namespaces/{namespace}/events/{name}",
    "/api/v1/watch/namespaces/{namespace}/limitranges",
    "/api/v1/watch/namespaces/{namespace}/limitranges/{name}",
    "/api/v1/watch/namespaces/{namespace}/persistentvolumeclaims",
    "/api/v1/watch/namespaces/{namespace}/persistentvolumeclaims/{name}",
    "/api/v1/watch/namespaces/{namespace}/pods",
    "/api/v1/watch/namespaces/{namespace}/pods/{name}",
    "/api/v1/watch/namespaces/{namespace}/podtemplates",
    "/api/v1/watch/namespaces/{namespace}/podtemplates/{name}",
    "/api/v1/watch/namespaces/{namespace}/replicationcontrollers",
    "/api/v1/watch/namespaces/{namespace}/replicationcontrollers/{name}",
    "/api/v1/watch/namespaces/{namespace}/resourcequotas",
    "/api/v1/watch/namespaces/{namespace}/resourcequotas/{name}",
    "/api/v1/watch/namespaces/{namespace}/secrets",
    "/api/v1/watch/namespaces/{namespace}/secrets/{name}",
    "/api/v1/watch/namespaces/{namespace}/serviceaccounts",
    "/api/v1/watch/namespaces/{namespace}/serviceaccounts/{name}",
    "/api/v1/watch/namespaces/{namespace}/services",
    "/api/v1/watch/namespaces/{namespace}/services/{name}",
    "/api/v1/watch/namespaces/{name}",
    "/api/v1/watch/nodes",
    "/api/v1/watch/nodes/{name}",
    "/api/v1/watch/persistentvolumeclaims",
    "/api/v1/watch/persistentvolumes",
    "/api/v1/watch/persistentvolumes/{name}",
    "/api/v1/watch/pods",
    "/api/v1/watch/podtemplates",
    "/api/v1/watch/replicationcontrollers",
    "/api/v1/watch/resourcequotas",
    "/api/v1/watch/secrets",
    "/api/v1/watch/serviceaccounts",
    "/api/v1/watch/services"
}

TEST_KUBE_WATCH_API_ENDPOINTS_V1_EXTENSION = {
    # There are 17 v1beta1 watch API endpoints
    # http://kubernetes.io/docs/api-reference/extensions/v1beta1/operations/
    # horizontalpodautoscalars and jobs has its own non-beta API endpoints so
    # we are not using the beta versions
    "/apis/extensions/v1beta1/watch/daemonsets",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/daemonsets",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/daemonsets/{name}",
    "/apis/extensions/v1beta1/watch/deployments",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/deployments",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/deployments/{name}",
    "/apis/extensions/v1beta1/watch/ingresses",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/ingresses",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/ingresses/{name}",
    "/apis/extensions/v1beta1/watch/replicasets",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/replicasets",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/replicasets/{name}",
    "/apis/extensions/v1beta1/watch/networkpolicies",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/networkpolicies",
    "/apis/extensions/v1beta1/watch/namespaces/{namespace}/networkpolicies/{name}",
    "/apis/extensions/v1beta1/watch/thirdpartyresources",
    "/apis/extensions/v1beta1/watch/thirdpartyresources/{name}"
}

TEST_KUBE_WATCH_API_ENDPOINTS_V1_AUTOSCALE = {
    # There are 3 v1 autoscaling watch API endpoints
    # http://kubernetes.io/docs/api-reference/autoscaling/v1/operations/
    "/apis/autoscaling/v1/watch/horizontalpodautoscalers",
    "/apis/autoscaling/v1/watch/namespaces/{namespace}/horizontalpodautoscalers",
    "/apis/autoscaling/v1/watch/namespaces/{namespace}/horizontalpodautoscalers/{name}"
}

TEST_KUBE_WATCH_API_ENDPOINTS_V1_BATCH = {
    # There are 3 v1 watch watch API endpoints
    # http://kubernetes.io/docs/api-reference/batch/v1/operations/
    "/apis/batch/v1/watch/jobs",
    "/apis/batch/v1/watch/namespaces/{namespace}/jobs",
    "/apis/batch/v1/watch/namespaces/{namespace}/jobs/{name}"
}
