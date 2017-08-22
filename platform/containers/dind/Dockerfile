# AX customized version of dind build file.
# This is based on https://github.com/docker-library/docker/blob/master/1.12/dind/Dockerfile
#
# AX docker uses Ubuntu 14.04. Use Ubuntu commands.

FROM %%ARGO_BASE_REGISTRY%%/argobase/docker:1.12.6

# https://github.com/docker/docker/blob/master/project/PACKAGERS.md#runtime-dependencies
RUN apt-get update && apt-get install -y \
        btrfs-tools \
        e2fsprogs \
        iptables \
        xfsprogs \
        xz-utils \
        wget

# TODO aufs-tools

# set up subuid/subgid so that "--userns-remap=default" works out-of-the-box
RUN set -x \
    && addgroup --system dockremap \
    && adduser --system --ingroup dockremap dockremap \
    && echo 'dockremap:165536:65536' >> /etc/subuid \
    && echo 'dockremap:165536:65536' >> /etc/subgid

RUN wget "https://raw.githubusercontent.com/docker/docker/v${DOCKER_VERSION}/hack/dind" -O /usr/local/bin/dind \
    && chmod +x /usr/local/bin/dind

COPY dockerd-entrypoint.sh /usr/local/bin/

VOLUME /var/lib/docker
EXPOSE 2375

ENTRYPOINT ["dockerd-entrypoint.sh"]
CMD []
