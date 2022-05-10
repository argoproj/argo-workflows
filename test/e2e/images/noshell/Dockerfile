# This image is for testing the killing of scratch containers.

# use this image as (a) probably already downloaded and (b) has build-essentials on it, which contains cc
FROM golang:1.18 as builder

ADD *.c /

RUN cc -static /sleep.c -o /sleep
RUN cc -static /oom.c -o /oom

FROM golang:1.18 as shell

COPY --from=builder /sleep /sleep
COPY --from=builder /oom /oom
