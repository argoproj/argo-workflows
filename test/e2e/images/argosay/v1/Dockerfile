FROM ubuntu:mantic-20230807.1

RUN apt-get update && \
    apt-get install -y apt-utils ca-certificates apt-transport-https cowsay --no-install-recommends && \
    apt-get clean \
    && rm -rf \
        /var/lib/apt/lists/* \
        /tmp/* \
        /var/tmp/* \
        /usr/share/man \
        /usr/share/doc \
        /usr/share/doc-base

# "cowsay" installs to /usr/games
ENV PATH $PATH:/usr/games

CMD ["cowsay"]
