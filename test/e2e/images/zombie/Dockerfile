#syntax=docker/dockerfile:1.2

# https://vitux.com/how-to-create-a-dummy-zombie-process-in-ubuntu/
FROM golang:1.18
ADD zombie.c /
RUN cc /zombie.c -o /zombie
ENTRYPOINT [ "/zombie" ]