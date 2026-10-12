# Build the controller binary for the dedicated local cluster architecture first.
FROM alpine:3.22
COPY --chmod=0555 controller /controller
USER 65534:65534
ENTRYPOINT ["/controller"]
