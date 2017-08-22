
base_registry=${ARGO_BASE_REGISTRY:-docker.io}
dev_registry=${ARGO_DEV_REGISTRY}

cat Dockerfile | sed "s#%%ARGO_BASE_REGISTRY%%#${base_registry}#g" | docker build -t ${dev_registry}/${NAMESPACE}/luceneindex_test:${VERSION} -
docker push ${dev_registry}/${NAMESPACE}/luceneindex_test:${VERSION}
