FROM ubuntu:14.04

RUN apt-get update && apt-get install -y cowsay --no-install-recommends && rm -rf /var/lib/apt/lists/* 

# "cowsay" installs to /usr/games
ENV PATH $PATH:/usr/games

CMD ["cowsay"]