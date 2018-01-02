FROM node:6.9.5-alpine

COPY ./tmp/api-dist /app/api
COPY ./tmp/dist /app/ui
COPY ./tmp/node_modules /app/node_modules
WORKDIR /app

EXPOSE 8001
CMD ["sh", "-c", "node api/api/main.js --uiDist /app/ui --inCluster ${IN_CLUSTER} --namespace ${ARGO_NAMESPACE} --enableWebConsole ${ENABLE_WEB_CONSOLE}"]
