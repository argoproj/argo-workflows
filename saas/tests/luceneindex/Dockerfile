From %%ARGO_BASE_REGISTRY%%/argobase/axdb-base:v1.1

WORKDIR /

COPY ./bin /axdbtest/bin

#copy seedprovider
CMD /axdbtest/bin/luceneindex_test http://axdb.axsys:8083/v1
