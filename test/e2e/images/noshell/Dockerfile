# This image is for testing the killing of scratch containers.

# use this image as (a) probably already downloaded and (b) has build-essentials on it, which contains cc
FROM golang:1.18 as builder

ADD *.c /

RUN cc -static /sleep.c -o /sleep
RUN cc -static /kill.c -o /kill

FROM scratch

COPY --from=builder /sleep /sleep
COPY --from=builder /kill /kill
