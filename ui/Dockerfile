FROM node:6.9.5-alpine

RUN apk update && apk add git

WORKDIR /app

COPY ["package.json", "yarn.lock", "tsconfig.json", "./"]
RUN npm install -g yarn && yarn install --production
COPY ["api", "./api"]
COPY ["dist", "./dist"]
COPY ["src/app/models", "./src/app/models"]

EXPOSE 8001

CMD ["sh", "-c", "yarn run ts-node ./api/main.ts --inCluster ${IN_CLUSTER} --namespace ${ARGO_NAMESPACE}"]
