FROM %%ARGO_BASE_REGISTRY%%/argobase/alpine:3.5

RUN apk add --no-cache --update bash socat
COPY start.sh /start.sh
COPY redirect.sh /redirect.sh
CMD /start.sh
