FROM node:6.9.5

WORKDIR /app

COPY ["package.json", "yarn.lock", "tsconfig.json", "./"]
RUN npm install -g yarn && yarn install
COPY ["api", "./api"]
COPY ["dist", "./dist"]

EXPOSE 8001

CMD ["sh", "-c", "yarn run ts-node ./api/main.ts --inCluster ${IN_CLUSTER} --namespace ${NAMESPACE}"]
