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

FROM node:6.9.5-alpine

WORKDIR /app

COPY --from=0 /src/api-dist ./api
COPY --from=0 /src/dist ./ui
COPY --from=0 /src/node_modules ./node_modules

EXPOSE 8001
CMD ["sh", "-c", "node api/api/main.js --uiDist /app/ui --inCluster ${IN_CLUSTER} --namespace ${ARGO_NAMESPACE}"]
