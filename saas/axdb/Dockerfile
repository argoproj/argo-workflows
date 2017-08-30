From %%ARGO_BASE_REGISTRY%%/argobase/axdb-base:v1.3

WORKDIR /

RUN sed -e 's/commitlog_sync_period_in_ms: 10000/commitlog_sync_period_in_ms: 1000/g' /etc/cassandra/cassandra.yaml > /tmp/ca.yaml && \
    cp /tmp/ca.yaml /etc/cassandra/cassandra.yaml

#copy seedprovider
COPY ./kubernetes-cassandra.jar /ax/axdb/kubernetes-cassandra.jar
COPY ./tools/seedprovider/image/run.sh /ax/axdb/
COPY ./tools/seedprovider/image/health.sh /ax/axdb/
COPY ./tools/seedprovider/image/flush.sh /ax/axdb/
COPY ./tools/html /ax/axdb/html

#CMD service cassandra stop && \
#    sed -e "\|/var/lib/cassandra|s|/var/lib/cassandra|$CASSANDRA_HOME|" /etc/cassandra/cassandra.yaml | sed -e 's/- class_name: .*SeedProvider$/- class_name: io.k8s.cassandra.KubernetesSeedProvider/'> /tmp/ca.yaml && \
#    cp /tmp/ca.yaml /etc/cassandra/cassandra.yaml && \
#    export CLASSPATH=/ax/axdb/kubernetes-cassandra.jar && \
#    /ax/axdb/run.sh && \
#    /axdb/bin/axdb_server $NUMBER_OF_NODES

COPY ./bin /axdb/bin
EXPOSE 8080
ENTRYPOINT ["/ax/axdb/run.sh"]
