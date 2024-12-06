# most of these configuration options might not do anything yet. 
# the plan is to gradually introduce more and more functionality into Nix 
# and move away from the Makefile. 
# At the moment, only the equivalent of "make start UI=true" is supported. 
# Even then the buildFlags are not passed into Go, meaning you won't see the correct version info yet. 
# This is only intended for quick developing at the moment, gradually more functionality will be pushed here. 
rec {
  version = "latest";
  env = {
    DEFAULT_REQUEUE_TIME = "1s";
    SECURE = "false";
    ALWAYS_OFFLOAD_NODE_STATUS = "false";
    LOG_LEVEL = "debug";
    UPPERIO_DB_DEBUG = "0";
    IMAGE_NAMESPACE = "quay.io/argoproj";
    VERSION = "${version}";
    AUTH_MODE = "hybrid";
    NAMESPACED = "true";
    KUBE_NAMESPACE = "argo";
    NAMESPACE = "${env.KUBE_NAMESPACE}";
    MANAGED_NAMESPACE = "${env.KUBE_NAMESPACE}"; # same as kubeNamespace
    CTRL = "true";
    LOGS = "true"; # same as CTRL - not acted upon
    UI = "true"; # same as CTRL
    API = "true"; # same as CTRL
    UI_SECURE = "false";
    PLUGINS = "false";
  };
  controller = {
    env = {
      CTRL = "${env.CTRL}";
      ARGO_EXECUTOR_PLUGINS = "${env.PLUGINS}";
      ARGO_REMOVE_PVC_PROTECTION_FINALIZER = "true";
      ARGO_PROGRESS_PATCH_TICK_DURATION = "7s";
      DEFAULT_REQUEUE_TIME = "${env.DEFAULT_REQUEUE_TIME}";
      LEADER_ELECTION_IDENTITY = "local";
      ALWAYS_OFFLOAD_NODE_STATUS = "${env.ALWAYS_OFFLOAD_NODE_STATUS}";
      OFFLOAD_NODE_STATUS_TTL = "30s";
      WORKFLOW_GC_PERIOD = "30s";
      UPPERIO_DB_DEBUG = "${env.UPPERIO_DB_DEBUG}";
      ARCHIVED_WORKFLOW_GC_PERIOD = "30s";
    };
    args = "--executor-image ${env.IMAGE_NAMESPACE}/argoexec:${env.VERSION} --namespaced=${env.NAMESPACED} --managed-namespace=${env.MANAGED_NAMESPACE} --loglevel ${env.LOG_LEVEL}";
  };

  argoServer = {
    env = {
      UPPERIO_DB_DEBUG = "${env.UPPERIO_DB_DEBUG}";
    };
    args = "--loglevel ${env.LOG_LEVEL} server --namespaced=${env.NAMESPACED} --auth-mode ${env.AUTH_MODE} --secure=${env.SECURE} --x-frame-options=SAMEORIGIN";
  };
  ui = {
    env = {
      ARGO_UI_SECURE = "${env.UI_SECURE}";
      ARGO_SECURE = "${env.SECURE}";
    };
    args = "--cwd ui start";
  };
}
