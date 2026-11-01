FROM hashicorp/vault:1.19.0 AS vault
FROM postgres:16-alpine
COPY --from=vault /bin/vault /usr/local/bin/vault
COPY query.sh /demo/query.sh
ENTRYPOINT []
