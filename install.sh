#!/usr/bin/env bash
set -eu -o pipefail

# Usage:
#   curl ... | ENV_VAR=... sh -
#       or
#   ENV_VAR=... ./install.sh


VERSION=${VERSION:-latest}
INSTALL_CLI=${INSTALL_CLI:-1}
INSTALL_MINIO=${INSTALL_MINIO:-0}
INSTALL_MYSQL=${INSTALL_MYSQL:-0}
INSTALL_POSTGRES=${INSTALL_POSTGRES:-0}
DEFAULT_ADMIN_ROLEBINDING=${DEFAULT_ADMIN_ROLEBINDING:-0}

if [[ "$(pwd)" = "$HOME/go/src/github.com/argoproj/argo" ]]; then
    VERSION="dev"
fi

# Inspired by https://raw.githubusercontent.com/rancher/k3s/master/install.sh

GITHUB_URL=https://github.com/argoproj/argo/releases

info() {
    echo '[INFO] ' "$@"
}

if [[ ${VERSION} = 'latest' ]]; then
    VERSION=$(curl -w '%{url_effective}' -I -L -s -S ${GITHUB_URL}/latest -o /dev/null | sed -e 's|.*/||')
fi

info "Installing $VERSION"
if [[ ${INSTALL_CLI} -eq 1 ]]; then
    info "Creating installing CLI"
    curl -sL -o /usr/local/bin/argo ${GITHUB_URL}/download/v${VERSION}/argo-$(uname | tr '[A-Z]' '[a-z'])-amd64
    chmod +x /usr/local/bin/argo
fi

info "Creating argo namespace if not exists"
kubectl get ns argo || kubectl create ns argo

info "Installing base manifests"
if [[ ${VERSION} = 'dev' ]]; then
    kubectl -n argo apply -f manifests/install.yaml
else
    kubectl -n argo apply -f https://raw.githubusercontent.com/argoproj/argo/v${VERSION}/manifests/install.yaml
fi

if [[ ${DEFAULT_ADMIN_ROLEBINDING} -eq 1 ]]; then
    kubectl -n argo apply -f manifests/extras/default-admin-rolebinding.yaml
fi

if [[ ${INSTALL_MINIO} -eq 1 ]]; then
    info "Installing MinIO (on port 9000 login admin/password)"
    kubectl -n argo apply -f manifests/extras/minio
else
    info "Removing MinIO"
    kubectl -n argo delete all -l app=minio
fi

if [[ ${INSTALL_MYSQL} -eq 1 ]]; then
    info "Installing MySQL (on port 3306 login mysql/password)"
    kubectl -n argo apply -f manifests/extras/mysql
else
    info "Removing MySQL"
    kubectl -n argo delete all -l app=mysql
fi

if [[ ${INSTALL_POSTGRES} -eq 1 ]]; then
    info "Installing Postgres (on port 5432 login postgres/password)"
    kubectl -n argo apply -f manifests/extras/postgres
else
    info "Removing Postgres"
    kubectl -n argo delete all -l app=postgres
fi

info "Configuring Argo"
kubectl -n argo apply -f - <<EOF
apiVersion: v1
data:
  config: |
    artifactRepository:
      archiveLogs: true
$([[ ${INSTALL_MINIO} -eq 1 ]] && cat <<MINO
      s3:
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        accessKeySecret:
          name: my-minio-cred
          key: accesskey
        secretKeySecret:
          name: my-minio-cred
          key: secretkey
MINO
)
$([[ ${INSTALL_MYSQL} -eq 1 ]] || [[ ${INSTALL_POSTGRES} -eq 1 ]] && cat <<PERSISTENCE
    persistence:
      connectionPool:
        maxIdleConns: 100
        maxOpenConns: 0
      nodeStatusOffLoad: true
      history: true
$([[ ${INSTALL_POSTGRES} -eq 1 ]] && cat <<POSTGRES
      postgresql:
        host: postgres
        port: 5432
        database: postgres
        tableName: argo_workflows
        userNameSecret:
          name: argo-postgres-config
          key: username
        passwordSecret:
          name: argo-postgres-config
          key: password
POSTGRES
)
$([[ ${INSTALL_MYSQL} -eq 1 ]] && cat <<MYSQL
      mysql:
        host: mysql
        port: 3306
        database: argo
        tableName: argo_workflows
        userNameSecret:
          name: argo-mysql-config
          key: username
        passwordSecret:
          name: argo-mysql-config
          key: password
MYSQL
)
PERSISTENCE
)
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
EOF