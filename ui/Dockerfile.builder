FROM node:6.9.5

RUN apt-get update && apt-get install -y apt-transport-https && \
  curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && \
  echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
  apt-get update && apt-get install yarn


WORKDIR /src
ADD ["package.json", "yarn.lock", "./"]

RUN yarn install

ADD [".", "."]

RUN yarn build && yarn run api:build && yarn install --production
